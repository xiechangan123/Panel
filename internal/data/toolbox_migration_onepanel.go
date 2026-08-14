package data

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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

type onePanelAdapter struct {
	t *gotext.Locale
	*migrationClient

	v1    bool // 来源面板为 1Panel v1
	known bool // 大版本已确定，不再回落重试
}

// resolve 拼接接口路径，v1 与 v2 的前缀不同，备份接口在 v1 挂在 /settings 下
func (a *onePanelAdapter) resolve(path string) string {
	if !a.v1 {
		return "/api/v2" + path
	}
	if rest, ok := strings.CutPrefix(path, "/backups"); ok {
		return "/api/v1/settings/backup" + rest
	}
	return "/api/v1" + path
}

// tryVersions 执行请求，大版本未确定时先按 v2 再按 v1 尝试
// 两版都失败则上报 v2 的错误，避免把 v1 的路由缺失当成真正的失败原因
func (a *onePanelAdapter) tryVersions(run func() error) error {
	err := run()
	if err == nil {
		a.known = true
		return nil
	}
	if a.known {
		return err
	}

	a.v1 = true
	if retry := run(); retry == nil {
		a.known = true
		return nil
	}
	a.v1 = false

	return err
}

// api 调用 1Panel 接口
func (a *onePanelAdapter) api(ctx context.Context, method, path string, body map[string]any) (any, error) {
	var data any
	err := a.tryVersions(func() error {
		var err error
		data, err = a.call(ctx, method, a.resolve(path), body)
		return err
	})

	return data, err
}

// newOnePanelClient 1Panel 以 JSON 提交，认证为请求头中的 md5("1panel" + 接口密钥 + 时间戳)
func newOnePanelClient(conn *request.ToolboxMigrationConnection) *migrationClient {
	return &migrationClient{
		url: conn.URL,
		sign: func(req *resty.Request) {
			timestamp := strconv.FormatInt(time.Now().Unix(), 10)
			token := md5.Sum([]byte("1panel" + conn.APIKey + timestamp))
			req.SetHeader("1Panel-Timestamp", timestamp).SetHeader("1Panel-Token", hex.EncodeToString(token[:]))
		},
		unwrap: func(body []byte) (any, error) {
			var result struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
				Data    any    `json:"data"`
			}
			if err := json.Unmarshal(body, &result); err != nil {
				return nil, fmt.Errorf("invalid response: %w", err)
			}
			if result.Code != http.StatusOK {
				return nil, errors.New(result.Message)
			}
			return result.Data, nil
		},
		downloadParam: "path",
	}
}

func (a *onePanelAdapter) Probe(ctx context.Context) (*types.MigrationSource, error) {
	data, err := a.api(ctx, http.MethodPost, "/settings/search", nil)
	if err != nil {
		return nil, err
	}
	return &types.MigrationSource{Panel: "onepanel", Version: cast.ToString(cast.ToStringMap(data)["systemVersion"])}, nil
}

func (a *onePanelAdapter) Items(ctx context.Context) ([]types.MigrationItem, error) {
	websites, err := a.websiteItems(ctx)
	if err != nil {
		return nil, err
	}
	return append(websites, a.databaseItems(ctx)...), nil
}

// websiteItems 列出静态、反代与 PHP 运行环境网站
func (a *onePanelAdapter) websiteItems(ctx context.Context) ([]types.MigrationItem, error) {
	data, err := a.api(ctx, http.MethodPost, "/websites/search", map[string]any{
		"page": 1, "pageSize": 10000, "name": "", "orderBy": "createdAt", "order": "descending",
	})
	if err != nil {
		return nil, fmt.Errorf("load websites: %w", err)
	}

	items := make([]types.MigrationItem, 0)
	for _, row := range a.rows(cast.ToStringMap(data)["items"]) {
		id := cast.ToString(row["id"])
		name := lo.CoalesceOrEmpty(cast.ToString(row["alias"]), cast.ToString(row["primaryDomain"]))
		if id == "" || name == "" {
			continue
		}
		item := types.MigrationItem{
			Key: biz.MigrationItemKey("website", id), Type: "website", Name: name,
			Status:     lo.Ternary(strings.EqualFold(cast.ToString(row["status"]), "running"), "running", "stopped"),
			TargetName: name, SourceID: id, SourcePath: cast.ToString(row["sitePath"]),
		}
		// PHP 站点跑在 1Panel 运行环境容器里，其余运行环境与应用商店站点依赖容器编排，无法迁移到 systemd 形态
		switch typ := strings.ToLower(cast.ToString(row["type"])); typ {
		case "static":
			item.Subtype = "static"
		case "proxy":
			item.Subtype = "proxy"
		case "runtime":
			item.Subtype = "php"
			if !strings.EqualFold(cast.ToString(row["runtimeType"]), "php") {
				item.Blockers = []string{a.t.Get("only PHP runtime websites can be migrated: %s", cast.ToString(row["runtimeType"]))}
			}
		default:
			item.Subtype = typ
			item.Blockers = []string{a.t.Get("this 1Panel website type is not supported for automatic migration: %s", typ)}
		}
		items = append(items, item)
	}
	return items, nil
}

// databaseItems 列出本机 MySQL / MariaDB / PostgreSQL 数据库
func (a *onePanelAdapter) databaseItems(ctx context.Context) []types.MigrationItem {
	items := make([]types.MigrationItem, 0)
	for _, subtype := range []string{"mysql", "mariadb", "postgresql"} {
		data, err := a.api(ctx, http.MethodGet, "/databases/db/list/"+subtype, nil)
		if err != nil {
			continue
		}
		for _, server := range a.rows(data) {
			// from 为 remote 表示外部数据库，其数据不在来源服务器上
			name := cast.ToString(server["database"])
			if name == "" || strings.EqualFold(cast.ToString(server["from"]), "remote") {
				continue
			}
			items = append(items, a.databasesOf(ctx, subtype, name, cast.ToString(server["version"]))...)
		}
	}
	return items
}

// databasesOf 列出指定数据库服务下的库
func (a *onePanelAdapter) databasesOf(ctx context.Context, subtype, server, version string) []types.MigrationItem {
	path := "/databases/search"
	if subtype == "postgresql" {
		path = "/databases/pg/search"
	}
	data, err := a.api(ctx, http.MethodPost, path, map[string]any{
		"page": 1, "pageSize": 10000, "info": "", "database": server, "orderBy": "createdAt", "order": "descending",
	})
	if err != nil {
		return nil
	}

	items := make([]types.MigrationItem, 0)
	for _, row := range a.rows(cast.ToStringMap(data)["items"]) {
		id, name := cast.ToString(row["id"]), cast.ToString(row["name"])
		if id == "" || name == "" {
			continue
		}
		items = append(items, types.MigrationItem{
			Key: biz.MigrationItemKey("database", subtype+":"+id), Type: "database", Subtype: subtype, Name: name,
			Status: "running", TargetName: name, SourceID: id, SourceGroup: server, Version: version,
		})
	}
	return items
}

func (a *onePanelAdapter) Detail(ctx context.Context, item types.MigrationItem) (*types.MigrationDetail, error) {
	detail := &types.MigrationDetail{Item: item}
	var err error
	switch item.Type {
	case "website":
		detail.Website, err = a.websiteDetail(ctx, item)
	case "database":
		detail.Database, err = a.databaseDetail(ctx, item)
	default:
		err = errors.New(a.t.Get("unsupported migration resource type: %s", item.Type))
	}
	return detail, err
}

func (a *onePanelAdapter) websiteDetail(ctx context.Context, item types.MigrationItem) (*types.MigrationWebsite, error) {
	data, err := a.api(ctx, http.MethodGet, "/websites/"+item.SourceID, nil)
	if err != nil {
		return nil, err
	}
	row := cast.ToStringMap(data)

	path := cast.ToString(row["sitePath"])
	website := &types.MigrationWebsite{
		Type: item.Subtype, Path: path, Root: path, Remark: cast.ToString(row["remark"]),
		Index: []string{"index.php", "index.html", "index.htm"}, Enabled: item.Status == "running",
		OpenBasedir: cast.ToBool(row["openBaseDir"]), Rewrite: cast.ToString(row["rewrite"]),
	}
	// 1Panel 的站点目录下 index 才是网站根目录
	if dir := cast.ToString(row["siteDir"]); dir != "" && dir != "/" {
		website.Root = strings.TrimRight(path, "/") + "/" + strings.TrimLeft(dir, "/")
	}
	if expire := cast.ToTime(row["expireDate"]); !expire.IsZero() && expire.Year() > 1970 {
		website.ExpireAt = &expire
	}
	if item.Subtype == "php" {
		website.PHP = a.runtimePHP(ctx, cast.ToString(row["runtimeID"]))
	}
	if item.Subtype == "proxy" {
		website.Proxies = []types.MigrationProxy{{Location: "/", Pass: cast.ToString(row["proxy"])}}
	}

	website.Domains, website.Listens = a.domains(ctx, item.SourceID)
	website.Proxies = append(website.Proxies, a.proxies(ctx, item.SourceID)...)
	website.Redirects = a.redirects(ctx, item.Name)
	a.applySSL(ctx, item.SourceID, website)
	return website, nil
}

// domains 读取站点域名与监听端口
func (a *onePanelAdapter) domains(ctx context.Context, websiteID string) ([]string, []string) {
	data, err := a.api(ctx, http.MethodGet, "/websites/domains/"+websiteID, nil)
	if err != nil {
		return nil, nil
	}
	domains, listens := make([]string, 0), make([]string, 0)
	for _, row := range a.rows(data) {
		domain, port := cast.ToString(row["domain"]), cast.ToString(row["port"])
		if domain != "" && !lo.Contains(domains, domain) {
			domains = append(domains, domain)
		}
		if port != "" && port != "0" && !lo.Contains(listens, port) {
			listens = append(listens, port)
		}
	}
	return domains, listens
}

// proxies 读取站点反向代理规则
func (a *onePanelAdapter) proxies(ctx context.Context, websiteID string) []types.MigrationProxy {
	data, err := a.api(ctx, http.MethodPost, "/websites/proxies", map[string]any{"id": cast.ToUint(websiteID)})
	if err != nil {
		return nil
	}
	proxies := make([]types.MigrationProxy, 0)
	for _, row := range a.rows(data) {
		pass := cast.ToString(row["proxyPass"])
		if pass == "" || !cast.ToBool(row["enable"]) {
			continue
		}
		proxies = append(proxies, types.MigrationProxy{
			Location: lo.CoalesceOrEmpty(cast.ToString(row["match"]), "/"), Pass: pass,
			Host: cast.ToString(row["proxyHost"]), Replaces: cast.ToStringMapString(row["replaces"]),
		})
	}
	return proxies
}

// redirects 读取站点重定向规则
func (a *onePanelAdapter) redirects(ctx context.Context, name string) []types.MigrationRedirect {
	data, err := a.api(ctx, http.MethodPost, "/websites/redirect", map[string]any{"websiteName": name})
	if err != nil {
		return nil
	}
	redirects := make([]types.MigrationRedirect, 0)
	for _, row := range a.rows(data) {
		target := cast.ToString(row["redirect"])
		if target == "" || !cast.ToBool(row["enable"]) {
			continue
		}
		redirect := types.MigrationRedirect{
			Type: "url", To: target, KeepURI: cast.ToBool(row["keepPath"]),
			StatusCode: cast.ToInt(row["redirectRoot"]),
		}
		switch cast.ToString(row["type"]) {
		case "domain":
			redirect.Type = "host"
			redirect.From = strings.Join(cast.ToStringSlice(row["domains"]), " ")
		default:
			redirect.From = strings.Join(cast.ToStringSlice(row["paths"]), " ")
		}
		if redirect.StatusCode == 0 {
			redirect.StatusCode = http.StatusMovedPermanently
		}
		redirects = append(redirects, redirect)
	}
	return redirects
}

// applySSL 读取证书与 HTTPS 配置
func (a *onePanelAdapter) applySSL(ctx context.Context, websiteID string, website *types.MigrationWebsite) {
	data, err := a.api(ctx, http.MethodGet, "/websites/"+websiteID+"/https", nil)
	if err != nil {
		return
	}
	config := cast.ToStringMap(data)
	ssl := cast.ToStringMap(config["SSL"])
	website.SSLCert = cast.ToString(ssl["pem"])
	website.SSLKey = cast.ToString(ssl["privateKey"])
	website.HTTPRedirect = cast.ToBool(config["httpConfig"] == "HTTPSOnly")
	website.HSTS = cast.ToBool(config["hsts"])
	website.SSL = cast.ToBool(config["enable"]) && website.SSLCert != "" && website.SSLKey != ""
	if !website.SSL {
		return
	}
	website.SSLProtocols = cast.ToStringSlice(config["SSLProtocol"])
	if !lo.Contains(website.Listens, "443") {
		website.Listens = append(website.Listens, "443")
	}
	website.SSLListens = append(website.SSLListens, "443")
}

// phpVersion 从运行环境详情读取 PHP 版本
func (a *onePanelAdapter) runtimePHP(ctx context.Context, runtimeID string) uint {
	if runtimeID == "" || runtimeID == "0" {
		return 0
	}
	data, err := a.api(ctx, http.MethodGet, "/runtimes/"+runtimeID, nil)
	if err != nil {
		return 0
	}
	return a.phpVersion(cast.ToString(cast.ToStringMap(data)["version"]))
}

func (a *onePanelAdapter) databaseDetail(ctx context.Context, item types.MigrationItem) (*types.MigrationDatabase, error) {
	path := "/databases/search"
	if item.Subtype == "postgresql" {
		path = "/databases/pg/search"
	}
	data, err := a.api(ctx, http.MethodPost, path, map[string]any{
		"page": 1, "pageSize": 10000, "info": item.Name, "database": item.SourceGroup,
		"orderBy": "createdAt", "order": "descending",
	})
	if err != nil {
		return nil, err
	}
	row, ok := lo.Find(a.rows(cast.ToStringMap(data)["items"]), func(row map[string]any) bool {
		return cast.ToString(row["id"]) == item.SourceID
	})
	if !ok {
		return nil, errors.New(a.t.Get("resource no longer exists on the source server"))
	}
	return &types.MigrationDatabase{
		Type: item.Subtype, Version: item.Version, Name: item.Name, Host: "localhost",
		Username: cast.ToString(row["username"]), Password: cast.ToString(row["password"]),
	}, nil
}

func (a *onePanelAdapter) SetRunning(ctx context.Context, item types.MigrationItem, running bool) error {
	if item.Type != "website" {
		return nil
	}
	_, err := a.api(ctx, http.MethodPost, "/websites/operate", map[string]any{
		"id": cast.ToUint(item.SourceID), "operate": lo.Ternary(running, "start", "stop"),
	})
	return err
}

func (a *onePanelAdapter) Backup(ctx context.Context, detail *types.MigrationDetail) (string, error) {
	item := detail.Item
	backup := map[string]any{"isImmediate": true}
	switch item.Type {
	case "website":
		backup["type"], backup["name"], backup["detailName"] = "website", item.Name, item.Name
	case "database":
		backup["type"], backup["name"], backup["detailName"] = item.Subtype, item.SourceGroup, item.Name
	default:
		return "", errors.New(a.t.Get("unsupported migration resource type: %s", item.Type))
	}

	// 备份接口同步执行，返回后最新的一条记录即本次备份
	if _, err := a.api(ctx, http.MethodPost, "/backups/backup", backup); err != nil {
		return "", err
	}
	data, err := a.api(ctx, http.MethodPost, "/backups/record/search", map[string]any{
		"page": 1, "pageSize": 1, "type": backup["type"], "name": backup["name"], "detailName": backup["detailName"],
	})
	if err != nil {
		return "", err
	}
	records := a.rows(cast.ToStringMap(data)["items"])
	if len(records) == 0 {
		return "", errors.New(a.t.Get("the source backup did not finish in time"))
	}

	// 备份记录只有目录与文件名，需换取来源服务器上的绝对路径
	// v1 用 source 指定备份账号，v2 改为 downloadAccountID，两者都带上即可兼容
	path, err := a.api(ctx, http.MethodPost, "/backups/record/download", map[string]any{
		"source": "LOCAL", "downloadAccountID": 1,
		"fileDir": records[0]["fileDir"], "fileName": records[0]["fileName"],
	})
	if err != nil {
		return "", err
	}
	return cast.ToString(path), nil
}

func (a *onePanelAdapter) Download(ctx context.Context, remote, local string) error {
	return a.tryVersions(func() error {
		a.downloadPath = a.resolve("/files/download")
		return a.download(ctx, remote, local)
	})
}
