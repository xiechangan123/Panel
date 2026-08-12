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

type onePanelMigrationAdapter struct {
	source *toolboxMigrationSourceRepo
	conn   *request.ToolboxMigrationConnection
}

func (a *onePanelMigrationAdapter) Probe(ctx context.Context) (*types.MigrationSourceInfo, error) {
	data, err := a.call(ctx, http.MethodGet, "/api/v2/dashboard/base/os", nil)
	if err != nil {
		return nil, err
	}
	info := cast.ToStringMap(data)
	version := ""
	if settingData, settingErr := a.call(ctx, http.MethodPost, "/api/v2/settings/search", nil); settingErr == nil {
		version = cast.ToString(cast.ToStringMap(settingData)["systemVersion"])
	}
	return &types.MigrationSourceInfo{
		Panel:        "onepanel",
		Version:      version,
		Architecture: cast.ToString(info["kernelArch"]),
		Capabilities: []string{"website", "database", "runtime", "container", "compose"},
	}, nil
}

func (a *onePanelMigrationAdapter) Items(ctx context.Context) ([]types.MigrationSourceItem, error) {
	websiteData, err := a.call(ctx, http.MethodPost, "/api/v2/websites/search", lo.Assign(map[string]any{"page": 1, "pageSize": 10000}, map[string]any{
		"name": "", "type": "", "orderBy": "createdAt", "order": "descending",
	}))
	if err != nil {
		return nil, fmt.Errorf("load websites: %w", err)
	}

	items := make([]types.MigrationSourceItem, 0)
	runtimeWebsites := make(map[string][]int)
	deploymentWebsites := make(map[string][]int)
	websiteDatabases := make(map[int]string)
	websiteDatabaseTypes := make(map[int]string)
	for _, row := range a.rows(websiteData) {
		id := cast.ToString(row["id"])
		name := lo.CoalesceOrEmpty(cast.ToString(row["alias"]), cast.ToString(row["primaryDomain"]))
		if id == "" || name == "" {
			continue
		}
		subtype := strings.ToLower(cast.ToString(row["type"]))
		supported := slices.Contains([]string{"static", "proxy", "runtime", "deployment"}, subtype)
		item := types.MigrationSourceItem{
			Key: "website" + ":" + base64.RawURLEncoding.EncodeToString([]byte(id)), Type: "website", Subtype: subtype, Name: name,
			Status: a.status(cast.ToString(row["status"])), Supported: supported,
			Features:   []string{"files", "domains", "php", "rewrite", "proxy", "redirect", "https"},
			TargetName: name, SourceID: id, SourcePath: cast.ToString(row["sitePath"]),
		}
		if !supported {
			item.Blockers = []string{a.source.t.Get("this 1Panel website type is not supported for automatic migration: %s", subtype)}
		}
		var websiteDetail map[string]any
		if detail, detailErr := a.call(ctx, http.MethodGet, "/api/v2/websites/"+id, nil); detailErr == nil {
			websiteDetail = cast.ToStringMap(detail)
			if subtype == "runtime" && strings.EqualFold(cast.ToString(websiteDetail["runtimeType"]), "php") {
				item.Subtype = "php"
			}
		}
		if subtype == "deployment" {
			item.Subtype = "appstore"
			item.Warnings = []string{a.source.t.Get("this AppStore website is migrated with its related Compose workload")}
		}
		runtime := cast.ToString(row["runtimeName"])
		if runtime != "" {
			item.Warnings = []string{a.source.t.Get("this website depends on a 1Panel Runtime and is migrated with the related Compose workload")}
		}
		items = append(items, item)
		if websiteDetail != nil {
			if databaseID := cast.ToString(websiteDetail["dbID"]); databaseID != "" && databaseID != "0" {
				websiteDatabases[len(items)-1] = databaseID
				websiteDatabaseTypes[len(items)-1] = strings.ToLower(cast.ToString(websiteDetail["dbType"]))
			}
		}
		if runtime != "" {
			runtimeWebsites[runtime] = append(runtimeWebsites[runtime], len(items)-1)
		}
		if appInstallID := cast.ToString(row["appInstallId"]); appInstallID != "" && appInstallID != "0" {
			deploymentWebsites[appInstallID] = append(deploymentWebsites[appInstallID], len(items)-1)
		}
	}

	runtimeNames := make(map[string]bool)
	if data, loadErr := a.call(
		ctx,
		http.MethodPost,
		"/api/v2/runtimes/search",
		lo.Assign(map[string]any{"page": 1, "pageSize": 10000}, map[string]any{"name": "", "type": "", "status": ""}),
	); loadErr == nil {
		for _, row := range a.rows(data) {
			id := cast.ToString(row["id"])
			name := cast.ToString(row["name"])
			typ := strings.ToLower(cast.ToString(row["type"]))
			if id == "" || name == "" || typ == "php" {
				continue
			}
			runtimeNames[name] = true
			key := "project" + ":" + base64.RawURLEncoding.EncodeToString([]byte("runtime:"+id))
			item := types.MigrationSourceItem{
				Key: key, Type: "project", Subtype: "runtime_" + typ, Name: name,
				Status: a.status(cast.ToString(row["status"])), Supported: true,
				Features:   []string{"code", "compose", "environment", "image", "volumes", "proxy"},
				TargetName: name, TargetPath: name, SourceID: id,
				SourcePath: lo.CoalesceOrEmpty(cast.ToString(row["path"]), cast.ToString(row["codeDir"])), SourceGroup: typ,
			}
			if typ == "node" || typ == "nodejs" || typ == "python" {
				item.Warnings = []string{
					a.source.t.Get(
						"the 1Panel Runtime is restored as AcePanel Compose; dependencies are not installed automatically and the target remains stopped",
					),
				}
			}
			for _, websiteIndex := range runtimeWebsites[name] {
				items[websiteIndex].DependsOn = append(items[websiteIndex].DependsOn, key)
			}
			items = append(items, item)
		}
	}

	appNames := make(map[string]bool)
	if len(deploymentWebsites) > 0 {
		query := lo.Assign(
			map[string]any{"page": 1, "pageSize": 10000},
			map[string]any{"name": "", "tags": []string{}, "all": true},
		)
		if data, loadErr := a.call(ctx, http.MethodPost, "/api/v2/apps/installed/search", query); loadErr == nil {
			for _, row := range a.rows(data) {
				id := cast.ToString(row["id"])
				websiteIndexes := deploymentWebsites[id]
				if id == "" || len(websiteIndexes) == 0 {
					continue
				}
				name := cast.ToString(row["name"])
				appKey := cast.ToString(row["appKey"])
				if name == "" || appKey == "" {
					continue
				}
				key := "project" + ":" + base64.RawURLEncoding.EncodeToString([]byte("appstore:"+id))
				item := types.MigrationSourceItem{
					Key: key, Type: "project", Subtype: "appstore", Name: name,
					Status: a.status(cast.ToString(row["status"])), Supported: true,
					Warnings:   []string{a.source.t.Get("the 1Panel AppStore workload is restored as an AcePanel Compose project")},
					Features:   []string{"compose", "environment", "images", "volumes", "binds", "proxy"},
					TargetName: name, TargetPath: name, SourceID: id,
					SourcePath: cast.ToString(row["path"]), SourceGroup: appKey,
				}
				for _, websiteIndex := range websiteIndexes {
					items[websiteIndex].DependsOn = append(items[websiteIndex].DependsOn, key)
				}
				appNames[name] = true
				items = append(items, item)
			}
		}
	}
	for _, websiteIndexes := range deploymentWebsites {
		for _, websiteIndex := range websiteIndexes {
			if len(items[websiteIndex].DependsOn) == 0 {
				items[websiteIndex].Blockers = append(items[websiteIndex].Blockers, a.source.t.Get("the related AppStore workload could not be read from 1Panel"))
			}
		}
	}

	for _, databaseType := range []string{"mysql", "mariadb", "postgresql"} {
		data, loadErr := a.call(ctx, http.MethodGet, "/api/v2/databases/db/list/"+databaseType, nil)
		if loadErr != nil {
			continue
		}
		for _, server := range a.rows(data) {
			typ := strings.ToLower(cast.ToString(server["type"]))
			if typ != "mysql" && typ != "mariadb" && typ != "postgresql" {
				continue
			}
			if strings.EqualFold(cast.ToString(server["from"]), "remote") {
				continue
			}
			serverName := cast.ToString(server["database"])
			serverID := cast.ToString(server["id"])
			if serverName == "" || serverID == "" {
				continue
			}
			path := "/api/v2/databases/search"
			if typ == "postgresql" {
				path = "/api/v2/databases/pg/search"
			}
			dbData, dbErr := a.call(ctx, http.MethodPost, path, lo.Assign(map[string]any{"page": 1, "pageSize": 10000}, map[string]any{
				"info": "", "database": serverName, "orderBy": "createdAt", "order": "descending",
			}))
			if dbErr != nil {
				continue
			}
			for _, row := range a.rows(dbData) {
				id := cast.ToString(row["id"])
				name := cast.ToString(row["name"])
				if id == "" || name == "" {
					continue
				}
				databaseKey := "database" + ":" + base64.RawURLEncoding.EncodeToString([]byte(typ+":"+serverID+":"+id))
				items = append(items, types.MigrationSourceItem{
					Key: databaseKey, Type: "database", Subtype: typ, Name: name,
					Status: "running", Supported: true,
					Warnings: []string{a.source.t.Get("verify database compatibility before importing across products or major versions")},
					Features: []string{"schema", "data", "user"}, TargetName: name,
					SourceID: id, SourceGroup: serverName,
				})
				for websiteIndex, databaseID := range websiteDatabases {
					databaseType := websiteDatabaseTypes[websiteIndex]
					typeMatches := databaseType == "" || databaseType == typ || databaseType == "mysql" && typ == "mariadb"
					if databaseID == id && typeMatches {
						items[websiteIndex].DependsOn = append(items[websiteIndex].DependsOn, databaseKey)
					}
				}
			}
		}
	}

	composeQuery := lo.Assign(map[string]any{"page": 1, "pageSize": 10000}, map[string]any{"info": "", "excludeAppStore": true})
	if data, loadErr := a.call(ctx, http.MethodPost, "/api/v2/containers/compose/search", composeQuery); loadErr == nil {
		for _, row := range a.rows(data) {
			name := cast.ToString(row["name"])
			if name == "" || runtimeNames[name] || appNames[name] {
				continue
			}
			status := "stopped"
			if cast.ToInt64(row["runningCount"]) > 0 {
				status = "running"
			}
			warnings := []string{
				a.source.t.Get("absolute bind paths are rewritten to the AcePanel managed directory; special mounts require manual verification"),
			}
			if strings.Contains(cast.ToString(row["configFile"]), ",") {
				warnings = append(warnings, a.source.t.Get("this Compose project uses multiple config files; only the primary config file is migrated automatically"))
			}
			items = append(items, types.MigrationSourceItem{
				Key: "compose" + ":" + base64.RawURLEncoding.EncodeToString([]byte(name)), Type: "compose", Subtype: "compose", Name: name,
				Status: status, Supported: true,
				Warnings:   warnings,
				Features:   []string{"compose", "environment", "images", "volumes", "binds", "networks"},
				TargetName: name, TargetPath: name, SourceID: name,
				SourcePath: lo.CoalesceOrEmpty(cast.ToString(row["configFile"]), cast.ToString(row["path"]), cast.ToString(row["workdir"])),
			})
		}
	}

	if data, loadErr := a.call(ctx, http.MethodPost, "/api/v2/containers/search", lo.Assign(map[string]any{"page": 1, "pageSize": 10000}, map[string]any{
		"name": "", "state": "all", "orderBy": "createdAt", "order": "descending", "excludeAppStore": true,
	})); loadErr == nil {
		for _, row := range a.rows(data) {
			name := strings.TrimPrefix(cast.ToString(row["name"]), "/")
			id := cast.ToString(row["containerID"])
			if name == "" || id == "" || cast.ToBool(row["isFromApp"]) || cast.ToBool(row["isFromCompose"]) {
				continue
			}
			items = append(items, types.MigrationSourceItem{
				Key: "container" + ":" + base64.RawURLEncoding.EncodeToString([]byte(id)), Type: "container", Subtype: "docker", Name: name,
				Status: a.status(cast.ToString(row["state"])), Supported: true,
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

func (a *onePanelMigrationAdapter) Detail(ctx context.Context, item types.MigrationSourceItem) (*types.MigrationSourceDetail, error) {
	var err error
	detail := &types.MigrationSourceDetail{Item: item}
	switch item.Type {
	case "website":
		data, err := a.call(ctx, http.MethodGet, "/api/v2/websites/"+item.SourceID, nil)
		if err != nil {
			return nil, err
		}
		sourceWebsite := cast.ToStringMap(data)
		sitePath := cast.ToString(sourceWebsite["sitePath"])
		if sitePath == "" {
			sitePath = item.SourcePath
		}
		website := &types.MigrationWebsiteDetail{
			Type: strings.ToLower(cast.ToString(sourceWebsite["type"])), Path: filepath.Join(sitePath, "index"),
			Domains: []string{cast.ToString(sourceWebsite["primaryDomain"])},
			Listens: []string{"80"}, Index: []string{"index.php", "index.html", "index.htm"},
			Remark: cast.ToString(sourceWebsite["remark"]), ExpireAt: a.source.parseTime(cast.ToString(sourceWebsite["expireDate"])),
			Enabled: item.Status == "running", OpenBasedir: cast.ToBool(sourceWebsite["openBaseDir"]),
		}
		website.Root = website.Path
		if siteDir := cast.ToString(sourceWebsite["siteDir"]); siteDir != "" && siteDir != "/" {
			website.Root = filepath.Join(website.Path, strings.TrimPrefix(siteDir, "/"))
		}
		if website.Type == "deployment" {
			website.Type = "static"
			if proxy := cast.ToString(sourceWebsite["proxy"]); proxy != "" {
				if !strings.Contains(proxy, "://") {
					proxy = "http://" + proxy
				}
				website.Type = "proxy"
				website.Proxies = append(website.Proxies, types.MigrationProxy{Location: "/", Pass: proxy, HTTPVersion: "1.1"})
			}
		}
		if website.Type == "runtime" && strings.EqualFold(cast.ToString(sourceWebsite["runtimeType"]), "php") {
			website.Type = "php"
		}
		if website.Type == "runtime" && !strings.EqualFold(cast.ToString(sourceWebsite["runtimeType"]), "php") {
			if proxy := cast.ToString(sourceWebsite["proxy"]); proxy != "" {
				if !strings.Contains(proxy, "://") && !strings.HasPrefix(proxy, "unix:") {
					proxy = "http://" + proxy
				}
				website.Type = "proxy"
				website.Proxies = append(website.Proxies, types.MigrationProxy{Location: "/", Pass: proxy, HTTPVersion: "1.1"})
			}
		}
		if website.Type != "php" && website.Type != "proxy" && website.Type != "static" {
			website.Type = "static"
		}

		if data, loadErr := a.call(ctx, http.MethodGet, "/api/v2/websites/domains/"+item.SourceID, nil); loadErr == nil {
			var domains []string
			var listens []string
			for _, row := range a.rows(data) {
				domain := cast.ToString(row["domain"])
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
		if data, loadErr := a.call(
			ctx,
			http.MethodPost,
			"/api/v2/websites/rewrite",
			map[string]any{"websiteId": cast.ToUint(item.SourceID), "name": "current"},
		); loadErr == nil {
			website.Rewrite = cast.ToString(cast.ToStringMap(data)["content"])
		}
		if data, loadErr := a.call(ctx, http.MethodPost, "/api/v2/websites/proxies", map[string]any{"id": cast.ToUint(item.SourceID)}); loadErr == nil {
			for _, row := range a.rows(data) {
				if !cast.ToBool(row["enable"]) && cast.ToString(row["enable"]) != "" {
					continue
				}
				pass := cast.ToString(row["proxyPass"])
				if pass == "" {
					continue
				}
				location := lo.CoalesceOrEmpty(cast.ToString(row["match"]), "/")
				if modifier := cast.ToString(row["modifier"]); modifier != "" {
					location = modifier + " " + location
				}
				proxy := types.MigrationProxy{
					Location: location, Pass: pass, Host: cast.ToString(row["proxyHost"]),
					Replaces: cast.ToStringMapString(row["replaces"]), HTTPVersion: "1.1",
				}
				if cast.ToBool(row["sni"]) {
					proxy.SNI = cast.ToString(row["proxySSLName"])
				}
				website.Proxies = append(website.Proxies, proxy)
			}
		}
		if data, loadErr := a.call(
			ctx,
			http.MethodPost,
			"/api/v2/websites/redirect",
			map[string]any{"websiteId": cast.ToUint(item.SourceID)},
		); loadErr == nil {
			for _, row := range a.rows(data) {
				if !cast.ToBool(row["enable"]) {
					continue
				}
				redirectType := strings.ToLower(cast.ToString(row["type"]))
				from := ""
				switch redirectType {
				case "domain":
					redirectType = "host"
					from = strings.Join(a.source.strings(row["domains"]), " ")
				case "path":
					redirectType = "url"
					from = cast.ToString(row["path"])
				default:
					continue
				}
				statusCode := cast.ToInt(row["redirect"])
				if statusCode == 0 {
					statusCode = http.StatusMovedPermanently
				}
				website.Redirects = append(website.Redirects, types.MigrationRedirect{
					Type: redirectType, From: from, To: cast.ToString(row["target"]),
					KeepURI: cast.ToBool(row["keepPath"]), StatusCode: statusCode,
				})
			}
		}
		if data, loadErr := a.call(ctx, http.MethodGet, "/api/v2/websites/"+item.SourceID+"/https", nil); loadErr == nil {
			https := cast.ToStringMap(data)
			ssl := cast.ToStringMap(https["SSL"])
			website.SSL = cast.ToBool(https["enable"])
			website.SSLCert = cast.ToString(ssl["pem"])
			website.SSLKey = cast.ToString(ssl["privateKey"])
			website.SSLProtocols = a.source.strings(https["SSLProtocol"])
			website.HSTS = cast.ToBool(https["hsts"])
			website.HTTPRedirect = strings.EqualFold(cast.ToString(https["httpConfig"]), "HTTPToHTTPS")
			for _, port := range a.source.strings(https["httpsPorts"]) {
				if !slices.Contains(website.Listens, port) {
					website.Listens = append(website.Listens, port)
				}
				if !slices.Contains(website.SSLListens, port) {
					website.SSLListens = append(website.SSLListens, port)
				}
			}
			for _, port := range strings.Split(cast.ToString(https["httpsPort"]), ",") {
				port = strings.TrimSpace(port)
				if port != "" && !slices.Contains(website.Listens, port) {
					website.Listens = append(website.Listens, port)
				}
				if port != "" && !slices.Contains(website.SSLListens, port) {
					website.SSLListens = append(website.SSLListens, port)
				}
			}
		}
		if cast.ToBool(sourceWebsite["IPV6"]) {
			for _, listen := range slices.Clone(website.Listens) {
				if _, portErr := strconv.ParseUint(listen, 10, 16); portErr != nil {
					continue
				}
				ipv6Listen := "[::]:" + listen
				if !slices.Contains(website.Listens, ipv6Listen) {
					website.Listens = append(website.Listens, ipv6Listen)
				}
				if slices.Contains(website.SSLListens, listen) {
					website.SSLListens = append(website.SSLListens, ipv6Listen)
				}
			}
		}
		if runtimeID := cast.ToString(sourceWebsite["runtimeID"]); runtimeID != "" && runtimeID != "0" {
			if runtime, runtimeErr := a.call(ctx, http.MethodGet, "/api/v2/runtimes/"+runtimeID, nil); runtimeErr == nil {
				website.PHP = a.source.runtimeVersion(cast.ToString(cast.ToStringMap(runtime)["version"]))
			}
		}
		detail.Website = website
	case "database":
		database := &types.MigrationDatabaseDetail{
			Type: item.Subtype, Server: item.SourceGroup, Name: item.Name, Host: "localhost",
		}
		if servers, loadErr := a.call(ctx, http.MethodGet, "/api/v2/databases/db/list/"+item.Subtype, nil); loadErr == nil {
			server, exists := lo.Find(a.rows(servers), func(server map[string]any) bool {
				return cast.ToString(server["database"]) == item.SourceGroup
			})
			if exists {
				database.Version = cast.ToString(server["version"])
			}
		}
		path := "/api/v2/databases/search"
		if item.Subtype == "postgresql" {
			path = "/api/v2/databases/pg/search"
		}
		var databases any
		databases, err = a.call(ctx, http.MethodPost, path, lo.Assign(map[string]any{"page": 1, "pageSize": 10000}, map[string]any{
			"info": item.Name, "database": item.SourceGroup,
			"orderBy": "createdAt", "order": "descending",
		}))
		if err == nil {
			row, exists := lo.Find(a.rows(databases), func(row map[string]any) bool {
				return cast.ToString(row["id"]) == item.SourceID
			})
			if exists {
				database.Username = cast.ToString(row["username"])
				database.Password = cast.ToString(row["password"])
			}
		}
		if item.Subtype == "mysql" || item.Subtype == "mariadb" {
			users, usersErr := a.call(ctx, http.MethodPost, "/api/v2/databases/users/search", map[string]any{"database": item.SourceGroup})
			grants, grantsErr := a.call(ctx, http.MethodPost, "/api/v2/databases/grants/search", map[string]any{"database": item.SourceGroup})
			if usersErr == nil && grantsErr == nil {
				grant, exists := lo.Find(a.rows(grants), func(row map[string]any) bool {
					return cast.ToString(row["database"]) == item.Name
				})
				if exists {
					user, found := lo.Find(a.rows(users), func(row map[string]any) bool {
						return cast.ToString(row["username"]) == cast.ToString(grant["username"]) &&
							cast.ToString(row["host"]) == cast.ToString(grant["host"])
					})
					if found {
						database.Username = cast.ToString(user["username"])
						database.Password = cast.ToString(user["password"])
						database.Host = lo.CoalesceOrEmpty(cast.ToString(user["host"]), "localhost")
					}
				}
			}
		}
		database.PasswordOK = database.Password != ""
		detail.Database = database
	case "project":
		if item.Subtype == "appstore" {
			var data any
			data, err = a.call(ctx, http.MethodPost, "/api/v2/apps/installed/search", lo.Assign(map[string]any{"page": 1, "pageSize": 10000}, map[string]any{
				"name": item.Name, "tags": []string{}, "all": true,
			}))
			if err != nil {
				break
			}
			row, exists := lo.Find(a.rows(data), func(row map[string]any) bool {
				return cast.ToString(row["id"]) == item.SourceID
			})
			if !exists {
				err = errors.New("appstore workload no longer exists")
				break
			}
			appPath := lo.CoalesceOrEmpty(cast.ToString(row["path"]), item.SourcePath)
			composePath := filepath.Join(appPath, "docker-compose.yml")
			composeContent := cast.ToString(row["dockerCompose"])
			if composeContent == "" {
				if content, loadErr := a.call(ctx, http.MethodPost, "/api/v2/files/content", map[string]any{
					"path": composePath, "isDetail": false,
				}); loadErr == nil {
					composeContent = cast.ToString(cast.ToStringMap(content)["content"])
				}
			}
			envs := a.source.environment(row["env"])
			if content, loadErr := a.call(ctx, http.MethodPost, "/api/v2/files/content", map[string]any{
				"path": filepath.Join(appPath, ".env"), "isDetail": false,
			}); loadErr == nil {
				envs = a.source.environment(cast.ToString(cast.ToStringMap(content)["content"]))
			}
			running := item.Status == "running"
			detail.Project = &types.MigrationProjectDetail{
				Type: types.ProjectTypeGeneral, Path: appPath, WorkingDir: appPath,
				User: "www", Running: running, Enabled: running,
			}
			detail.Compose = &types.MigrationComposeDetail{
				Path: composePath, Compose: composeContent, Envs: envs, Running: running,
			}
		} else {
			var data any
			data, err = a.call(ctx, http.MethodGet, "/api/v2/runtimes/"+item.SourceID, nil)
			if err != nil {
				break
			}
			runtime := cast.ToStringMap(data)
			codeDir := lo.CoalesceOrEmpty(
				cast.ToString(runtime["codeDir"]),
				cast.ToString(runtime["path"]),
				item.SourcePath,
			)
			detail.Project = &types.MigrationProjectDetail{
				Type: a.source.projectType(strings.TrimPrefix(item.Subtype, "runtime_")),
				Path: codeDir, WorkingDir: codeDir, User: "www", Restart: "on-failure",
				Environments: a.source.environment(runtime["environments"]),
				Running:      item.Status == "running", Enabled: item.Status == "running",
			}
			composePath := ""
			if runtimePath := cast.ToString(runtime["path"]); runtimePath != "" {
				composePath = filepath.Join(runtimePath, "docker-compose.yml")
			}
			detail.Compose = &types.MigrationComposeDetail{
				Path: composePath, Envs: detail.Project.Environments, Running: detail.Project.Running,
			}
			if detail.Project.Type == types.ProjectTypeNodejs || detail.Project.Type == types.ProjectTypePython {
				detail.Compose.Running = false
			}
			if composePath != "" {
				if content, loadErr := a.call(ctx, http.MethodPost, "/api/v2/files/content", map[string]any{
					"path": composePath, "isDetail": false,
				}); loadErr == nil {
					detail.Compose.Compose = cast.ToString(cast.ToStringMap(content)["content"])
				}
				if content, loadErr := a.call(ctx, http.MethodPost, "/api/v2/files/content", map[string]any{
					"path": filepath.Join(filepath.Dir(composePath), ".env"), "isDetail": false,
				}); loadErr == nil {
					detail.Compose.Envs = a.source.environment(cast.ToString(cast.ToStringMap(content)["content"]))
				}
			}
		}
	case "container":
		var raw any
		raw, err = a.call(ctx, http.MethodPost, "/api/v2/containers/inspect", map[string]any{
			"id": item.SourceID, "type": "container",
		})
		if err == nil {
			detail.Container = a.source.parseContainerDetail(raw, &item)
		}
	case "compose":
		var data any
		query := lo.Assign(map[string]any{"page": 1, "pageSize": 10000}, map[string]any{"info": item.Name})
		data, err = a.call(ctx, http.MethodPost, "/api/v2/containers/compose/search", query)
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
		composePath := lo.CoalesceOrEmpty(cast.ToString(row["configFile"]), cast.ToString(row["path"]), item.SourcePath)
		if strings.Contains(composePath, ",") {
			composePath = strings.TrimSpace(strings.Split(composePath, ",")[0])
		}
		compose := &types.MigrationComposeDetail{
			Path: composePath, Envs: a.source.environment(row["env"]), Running: item.Status == "running",
		}
		if compose.Compose == "" && compose.Path != "" {
			if content, loadErr := a.call(ctx, http.MethodPost, "/api/v2/files/content", map[string]any{
				"path": compose.Path, "isDetail": false,
			}); loadErr == nil {
				compose.Compose = cast.ToString(cast.ToStringMap(content)["content"])
			}
		}
		if compose.Path != "" {
			envPath := filepath.Join(filepath.Dir(compose.Path), ".env")
			if content, loadErr := a.call(ctx, http.MethodPost, "/api/v2/files/content", map[string]any{
				"path": envPath, "isDetail": false,
			}); loadErr == nil {
				compose.Envs = a.source.environment(cast.ToString(cast.ToStringMap(content)["content"]))
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

func (a *onePanelMigrationAdapter) SetRunning(ctx context.Context, detail *types.MigrationSourceDetail, running bool) error {
	operate := "stop"
	if running {
		operate = "start"
	}
	item := detail.Item
	switch item.Type {
	case "website":
		_, err := a.call(ctx, http.MethodPost, "/api/v2/websites/operate", map[string]any{"id": cast.ToUint(item.SourceID), "operate": operate})
		return err
	case "container":
		taskID := fmt.Sprintf("acepanel-migration-container-%s-%d", operate, time.Now().UnixNano())
		if _, err := a.call(ctx, http.MethodPost, "/api/v2/containers/operate", map[string]any{
			"names": []string{item.Name}, "operation": operate, "taskID": taskID,
		}); err != nil {
			return err
		}
		return a.waitTask(ctx, taskID)
	case "compose":
		operation := "down"
		if running {
			operation = "up"
		}
		_, err := a.call(
			ctx,
			http.MethodPost,
			"/api/v2/containers/compose/operate",
			map[string]any{"name": item.Name, "path": item.SourcePath, "operation": operation},
		)
		return err
	case "project":
		if item.Subtype == "appstore" {
			_, err := a.call(ctx, http.MethodPost, "/api/v2/apps/installed/op", map[string]any{
				"installId": cast.ToUint(item.SourceID), "operate": operate,
			})
			return err
		}
		runtimeOperate := "down"
		if running {
			runtimeOperate = "up"
		}
		_, err := a.call(ctx, http.MethodPost, "/api/v2/runtimes/operate", map[string]any{"ID": cast.ToUint(item.SourceID), "operate": runtimeOperate})
		return err
	default:
		return nil
	}
}

func (a *onePanelMigrationAdapter) Prepare(ctx context.Context, detail *types.MigrationSourceDetail) (*types.MigrationArtifact, error) {
	item := detail.Item
	if item.Type == "project" && item.Subtype == "appstore" {
		recordPath, err := a.backup(ctx, "app", item.SourceGroup, item.Name)
		if err != nil {
			return nil, err
		}
		artifact := &types.MigrationArtifact{RemotePath: recordPath, FileName: filepath.Base(recordPath), Kind: "appstore"}
		return a.addComposeImages(ctx, detail.Compose, artifact, item.Name)
	}
	if item.Type == "project" {
		sourcePath := item.SourcePath
		if detail.Project != nil && detail.Project.Path != "" {
			sourcePath = detail.Project.Path
		}
		artifact, err := a.archiveDirectory(ctx, item, sourcePath, "code")
		if err != nil || detail.Compose == nil {
			return artifact, err
		}
		runtimePath := filepath.Dir(detail.Compose.Path)
		if runtimePath != "" && runtimePath != "." && filepath.Clean(runtimePath) != filepath.Clean(sourcePath) {
			runtimeArtifact, archiveErr := a.archiveDirectory(ctx, item, runtimePath, "runtime")
			if archiveErr != nil {
				return nil, archiveErr
			}
			artifact.RemotePaths = []string{artifact.RemotePath, runtimeArtifact.RemotePath}
			artifact.Kind = "project_bundle"
		}
		return a.addComposeImages(ctx, detail.Compose, artifact, item.Name)
	}

	backupType := item.Type
	name := item.Name
	detailName := item.Name
	if item.Type == "database" && detail.Database != nil {
		backupType = detail.Database.Type
		name = detail.Database.Server
		detailName = detail.Database.Name
	}
	if item.Type == "container" || item.Type == "compose" {
		detailName = ""
	}
	recordPath, err := a.backup(ctx, backupType, name, detailName)
	if err != nil {
		return nil, err
	}
	artifact := &types.MigrationArtifact{RemotePath: recordPath, FileName: filepath.Base(recordPath), Kind: item.Type}
	if item.Type != "container" || detail.Container == nil || detail.Container.Image == "" {
		if item.Type == "compose" && detail.Compose != nil {
			return a.addComposeImages(ctx, detail.Compose, artifact, item.Name)
		}
		return artifact, nil
	}

	remoteDir := "/opt/1panel/backup/acepanel-migration"
	_, _ = a.call(ctx, http.MethodPost, "/api/v2/files", map[string]any{"path": remoteDir, "isDir": true, "mode": 0755})
	imageName := "acepanel-migration-" + a.source.safeFileName(item.Name)
	tag := strconv.FormatInt(time.Now().Unix(), 10)
	tagName := imageName + ":" + tag
	commitTaskID := fmt.Sprintf("acepanel-migration-commit-%d", time.Now().UnixNano())
	_, commitErr := a.call(ctx, http.MethodPost, "/api/v2/containers/commit", map[string]any{
		"containerID": item.SourceID, "containerName": item.Name, "newImageName": tagName,
		"comment": "AcePanel migration", "author": "AcePanel", "pause": true, "taskID": commitTaskID,
	})
	if commitErr == nil {
		if commitErr = a.waitTask(ctx, commitTaskID); commitErr != nil {
			return nil, commitErr
		}
		detail.Container.Image = tagName
	} else {
		return nil, commitErr
	}
	exportName := fmt.Sprintf("%s-%d", imageName, time.Now().UnixNano())
	exportTaskID := fmt.Sprintf("acepanel-migration-image-save-%d", time.Now().UnixNano())
	_, err = a.call(ctx, http.MethodPost, "/api/v2/containers/image/save", map[string]any{
		"tagName": tagName, "path": remoteDir, "name": exportName, "taskID": exportTaskID,
	})
	if err != nil {
		return nil, err
	}
	if err = a.waitTask(ctx, exportTaskID); err != nil {
		return nil, err
	}
	artifact.RemotePaths = []string{recordPath, filepath.Join(remoteDir, exportName+".tar")}
	artifact.Kind = "container_bundle"
	return artifact, nil
}

func (a *onePanelMigrationAdapter) addComposeImages(
	ctx context.Context,
	compose *types.MigrationComposeDetail,
	artifact *types.MigrationArtifact,
	name string,
) (*types.MigrationArtifact, error) {
	images := a.source.composeImages(compose.Compose, compose.Envs)
	if len(images) == 0 {
		return artifact, nil
	}
	remoteDir := "/opt/1panel/backup/acepanel-migration"
	_, _ = a.call(ctx, http.MethodPost, "/api/v2/files", map[string]any{"path": remoteDir, "isDir": true, "mode": 0755})
	paths := append([]string(nil), artifact.RemotePaths...)
	if len(paths) == 0 {
		paths = []string{artifact.RemotePath}
	}
	compose.ImageTags = make(map[string]string, len(images))
	compose.ImageSources = make(map[string]string, len(images))
	i := 0
	for source, image := range images {
		i++
		migrationTag := fmt.Sprintf("acepanel-migration/%s-%d:%d", a.source.safeFileName(name), i, time.Now().UnixNano())
		if _, err := a.call(ctx, http.MethodPost, "/api/v2/containers/image/tag", map[string]any{
			"sourceID": image, "tags": []string{migrationTag},
		}); err != nil {
			return nil, err
		}
		fileName := fmt.Sprintf("%s-image-%d-%d", a.source.safeFileName(name), i, time.Now().UnixNano())
		taskID := fmt.Sprintf("acepanel-migration-image-save-%d", time.Now().UnixNano())
		if _, err := a.call(ctx, http.MethodPost, "/api/v2/containers/image/save", map[string]any{
			"tagName": migrationTag, "path": remoteDir, "name": fileName, "taskID": taskID,
		}); err != nil {
			return nil, err
		}
		if err := a.waitTask(ctx, taskID); err != nil {
			return nil, err
		}
		compose.ImageTags[source] = migrationTag
		compose.ImageSources[source] = image
		paths = append(paths, filepath.Join(remoteDir, fileName+".tar"))
	}
	if len(paths) > 1 {
		artifact.RemotePaths = paths
		artifact.Kind = "compose_bundle"
	}
	return artifact, nil
}

func (a *onePanelMigrationAdapter) archiveDirectory(
	ctx context.Context,
	item types.MigrationSourceItem,
	sourcePath string,
	suffix string,
) (*types.MigrationArtifact, error) {
	if sourcePath == "" || sourcePath == "/" {
		return nil, errors.New("source path is empty")
	}
	remoteDir := "/opt/1panel/backup/acepanel-migration"
	_, _ = a.call(ctx, http.MethodPost, "/api/v2/files", map[string]any{"path": remoteDir, "isDir": true, "mode": 0755})
	name := a.source.safeFileName(item.Name)
	if suffix != "" {
		name += "-" + suffix
	}
	fileName := fmt.Sprintf("%s-%d.tar.gz", name, time.Now().UnixNano())
	taskID := fmt.Sprintf("acepanel-migration-compress-%d", time.Now().UnixNano())
	_, err := a.call(ctx, http.MethodPost, "/api/v2/files/compress", map[string]any{
		"files": []string{sourcePath}, "dst": remoteDir, "type": "tar.gz", "name": fileName, "replace": false, "taskID": taskID,
	})
	if err != nil {
		return nil, err
	}
	if err = a.waitTask(ctx, taskID); err != nil {
		return nil, err
	}
	return &types.MigrationArtifact{RemotePath: filepath.Join(remoteDir, fileName), FileName: fileName, Kind: item.Type}, nil
}

func (a *onePanelMigrationAdapter) waitTask(ctx context.Context, taskID string) error {
	missing := 0
	for {
		query := lo.Assign(map[string]any{"page": 1, "pageSize": 10000}, map[string]any{"taskID": taskID})
		data, err := a.call(ctx, http.MethodPost, "/api/v2/logs/tasks/search", query)
		if err == nil {
			rows := a.rows(data)
			if len(rows) == 0 {
				missing++
			} else {
				status := strings.ToLower(cast.ToString(rows[0]["status"]))
				switch status {
				case "success":
					return nil
				case "failed":
					return fmt.Errorf("source task failed: %s", cast.ToString(rows[0]["errorMsg"]))
				}
			}
		}
		if missing >= 60 {
			return errors.New("source task was not found")
		}
		if err := a.source.wait(ctx, time.Second); err != nil {
			return err
		}
	}
}

func (a *onePanelMigrationAdapter) backup(ctx context.Context, typ, name, detailName string) (string, error) {
	query := lo.Assign(map[string]any{"page": 1, "pageSize": 10000}, map[string]any{"type": typ, "name": name, "detailName": detailName})
	existing := make(map[string]bool)
	if data, err := a.call(ctx, http.MethodPost, "/api/v2/backups/record/search", query); err == nil {
		for _, row := range a.rows(data) {
			if id := cast.ToString(row["id"]); id != "" {
				existing[id] = true
			}
		}
	}
	_, err := a.call(ctx, http.MethodPost, "/api/v2/backups/backup", map[string]any{
		"type": typ, "name": name, "detailName": detailName, "isImmediate": false, "stopBefore": false,
	})
	if err != nil {
		return "", err
	}
	for attempt := 0; attempt < 180; attempt++ {
		data, loadErr := a.call(ctx, http.MethodPost, "/api/v2/backups/record/search", lo.Assign(map[string]any{"page": 1, "pageSize": 10000}, map[string]any{
			"type": typ, "name": name, "detailName": detailName,
		}))
		if loadErr == nil {
			for _, row := range a.rows(data) {
				if existing[cast.ToString(row["id"])] {
					continue
				}
				status := strings.ToLower(cast.ToString(row["status"]))
				if status == "failed" {
					return "", fmt.Errorf("source backup failed: %s", cast.ToString(row["message"]))
				}
				if status != "success" {
					continue
				}
				fileDir := cast.ToString(row["fileDir"])
				fileName := cast.ToString(row["fileName"])
				accountID := cast.ToUint(cast.ToString(row["downloadAccountID"]))
				if accountID == 0 || fileDir == "" || fileName == "" {
					continue
				}
				pathData, downloadErr := a.call(ctx, http.MethodPost, "/api/v2/backups/record/download", map[string]any{
					"downloadAccountID": accountID, "fileDir": fileDir, "fileName": fileName,
				})
				if downloadErr != nil {
					return "", downloadErr
				}
				if pathValue, ok := pathData.(string); ok && pathValue != "" {
					return pathValue, nil
				}
			}
		}
		if err := a.source.wait(ctx, time.Second); err != nil {
			return "", err
		}
	}
	return "", errors.New("source backup did not finish in time")
}

func (a *onePanelMigrationAdapter) Download(ctx context.Context, artifact *types.MigrationArtifact, target string) error {
	return a.source.download(ctx, artifact, target, a.downloadFile)
}

func (a *onePanelMigrationAdapter) rows(value any) []map[string]any {
	if values, ok := value.([]any); ok {
		return lo.FilterMap(values, func(value any, _ int) (map[string]any, bool) {
			row := cast.ToStringMap(value)
			return row, len(row) > 0
		})
	}
	if items, ok := cast.ToStringMap(value)["items"]; ok {
		return a.rows(items)
	}
	return nil
}

func (a *onePanelMigrationAdapter) status(value string) string {
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

func (a *onePanelMigrationAdapter) apiResponse(resp *resty.Response, err error) (any, error) {
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
	if code := cast.ToInt64(result["code"]); code != http.StatusOK {
		return nil, fmt.Errorf("source API: %s", cast.ToString(result["message"]))
	}
	if data, ok := result["data"]; ok {
		return data, nil
	}
	return raw, nil
}

func (a *onePanelMigrationAdapter) call(ctx context.Context, method, path string, body any) (any, error) {
	client := resty.New().
		SetBaseURL(strings.TrimRight(a.conn.URL, "/")).
		SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true}).
		SetTimeout(45 * time.Second)
	defer func() { _ = client.Close() }()

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	token := md5.Sum([]byte("1panel" + a.conn.APIKey + timestamp))
	req := client.R().
		SetContext(ctx).
		SetHeader("1Panel-Timestamp", timestamp).
		SetHeader("1Panel-Token", hex.EncodeToString(token[:])).
		SetHeader("Content-Type", "application/json")
	if method == http.MethodGet {
		req.SetQueryParams(cast.ToStringMapString(body))
	} else if body != nil {
		req.SetBody(body)
	}
	return a.apiResponse(req.Execute(method, path))
}

func (a *onePanelMigrationAdapter) downloadFile(ctx context.Context, remotePath, target string) error {
	client := resty.New().
		SetBaseURL(strings.TrimRight(a.conn.URL, "/")).
		SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true}).
		SetTimeout(30 * time.Minute)
	defer func() { _ = client.Close() }()
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	token := md5.Sum([]byte("1panel" + a.conn.APIKey + timestamp))
	resp, err := client.R().
		SetContext(ctx).
		SetResponseDoNotParse(true).
		SetHeader("1Panel-Timestamp", timestamp).
		SetHeader("1Panel-Token", hex.EncodeToString(token[:])).
		SetQueryParam("path", remotePath).
		Get("/api/v2/files/download")
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
