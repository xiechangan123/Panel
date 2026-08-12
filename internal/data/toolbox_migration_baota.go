package data

import (
	"context"
	"crypto/md5"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/samber/lo"
	"github.com/spf13/cast"
	"resty.dev/v3"

	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/types"
)

type baotaMigrationAdapter struct {
	source *toolboxMigrationSourceRepo
	conn   *request.ToolboxMigrationConnection
}

func (a *baotaMigrationAdapter) Probe(ctx context.Context) (*types.MigrationSourceInfo, error) {
	data, err := a.call(ctx, http.MethodPost, "/system?action=GetSystemTotal", nil)
	if err != nil {
		return nil, err
	}
	info := cast.ToStringMap(data)
	return &types.MigrationSourceInfo{
		Panel:        "baota",
		Version:      cast.ToString(info["version"]),
		Capabilities: []string{"website", "database", "project", "container", "compose"},
	}, nil
}

func (a *baotaMigrationAdapter) Items(ctx context.Context) ([]types.MigrationSourceItem, error) {
	siteData, err := a.call(ctx, http.MethodPost, "/data?action=getData", map[string]string{
		"table": "sites", "p": "1", "limit": "10000", "search": "",
	})
	if err != nil {
		return nil, fmt.Errorf("load websites: %w", err)
	}

	items := make([]types.MigrationSourceItem, 0)
	websiteByName := make(map[string]int)
	for _, row := range a.rows(siteData) {
		id := cast.ToString(row["id"])
		name := cast.ToString(row["name"])
		if id == "" || name == "" {
			continue
		}
		projectType := strings.ToLower(cast.ToString(row["project_type"]))
		if slices.Contains([]string{"node", "python", "go", "net", "java", "other"}, projectType) {
			continue
		}
		subtype := projectType
		switch projectType {
		case "php", "wp2":
			subtype = "php"
			if cast.ToString(row["php_version"]) == "静态" {
				subtype = "static"
			}
		case "html":
			subtype = "static"
		case "proxy":
		default:
			continue
		}
		quota := cast.ToStringMap(row["quota"])
		items = append(items, types.MigrationSourceItem{
			Key:         "website" + ":" + base64.RawURLEncoding.EncodeToString([]byte(id)),
			Type:        "website",
			Subtype:     subtype,
			Name:        name,
			Status:      a.status(cast.ToString(row["status"])),
			Size:        cast.ToInt64(quota["used"]),
			Supported:   true,
			Features:    []string{"files", "domains", "php", "rewrite", "proxy", "redirect", "https"},
			TargetName:  name,
			SourceID:    id,
			SourcePath:  cast.ToString(row["path"]),
			SourceGroup: "site",
		})
		websiteByName[name] = len(items) - 1
	}

	databaseType := "mysql"
	if versionData, loadErr := a.call(ctx, http.MethodPost, "/files?action=GetFileBody", map[string]string{
		"path": "/www/server/mysql/version.pl",
	}); loadErr == nil && strings.Contains(strings.ToLower(cast.ToString(versionData)), "mariadb") {
		databaseType = "mariadb"
	}
	if data, loadErr := a.call(ctx, http.MethodPost, "/data?action=getData", map[string]string{
		"table": "databases", "p": "1", "limit": "10000", "search": "",
	}); loadErr == nil {
		for _, row := range a.rows(data) {
			if !strings.EqualFold(cast.ToString(row["type"]), "mysql") || cast.ToInt(row["db_type"]) != 0 || cast.ToInt(row["sid"]) != 0 {
				continue
			}
			id := cast.ToString(row["id"])
			name := cast.ToString(row["name"])
			if id == "" || name == "" {
				continue
			}
			quota := cast.ToStringMap(row["quota"])
			databaseKey := "database" + ":" + base64.RawURLEncoding.EncodeToString([]byte(id))
			items = append(items, types.MigrationSourceItem{
				Key:         databaseKey,
				Type:        "database",
				Subtype:     databaseType,
				Name:        name,
				Status:      "running",
				Size:        cast.ToInt64(quota["used"]),
				Supported:   true,
				Warnings:    []string{a.source.t.Get("verify compatibility when importing between MySQL and MariaDB or across major versions")},
				Features:    []string{"schema", "data", "user"},
				TargetName:  name,
				SourceID:    id,
				SourceGroup: cast.ToString(row["sid"]),
			})
			if websiteIndex, ok := websiteByName[name]; ok {
				items[websiteIndex].DependsOn = append(items[websiteIndex].DependsOn, databaseKey)
			}
		}
	}

	projectTypes := []struct {
		module  string
		subtype string
	}{
		{"nodejs", "nodejs"}, {"python", "python"}, {"go", "go"},
		{"net", "dotnet"}, {"java", "java"}, {"other", "general"},
	}
	for _, projectType := range projectTypes {
		action := "get_project_list"
		if projectType.module == "python" {
			action = "GetProjectList"
		}
		path := fmt.Sprintf("/project/%s/%s/%s", projectType.module, action, projectType.module)
		data, loadErr := a.call(ctx, http.MethodPost, path, map[string]string{"p": "1", "limit": "10000"})
		if loadErr != nil {
			continue
		}
		for _, row := range a.rows(data) {
			id := cast.ToString(row["id"])
			name := cast.ToString(row["name"])
			if id == "" || name == "" {
				continue
			}
			subtype := projectType.subtype
			config := cast.ToStringMap(row["project_config"])
			if len(config) == 0 {
				_ = json.Unmarshal([]byte(cast.ToString(row["project_config"])), &config)
			}
			if projectType.module == "java" {
				javaType := strings.ToLower(cast.ToString(config["java_type"]))
				if javaType != "" && javaType != "springboot" {
					subtype = "tomcat"
				}
			}
			supported := subtype != "tomcat"
			status := "stopped"
			if cast.ToBool(row["run"]) {
				status = "running"
			}
			item := types.MigrationSourceItem{
				Key:            "project" + ":" + base64.RawURLEncoding.EncodeToString([]byte(projectType.module+":"+id)),
				Type:           "project",
				Subtype:        subtype,
				Name:           name,
				Status:         status,
				Supported:      supported,
				Features:       []string{"files", "command", "environment", "user", "runtime"},
				TargetName:     name,
				TargetPath:     name,
				SourceID:       id,
				SourcePath:     cast.ToString(row["path"]),
				SourceGroup:    projectType.module,
				RuntimeVersion: a.projectRuntimeVersion(config, subtype),
			}
			if item.RuntimeVersion != "" {
				item.Features = append(item.Features, "runtime "+item.RuntimeVersion)
			}
			if !supported {
				item.Blockers = []string{a.source.t.Get("Tomcat projects are not supported for automatic migration")}
			}
			if subtype == "nodejs" || subtype == "python" {
				item.Warnings = []string{a.source.t.Get("project files are migrated, but dependencies are not installed and the target project remains stopped")}
			} else if subtype == "go" && item.RuntimeVersion == "" {
				item.Warnings = []string{a.source.t.Get("BaoTa does not expose the build architecture for this Go project; verify the executable architecture before starting it")}
			}
			items = append(items, item)
		}
	}

	if data, loadErr := a.call(ctx, http.MethodPost, "/btdocker/compose/compose_project_list", map[string]string{}); loadErr == nil {
		for _, row := range a.rows(data) {
			name := cast.ToString(row["name"])
			id := cast.ToString(row["id"])
			if name == "" || id == "" {
				continue
			}
			items = append(items, types.MigrationSourceItem{
				Key: "compose" + ":" + base64.RawURLEncoding.EncodeToString([]byte(name)), Type: "compose", Subtype: "compose", Name: name,
				Status: a.status(cast.ToString(row["run_status"])), Supported: true,
				Warnings: []string{
					a.source.t.Get("named volumes and bind data outside the Compose project directory must be migrated manually"),
				},
				Features:   []string{"compose", "environment", "images", "project-files", "networks"},
				TargetName: name, TargetPath: name, SourceID: id,
				SourcePath: cast.ToString(row["path"]),
			})
		}
	}

	if data, loadErr := a.call(ctx, http.MethodPost, "/btdocker/container/get_list", map[string]string{"p": "1", "limit": "10000"}); loadErr == nil {
		for _, row := range a.rows(data) {
			name := strings.TrimPrefix(cast.ToString(row["name"]), "/")
			id := cast.ToString(row["id"])
			if name == "" || id == "" {
				continue
			}
			if detail, detailErr := a.call(ctx, http.MethodPost, "/btdocker/container/get_container_info", map[string]string{"id": id}); detailErr == nil {
				labels := cast.ToStringMap(cast.ToStringMap(detail)["Config"])["Labels"]
				if cast.ToString(cast.ToStringMap(labels)["com.docker.compose.project"]) != "" {
					continue
				}
			}
			items = append(items, types.MigrationSourceItem{
				Key: "container" + ":" + base64.RawURLEncoding.EncodeToString([]byte(id)), Type: "container", Subtype: "docker", Name: name,
				Status: a.status(cast.ToString(row["status"])), Supported: true,
				Warnings: []string{
					a.source.t.Get("devices, system directories, Docker sockets, GPUs, and special namespaces require verification on the target"),
				},
				Features:   []string{"image", "writable-layer", "ports", "environment", "volumes", "networks"},
				TargetName: name, SourceID: id,
			})
		}
	}

	return items, nil
}

func (a *baotaMigrationAdapter) projectRuntimeVersion(config map[string]any, subtype string) string {
	switch subtype {
	case "nodejs", "node":
		return cast.ToString(config["nodejs_version"])
	case "python":
		return lo.CoalesceOrEmpty(cast.ToString(config["version"]), cast.ToString(config["python_bin"]))
	case "java":
		return lo.CoalesceOrEmpty(cast.ToString(config["project_jdk"]), cast.ToString(config["jdk_path"]))
	case "dotnet", "net":
		return cast.ToString(config["dotnet_version"])
	default:
		return ""
	}
}

func (a *baotaMigrationAdapter) Detail(ctx context.Context, item types.MigrationSourceItem) (*types.MigrationSourceDetail, error) {
	var err error
	detail := &types.MigrationSourceDetail{Item: item}
	switch item.Type {
	case "website":
		website := &types.MigrationWebsiteDetail{
			Type: "static", Path: item.SourcePath, Root: item.SourcePath,
			Domains: []string{item.Name}, Listens: []string{"80"}, Index: []string{"index.php", "index.html", "index.htm"},
			Enabled: item.Status == "running",
		}
		if item.Subtype == "proxy" {
			website.Type = "proxy"
		}
		if data, err := a.call(ctx, http.MethodPost, "/data?action=getData", map[string]string{
			"table": "sites", "p": "1", "limit": "10000", "search": item.Name,
		}); err == nil {
			for _, row := range a.rows(data) {
				if cast.ToString(row["id"]) == item.SourceID {
					website.Remark = cast.ToString(row["ps"])
					website.ExpireAt = a.source.parseTime(cast.ToString(row["edate"]))
					break
				}
			}
		}

		if data, err := a.call(ctx, http.MethodPost, "/data?action=getData", map[string]string{
			"table": "domain", "p": "1", "limit": "10000", "search": item.SourceID,
		}); err == nil {
			var domains []string
			var listens []string
			for _, row := range a.rows(data) {
				if cast.ToString(row["pid"]) != item.SourceID {
					continue
				}
				domain := cast.ToString(row["name"])
				port := cast.ToString(row["port"])
				if domain != "" && !slices.Contains(domains, domain) {
					domains = append(domains, domain)
				}
				if port != "" && !slices.Contains(listens, port) {
					listens = append(listens, port)
				}
			}
			if len(domains) > 0 {
				website.Domains = domains
			}
			if len(listens) > 0 {
				website.Listens = listens
			}
		}

		if data, err := a.call(ctx, http.MethodPost, "/site?action=GetSitePHPVersion", map[string]string{"siteName": item.Name}); err == nil {
			info := cast.ToStringMap(data)
			website.PHP = a.source.runtimeVersion(cast.ToString(info["phpversion"]))
			if website.PHP > 0 {
				website.Type = "php"
			}
		}
		if data, err := a.call(ctx, http.MethodPost, "/site?action=GetSiteRunPath", map[string]string{"id": item.SourceID}); err == nil {
			runPath := cast.ToString(cast.ToStringMap(data)["runPath"])
			if runPath != "" && runPath != "/" {
				website.Root = filepath.Join(website.Path, strings.TrimPrefix(runPath, "/"))
			}
		}
		if data, err := a.call(ctx, http.MethodPost, "/site?action=GetDirUserINI", map[string]string{
			"id": item.SourceID, "path": website.Path,
		}); err == nil {
			website.OpenBasedir = cast.ToBool(cast.ToStringMap(data)["userini"])
		}
		if data, err := a.call(ctx, http.MethodPost, "/site?action=GetIndex", map[string]string{"id": item.SourceID}); err == nil {
			if value, ok := data.(string); ok && value != "" {
				website.Index = strings.FieldsFunc(value, func(char rune) bool { return char == ',' || char == ' ' || char == '\t' })
			}
		}
		if data, err := a.call(ctx, http.MethodPost, "/site?action=GetRewriteLists", map[string]string{
			"site_ids": "[" + item.SourceID + "]", "site_type": item.Subtype,
		}); err == nil {
			for _, rewrite := range a.rows(cast.ToStringMap(data)["site_rewrites"]) {
				if cast.ToString(rewrite["id"]) != item.SourceID || cast.ToString(rewrite["file"]) == "" {
					continue
				}
				if content, loadErr := a.call(ctx, http.MethodPost, "/files?action=GetFileBody", map[string]string{
					"path": cast.ToString(rewrite["file"]),
				}); loadErr == nil {
					website.Rewrite = cast.ToString(content)
				}
				break
			}
		}
		if data, err := a.call(ctx, http.MethodPost, "/site?action=GetProxyList", map[string]string{"sitename": item.Name}); err == nil {
			for _, row := range a.rows(data) {
				pass := cast.ToString(row["proxysite"])
				if pass == "" || cast.ToInt(row["type"]) != 1 {
					continue
				}
				replaces := lo.SliceToMap(a.rows(row["subfilter"]), func(filter map[string]any) (string, string) {
					return cast.ToString(filter["sub1"]), cast.ToString(filter["sub2"])
				})
				delete(replaces, "")
				website.Proxies = append(website.Proxies, types.MigrationProxy{
					Location: lo.CoalesceOrEmpty(cast.ToString(row["proxydir"]), "/"),
					Pass:     pass, Host: cast.ToString(row["todomain"]), Replaces: replaces, HTTPVersion: "1.1",
				})
			}
			if website.Type == "static" && slices.ContainsFunc(website.Proxies, func(proxy types.MigrationProxy) bool {
				return proxy.Location == "/" || proxy.Location == "^~ /"
			}) {
				website.Type = "proxy"
			}
		}
		if data, err := a.call(
			ctx,
			http.MethodPost,
			"/site?action=GetRedirectList",
			map[string]string{"sitename": item.Name, "errorpage": "0"},
		); err == nil {
			for _, row := range a.rows(data) {
				if cast.ToString(row["type"]) == "0" || cast.ToString(row["tourl"]) == "" {
					continue
				}
				redirectType := "url"
				from := cast.ToString(row["redirectpath"])
				if cast.ToString(row["domainorpath"]) == "domain" {
					redirectType = "host"
					from = strings.Join(a.source.strings(row["redirectdomain"]), " ")
				}
				statusCode := int(cast.ToInt64(row["redirecttype"]))
				if statusCode == 0 {
					statusCode = 301
				}
				website.Redirects = append(website.Redirects, types.MigrationRedirect{
					Type: redirectType, From: from, To: cast.ToString(row["tourl"]), KeepURI: cast.ToBool(row["holdpath"]), StatusCode: statusCode,
				})
			}
		}
		if data, err := a.call(ctx, http.MethodPost, "/site?action=GetSSL", map[string]string{"siteName": item.Name}); err == nil {
			ssl := cast.ToStringMap(data)
			website.SSLCert = cast.ToString(ssl["csr"])
			website.SSLKey = cast.ToString(ssl["key"])
			website.SSL = cast.ToBool(ssl["status"]) && website.SSLCert != "" && website.SSLKey != ""
			website.HTTPRedirect = cast.ToBool(ssl["httpTohttps"])
			if website.SSL {
				tlsVersions := cast.ToStringMap(ssl["tls_versions"])
				for _, protocol := range []string{"TLSv1", "TLSv1.1", "TLSv1.2", "TLSv1.3"} {
					if cast.ToBool(tlsVersions[protocol]) {
						website.SSLProtocols = append(website.SSLProtocols, protocol)
					}
				}
				port := "443"
				if portData, loadErr := a.call(ctx, http.MethodPost, "/data?action=get_https_port", map[string]string{
					"siteName": item.Name,
				}); loadErr == nil && cast.ToInt(portData) > 0 {
					port = cast.ToString(portData)
				}
				if !slices.Contains(website.Listens, port) {
					website.Listens = append(website.Listens, port)
				}
				website.SSLListens = append(website.SSLListens, port)
			}
		}
		detail.Website = website
	case "database":
		database := &types.MigrationDatabaseDetail{
			Type: item.Subtype, Server: item.SourceGroup, Name: item.Name, Host: "localhost",
		}
		var data any
		data, err = a.call(ctx, http.MethodPost, "/data?action=getData", map[string]string{
			"table": "databases", "p": "1", "limit": "10000", "search": item.Name,
		})
		if err == nil {
			row, exists := lo.Find(a.rows(data), func(row map[string]any) bool {
				return cast.ToString(row["id"]) == item.SourceID
			})
			if exists {
				database.Username = cast.ToString(row["username"])
				database.Password = cast.ToString(row["password"])
				database.Host = lo.CoalesceOrEmpty(cast.ToString(row["accept"]), "localhost")
				database.PasswordOK = database.Password != ""
			}
		}
		if versionData, loadErr := a.call(ctx, http.MethodPost, "/files?action=GetFileBody", map[string]string{
			"path": "/www/server/mysql/version.pl",
		}); loadErr == nil {
			database.Version = strings.TrimSpace(cast.ToString(versionData))
		}
		detail.Database = database
	case "project":
		action := "get_project_list"
		if item.SourceGroup == "python" {
			action = "GetProjectList"
		}
		var data any
		data, err = a.call(
			ctx,
			http.MethodPost,
			fmt.Sprintf("/project/%s/%s/%s", item.SourceGroup, action, item.SourceGroup),
			map[string]string{"p": "1", "limit": "10000"},
		)
		if err != nil {
			break
		}
		row, exists := lo.Find(a.rows(data), func(row map[string]any) bool {
			return cast.ToString(row["id"]) == item.SourceID || cast.ToString(row["name"]) == item.Name
		})
		if !exists {
			err = errors.New("project no longer exists")
			break
		}
		config := cast.ToStringMap(row["project_config"])
		if len(config) == 0 {
			_ = json.Unmarshal([]byte(cast.ToString(row["project_config"])), &config)
		}
		projectPath := cast.ToString(row["path"])
		switch item.SourceGroup {
		case "nodejs":
			projectPath = lo.CoalesceOrEmpty(cast.ToString(config["project_cwd"]), projectPath)
		case "python":
			projectPath = lo.CoalesceOrEmpty(cast.ToString(config["path"]), projectPath)
		case "go", "net":
			projectPath = lo.CoalesceOrEmpty(cast.ToString(config["project_path"]), projectPath)
		case "java":
			projectPath = cast.ToString(config["jar_path"])
			if projectPath == "" {
				projectPath = filepath.Dir(lo.CoalesceOrEmpty(cast.ToString(config["project_jar"]), cast.ToString(row["path"])))
			}
		case "other":
			projectPath = filepath.Dir(lo.CoalesceOrEmpty(cast.ToString(config["project_exe"]), cast.ToString(row["path"])))
		}
		execStart := cast.ToString(config["project_cmd"])
		if item.SourceGroup == "nodejs" && execStart == "" {
			execStart = cast.ToString(config["project_script"])
			if content, loadErr := a.call(ctx, http.MethodPost, "/files?action=GetFileBody", map[string]string{
				"path": filepath.Join(projectPath, "package.json"),
			}); loadErr == nil {
				packageInfo := make(map[string]any)
				if json.Unmarshal([]byte(cast.ToString(content)), &packageInfo) == nil {
					if _, exists := cast.ToStringMap(packageInfo["scripts"])[execStart]; exists {
						execStart = "npm run " + execStart
					}
				}
			}
			if fields := strings.Fields(execStart); len(fields) > 0 {
				if extension := strings.ToLower(filepath.Ext(fields[0])); extension == ".js" || extension == ".cjs" || extension == ".mjs" {
					execStart = "node " + execStart
				}
			}
		}
		if item.SourceGroup == "python" && execStart == "" {
			switch strings.ToLower(cast.ToString(config["stype"])) {
			case "python":
				execStart = strings.TrimSpace("python -u " + cast.ToString(config["rfile"]) + " " + cast.ToString(config["parm"]))
			case "uwsgi":
				execStart = "uwsgi --ini " + filepath.Join(projectPath, "uwsgi.ini")
			case "gunicorn":
				module := strings.TrimSuffix(strings.TrimPrefix(cast.ToString(config["rfile"]), projectPath+"/"), ".py")
				module = strings.ReplaceAll(module, "/", ".")
				execStart = "gunicorn -c " + filepath.Join(projectPath, "gunicorn_conf.py") + " " + module + ":" + cast.ToString(config["call_app"])
			}
		}
		environments := a.source.environment(config["env_list"])
		environments = append(environments, a.source.environment(config["environment"])...)
		if envFile := cast.ToString(config["env_file"]); envFile != "" {
			if content, loadErr := a.call(ctx, http.MethodPost, "/files?action=GetFileBody", map[string]string{
				"path": envFile,
			}); loadErr == nil {
				environments = append(environments, a.source.environment(cast.ToString(content))...)
			}
		}
		enabled := cast.ToBool(config["is_power_on"])
		if item.SourceGroup == "python" {
			enabled = cast.ToBool(config["auto_run"])
		}
		domains := make([]string, 0)
		listens := make([]string, 0)
		if domainData, loadErr := a.call(ctx, http.MethodPost, "/data?action=getData", map[string]string{
			"table": "domain", "p": "1", "limit": "10000", "search": item.SourceID,
		}); loadErr == nil {
			for _, domain := range a.rows(domainData) {
				if cast.ToString(domain["pid"]) != item.SourceID {
					continue
				}
				if name := cast.ToString(domain["name"]); name != "" && !slices.Contains(domains, name) {
					domains = append(domains, name)
				}
				if port := cast.ToString(domain["port"]); port != "" && !slices.Contains(listens, port) {
					listens = append(listens, port)
				}
			}
		}
		detail.Project = &types.MigrationProjectDetail{
			Type:         a.source.projectType(item.Subtype),
			Version:      a.projectRuntimeVersion(config, item.Subtype),
			Path:         projectPath,
			WorkingDir:   projectPath,
			ExecStart:    execStart,
			User:         lo.CoalesceOrEmpty(lo.CoalesceOrEmpty(cast.ToString(config["run_user"]), cast.ToString(config["user"])), "www"),
			Restart:      "on-failure",
			Port:         cast.ToUint(config["port"]),
			Domains:      domains,
			Listens:      listens,
			Environments: environments,
			Running:      item.Status == "running",
			Enabled:      enabled,
		}
	case "container":
		var raw any
		raw, err = a.call(ctx, http.MethodPost, "/btdocker/container/get_container_info", map[string]string{"id": item.SourceID})
		if err == nil {
			detail.Container = a.source.parseContainerDetail(raw, &item)
		}
	case "compose":
		var data any
		data, err = a.call(ctx, http.MethodPost, "/btdocker/compose/compose_project_list", map[string]string{})
		if err != nil {
			break
		}
		row, exists := lo.Find(a.rows(data), func(row map[string]any) bool {
			return cast.ToString(row["name"]) == item.Name
		})
		if !exists {
			err = errors.New("compose project no longer exists")
			break
		}
		composePath := cast.ToString(row["path"])
		compose := &types.MigrationComposeDetail{
			Path: composePath, Running: item.Status == "running",
		}
		if compose.Compose == "" && compose.Path != "" {
			candidates := []string{compose.Path}
			if filepath.Ext(compose.Path) == "" {
				candidates = []string{
					filepath.Join(compose.Path, "docker-compose.yaml"),
					filepath.Join(compose.Path, "docker-compose.yml"),
				}
			}
			for _, candidate := range candidates {
				content, loadErr := a.call(ctx, http.MethodPost, "/files?action=GetFileBody", map[string]string{"path": candidate})
				if loadErr != nil || cast.ToString(content) == "" {
					continue
				}
				compose.Path = candidate
				compose.Compose = cast.ToString(content)
				break
			}
		}
		if compose.Path != "" {
			envPath := filepath.Join(filepath.Dir(compose.Path), ".env")
			if content, loadErr := a.call(ctx, http.MethodPost, "/files?action=GetFileBody", map[string]string{"path": envPath}); loadErr == nil {
				compose.Envs = a.source.environment(cast.ToString(content))
			}
		}
		detail.Compose = compose
	default:
		err = errors.New("unsupported migration resource")
	}
	if err != nil {
		return nil, err
	}
	return detail, nil
}

func (a *baotaMigrationAdapter) SetRunning(ctx context.Context, detail *types.MigrationSourceDetail, running bool) error {
	operate := "stop"
	if running {
		operate = "start"
	}
	item := detail.Item
	switch item.Type {
	case "website":
		_, err := a.call(
			ctx,
			http.MethodPost,
			"/site?action=Site"+strings.ToUpper(operate[:1])+operate[1:],
			map[string]string{"id": item.SourceID, "name": item.Name},
		)
		return err
	case "project":
		_, err := a.call(
			ctx,
			http.MethodPost,
			fmt.Sprintf("/project/%s/%s_project/%s", item.SourceGroup, operate, item.SourceGroup),
			map[string]string{"project_name": item.Name, "id": item.SourceID},
		)
		return err
	case "container":
		_, err := a.call(ctx, http.MethodPost, "/btdocker/container/"+operate, map[string]string{"id": item.SourceID})
		return err
	case "compose":
		_, err := a.call(ctx, http.MethodPost, "/btdocker/compose/set_compose_status", map[string]string{"project_id": item.SourceID, "status": operate})
		return err
	default:
		return nil
	}
}

func (a *baotaMigrationAdapter) Prepare(ctx context.Context, detail *types.MigrationSourceDetail) (*types.MigrationArtifact, error) {
	item := detail.Item
	switch item.Type {
	case "website", "database":
		existing := a.backupFiles(ctx, item)
		path := "/site?action=ToBackup"
		if item.Type == "database" {
			path = "/database?action=ToBackup"
		}
		if _, err := a.call(ctx, http.MethodPost, path, map[string]string{"id": item.SourceID}); err != nil {
			return nil, err
		}
		remotePath, err := a.waitBackup(ctx, item, existing)
		if err != nil {
			return nil, err
		}
		return &types.MigrationArtifact{RemotePath: remotePath, FileName: filepath.Base(remotePath), Kind: item.Type}, nil
	case "project", "compose":
		sourcePath := item.SourcePath
		if item.Type == "project" && detail.Project != nil {
			sourcePath = detail.Project.Path
		}
		if item.Type == "compose" && detail.Compose != nil {
			sourcePath = detail.Compose.Path
			if filepath.Ext(sourcePath) != "" {
				sourcePath = filepath.Dir(sourcePath)
			}
		}
		artifact, err := a.archiveDirectory(ctx, item, sourcePath)
		if err != nil || item.Type != "compose" || detail.Compose == nil {
			return artifact, err
		}
		return a.addComposeImages(ctx, detail.Compose, artifact, item.Name)
	case "container":
		if detail.Container == nil {
			return nil, errors.New("container detail is missing")
		}
		remoteDir := "/www/backup/acepanel-migration"
		name := "acepanel-migration-" + a.source.safeFileName(item.Name)
		tag := strconv.FormatInt(time.Now().UnixNano(), 10)
		exportName := fmt.Sprintf("%s-%d", name, time.Now().UnixNano())
		_, err := a.call(ctx, http.MethodPost, "/btdocker/container/commit", map[string]string{
			"id": item.SourceID, "repository": name, "name": exportName, "tag": tag,
			"path": remoteDir,
		})
		if err != nil {
			return nil, err
		}
		detail.Container.Image = name + ":" + tag
		remotePath := filepath.Join(remoteDir, exportName+".tar")
		paths := []string{remotePath}
		for i := range detail.Container.Volumes {
			mount := &detail.Container.Volumes[i]
			cleanSource := filepath.Clean(mount.Source)
			if mount.Source == "" || cleanSource == "/" || cleanSource == "/var/run/docker.sock" || cleanSource == "/run/docker.sock" ||
				cleanSource == "/dev" || cleanSource == "/proc" || cleanSource == "/sys" || strings.HasPrefix(cleanSource, "/dev/") ||
				strings.HasPrefix(cleanSource, "/proc/") || strings.HasPrefix(cleanSource, "/sys/") {
				continue
			}
			mountPath := filepath.Join(remoteDir, fmt.Sprintf("%s-mount-%d-%d.tar.gz", name, i+1, time.Now().UnixNano()))
			archiveResult, archiveErr := a.call(ctx, http.MethodPost, "/files?action=ZipAndDownload", map[string]string{
				"path": filepath.Dir(mount.Source), "sfile": filepath.Base(mount.Source), "dfile": mountPath, "z_type": "tar.gz",
			})
			if archiveErr != nil {
				continue
			}
			if taskID := cast.ToString(cast.ToStringMap(archiveResult)["task_id"]); taskID != "" {
				if archiveErr = a.waitTask(ctx, taskID); archiveErr != nil {
					continue
				}
			}
			mount.BackupPath = mountPath
			paths = append(paths, mountPath)
		}
		artifact := &types.MigrationArtifact{RemotePath: remotePath, FileName: filepath.Base(remotePath), Kind: "container_image"}
		if len(paths) > 1 {
			artifact.RemotePaths = paths
			artifact.Kind = "container_bundle"
		}
		return artifact, nil
	default:
		return nil, errors.New("unsupported migration resource")
	}
}

func (a *baotaMigrationAdapter) addComposeImages(
	ctx context.Context,
	compose *types.MigrationComposeDetail,
	artifact *types.MigrationArtifact,
	name string,
) (*types.MigrationArtifact, error) {
	images := a.source.composeImages(compose.Compose, compose.Envs)
	if len(images) == 0 {
		return artifact, nil
	}
	remoteDir := "/www/backup/acepanel-migration"
	paths := []string{artifact.RemotePath}
	compose.ImageTags = make(map[string]string, len(images))
	compose.ImageSources = make(map[string]string, len(images))
	i := 0
	for source, image := range images {
		i++
		fileName := fmt.Sprintf("%s-image-%d-%d", a.source.safeFileName(name), i, time.Now().UnixNano())
		if _, err := a.call(ctx, http.MethodPost, "/btdocker/image/save", map[string]string{
			"id": image, "path": remoteDir, "name": fileName,
		}); err != nil {
			return nil, err
		}
		compose.ImageTags[source] = fmt.Sprintf("acepanel-migration/%s-%d:%d", a.source.safeFileName(name), i, time.Now().UnixNano())
		compose.ImageSources[source] = image
		paths = append(paths, filepath.Join(remoteDir, fileName+".tar"))
	}
	if len(paths) > 1 {
		artifact.RemotePaths = paths
		artifact.Kind = "compose_bundle"
	}
	return artifact, nil
}

func (a *baotaMigrationAdapter) archiveDirectory(
	ctx context.Context,
	item types.MigrationSourceItem,
	sourcePath string,
) (*types.MigrationArtifact, error) {
	if sourcePath == "" || sourcePath == "/" {
		return nil, errors.New("source path is empty")
	}
	remotePath := filepath.Join("/www/backup/acepanel-migration", fmt.Sprintf("%s-%d.tar.gz", a.source.safeFileName(item.Name), time.Now().UnixNano()))
	result, err := a.call(ctx, http.MethodPost, "/files?action=ZipAndDownload", map[string]string{
		"path": filepath.Dir(sourcePath), "sfile": filepath.Base(sourcePath), "dfile": remotePath, "z_type": "tar.gz",
	})
	if err != nil {
		return nil, err
	}
	if taskID := cast.ToString(cast.ToStringMap(result)["task_id"]); taskID != "" {
		if err = a.waitTask(ctx, taskID); err != nil {
			return nil, err
		}
	}
	return &types.MigrationArtifact{RemotePath: remotePath, FileName: filepath.Base(remotePath), Kind: item.Type}, nil
}

func (a *baotaMigrationAdapter) waitTask(ctx context.Context, taskID string) error {
	for {
		data, err := a.call(ctx, http.MethodPost, "/task?action=get_task_lists", map[string]string{"task_id": taskID})
		if err == nil {
			rows := a.rows(data)
			if len(rows) > 0 {
				switch cast.ToInt(rows[0]["status"]) {
				case 1:
					return nil
				case 0, -1:
				default:
					return errors.New("source task failed")
				}
			}
		}
		if err := a.source.wait(ctx, time.Second); err != nil {
			return err
		}
	}
}

func (a *baotaMigrationAdapter) backupFiles(ctx context.Context, item types.MigrationSourceItem) map[string]bool {
	result := make(map[string]bool)
	backupType := "0"
	if item.Type == "database" {
		backupType = "1"
	}
	data, err := a.call(ctx, http.MethodPost, "/data?action=getData", map[string]string{
		"table": "backup", "p": "1", "limit": "100", "search": item.SourceID, "type": backupType,
	})
	if err != nil {
		return result
	}
	for _, row := range a.rows(data) {
		if filename := cast.ToString(row["filename"]); filename != "" {
			result[filename] = true
		}
	}
	return result
}

func (a *baotaMigrationAdapter) waitBackup(ctx context.Context, item types.MigrationSourceItem, existing map[string]bool) (string, error) {
	backupType := "0"
	if item.Type == "database" {
		backupType = "1"
	}
	for attempt := 0; attempt < 120; attempt++ {
		data, err := a.call(ctx, http.MethodPost, "/data?action=getData", map[string]string{
			"table": "backup", "p": "1", "limit": "100", "search": item.SourceID, "type": backupType,
		})
		if err == nil {
			for _, row := range a.rows(data) {
				filename := cast.ToString(row["filename"])
				if filename == "" || existing[filename] {
					continue
				}
				pid := cast.ToString(row["pid"])
				if pid == item.SourceID {
					return lo.CoalesceOrEmpty(cast.ToString(row["local"]), strings.Split(filename, "|")[0]), nil
				}
			}
		}
		if err := a.source.wait(ctx, time.Second); err != nil {
			return "", err
		}
	}
	return "", errors.New("source backup did not finish in time")
}

func (a *baotaMigrationAdapter) Download(ctx context.Context, artifact *types.MigrationArtifact, target string) error {
	return a.source.download(ctx, artifact, target, a.downloadFile)
}

func (a *baotaMigrationAdapter) rows(value any) []map[string]any {
	if values, ok := value.([]any); ok {
		return lo.FilterMap(values, func(value any, _ int) (map[string]any, bool) {
			row := cast.ToStringMap(value)
			return row, len(row) > 0
		})
	}
	if containers, ok := cast.ToStringMap(value)["container_list"]; ok {
		return a.rows(containers)
	}
	return nil
}

func (a *baotaMigrationAdapter) status(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "running", "run", "started", "active", "enabled", "normal", "up":
		return "running"
	case "paused", "pause":
		return "paused"
	case "", "0", "stopped", "stop", "disabled", "down", "exited":
		return "stopped"
	default:
		return value
	}
}

func (a *baotaMigrationAdapter) apiResponse(resp *resty.Response, err error) (any, error) {
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode(), strings.TrimSpace(resp.String()))
	}
	var raw any
	if err = json.Unmarshal(resp.Bytes(), &raw); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}
	result := cast.ToStringMap(raw)
	if status, ok := result["status"].(bool); ok && !status {
		return nil, fmt.Errorf("source API: %s", lo.CoalesceOrEmpty(cast.ToString(result["msg"]), cast.ToString(result["message"])))
	}
	if data, ok := result["data"]; ok {
		return data, nil
	}
	return raw, nil
}

func (a *baotaMigrationAdapter) call(ctx context.Context, method, path string, body any) (any, error) {
	client := resty.New().
		SetBaseURL(strings.TrimRight(a.conn.URL, "/")).
		SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true}).
		SetTimeout(45 * time.Second)
	defer func() { _ = client.Close() }()

	form := cast.ToStringMapString(body)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	secret := md5.Sum([]byte(a.conn.APIKey))
	token := md5.Sum([]byte(timestamp + hex.EncodeToString(secret[:])))
	form["request_time"] = timestamp
	form["request_token"] = hex.EncodeToString(token[:])
	req := client.R().SetContext(ctx)
	if method == http.MethodGet {
		req.SetQueryParams(form)
	} else {
		req.SetFormData(form)
	}
	return a.apiResponse(req.Execute(method, path))
}

func (a *baotaMigrationAdapter) downloadFile(ctx context.Context, remotePath, target string) error {
	client := resty.New().
		SetBaseURL(strings.TrimRight(a.conn.URL, "/")).
		SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true}).
		SetTimeout(30 * time.Minute)
	defer func() { _ = client.Close() }()
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	secret := md5.Sum([]byte(a.conn.APIKey))
	token := md5.Sum([]byte(timestamp + hex.EncodeToString(secret[:])))
	resp, err := client.R().
		SetContext(ctx).
		SetResponseDoNotParse(true).
		SetQueryParams(map[string]string{
			"filename": remotePath, "request_time": timestamp, "request_token": hex.EncodeToString(token[:]),
		}).
		Get("/download")
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode() != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("status %d: %s", resp.StatusCode(), strings.TrimSpace(string(body)))
	}
	if err = os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	file, err := os.Create(target)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(file, resp.Body)
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}
