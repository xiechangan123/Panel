package data

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/leonelquinteros/gotext"
	"github.com/samber/lo"
	"github.com/spf13/cast"
	"resty.dev/v3"

	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/types"
)

// 宝塔项目模块与 AcePanel 项目类型的对应关系，nodejs 等模块名同时用于拼接接口路径
var baotaProjectModules = map[string]types.ProjectType{
	"nodejs": types.ProjectTypeNodejs,
	"python": types.ProjectTypePython,
	"go":     types.ProjectTypeGo,
	"net":    types.ProjectTypeDotnet,
	"java":   types.ProjectTypeJava,
	"other":  types.ProjectTypeGeneral,
}

type baotaAdapter struct {
	t *gotext.Locale
	*migrationClient
}

// newBaotaClient 宝塔以表单提交，认证为 md5(时间戳 + md5(接口密钥))
func newBaotaClient(conn *request.ToolboxMigrationConnection) *migrationClient {
	return &migrationClient{
		url:  conn.URL,
		form: true,
		sign: func(req *resty.Request) {
			timestamp := strconv.FormatInt(time.Now().Unix(), 10)
			secret := md5.Sum([]byte(conn.APIKey))
			token := md5.Sum([]byte(timestamp + hex.EncodeToString(secret[:])))
			values := map[string]string{"request_time": timestamp, "request_token": hex.EncodeToString(token[:])}
			if req.Method == http.MethodGet {
				req.SetQueryParams(values)
			} else {
				req.SetFormData(values)
			}
		},
		// 宝塔失败响应为 {status: false, msg: "..."}，部分接口的业务数据也含 status 字段，需同时判断 msg
		unwrap: func(body []byte) (any, error) {
			var raw any
			if err := json.Unmarshal(body, &raw); err != nil {
				return nil, fmt.Errorf("invalid response: %w", err)
			}
			result := cast.ToStringMap(raw)
			if status, ok := result["status"].(bool); ok && !status && result["msg"] != nil {
				return nil, errors.New(cast.ToString(result["msg"]))
			}
			if data, ok := result["data"]; ok {
				return data, nil
			}
			return raw, nil
		},
		downloadPath:  "/download",
		downloadParam: "filename",
	}
}

func (a *baotaAdapter) Probe(ctx context.Context) (*types.MigrationSource, error) {
	data, err := a.call(ctx, http.MethodPost, "/system?action=GetSystemTotal", nil)
	if err != nil {
		return nil, err
	}
	return &types.MigrationSource{Panel: "baota", Version: cast.ToString(cast.ToStringMap(data)["version"])}, nil
}

func (a *baotaAdapter) Items(ctx context.Context) ([]types.MigrationItem, error) {
	websites, err := a.websiteItems(ctx)
	if err != nil {
		return nil, err
	}
	databases := a.databaseItems(ctx)
	// 宝塔一键部署的站点与数据库同名，据此建立依赖关系
	for i := range websites {
		if index := slices.IndexFunc(databases, func(database types.MigrationItem) bool {
			return database.Name == websites[i].Name
		}); index >= 0 {
			websites[i].DependsOn = append(websites[i].DependsOn, databases[index].Key)
		}
	}
	return slices.Concat(websites, databases, a.projectItems(ctx)), nil
}

// websiteItems 列出静态、PHP 与反代网站
func (a *baotaAdapter) websiteItems(ctx context.Context) ([]types.MigrationItem, error) {
	data, err := a.call(ctx, http.MethodPost, "/data?action=getData", map[string]any{
		"table": "sites", "p": 1, "limit": 10000,
	})
	if err != nil {
		return nil, fmt.Errorf("load websites: %w", err)
	}

	items := make([]types.MigrationItem, 0)
	for _, row := range a.rows(data) {
		id, name := cast.ToString(row["id"]), cast.ToString(row["name"])
		if id == "" || name == "" {
			continue
		}
		// PHP / WP2 为 PHP 站点，HTML 为静态站点，Proxy 为反代站点，其余是项目
		subtype := ""
		switch strings.ToLower(cast.ToString(row["project_type"])) {
		case "php", "wp2":
			subtype = lo.Ternary(cast.ToString(row["php_version"]) == "静态", "static", "php")
		case "html":
			subtype = "static"
		case "proxy":
			subtype = "proxy"
		default:
			continue
		}
		items = append(items, types.MigrationItem{
			Key: biz.MigrationItemKey("website", id), Type: "website", Subtype: subtype, Name: name,
			Status:     lo.Ternary(cast.ToString(row["status"]) == "1", "running", "stopped"),
			Size:       cast.ToInt64(cast.ToStringMap(row["quota"])["used"]),
			TargetName: name, SourceID: id, SourcePath: cast.ToString(row["path"]),
		})
	}
	return items, nil
}

// databaseItems 列出本机 MySQL 数据库
func (a *baotaAdapter) databaseItems(ctx context.Context) []types.MigrationItem {
	data, err := a.call(ctx, http.MethodPost, "/data?action=getData", map[string]any{
		"table": "databases", "p": 1, "limit": 10000,
	})
	if err != nil {
		return nil
	}
	// 宝塔的 MySQL 可能实际是 MariaDB，版本文件里能区分
	subtype := "mysql"
	if version, versionErr := a.fileContent(ctx, "/www/server/mysql/version.pl"); versionErr == nil &&
		strings.Contains(strings.ToLower(version), "mariadb") {
		subtype = "mariadb"
	}

	items := make([]types.MigrationItem, 0)
	for _, row := range a.rows(data) {
		id, name := cast.ToString(row["id"]), cast.ToString(row["name"])
		// db_type 与 sid 非 0 表示远程数据库，不在迁移范围内
		if id == "" || name == "" || !strings.EqualFold(cast.ToString(row["type"]), "mysql") ||
			cast.ToInt(row["db_type"]) != 0 || cast.ToInt(row["sid"]) != 0 {
			continue
		}
		items = append(items, types.MigrationItem{
			Key: biz.MigrationItemKey("database", id), Type: "database", Subtype: subtype, Name: name, Status: "running",
			Size: cast.ToInt64(cast.ToStringMap(row["quota"])["used"]), TargetName: name, SourceID: id,
		})
	}
	return items
}

// projectItems 列出各语言项目
func (a *baotaAdapter) projectItems(ctx context.Context) []types.MigrationItem {
	items := make([]types.MigrationItem, 0)
	for module, projectType := range baotaProjectModules {
		data, err := a.call(ctx, http.MethodPost, a.projectPath(module, "list"), map[string]any{"p": 1, "limit": 10000})
		if err != nil {
			continue
		}
		for _, row := range a.rows(data) {
			id, name := cast.ToString(row["id"]), cast.ToString(row["name"])
			if id == "" || name == "" {
				continue
			}
			config := cast.ToStringMap(row["project_config"])
			item := types.MigrationItem{
				Key: biz.MigrationItemKey("project", module+":"+id), Type: "project", Subtype: string(projectType), Name: name,
				Status:     lo.Ternary(cast.ToBool(row["run"]), "running", "stopped"),
				TargetName: name, SourceID: id, SourcePath: cast.ToString(row["path"]),
				SourceGroup: module, Version: a.projectVersion(config, module),
			}
			// Tomcat 项目的部署形态与 AcePanel 的 systemd 项目不兼容
			if module == "java" && !strings.EqualFold(cast.ToString(config["java_type"]), "springboot") &&
				cast.ToString(config["java_type"]) != "" {
				item.Blockers = []string{a.t.Get("Tomcat projects are not supported for automatic migration")}
			}
			items = append(items, item)
		}
	}
	return items
}

// projectPath 拼接项目接口路径，python 模块的方法名与其他模块不同
func (a *baotaAdapter) projectPath(module, action string) string {
	name := map[string]string{"list": "get_project_list", "start": "start_project", "stop": "stop_project"}[action]
	if module == "python" && action == "list" {
		name = "GetProjectList"
	}
	return fmt.Sprintf("/project/%s/%s/%s", module, name, module)
}

// projectVersion 取项目配置中的运行时版本
func (a *baotaAdapter) projectVersion(config map[string]any, module string) string {
	switch module {
	case "nodejs":
		return cast.ToString(config["nodejs_version"])
	case "python":
		return lo.CoalesceOrEmpty(cast.ToString(config["version"]), cast.ToString(config["python_bin"]))
	case "java":
		return lo.CoalesceOrEmpty(cast.ToString(config["project_jdk"]), cast.ToString(config["jdk_path"]))
	case "net":
		return cast.ToString(config["dotnet_version"])
	default:
		return ""
	}
}

func (a *baotaAdapter) Detail(ctx context.Context, item types.MigrationItem) (*types.MigrationDetail, error) {
	detail := &types.MigrationDetail{Item: item}
	var err error
	switch item.Type {
	case "website":
		detail.Website, err = a.websiteDetail(ctx, item)
	case "database":
		detail.Database, err = a.databaseDetail(ctx, item)
	case "project":
		detail.Project, err = a.projectDetail(ctx, item)
	default:
		err = errors.New(a.t.Get("unsupported migration resource type: %s", item.Type))
	}
	return detail, err
}

func (a *baotaAdapter) websiteDetail(ctx context.Context, item types.MigrationItem) (*types.MigrationWebsite, error) {
	website := &types.MigrationWebsite{
		Type: item.Subtype, Path: item.SourcePath, Root: item.SourcePath,
		Domains: []string{item.Name}, Listens: []string{"80"},
		Index: []string{"index.php", "index.html", "index.htm"}, Enabled: item.Status == "running",
	}

	// 备注与到期时间来自站点表
	if data, err := a.call(ctx, http.MethodPost, "/data?action=getData", map[string]any{
		"table": "sites", "p": 1, "limit": 10000, "search": item.Name,
	}); err == nil {
		if row, ok := lo.Find(a.rows(data), func(row map[string]any) bool { return cast.ToString(row["id"]) == item.SourceID }); ok {
			website.Remark = cast.ToString(row["ps"])
			website.ExpireAt = a.parseDate(cast.ToString(row["edate"]))
		}
	}

	if domains, listens := a.domains(ctx, item.SourceID); len(domains) > 0 {
		website.Domains, website.Listens = domains, listens
	}

	if data, err := a.call(ctx, http.MethodPost, "/site?action=GetSitePHPVersion", map[string]any{"siteName": item.Name}); err == nil {
		website.PHP = a.phpVersion(cast.ToString(cast.ToStringMap(data)["phpversion"]))
	}
	if website.PHP > 0 {
		website.Type = "php"
	}

	// 运行目录相对站点目录，同时带出 open_basedir 状态
	if data, err := a.call(ctx, http.MethodPost, "/site?action=GetDirUserINI", map[string]any{
		"id": item.SourceID, "path": website.Path,
	}); err == nil {
		info := cast.ToStringMap(data)
		website.OpenBasedir = cast.ToBool(info["userini"])
		if runPath := cast.ToString(cast.ToStringMap(info["runPath"])["runPath"]); runPath != "" && runPath != "/" {
			website.Root = filepath.Join(website.Path, strings.TrimPrefix(runPath, "/"))
		}
	}
	if data, err := a.call(ctx, http.MethodPost, "/site?action=GetIndex", map[string]any{"id": item.SourceID}); err == nil {
		if index := strings.FieldsFunc(cast.ToString(data), func(char rune) bool {
			return char == ',' || char == ' ' || char == '\t'
		}); len(index) > 0 {
			website.Index = index
		}
	}
	website.Rewrite = a.rewrite(ctx, item)
	website.Proxies = a.proxies(ctx, item.Name)
	if website.Type == "static" && slices.ContainsFunc(website.Proxies, func(proxy types.MigrationProxy) bool { return proxy.Location == "/" }) {
		website.Type = "proxy"
	}
	website.Redirects = a.redirects(ctx, item.Name)
	a.applySSL(ctx, item.Name, website)
	return website, nil
}

// domains 读取站点绑定的域名与端口
func (a *baotaAdapter) domains(ctx context.Context, siteID string) ([]string, []string) {
	data, err := a.call(ctx, http.MethodPost, "/data?action=getData", map[string]any{
		"table": "domain", "p": 1, "limit": 10000, "search": siteID,
	})
	if err != nil {
		return nil, nil
	}
	domains, listens := make([]string, 0), make([]string, 0)
	for _, row := range a.rows(data) {
		if cast.ToString(row["pid"]) != siteID {
			continue
		}
		if domain := cast.ToString(row["name"]); domain != "" && !slices.Contains(domains, domain) {
			domains = append(domains, domain)
		}
		if port := cast.ToString(row["port"]); port != "" && !slices.Contains(listens, port) {
			listens = append(listens, port)
		}
	}
	return domains, listens
}

// rewrite 读取站点伪静态规则内容
func (a *baotaAdapter) rewrite(ctx context.Context, item types.MigrationItem) string {
	data, err := a.call(ctx, http.MethodPost, "/site?action=GetRewriteLists", map[string]any{
		"site_ids": "[" + item.SourceID + "]", "site_type": item.Subtype,
	})
	if err != nil {
		return ""
	}
	for _, row := range a.rows(cast.ToStringMap(data)["site_rewrites"]) {
		file := cast.ToString(row["file"])
		if cast.ToString(row["id"]) != item.SourceID || file == "" {
			continue
		}
		content, contentErr := a.fileContent(ctx, file)
		if contentErr != nil {
			return ""
		}
		return content
	}
	return ""
}

// proxies 读取站点反向代理规则
func (a *baotaAdapter) proxies(ctx context.Context, name string) []types.MigrationProxy {
	data, err := a.call(ctx, http.MethodPost, "/site?action=GetProxyList", map[string]any{"sitename": name})
	if err != nil {
		return nil
	}
	proxies := make([]types.MigrationProxy, 0)
	for _, row := range a.rows(data) {
		// type 为 0 表示该代理规则已停用
		pass := cast.ToString(row["proxysite"])
		if pass == "" || cast.ToInt(row["type"]) != 1 {
			continue
		}
		replaces := lo.SliceToMap(a.rows(row["subfilter"]), func(filter map[string]any) (string, string) {
			return cast.ToString(filter["sub1"]), cast.ToString(filter["sub2"])
		})
		delete(replaces, "")
		proxies = append(proxies, types.MigrationProxy{
			Location: lo.CoalesceOrEmpty(cast.ToString(row["proxydir"]), "/"),
			Pass:     pass, Host: cast.ToString(row["todomain"]), Replaces: replaces,
		})
	}
	return proxies
}

// redirects 读取站点重定向规则
func (a *baotaAdapter) redirects(ctx context.Context, name string) []types.MigrationRedirect {
	data, err := a.call(ctx, http.MethodPost, "/site?action=GetRedirectList", map[string]any{"sitename": name, "errorpage": "0"})
	if err != nil {
		return nil
	}
	redirects := make([]types.MigrationRedirect, 0)
	for _, row := range a.rows(data) {
		target := cast.ToString(row["tourl"])
		if target == "" || cast.ToString(row["type"]) == "0" {
			continue
		}
		redirect := types.MigrationRedirect{
			Type: "url", From: cast.ToString(row["redirectpath"]), To: target,
			KeepURI: cast.ToBool(row["holdpath"]), StatusCode: cast.ToInt(row["redirecttype"]),
		}
		if cast.ToString(row["domainorpath"]) == "domain" {
			redirect.Type = "host"
			redirect.From = strings.Join(cast.ToStringSlice(row["redirectdomain"]), " ")
		}
		if redirect.StatusCode == 0 {
			redirect.StatusCode = http.StatusMovedPermanently
		}
		redirects = append(redirects, redirect)
	}
	return redirects
}

// applySSL 读取证书与 TLS 配置
func (a *baotaAdapter) applySSL(ctx context.Context, name string, website *types.MigrationWebsite) {
	data, err := a.call(ctx, http.MethodPost, "/site?action=GetSSL", map[string]any{"siteName": name})
	if err != nil {
		return
	}
	ssl := cast.ToStringMap(data)
	website.SSLCert = cast.ToString(ssl["csr"])
	website.SSLKey = cast.ToString(ssl["key"])
	website.HTTPRedirect = cast.ToBool(ssl["httpTohttps"])
	website.SSL = cast.ToBool(ssl["status"]) && website.SSLCert != "" && website.SSLKey != ""
	if !website.SSL {
		return
	}
	protocols := cast.ToStringMap(ssl["tls_versions"])
	for _, protocol := range []string{"TLSv1", "TLSv1.1", "TLSv1.2", "TLSv1.3"} {
		if cast.ToBool(protocols[protocol]) {
			website.SSLProtocols = append(website.SSLProtocols, protocol)
		}
	}
	port := "443"
	if data, err = a.call(ctx, http.MethodPost, "/data?action=get_https_port", map[string]any{"siteName": name}); err == nil {
		if value := cast.ToString(data); value != "" && value != "0" {
			port = value
		}
	}
	if !slices.Contains(website.Listens, port) {
		website.Listens = append(website.Listens, port)
	}
	website.SSLListens = append(website.SSLListens, port)
}

func (a *baotaAdapter) databaseDetail(ctx context.Context, item types.MigrationItem) (*types.MigrationDatabase, error) {
	database := &types.MigrationDatabase{Type: item.Subtype, Name: item.Name, Host: "localhost"}
	data, err := a.call(ctx, http.MethodPost, "/data?action=getData", map[string]any{
		"table": "databases", "p": 1, "limit": 10000, "search": item.Name,
	})
	if err != nil {
		return nil, err
	}
	row, ok := lo.Find(a.rows(data), func(row map[string]any) bool { return cast.ToString(row["id"]) == item.SourceID })
	if !ok {
		return nil, errors.New(a.t.Get("resource no longer exists on the source server"))
	}
	database.Username = cast.ToString(row["username"])
	database.Password = cast.ToString(row["password"])
	database.Host = lo.CoalesceOrEmpty(cast.ToString(row["accept"]), "localhost")
	if version, versionErr := a.fileContent(ctx, "/www/server/mysql/version.pl"); versionErr == nil {
		database.Version = strings.TrimSpace(version)
	}
	return database, nil
}

func (a *baotaAdapter) projectDetail(ctx context.Context, item types.MigrationItem) (*types.MigrationProject, error) {
	data, err := a.call(ctx, http.MethodPost, a.projectPath(item.SourceGroup, "list"), map[string]any{"p": 1, "limit": 10000})
	if err != nil {
		return nil, err
	}
	row, ok := lo.Find(a.rows(data), func(row map[string]any) bool { return cast.ToString(row["id"]) == item.SourceID })
	if !ok {
		return nil, errors.New(a.t.Get("resource no longer exists on the source server"))
	}

	config := cast.ToStringMap(row["project_config"])
	path := a.projectRoot(row, config, item.SourceGroup)
	project := &types.MigrationProject{
		Type: types.ProjectType(item.Subtype), Version: item.Version, Path: path, WorkingDir: path,
		ExecStart: a.projectExecStart(ctx, config, item.SourceGroup, path),
		User:      lo.CoalesceOrEmpty(cast.ToString(config["run_user"]), "www"),
		Port:      cast.ToUint(config["port"]), Running: item.Status == "running",
		Enabled: cast.ToBool(config[lo.Ternary(item.SourceGroup == "python", "auto_run", "is_power_on")]),
	}
	project.Domains, project.Listens = a.domains(ctx, item.SourceID)
	project.Environments = a.projectEnvironments(ctx, config)
	if project.ExecStart == "" {
		return nil, errors.New(a.t.Get("the source project start command is missing"))
	}
	return project, nil
}

// projectRoot 各语言项目的代码目录字段不同
func (a *baotaAdapter) projectRoot(row, config map[string]any, module string) string {
	path := cast.ToString(row["path"])
	switch module {
	case "nodejs":
		return lo.CoalesceOrEmpty(cast.ToString(config["project_cwd"]), path)
	case "python":
		return lo.CoalesceOrEmpty(cast.ToString(config["path"]), path)
	case "go", "net":
		return lo.CoalesceOrEmpty(cast.ToString(config["project_path"]), path)
	case "java":
		return lo.CoalesceOrEmpty(cast.ToString(config["jar_path"]), filepath.Dir(cast.ToString(config["project_jar"])), path)
	case "other":
		return lo.CoalesceOrEmpty(filepath.Dir(cast.ToString(config["project_exe"])), path)
	default:
		return path
	}
}

// projectExecStart 还原项目启动命令
func (a *baotaAdapter) projectExecStart(ctx context.Context, config map[string]any, module, path string) string {
	if command := cast.ToString(config["project_cmd"]); command != "" {
		return command
	}
	switch module {
	case "nodejs":
		return a.nodejsExecStart(ctx, cast.ToString(config["project_script"]), path)
	case "python":
		return a.pythonExecStart(config, path)
	default:
		return ""
	}
}

// nodejsExecStart Node.js 项目的启动脚本可能是 package.json 中的 script 名或入口文件
func (a *baotaAdapter) nodejsExecStart(ctx context.Context, script, path string) string {
	if script == "" {
		return ""
	}
	if content, err := a.fileContent(ctx, filepath.Join(path, "package.json")); err == nil {
		var packageInfo struct {
			Scripts map[string]string `json:"scripts"`
		}
		if json.Unmarshal([]byte(content), &packageInfo) == nil {
			if _, ok := packageInfo.Scripts[script]; ok {
				return "npm run " + script
			}
		}
	}
	if fields := strings.Fields(script); len(fields) > 0 &&
		slices.Contains([]string{".js", ".cjs", ".mjs"}, strings.ToLower(filepath.Ext(fields[0]))) {
		return "node " + script
	}
	return script
}

// pythonExecStart Python 项目按部署方式生成启动命令
func (a *baotaAdapter) pythonExecStart(config map[string]any, path string) string {
	entry := cast.ToString(config["rfile"])
	switch strings.ToLower(cast.ToString(config["stype"])) {
	case "python":
		return strings.TrimSpace("python -u " + entry + " " + cast.ToString(config["parm"]))
	case "uwsgi":
		return "uwsgi --ini " + filepath.Join(path, "uwsgi.ini")
	case "gunicorn":
		module := strings.ReplaceAll(strings.TrimSuffix(strings.TrimPrefix(entry, path+"/"), ".py"), "/", ".")
		return "gunicorn -c " + filepath.Join(path, "gunicorn_conf.py") + " " + module + ":" + cast.ToString(config["call_app"])
	default:
		return ""
	}
}

// projectEnvironments 合并项目配置与环境变量文件中的变量
func (a *baotaAdapter) projectEnvironments(ctx context.Context, config map[string]any) []types.KV {
	values := a.environments(cast.ToString(config["env_list"]))
	if envFile := cast.ToString(config["env_file"]); envFile != "" {
		if content, err := a.fileContent(ctx, envFile); err == nil {
			values = append(values, a.environments(content)...)
		}
	}
	return values
}

func (a *baotaAdapter) SetRunning(ctx context.Context, item types.MigrationItem, running bool) error {
	action := lo.Ternary(running, "start", "stop")
	switch item.Type {
	case "website":
		path := lo.Ternary(running, "/site?action=SiteStart", "/site?action=SiteStop")
		_, err := a.call(ctx, http.MethodPost, path, map[string]any{"id": item.SourceID, "name": item.Name})
		return err
	case "project":
		_, err := a.call(ctx, http.MethodPost, a.projectPath(item.SourceGroup, action), map[string]any{
			"project_name": item.Name, "id": item.SourceID,
		})
		return err
	default:
		return nil
	}
}

func (a *baotaAdapter) Backup(ctx context.Context, detail *types.MigrationDetail) (string, error) {
	item := detail.Item
	switch item.Type {
	case "website", "database":
		return a.panelBackup(ctx, item)
	case "project":
		return a.archive(ctx, item.Name, detail.Project.Path)
	default:
		return "", errors.New(a.t.Get("unsupported migration resource type: %s", item.Type))
	}
}

// panelBackup 触发宝塔自带备份并等待新备份文件生成
func (a *baotaAdapter) panelBackup(ctx context.Context, item types.MigrationItem) (string, error) {
	backupType := lo.Ternary(item.Type == "database", "1", "0")
	existing, err := a.backupFiles(ctx, item.SourceID, backupType)
	if err != nil {
		return "", err
	}
	path := lo.Ternary(item.Type == "database", "/database?action=ToBackup", "/site?action=ToBackup")
	if _, err = a.call(ctx, http.MethodPost, path, map[string]any{"id": item.SourceID}); err != nil {
		return "", err
	}

	// 备份为后台任务，轮询备份表直到出现新文件
	for range 600 {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(time.Second):
		}
		current, listErr := a.backupFiles(ctx, item.SourceID, backupType)
		if listErr != nil {
			continue
		}
		for file := range current {
			if !existing[file] {
				return file, nil
			}
		}
	}
	return "", errors.New(a.t.Get("the source backup did not finish in time"))
}

// backupFiles 列出指定资源已有的备份文件
func (a *baotaAdapter) backupFiles(ctx context.Context, sourceID, backupType string) (map[string]bool, error) {
	data, err := a.call(ctx, http.MethodPost, "/data?action=getData", map[string]any{
		"table": "backup", "p": 1, "limit": 100, "search": sourceID, "type": backupType,
	})
	if err != nil {
		return nil, err
	}
	files := make(map[string]bool)
	for _, row := range a.rows(data) {
		if cast.ToString(row["pid"]) != sourceID {
			continue
		}
		// filename 可能是「本地路径|云存储路径」的组合
		if file := cast.ToString(row["local"]); file != "" {
			files[file] = true
		} else if file = strings.Split(cast.ToString(row["filename"]), "|")[0]; file != "" {
			files[file] = true
		}
	}
	return files, nil
}

// archive 打包来源目录为 tar.gz
func (a *baotaAdapter) archive(ctx context.Context, name, path string) (string, error) {
	if path == "" || path == "/" {
		return "", errors.New(a.t.Get("the source path is empty"))
	}
	target := filepath.Join("/www/backup", "acepanel-migration", fmt.Sprintf("%s-%d.tar.gz", a.safeName(name), time.Now().UnixNano()))
	data, err := a.call(ctx, http.MethodPost, "/files?action=ZipAndDownload", map[string]any{
		"path": filepath.Dir(path), "sfile": filepath.Base(path), "dfile": target, "z_type": "tar.gz",
	})
	if err != nil {
		return "", err
	}
	if taskID := cast.ToString(cast.ToStringMap(data)["task_id"]); taskID != "" {
		if err = a.waitTask(ctx, taskID); err != nil {
			return "", err
		}
	}
	return target, nil
}

// waitTask 等待宝塔后台任务完成
func (a *baotaAdapter) waitTask(ctx context.Context, taskID string) error {
	for range 3600 {
		data, err := a.call(ctx, http.MethodPost, "/task?action=get_task_lists", map[string]any{"task_id": taskID})
		if err == nil {
			if list := a.rows(data); len(list) > 0 {
				switch cast.ToInt(list[0]["status"]) {
				case 1:
					return nil
				case 0, -1: // 排队中或执行中
				default:
					return errors.New(a.t.Get("the source task failed"))
				}
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
	return errors.New(a.t.Get("the source backup did not finish in time"))
}

func (a *baotaAdapter) Download(ctx context.Context, remote, local string) error {
	return a.download(ctx, remote, local)
}

// fileContent 读取来源服务器上的文件内容
func (a *baotaAdapter) fileContent(ctx context.Context, path string) (string, error) {
	data, err := a.call(ctx, http.MethodPost, "/files?action=GetFileBody", map[string]any{"path": path})
	if err != nil {
		return "", err
	}
	return cast.ToString(data), nil
}

// parseDate 解析宝塔的日期字段，0000-00-00 表示未设置
func (a *baotaAdapter) parseDate(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" || value == "0000-00-00" {
		return nil
	}
	parsed, err := time.ParseInLocation(time.DateOnly, value, time.Local)
	if err != nil {
		return nil
	}
	return &parsed
}

// environments 把 KEY=VALUE 文本解析为环境变量
func (a *baotaAdapter) environments(content string) []types.KV {
	return types.SliceToKV(lo.FilterMap(strings.Split(content, "\n"), func(line string, _ int) (string, bool) {
		line = strings.TrimSpace(line)
		return line, line != "" && !strings.HasPrefix(line, "#") && strings.Contains(line, "=")
	}))
}

// safeName 过滤出可安全用于文件名的字符
func (a *baotaAdapter) safeName(name string) string {
	var result strings.Builder
	for _, char := range name {
		switch {
		case char >= 'a' && char <= 'z', char >= 'A' && char <= 'Z', char >= '0' && char <= '9', char == '-', char == '_', char == '.':
			result.WriteRune(char)
		default:
			result.WriteRune('_')
		}
	}
	if result.Len() == 0 {
		return "resource"
	}
	return result.String()
}
