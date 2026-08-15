package data

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/leonelquinteros/gotext"
	"github.com/samber/lo"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/cert"
	"github.com/acepanel/panel/v3/pkg/embed"
	"github.com/acepanel/panel/v3/pkg/io"
	"github.com/acepanel/panel/v3/pkg/punycode"
	"github.com/acepanel/panel/v3/pkg/shell"
	"github.com/acepanel/panel/v3/pkg/systemctl"
	"github.com/acepanel/panel/v3/pkg/tools"
	"github.com/acepanel/panel/v3/pkg/types"
	"github.com/acepanel/panel/v3/pkg/webserver"
	webservertypes "github.com/acepanel/panel/v3/pkg/webserver/types"
)

type websiteRepo struct {
	t       *gotext.Locale
	db      *gorm.DB
	setting biz.SettingRepo
}

func NewWebsiteRepo(db *gorm.DB, t *gotext.Locale, settingRepo biz.SettingRepo) biz.WebsiteRepo {
	return &websiteRepo{
		t:       t,
		db:      db,
		setting: settingRepo,
	}
}

const nginxPHPCacheConfig = `# browser cache
location ~ .*\.(bmp|jpg|jpeg|png|gif|svg|ico|tiff|webp|avif|heif|heic|jxl)$ {
    expires 30d;
    access_log /dev/null;
    error_log /dev/null;
}
location ~ .*\.(js|css|ttf|otf|woff|woff2|eot)$ {
    expires 6h;
    access_log /dev/null;
    error_log /dev/null;
}
# deny sensitive files
location ~ ^/(\.user.ini|\.htaccess|\.git|\.svn|\.env) {
    return 404;
}
`

const apachePHPCacheConfig = `# browser cache
<IfModule mod_expires.c>
    ExpiresActive On
    ExpiresByType image/bmp "access plus 30 days"
    ExpiresByType image/jpeg "access plus 30 days"
    ExpiresByType image/png "access plus 30 days"
    ExpiresByType image/gif "access plus 30 days"
    ExpiresByType image/svg+xml "access plus 30 days"
    ExpiresByType image/x-icon "access plus 30 days"
    ExpiresByType image/tiff "access plus 30 days"
    ExpiresByType image/webp "access plus 30 days"
    ExpiresByType image/avif "access plus 30 days"
    ExpiresByType image/heif "access plus 30 days"
    ExpiresByType image/heic "access plus 30 days"
    ExpiresByType image/jxl "access plus 30 days"
    ExpiresByType text/css "access plus 6 hours"
    ExpiresByType application/javascript "access plus 6 hours"
    ExpiresByType font/ttf "access plus 6 hours"
    ExpiresByType font/otf "access plus 6 hours"
    ExpiresByType font/woff "access plus 6 hours"
    ExpiresByType font/woff2 "access plus 6 hours"
    ExpiresByType application/vnd.ms-fontobject "access plus 6 hours"
</IfModule>
# deny sensitive files
<FilesMatch "^(\.user\.ini|\.htaccess|\.git|\.svn|\.env)">
    Require all denied
</FilesMatch>
`

const nginxSPAConfig = `# single-page application route fallback, remove if not needed
location / {
    try_files $uri $uri/ /index.html;
}
`

const apacheSPAConfig = `# single-page application route fallback, remove if not needed
FallbackResource /index.html
`

func (r *websiteRepo) GetRewrites() (map[string]string, error) {
	webServer, err := r.setting.Get(biz.SettingKeyWebserver)
	if err != nil {
		return make(map[string]string), nil
	}

	entries, err := embed.RewritesFS.ReadDir(filepath.Join("rewrites", webServer))
	if err != nil {
		return make(map[string]string), nil
	}

	rw := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if content, err := embed.RewritesFS.ReadFile(filepath.Join("rewrites", webServer, entry.Name())); err == nil {
			rw[strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))] = string(content)
		}
	}

	return rw, nil
}

func (r *websiteRepo) UpdateDefaultConfig(req *request.WebsiteDefaultConfig) error {
	webServer, err := r.setting.Get(biz.SettingKeyWebserver)
	if err != nil {
		return err
	}
	var htmlPath string
	switch webServer {
	case "nginx":
		htmlPath = filepath.Join(app.Root, "server/nginx/html")
	case "apache":
		htmlPath = filepath.Join(app.Root, "server/apache/htdocs")
	default:
		htmlPath = filepath.Join(app.Root, "server/nginx/html")
	}

	if err = io.Write(filepath.Join(htmlPath, "index.html"), req.Index, 0644); err != nil {
		return err
	}
	if err = io.Write(filepath.Join(htmlPath, "stop.html"), req.Stop, 0644); err != nil {
		return err
	}
	if req.NotFound != "" {
		if err = io.Write(filepath.Join(htmlPath, "404.html"), req.NotFound, 0644); err != nil {
			return err
		}
	}
	if err = r.setting.SetSlice(biz.SettingKeyWebsiteTLSVersions, req.TLSVersions); err != nil {
		return err
	}
	if err = r.setting.Set(biz.SettingKeyWebsiteListenIPv6, strconv.FormatBool(req.ListenIPv6)); err != nil {
		return err
	}

	return r.ReloadWebServer()
}

func (r *websiteRepo) Count() (int64, error) {
	var count int64
	if err := r.db.Model(&biz.Website{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *websiteRepo) Get(id uint) (*types.WebsiteSetting, error) {
	website := new(biz.Website)
	if err := r.db.Where("id", id).First(website).Error; err != nil {
		return nil, err
	}

	vhost, err := r.getVhost(website)
	if err != nil {
		return nil, err
	}

	setting := new(types.WebsiteSetting)
	setting.ID = website.ID
	setting.Name = website.Name
	setting.Type = string(website.Type)
	setting.Path = website.Path
	setting.SSL = website.SSL
	// 监听地址
	setting.Listens = vhost.Listen()
	// 域名
	domains := vhost.ServerName()
	domains = lo.Map(domains, func(d string, _ int) string { return tools.UnwrapIPv6(d) })
	domains, err = punycode.DecodeDomains(domains)
	if err != nil {
		return nil, err
	}
	setting.Domains = domains
	// 运行目录
	setting.Root = vhost.Root()
	// 默认文档
	setting.Index = vhost.Index()
	// 防跨站
	if website.Type == biz.WebsiteTypePHP && io.Exists(filepath.Join(setting.Root, ".user.ini")) {
		userIni, _ := io.Read(filepath.Join(setting.Root, ".user.ini"))
		if strings.Contains(userIni, "open_basedir") {
			setting.OpenBasedir = true
		}
	}
	// SSL
	if setting.SSL {
		sslConfig := vhost.SSLConfig()
		setting.HTTPRedirect = sslConfig.HTTPRedirect
		setting.HSTS = sslConfig.HSTS
		setting.OCSP = sslConfig.OCSP
		setting.SSLProtocols = sslConfig.Protocols
	}
	// 证书
	crt, _ := os.ReadFile(filepath.Join(app.Root, "sites", website.Name, "config", "fullchain.pem"))
	setting.SSLCert = string(crt)
	key, _ := os.ReadFile(filepath.Join(app.Root, "sites", website.Name, "config", "private.key"))
	setting.SSLKey = string(key)
	// 解析证书信息
	if decode, err := cert.ParseCert(crt); err == nil {
		setting.SSLNotBefore = decode.NotBefore.Format(time.DateTime)
		setting.SSLNotAfter = decode.NotAfter.Format(time.DateTime)
		setting.SSLIssuer = decode.Issuer.CommonName
		setting.SSLOCSPServer = decode.OCSPServer
		// 合并 DNSNames 和 IPAddresses
		setting.SSLDNSNames = decode.DNSNames
		for _, ip := range decode.IPAddresses {
			setting.SSLDNSNames = append(setting.SSLDNSNames, ip.String())
		}
	}
	// 访问日志
	if setting.AccessLog = vhost.AccessLog(); setting.AccessLog == "" {
		setting.AccessLog = fmt.Sprintf("%s/sites/%s/log/access.log", app.Root, website.Name)
	}
	// 错误日志
	if setting.ErrorLog = vhost.ErrorLog(); setting.ErrorLog == "" {
		setting.ErrorLog = fmt.Sprintf("%s/sites/%s/log/error.log", app.Root, website.Name)
	}

	// PHP 网站特有
	if phpVhost, ok := vhost.(webservertypes.PHPVhost); ok {
		setting.PHP = phpVhost.PHP()
		// 伪静态
		setting.Rewrite = phpVhost.Config("010-rewrite.conf", webservertypes.ScopeSite)
	}

	// 反向代理网站特有
	if proxyVhost, ok := vhost.(webservertypes.ProxyVhost); ok {
		setting.Upstreams = proxyVhost.Upstreams()
		setting.Proxies = proxyVhost.Proxies()
	}

	// 重定向配置
	if redirectVhost, ok := vhost.(webservertypes.VhostRedirect); ok {
		setting.Redirects = redirectVhost.Redirects()
	}

	// 高级设置（限流限速、真实 IP、基本认证）
	setting.RateLimit = vhost.RateLimit()
	setting.RealIP = vhost.RealIP()
	// 读取基本认证用户列表
	setting.BasicAuth = r.readBasicAuthUsers(website.Name)

	// 自定义配置
	configDir := filepath.Join(app.Root, "sites", website.Name, "config")
	setting.CustomConfigs = r.getCustomConfigs(configDir)

	// 访问统计
	setting.StatEnabled = vhost.Config("021-stats-log.conf", webservertypes.ScopeSite) != ""

	return setting, err
}

func (r *websiteRepo) GetByName(name string) (*types.WebsiteSetting, error) {
	website := new(biz.Website)
	if err := r.db.Where("name", name).First(website).Error; err != nil {
		return nil, err
	}

	return r.Get(website.ID)
}

func (r *websiteRepo) List(typ string, page, limit uint) ([]*biz.Website, int64, error) {
	websites := make([]*biz.Website, 0)
	var total int64

	if err := r.db.Model(&biz.Website{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query := r.db
	if typ != "" && typ != "all" {
		query = query.Where("type = ?", typ)
	}
	if err := query.Order("id DESC").Offset(int((page - 1) * limit)).Limit(int(limit)).Find(&websites).Error; err != nil {
		return nil, 0, err
	}

	// 取证书剩余有效时间和PHP版本
	webServer, wsErr := r.setting.Get(biz.SettingKeyWebserver)
	for _, website := range websites {
		crt, _ := os.ReadFile(filepath.Join(app.Root, "sites", website.Name, "config", "fullchain.pem"))
		if decode, err := cert.ParseCert(crt); err == nil {
			hours := time.Until(decode.NotAfter).Hours()
			website.CertExpire = fmt.Sprintf("%.2f", hours/24)
		}
		if wsErr != nil {
			continue
		}
		vhost, err := r.newVhost(webServer, website)
		if err != nil {
			continue
		}
		if php, ok := vhost.(webservertypes.PHPVhost); ok {
			website.PHP = php.PHP()
		}
		// 获取域名
		if domains, err := punycode.DecodeDomains(vhost.ServerName()); err == nil {
			website.Domains = domains
		}
	}

	return websites, total, nil
}

func (r *websiteRepo) Create(req *request.WebsiteCreate) (*biz.Website, error) {
	w := &biz.Website{
		Name:   req.Name,
		Type:   biz.WebsiteType(req.Type),
		Status: true,
		Path:   req.Path,
		SSL:    false,
		Remark: req.Remark,
	}

	webServer, err := r.setting.Get(biz.SettingKeyWebserver)
	if err != nil {
		return nil, err
	}

	vhost, err := r.getVhost(w)
	if err != nil {
		return nil, err
	}

	// 创建配置文件目录
	if err = os.MkdirAll(filepath.Join(app.Root, "sites", req.Name, "config", "site"), 0600); err != nil {
		return nil, err
	}
	if err = os.MkdirAll(filepath.Join(app.Root, "sites", req.Name, "config", "shared"), 0600); err != nil {
		return nil, err
	}
	// 创建日志目录
	if err = os.MkdirAll(filepath.Join(app.Root, "sites", req.Name, "log"), 0755); err != nil {
		return nil, err
	}

	// 监听地址
	listens := lo.Map(req.Listens, func(listen string, _ int) webservertypes.Listen {
		return webservertypes.Listen{Address: listen}
	})
	if webServer == "nginx" {
		listenIPv6, getErr := r.setting.GetBool(biz.SettingKeyWebsiteListenIPv6, false)
		if getErr != nil {
			return nil, getErr
		}
		if listenIPv6 {
			ipv6Listens := lo.FilterMap(listens, func(listen webservertypes.Listen, _ int) (webservertypes.Listen, bool) {
				port := listen.Address
				if strings.Contains(listen.Address, ":") {
					_, parsedPort, splitErr := net.SplitHostPort(listen.Address)
					if splitErr != nil {
						return webservertypes.Listen{}, false
					}
					port = parsedPort
				}
				value, parseErr := strconv.ParseUint(port, 10, 16)
				if parseErr != nil || value == 0 {
					return webservertypes.Listen{}, false
				}
				return webservertypes.Listen{
					Address: "[::]:" + port,
					Args:    slices.Clone(listen.Args),
				}, true
			})
			listens = lo.UniqBy(slices.Concat(listens, ipv6Listens), func(listen webservertypes.Listen) string {
				return listen.Address
			})
		}
	}
	if err = vhost.SetListen(listens); err != nil {
		return nil, err
	}
	// 域名
	domains, err := punycode.EncodeDomains(req.Domains)
	if err != nil {
		return nil, err
	}
	domains = lo.Map(domains, func(d string, _ int) string { return tools.WrapIPv6(d) })
	if err = vhost.SetServerName(domains); err != nil {
		return nil, err
	}
	// 运行目录
	if err = vhost.SetRoot(req.Path); err != nil {
		return nil, err
	}
	// 日志
	if err = vhost.SetAccessLog(filepath.Join(app.Root, "sites", req.Name, "log", "access.log")); err != nil {
		return nil, err
	}
	if err = vhost.SetErrorLog(filepath.Join(app.Root, "sites", req.Name, "log", "error.log")); err != nil {
		return nil, err
	}
	// 404 页面
	var errorPageConfig string
	switch webServer {
	case "nginx":
		errorPageConfig = `error_page 404 /404.html;`
	case "apache":
		errorPageConfig = `ErrorDocument 404 /404.html`
	}
	if err = vhost.SetConfig("010-error-404.conf", webservertypes.ScopeSite, errorPageConfig); err != nil {
		return nil, err
	}

	// 反向代理支持
	if proxyVhost, ok := vhost.(webservertypes.ProxyVhost); ok {
		if err = proxyVhost.SetProxies([]webservertypes.Proxy{
			{
				Location: "^~ /",
				Pass:     req.Proxy,
			},
		}); err != nil {
			return nil, err
		}
	}

	// PHP 支持
	if phpVhost, ok := vhost.(webservertypes.PHPVhost); ok {
		if err = phpVhost.SetPHP(req.PHP); err != nil {
			return nil, err
		}
		if err = phpVhost.SetIndex([]string{"index.php", "index.html"}); err != nil {
			return nil, err
		}
		if err = phpVhost.SetRawConfig("010-rewrite.conf", webservertypes.ScopeSite, ""); err != nil {
			return nil, err
		}
		cacheConfig := nginxPHPCacheConfig
		if webServer == "apache" {
			cacheConfig = apachePHPCacheConfig
		}
		if err = phpVhost.SetConfig("010-cache.conf", webservertypes.ScopeSite, cacheConfig); err != nil {
			return nil, err
		}
	}

	// 纯静态网站默认写入单页应用（SPA）前端路由回退配置
	if w.Type == biz.WebsiteTypeStatic {
		spaConfig := nginxSPAConfig
		if webServer == "apache" {
			spaConfig = apacheSPAConfig
		}
		if spaConfig != "" {
			if err = vhost.SetRawConfig("799-spa.conf", webservertypes.ScopeSite, spaConfig); err != nil {
				return nil, err
			}
		}
	}

	// 初始化网站目录
	if err = os.MkdirAll(req.Path, 0755); err != nil {
		return nil, err
	}
	var index []byte
	switch app.Locale {
	case "zh_CN":
		index, err = embed.WebsiteFS.ReadFile(filepath.Join("website", "index_zh_CN.html"))
	case "zh_TW":
		index, err = embed.WebsiteFS.ReadFile(filepath.Join("website", "index_zh_TW.html"))
	default:
		index, err = embed.WebsiteFS.ReadFile(filepath.Join("website", "index.html"))
	}
	if err != nil {
		return nil, errors.New(r.t.Get("failed to get index template file: %v", err))
	}
	if err = io.Write(filepath.Join(req.Path, "index.html"), string(index), 0644); err != nil {
		return nil, err
	}
	var notFound []byte

	// 如果存在自定义 404 页面，则使用自定义的
	var custom404Path string
	switch webServer {
	case "nginx":
		custom404Path = filepath.Join(app.Root, "server/nginx/html/404.html")
	case "apache":
		custom404Path = filepath.Join(app.Root, "server/apache/htdocs/404.html")
	}
	if io.Exists(custom404Path) {
		notFound, _ = os.ReadFile(custom404Path)
	} else {
		switch app.Locale {
		case "zh_CN":
			notFound, _ = embed.WebsiteFS.ReadFile(filepath.Join("website", "404_zh_CN.html"))
		case "zh_TW":
			notFound, _ = embed.WebsiteFS.ReadFile(filepath.Join("website", "404_zh_TW.html"))
		default:
			notFound, _ = embed.WebsiteFS.ReadFile(filepath.Join("website", "404.html"))
		}
	}

	if err = io.Write(filepath.Join(req.Path, "404.html"), string(notFound), 0644); err != nil {
		return nil, err
	}

	// 写配置
	if err = vhost.SetConfig("001-acme.conf", webservertypes.ScopeSite, ""); err != nil {
		return nil, err
	}

	// 访问统计（nginx 默认启用）
	if webServer == "nginx" {
		if err = r.enableStat(vhost, req.Name); err != nil {
			return nil, err
		}
	}

	if err = vhost.Save(); err != nil {
		return nil, err
	}

	if err = io.Write(filepath.Join(app.Root, "sites", req.Name, "config", "fullchain.pem"), "", 0600); err != nil {
		return nil, err
	}
	if err = io.Write(filepath.Join(app.Root, "sites", req.Name, "config", "private.key"), "", 0600); err != nil {
		return nil, err
	}

	// 设置目录权限
	// sites/site_name 0755 root
	// sites/site_name/config 0600 root
	// sites/site_name/log 0701 root
	// sites/site_name/public 0755 www
	if err = io.Chmod(filepath.Join(app.Root, "sites", req.Name), 0755); err != nil {
		return nil, err
	}
	if err = io.Chmod(req.Path, 0755); err != nil {
		return nil, err
	}
	if err = io.Chown(req.Path, "www", "www"); err != nil {
		return nil, err
	}
	if err = io.Chmod(filepath.Join(app.Root, "sites", req.Name, "log"), 0701); err != nil {
		return nil, err
	}
	if err = io.Chmod(filepath.Join(app.Root, "sites", req.Name, "config"), 0600); err != nil {
		return nil, err
	}

	// PHP 网站默认开启防跨站
	if req.Type == "php" {
		userIni := filepath.Join(req.Path, ".user.ini")
		if !io.Exists(userIni) {
			if err = io.Write(userIni, fmt.Sprintf("open_basedir=%s:/tmp/", req.Path), 0644); err != nil {
				return nil, err
			}
		}
		_, _ = shell.Execf(`chattr +i '%s'`, userIni)
	}

	// 创建面板网站
	if err = r.db.Create(w).Error; err != nil {
		return nil, err
	}

	return w, nil
}

func (r *websiteRepo) Update(req *request.WebsiteUpdate) (*biz.Website, error) {
	website := new(biz.Website)
	if err := r.db.Where("id", req.ID).First(website).Error; err != nil {
		return nil, err
	}

	if err := r.applyUpdate(req, website); err != nil {
		return nil, err
	}

	return website, nil
}

func (r *websiteRepo) SwitchType(req *request.WebsiteSwitchType) (*biz.Website, error) {
	website := new(biz.Website)
	if err := r.db.Where("id", req.ID).First(website).Error; err != nil {
		return nil, err
	}

	targetType := biz.WebsiteType(req.Type)
	if targetType == website.Type {
		return nil, errors.New(r.t.Get("website type is unchanged"))
	}

	setting, err := r.Get(req.ID)
	if err != nil {
		return nil, err
	}
	webServer, err := r.setting.Get(biz.SettingKeyWebserver)
	if err != nil {
		return nil, err
	}

	customConfigs := lo.Map(setting.CustomConfigs, func(config types.WebsiteCustomConfig, _ int) request.WebsiteCustomConfig {
		return request.WebsiteCustomConfig{
			Name:    config.Name,
			Scope:   config.Scope,
			Content: config.Content,
		}
	})

	update := &request.WebsiteUpdate{
		ID:            website.ID,
		Listens:       setting.Listens,
		Domains:       setting.Domains,
		Path:          setting.Path,
		Root:          setting.Root,
		Index:         []string{"index.html"},
		SSL:           setting.SSL,
		SSLCert:       setting.SSLCert,
		SSLKey:        setting.SSLKey,
		HSTS:          setting.HSTS,
		OCSP:          setting.OCSP,
		HTTPRedirect:  setting.HTTPRedirect,
		SSLProtocols:  setting.SSLProtocols,
		Redirects:     setting.Redirects,
		StatEnabled:   setting.StatEnabled,
		AccessLog:     setting.AccessLog,
		ErrorLog:      setting.ErrorLog,
		RateLimit:     setting.RateLimit,
		RealIP:        setting.RealIP,
		BasicAuth:     setting.BasicAuth,
		CustomConfigs: customConfigs,
	}
	switch targetType {
	case biz.WebsiteTypePHP:
		update.Index = []string{"index.php", "index.html"}
		update.PHP = req.PHP
		update.OpenBasedir = true
	case biz.WebsiteTypeProxy:
		update.Proxies = []webservertypes.Proxy{{
			Location: "^~ /",
			Pass:     req.Proxy,
		}}
	}

	configDir := filepath.Join(app.Root, "sites", website.Name, "config")
	backupDir := fmt.Sprintf("%s.switch-backup-%d", configDir, time.Now().UnixNano())
	if err = os.Rename(configDir, backupDir); err != nil {
		return nil, err
	}
	if err = os.MkdirAll(filepath.Join(configDir, "site"), 0600); err != nil {
		_ = io.Remove(configDir)
		_ = os.Rename(backupDir, configDir)
		return nil, err
	}
	if err = os.MkdirAll(filepath.Join(configDir, "shared"), 0600); err != nil {
		_ = io.Remove(configDir)
		_ = os.Rename(backupDir, configDir)
		return nil, err
	}

	oldType := website.Type
	oldPath := website.Path
	oldSSL := website.SSL
	oldUpdatedAt := website.UpdatedAt
	userIniPath := filepath.Join(setting.Root, ".user.ini")
	oldUserIni, userIniErr := os.ReadFile(userIniPath)
	if userIniErr != nil && !os.IsNotExist(userIniErr) {
		_ = io.Remove(configDir)
		_ = os.Rename(backupDir, configDir)
		return nil, userIniErr
	}
	oldUserIniExists := userIniErr == nil
	databaseUpdated := false
	restore := func(switchErr error) (*biz.Website, error) {
		var restoreErr error
		restoreErr = errors.Join(restoreErr, io.Remove(configDir))
		restoreErr = errors.Join(restoreErr, os.Rename(backupDir, configDir))
		if databaseUpdated {
			restoreErr = errors.Join(restoreErr, r.db.Model(&biz.Website{}).Where("id = ?", website.ID).UpdateColumns(map[string]any{
				"type":       oldType,
				"path":       oldPath,
				"ssl":        oldSSL,
				"updated_at": oldUpdatedAt,
			}).Error)
		}
		restoreErr = errors.Join(restoreErr, io.Remove(userIniPath))
		if oldUserIniExists {
			restoreErr = errors.Join(restoreErr, io.Write(userIniPath, string(oldUserIni), 0644))
			if setting.OpenBasedir {
				_, attrErr := shell.Execf(`chattr +i '%s'`, userIniPath)
				restoreErr = errors.Join(restoreErr, attrErr)
			}
		}
		return nil, errors.Join(switchErr, restoreErr)
	}

	website.Type = targetType
	if err = r.applyUpdate(update, website); err != nil {
		return restore(err)
	}
	databaseUpdated = true

	vhost, err := r.getVhost(website)
	if err != nil {
		return restore(err)
	}
	if err = vhost.SetConfig("001-acme.conf", webservertypes.ScopeSite, ""); err != nil {
		return restore(err)
	}
	var errorPageConfig string
	switch webServer {
	case "nginx":
		errorPageConfig = `error_page 404 /404.html;`
	case "apache":
		errorPageConfig = `ErrorDocument 404 /404.html`
	}
	if err = vhost.SetConfig("010-error-404.conf", webservertypes.ScopeSite, errorPageConfig); err != nil {
		return restore(err)
	}
	if err = vhost.SetEnable(website.Status); err != nil {
		return restore(err)
	}
	switch targetType {
	case biz.WebsiteTypePHP:
		cacheConfig := nginxPHPCacheConfig
		if webServer == "apache" {
			cacheConfig = apachePHPCacheConfig
		}
		err = vhost.SetConfig("010-cache.conf", webservertypes.ScopeSite, cacheConfig)
	case biz.WebsiteTypeStatic:
		spaConfig := nginxSPAConfig
		if webServer == "apache" {
			spaConfig = apacheSPAConfig
		}
		err = vhost.SetRawConfig("799-spa.conf", webservertypes.ScopeSite, spaConfig)
	}
	if err != nil {
		return restore(err)
	}
	if err = vhost.Save(); err != nil {
		return restore(err)
	}

	if oldType == biz.WebsiteTypePHP && targetType != biz.WebsiteTypePHP && setting.OpenBasedir {
		if err = io.Remove(userIniPath); err != nil {
			return restore(err)
		}
	}
	if err = r.ReloadWebServer(); err != nil {
		_, switchErr := restore(err)
		if reloadErr := r.ReloadWebServer(); reloadErr != nil {
			switchErr = errors.Join(switchErr, reloadErr)
		}
		return nil, switchErr
	}

	_ = io.Remove(backupDir)
	return website, nil
}

// applyUpdate 将更新请求应用到网站配置与实体，供 Update 复用
func (r *websiteRepo) applyUpdate(req *request.WebsiteUpdate, website *biz.Website) error {
	vhost, err := r.getVhost(website)
	if err != nil {
		return err
	}

	// 监听地址
	if req.SSL && !website.SSL {
		webServer, getErr := r.setting.Get(biz.SettingKeyWebserver)
		if getErr != nil {
			return getErr
		}
		listenIPv6 := false
		if webServer == "nginx" {
			listenIPv6, getErr = r.setting.GetBool(biz.SettingKeyWebsiteListenIPv6, false)
			if getErr != nil {
				return getErr
			}
		}
		args := []string{"ssl"}
		if webServer == "nginx" {
			args = append(args, "quic")
		}
		addresses := []string{"443"}
		if webServer == "nginx" && listenIPv6 {
			addresses = append(addresses, "[::]:443")
		}
		httpsListens := lo.Map(addresses, func(address string, _ int) webservertypes.Listen {
			return webservertypes.Listen{Address: address, Args: slices.Clone(args)}
		})
		req.Listens = lo.UniqBy(lo.Map(slices.Concat(req.Listens, httpsListens), func(listen webservertypes.Listen, _ int) webservertypes.Listen {
			if slices.Contains(addresses, listen.Address) {
				listen.Args = lo.Uniq(slices.Concat(listen.Args, args))
			}
			return listen
		}), func(listen webservertypes.Listen) string {
			return listen.Address
		})
	}
	// 关闭 HTTPS 时移除 SSL 专用监听（含 IPv6），避免残留 ssl/quic 参数导致 nginx 无法启动
	if !req.SSL {
		req.Listens = lo.Filter(req.Listens, func(l webservertypes.Listen, _ int) bool {
			return !slices.Contains(l.Args, "ssl") && !slices.Contains(l.Args, "quic")
		})
	}
	if err = vhost.SetListen(req.Listens); err != nil {
		return err
	}
	// 域名
	domains, err := punycode.EncodeDomains(req.Domains)
	if err != nil {
		return err
	}
	domains = lo.Map(domains, func(d string, _ int) string { return tools.WrapIPv6(d) })
	if err = vhost.SetServerName(domains); err != nil {
		return err
	}
	// 首页文件
	if err = vhost.SetIndex(req.Index); err != nil {
		return err
	}
	// 运行目录
	if !io.Exists(req.Root) {
		return errors.New(r.t.Get("runtime directory does not exist"))
	}
	if err = vhost.SetRoot(req.Root); err != nil {
		return err
	}
	// 运行目录
	if !io.Exists(req.Path) {
		return errors.New(r.t.Get("website directory does not exist"))
	}
	website.Path = req.Path
	// SSL
	certPath := filepath.Join(app.Root, "sites", website.Name, "config", "fullchain.pem")
	keyPath := filepath.Join(app.Root, "sites", website.Name, "config", "private.key")
	if err = io.Write(certPath, req.SSLCert, 0600); err != nil {
		return err
	}
	if err = io.Write(keyPath, req.SSLKey, 0600); err != nil {
		return err
	}
	website.SSL = req.SSL
	if req.SSL {
		if _, err = cert.ParseCert([]byte(req.SSLCert)); err != nil {
			return errors.New(r.t.Get("failed to parse certificate: %v", err))
		}
		if _, err = cert.ParseKey([]byte(req.SSLKey)); err != nil {
			return errors.New(r.t.Get("failed to parse private key: %v", err))
		}
		// 检查证书是否已存在于面板的证书管理中，如果不存在则作为本地证书上传
		existing := new(biz.Cert)
		err = r.db.Where("TRIM(cert, char(9) || char(10) || char(13) || ' ') = ?", strings.TrimSpace(req.SSLCert)).First(existing).Error
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			certInfo, _ := cert.ParseCert([]byte(req.SSLCert))
			sans := certInfo.DNSNames
			for _, ip := range certInfo.IPAddresses {
				sans = append(sans, ip.String())
			}
			existing = &biz.Cert{
				Type:    "upload",
				Domains: sans,
				Cert:    strings.TrimSpace(req.SSLCert),
				Key:     strings.TrimSpace(req.SSLKey),
			}
			if err = r.db.Create(existing).Error; err != nil {
				return err
			}
		}
		// 绑定证书，使续签后能自动部署
		website.CertID = existing.ID
		quic := false
		for _, listen := range req.Listens {
			if slices.Contains(listen.Args, "quic") {
				quic = true
				break
			}
		}
		defaultTLSVersions, _ := r.setting.GetSlice(biz.SettingKeyWebsiteTLSVersions)
		if err = vhost.SetSSLConfig(&webservertypes.SSLConfig{
			Cert:         certPath,
			Key:          keyPath,
			Protocols:    lo.If(len(req.SSLProtocols) > 0, req.SSLProtocols).Else(defaultTLSVersions),
			HSTS:         req.HSTS,
			OCSP:         req.OCSP,
			HTTPRedirect: req.HTTPRedirect,
			AltSvc:       lo.If(quic, `'h3=":$server_port"; ma=2592000'`).Else(``),
		}); err != nil {
			return err
		}
	} else {
		if err = vhost.ClearSSL(); err != nil {
			return err
		}
		// 关闭 HTTPS 后不应再被证书续签覆盖
		website.CertID = 0
	}

	// PHP
	if phpVhost, ok := vhost.(webservertypes.PHPVhost); ok {
		if err = phpVhost.SetPHP(req.PHP); err != nil {
			return err
		}
		// 伪静态
		if err = phpVhost.SetRawConfig("010-rewrite.conf", webservertypes.ScopeSite, req.Rewrite); err != nil {
			return err
		}
		// 防跨站
		if !strings.HasSuffix(req.Root, "/") {
			req.Root += "/"
		}
		userIni := filepath.Join(req.Root, ".user.ini")
		if req.OpenBasedir {
			if !io.Exists(userIni) || req.Root != website.Path {
				// 之前没有开启，或者修改了运行目录，重新写入
				if err = io.Write(userIni, fmt.Sprintf("open_basedir=%s:%s:/tmp/", req.Root, req.Path), 0644); err != nil {
					return err
				}
			}
			_, _ = shell.Execf(`chattr +i '%s'`, userIni)
		} else if io.Exists(userIni) {
			if err = io.Remove(userIni); err != nil {
				return err
			}
		}
	}

	// 反向代理
	if proxyVhost, ok := vhost.(webservertypes.ProxyVhost); ok {
		if err = proxyVhost.SetUpstreams(req.Upstreams); err != nil {
			return err
		}
		if err = proxyVhost.SetProxies(req.Proxies); err != nil {
			return err
		}
	}

	// 重定向配置
	if redirectVhost, ok := vhost.(webservertypes.VhostRedirect); ok {
		if err = redirectVhost.SetRedirects(req.Redirects); err != nil {
			return err
		}
	}

	// 高级设置（日志路径、限流限速、真实 IP、基本认证）
	// 日志路径
	if req.AccessLog != "" {
		if err = vhost.SetAccessLog(req.AccessLog); err != nil {
			return err
		}
	}
	if req.ErrorLog != "" {
		if err = vhost.SetErrorLog(req.ErrorLog); err != nil {
			return err
		}
	}
	// 限流限速
	if req.RateLimit != nil {
		if err = vhost.SetRateLimit(req.RateLimit); err != nil {
			return err
		}
	} else {
		if err = vhost.ClearRateLimit(); err != nil {
			return err
		}
	}
	// 真实 IP 配置
	if req.RealIP != nil {
		if err = vhost.SetRealIP(req.RealIP); err != nil {
			return err
		}
	} else {
		if err = vhost.ClearRealIP(); err != nil {
			return err
		}
	}
	// 基本认证创建 htpasswd 文件
	if len(req.BasicAuth) > 0 {
		htpasswdPath := filepath.Join(app.Root, "sites", website.Name, "htpasswd")
		if err = r.writeBasicAuthUsers(htpasswdPath, req.BasicAuth); err != nil {
			return err
		}
		if err = vhost.SetBasicAuth(map[string]string{"user_file": htpasswdPath}); err != nil {
			return err
		}
	} else {
		// 清除基本认证配置和 htpasswd 文件
		htpasswdPath := filepath.Join(app.Root, "sites", website.Name, "htpasswd")
		_ = io.Remove(htpasswdPath)
		if err = vhost.ClearBasicAuth(); err != nil {
			return err
		}
	}

	// 访问统计
	webServer, _ := r.setting.Get(biz.SettingKeyWebserver)
	if webServer == "nginx" {
		if req.StatEnabled {
			if err = r.enableStat(vhost, website.Name); err != nil {
				return err
			}
		} else {
			_ = vhost.RemoveConfig("010-stat-format.conf", webservertypes.ScopeShared)
			_ = vhost.RemoveConfig("021-stats-log.conf", webservertypes.ScopeSite)
		}
	}

	// 自定义配置
	configDir := filepath.Join(app.Root, "sites", website.Name, "config")
	if err = r.saveCustomConfigs(configDir, req.CustomConfigs); err != nil {
		return err
	}

	// 保存配置
	if err = vhost.Save(); err != nil {
		return err
	}
	if err = r.db.Save(website).Error; err != nil {
		return err
	}

	return nil
}

func (r *websiteRepo) GetForDelete(id uint) (*biz.Website, error) {
	website := new(biz.Website)
	if err := r.db.Where("id", id).First(website).Error; err != nil {
		return nil, err
	}
	return website, nil
}

func (r *websiteRepo) RemoveFiles(name string, removePath bool) error {
	if removePath {
		_ = io.Remove(filepath.Join(app.Root, "sites", name))
	} else {
		// 仅删除配置和日志
		_ = io.Remove(filepath.Join(app.Root, "sites", name, "config"))
		_ = io.Remove(filepath.Join(app.Root, "sites", name, "log"))
		_ = io.Remove(filepath.Join(app.Root, "sites", name, "htpasswd"))
	}
	return nil
}

func (r *websiteRepo) Delete(website *biz.Website) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(website).Error; err != nil {
			return err
		}

		// HTTP 验证依赖网站，证书失去全部网站后无法继续自动续签
		return tx.Model(&biz.Cert{}).
			Where("id = ? AND dns_id = 0", website.CertID).
			Where("NOT EXISTS (SELECT 1 FROM websites WHERE cert_id = ?)", website.CertID).
			Update("auto_renewal", false).Error
	})
}

func (r *websiteRepo) UpdateRemark(id uint, remark string) error {
	website := new(biz.Website)
	if err := r.db.Where("id", id).First(website).Error; err != nil {
		return err
	}

	website.Remark = remark
	return r.db.Save(website).Error
}

func (r *websiteRepo) ResetConfig(id uint) error {
	website := new(biz.Website)
	if err := r.db.Where("id", id).First(&website).Error; err != nil {
		return err
	}

	webServer, err := r.setting.Get(biz.SettingKeyWebserver)
	if err != nil {
		return err
	}

	setting, err := r.Get(id)
	if err != nil {
		return err
	}
	listens := lo.Filter(setting.Listens, func(listen webservertypes.Listen, _ int) bool {
		if slices.Contains(listen.Args, "ssl") || slices.Contains(listen.Args, "quic") {
			return false
		}
		port := listen.Address
		if strings.Contains(listen.Address, ":") {
			if _, parsedPort, splitErr := net.SplitHostPort(listen.Address); splitErr == nil {
				port = parsedPort
			}
		}
		return !website.SSL || port != "443"
	})
	if len(listens) == 0 {
		listens = []webservertypes.Listen{{Address: "80"}}
	}

	update := &request.WebsiteUpdate{
		ID:          website.ID,
		Listens:     listens,
		Domains:     setting.Domains,
		Path:        website.Path,
		Root:        setting.Root,
		Index:       []string{"index.html"},
		SSL:         false,
		StatEnabled: webServer == "nginx",
		AccessLog:   filepath.Join(app.Root, "sites", website.Name, "log", "access.log"),
		ErrorLog:    filepath.Join(app.Root, "sites", website.Name, "log", "error.log"),
	}
	switch website.Type {
	case biz.WebsiteTypePHP:
		update.Index = []string{"index.php", "index.html"}
		update.PHP = setting.PHP
		update.OpenBasedir = true
	case biz.WebsiteTypeProxy:
		update.Upstreams = setting.Upstreams
		if len(setting.Proxies) > 0 && setting.Proxies[0].Pass != "" {
			update.Proxies = []webservertypes.Proxy{{
				Location: "^~ /",
				Pass:     setting.Proxies[0].Pass,
			}}
		}
	}

	if website.Type == biz.WebsiteTypePHP {
		if err = io.Remove(filepath.Join(setting.Root, ".user.ini")); err != nil {
			return err
		}
	}
	configDir := filepath.Join(app.Root, "sites", website.Name, "config")
	if err = io.Remove(configDir); err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Join(configDir, "site"), 0600); err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Join(configDir, "shared"), 0600); err != nil {
		return err
	}

	website.Status = true
	if err = r.applyUpdate(update, website); err != nil {
		return err
	}

	vhost, err := r.getVhost(website)
	if err != nil {
		return err
	}
	if err = vhost.SetConfig("001-acme.conf", webservertypes.ScopeSite, ""); err != nil {
		return err
	}
	var errorPageConfig string
	switch webServer {
	case "nginx":
		errorPageConfig = `error_page 404 /404.html;`
	case "apache":
		errorPageConfig = `ErrorDocument 404 /404.html`
	}
	if err = vhost.SetConfig("010-error-404.conf", webservertypes.ScopeSite, errorPageConfig); err != nil {
		return err
	}
	switch website.Type {
	case biz.WebsiteTypePHP:
		cacheConfig := nginxPHPCacheConfig
		if webServer == "apache" {
			cacheConfig = apachePHPCacheConfig
		}
		err = vhost.SetConfig("010-cache.conf", webservertypes.ScopeSite, cacheConfig)
	case biz.WebsiteTypeStatic:
		spaConfig := nginxSPAConfig
		if webServer == "apache" {
			spaConfig = apacheSPAConfig
		}
		err = vhost.SetRawConfig("799-spa.conf", webservertypes.ScopeSite, spaConfig)
	}
	if err != nil {
		return err
	}
	if err = vhost.Save(); err != nil {
		return err
	}
	if err = io.Chmod(filepath.Join(app.Root, "sites", website.Name, "config"), 0600); err != nil {
		return err
	}
	if err = r.ReloadWebServer(); err != nil {
		return err
	}

	return nil
}

func (r *websiteRepo) UpdateStatus(id uint, status bool) error {
	website := new(biz.Website)
	if err := r.db.Where("id", id).First(&website).Error; err != nil {
		return err
	}

	vhost, err := r.getVhost(website)
	if err != nil {
		return err
	}
	if err = vhost.SetEnable(status); err != nil {
		return err
	}
	if err = vhost.Save(); err != nil {
		return err
	}

	website.Status = status
	if err = r.db.Save(website).Error; err != nil {
		return err
	}

	return r.ReloadWebServer()
}

func (r *websiteRepo) UpdateExpireAt(id uint, expireAt *time.Time) error {
	return r.db.Model(&biz.Website{}).Where("id = ?", id).Update("expire_at", expireAt).Error
}

func (r *websiteRepo) UpdateCert(req *request.WebsiteUpdateCert) error {
	website := new(biz.Website)
	if err := r.db.Where("name", req.Name).First(&website).Error; err != nil {
		return err
	}

	if _, err := cert.ParseCert([]byte(req.Cert)); err != nil {
		return errors.New(r.t.Get("failed to parse certificate: %v", err))
	}
	if _, err := cert.ParseKey([]byte(req.Key)); err != nil {
		return errors.New(r.t.Get("failed to parse private key: %v", err))
	}

	certPath := filepath.Join(app.Root, "sites", website.Name, "config", "fullchain.pem")
	keyPath := filepath.Join(app.Root, "sites", website.Name, "config", "private.key")
	if err := io.Write(certPath, req.Cert, 0600); err != nil {
		return err
	}
	if err := io.Write(keyPath, req.Key, 0600); err != nil {
		return err
	}

	if website.SSL {
		return r.ReloadWebServer()
	}

	return nil
}

// customConfigStartNum 自定义配置起始序号
const customConfigStartNum = 800

// customConfigEndNum 自定义配置结束序号
const customConfigEndNum = 999

// getCustomConfigs 获取网站自定义配置列表
func (r *websiteRepo) getCustomConfigs(configDir string) []types.WebsiteCustomConfig {
	var configs []types.WebsiteCustomConfig

	// 从 site 和 shared 目录读取自定义配置
	for _, scope := range []string{"site", "shared"} {
		scopeDir := filepath.Join(configDir, scope)
		entries, err := os.ReadDir(scopeDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			// 匹配文件名格式: 800-999-name.conf
			name := entry.Name()
			if !strings.HasSuffix(name, ".conf") {
				continue
			}
			// 解析序号
			parts := strings.SplitN(name, "-", 2)
			if len(parts) < 2 {
				continue
			}
			num, err := strconv.Atoi(parts[0])
			if err != nil || num < customConfigStartNum || num > customConfigEndNum {
				continue
			}
			// 提取配置名称（去掉序号前缀和.conf后缀）
			configName := strings.TrimSuffix(parts[1], ".conf")
			if configName == "" {
				continue
			}
			// 读取配置内容
			content, err := io.Read(filepath.Join(scopeDir, name))
			if err != nil {
				continue
			}

			configs = append(configs, types.WebsiteCustomConfig{
				Name:    configName,
				Scope:   scope,
				Content: content,
			})
		}
	}

	return configs
}

// saveCustomConfigs 保存网站自定义配置
func (r *websiteRepo) saveCustomConfigs(configDir string, configs []request.WebsiteCustomConfig) error {
	if err := r.clearCustomConfigs(configDir); err != nil {
		return err
	}

	// 分别跟踪 site 和 shared 目录的序号
	siteNum := customConfigStartNum
	sharedNum := customConfigStartNum

	for _, cfg := range configs {
		var num int
		switch cfg.Scope {
		case "site":
			num = siteNum
			siteNum++
		case "shared":
			num = sharedNum
			sharedNum++
		default:
			return fmt.Errorf("invalid config scope: %s", cfg.Scope)
		}

		if num > customConfigEndNum {
			return errors.New(r.t.Get("maximum number of custom configurations reached (limit: %d)", customConfigEndNum-customConfigStartNum+1))
		}

		fileName := fmt.Sprintf("%03d-%s.conf", num, cfg.Name)
		filePath := filepath.Join(configDir, cfg.Scope, fileName)

		if err := io.Write(filePath, cfg.Content, 0600); err != nil {
			return fmt.Errorf("failed to write custom config: %w", err)
		}
	}

	return nil
}

// clearCustomConfigs 清除网站自定义配置文件
func (r *websiteRepo) clearCustomConfigs(configDir string) error {
	for _, scope := range []string{"site", "shared"} {
		scopeDir := filepath.Join(configDir, scope)
		entries, err := os.ReadDir(scopeDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			name := entry.Name()
			if !strings.HasSuffix(name, ".conf") {
				continue
			}
			parts := strings.SplitN(name, "-", 2)
			if len(parts) < 2 {
				continue
			}
			num, err := strconv.Atoi(parts[0])
			if err != nil || num < customConfigStartNum || num > customConfigEndNum {
				continue
			}
			filePath := filepath.Join(scopeDir, name)
			if err = os.Remove(filePath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove custom config: %w", err)
			}
		}
	}

	return nil
}

// newVhost 按给定网页服务器类型构造站点 vhost
func (r *websiteRepo) newVhost(webServer string, website *biz.Website) (webservertypes.Vhost, error) {
	configDir := filepath.Join(app.Root, "sites", website.Name, "config")
	switch website.Type {
	case biz.WebsiteTypeProxy:
		return webserver.NewProxyVhost(webserver.Type(webServer), configDir)
	case biz.WebsiteTypePHP:
		return webserver.NewPHPVhost(webserver.Type(webServer), configDir)
	case biz.WebsiteTypeStatic:
		return webserver.NewStaticVhost(webserver.Type(webServer), configDir)
	default:
		return nil, errors.New(r.t.Get("unsupported website type: %s", website.Type))
	}
}

func (r *websiteRepo) getVhost(website *biz.Website) (webservertypes.Vhost, error) {
	webServer, err := r.setting.Get(biz.SettingKeyWebserver)
	if err != nil {
		return nil, err
	}

	return r.newVhost(webServer, website)
}

func (r *websiteRepo) ReloadWebServer() error {
	webServer, err := r.setting.Get(biz.SettingKeyWebserver, "unknown")
	if err != nil {
		return err
	}
	var test string
	switch webServer {
	case "nginx":
		test = "nginx -t 2>&1"
	case "apache":
		test = "apachectl configtest 2>&1"
	default:
		return errors.New(r.t.Get("unsupported web server: %s", webServer))
	}

	// 服务未运行时无需重载，配置会在下次启动时生效
	if running, _ := systemctl.Status(webServer); !running {
		return nil
	}

	if err = systemctl.Reload(webServer); err != nil {
		out, _ := shell.Execf(test)
		return fmt.Errorf("failed to reload %s: %w; config test: %s", webServer, err, out)
	}

	return nil
}

// readBasicAuthUsers 读取 htpasswd 文件中的用户列表
func (r *websiteRepo) readBasicAuthUsers(siteName string) map[string]string {
	htpasswdPath := filepath.Join(app.Root, "sites", siteName, "htpasswd")
	if !io.Exists(htpasswdPath) {
		return make(map[string]string)
	}

	file, err := os.Open(htpasswdPath)
	if err != nil {
		return make(map[string]string)
	}
	defer func(file *os.File) { _ = file.Close() }(file)

	users := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// htpasswd 格式: username:{PLAIN}password或直接username:password
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			users[parts[0]] = strings.TrimPrefix(parts[1], "{PLAIN}")
		}
	}

	return users
}

// writeBasicAuthUsers 将用户凭证写入 htpasswd 文件
func (r *websiteRepo) writeBasicAuthUsers(htpasswdPath string, users map[string]string) error {
	webServer, err := r.setting.Get(biz.SettingKeyWebserver, "unknown")
	if err != nil {
		return err
	}

	var lines []string
	for username, password := range users {
		if username == "" || password == "" {
			continue
		}
		switch webServer {
		case "nginx":
			lines = append(lines, fmt.Sprintf("%s:%s", username, "{PLAIN}"+password))
		case "apache":
			lines = append(lines, fmt.Sprintf("%s:%s", username, password))
		default:
			return errors.New(r.t.Get("unsupported web server: %s", webServer))
		}
	}

	content := strings.Join(lines, "\n")
	if content != "" {
		content += "\n"
	}
	return io.Write(htpasswdPath, content, 0644) // 必须 0644，Nginx 在运行中以 www 用户读取
}

// enableStat 写入 nginx 访问统计配置（log_format + syslog access_log）
func (r *websiteRepo) enableStat(vhost webservertypes.Vhost, name string) error {
	// nginx 的 syslog tag 与 log_format 名只允许字母数字和下划线
	safeName := strings.Map(func(char rune) rune {
		if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' {
			return char
		}
		return '_'
	}, name)
	formatConf := fmt.Sprintf(`log_format ace_stat_%s escape=json
  '{"site":"%s",'
  '"uri":"$request_uri",'
  '"status":$status,'
  '"bytes":$body_bytes_sent,'
  '"ua":"$http_user_agent",'
  '"ip":"$remote_addr",'
  '"host":"$host",'
  '"method":"$request_method",'
  '"referer":"$http_referer",'
  '"xff":"$http_x_forwarded_for",'
  '"rt":$request_time,'
  '"proto":"$server_protocol",'
  '"port":"$remote_port",'
  '"body":"$request_body",'
  '"content_type":"$sent_http_content_type",'
  '"req_length":$request_length,'
  '"https":"$https",'
  '"upstream_time":"$upstream_response_time",'
  '"upstream_status":"$upstream_status"}';`, safeName, name)
	if err := vhost.SetConfig("010-stat-format.conf", webservertypes.ScopeShared, formatConf); err != nil {
		return err
	}
	logConf := fmt.Sprintf("client_body_in_single_buffer on;\naccess_log syslog:server=unix:/tmp/ace_stats.sock,nohostname,tag=%s ace_stat_%s;", safeName, safeName)
	return vhost.SetConfig("021-stats-log.conf", webservertypes.ScopeSite, logConf)
}
