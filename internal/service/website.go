package service

import (
	"fmt"
	"net/http"
	"path/filepath"
	"slices"
	"time"

	"github.com/leonelquinteros/gotext"
	"github.com/libtnb/chix/v2"
	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/io"
	"github.com/acepanel/panel/v3/pkg/shell"
	"github.com/acepanel/panel/v3/pkg/systemctl"
	"github.com/acepanel/panel/v3/pkg/webserver"
)

type WebsiteService struct {
	t           *gotext.Locale
	websiteRepo *biz.WebsiteUsecase
	settingRepo *biz.SettingUsecase
}

func NewWebsiteService(i do.Injector) (*WebsiteService, error) {
	return &WebsiteService{
		t:           do.MustInvoke[*gotext.Locale](i),
		websiteRepo: do.MustInvoke[*biz.WebsiteUsecase](i),
		settingRepo: do.MustInvoke[*biz.SettingUsecase](i),
	}, nil
}

func (s *WebsiteService) GetRewrites(w http.ResponseWriter, r *http.Request) {
	rewrites, err := s.websiteRepo.GetRewrites()
	if err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, rewrites)
}

func (s *WebsiteService) GetDefaultConfig(w http.ResponseWriter, r *http.Request) {
	webServer, _ := s.settingRepo.Get(biz.SettingKeyWebserver)
	var htmlPath string
	switch webServer {
	case "nginx":
		htmlPath = filepath.Join(app.Root, "server/nginx/html")
	case "apache":
		htmlPath = filepath.Join(app.Root, "server/apache/htdocs")
	default:
		htmlPath = filepath.Join(app.Root, "server/nginx/html")
	}

	index, _ := io.Read(filepath.Join(htmlPath, "index.html"))
	stop, _ := io.Read(filepath.Join(htmlPath, "stop.html"))
	notFound, _ := io.Read(filepath.Join(htmlPath, "404.html"))
	tlsVersions, _ := s.settingRepo.GetSlice(biz.SettingKeyWebsiteTLSVersions)

	Success(w, chix.M{
		"index":        index,
		"stop":         stop,
		"not_found":    notFound,
		"tls_versions": tlsVersions,
	})
}

func (s *WebsiteService) UpdateDefaultConfig(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.WebsiteDefaultConfig](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = s.websiteRepo.UpdateDefaultConfig(req); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, nil)
}

// nginxDefaultConf 生成内置默认站点配置,asDefault 控制是否持有 default_server
func nginxDefaultConf(asDefault bool) string {
	flag := ""
	if asDefault {
		flag = " default_server"
	}
	return fmt.Sprintf(`server
{
    listen 80%[1]s reuseport;
    listen [::]:80%[1]s reuseport;
    listen 443 ssl%[1]s reuseport;
    listen [::]:443 ssl%[1]s reuseport;
    listen 443 quic%[1]s reuseport;
    listen [::]:443 quic%[1]s reuseport;
    server_name _;
    index index.html;
    root %[2]s/html;
    ssl_reject_handshake on;
}
`, flag, filepath.Join(app.Root, "server/nginx"))
}

// filterDefaultHolders 过滤出配置中持有 default_server 的网站
func filterDefaultHolders(websites []*biz.Website) []*biz.Website {
	var holders []*biz.Website
	for _, website := range websites {
		vhost, err := webserver.NewStaticVhost(webserver.TypeNginx, filepath.Join(app.Root, "sites", website.Name, "config"))
		if err != nil {
			continue
		}
		for _, listen := range vhost.Listen() {
			if slices.Contains(listen.Args, "default_server") {
				holders = append(holders, website)
				break
			}
		}
	}

	return holders
}

// setWebsiteDefaultServer 增删网站配置中的 default_server 标志
func (s *WebsiteService) setWebsiteDefaultServer(name string, isDefault bool) error {
	vhost, err := webserver.NewStaticVhost(webserver.TypeNginx, filepath.Join(app.Root, "sites", name, "config"))
	if err != nil {
		return err
	}

	listens := vhost.Listen()
	for i := range listens {
		if isDefault {
			if !slices.Contains(listens[i].Args, "default_server") {
				listens[i].Args = append(listens[i].Args, "default_server")
			}
		} else {
			listens[i].Args = slices.DeleteFunc(listens[i].Args, func(arg string) bool {
				return arg == "default_server"
			})
		}
	}
	if err = vhost.SetListen(listens); err != nil {
		return err
	}

	return vhost.Save()
}

// GetDefaultSite 获取当前默认站点,0 表示面板内置默认页
func (s *WebsiteService) GetDefaultSite(w http.ResponseWriter, r *http.Request) {
	websites, _, err := s.websiteRepo.List("", 1, 10000)
	if err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	var id uint
	if holders := filterDefaultHolders(websites); len(holders) > 0 {
		id = holders[0].ID
	}

	Success(w, chix.M{"id": id})
}

// UpdateDefaultSite 切换默认站点
// 在内置默认配置与网站配置之间迁移 default_server 标志,ID 为 0 表示恢复内置默认页
func (s *WebsiteService) UpdateDefaultSite(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.WebsiteDefaultSite](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	webServer, _ := s.settingRepo.Get(biz.SettingKeyWebserver)
	if webServer != "nginx" {
		Error(w, http.StatusUnprocessableEntity, s.t.Get("default site is only supported with nginx"))
		return
	}

	websites, _, err := s.websiteRepo.List("", 1, 10000)
	if err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	var target *biz.Website
	if req.ID > 0 {
		idx := slices.IndexFunc(websites, func(item *biz.Website) bool { return item.ID == req.ID })
		if idx < 0 {
			Error(w, http.StatusUnprocessableEntity, s.t.Get("website not found"))
			return
		}
		target = websites[idx]
	}

	holders := filterDefaultHolders(websites)

	// 备份待改文件,校验失败时整体回滚
	defaultConf := filepath.Join(app.Root, "server/nginx/conf/default.conf")
	backups := make(map[string]string)
	backup := func(path string) {
		if content, err := io.Read(path); err == nil {
			backups[path] = content
		}
	}
	backup(defaultConf)
	for _, website := range holders {
		backup(filepath.Join(app.Root, "sites", website.Name, "config/nginx.conf"))
	}
	if target != nil {
		backup(filepath.Join(app.Root, "sites", target.Name, "config/nginx.conf"))
	}
	restore := func() {
		for path, content := range backups {
			_ = io.Write(path, content, 0600)
		}
	}

	// 移除原有网站上的 default_server
	for _, website := range holders {
		if target != nil && website.ID == target.ID {
			continue
		}
		if err = s.setWebsiteDefaultServer(website.Name, false); err != nil {
			restore()
			Error(w, http.StatusInternalServerError, "%v", err)
			return
		}
	}
	// 目标网站添加 default_server
	if target != nil {
		if err = s.setWebsiteDefaultServer(target.Name, true); err != nil {
			restore()
			Error(w, http.StatusInternalServerError, "%v", err)
			return
		}
	}
	// 重写内置默认配置,目标为空时由其持有 default_server
	if err = io.Write(defaultConf, nginxDefaultConf(target == nil), 0600); err != nil {
		restore()
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	if _, err = shell.Execf("nginx -t"); err != nil {
		restore()
		Error(w, http.StatusInternalServerError, s.t.Get("nginx config test failed: %v", err))
		return
	}
	if err = systemctl.Reload("nginx"); err != nil {
		Error(w, http.StatusInternalServerError, s.t.Get("failed to reload nginx: %v", err))
		return
	}

	Success(w, nil)
}

// UpdateCert 用于自动化工具更新证书
func (s *WebsiteService) UpdateCert(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.WebsiteUpdateCert](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = s.websiteRepo.UpdateCert(req); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, nil)
}

// List 网站列表
func (s *WebsiteService) List(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.WebsiteList](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	websites, total, err := s.websiteRepo.List(req.Type, req.Page, req.Limit)
	if err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, chix.M{
		"total": total,
		"items": websites,
	})
}

func (s *WebsiteService) Create(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.WebsiteCreate](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if len(req.Path) == 0 {
		req.Path, _ = s.settingRepo.Get(biz.SettingKeyWebsitePath)
		req.Path = filepath.Join(req.Path, req.Name, "public")
	}

	if _, err = s.websiteRepo.Create(r.Context(), req); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, nil)
}

func (s *WebsiteService) Get(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.ID](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	config, err := s.websiteRepo.Get(req.ID)
	if err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, config)
}

func (s *WebsiteService) Update(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.WebsiteUpdate](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = s.websiteRepo.Update(r.Context(), req); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, nil)
}

func (s *WebsiteService) Delete(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.WebsiteDelete](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = s.websiteRepo.Delete(r.Context(), req); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, nil)
}

func (s *WebsiteService) UpdateRemark(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.WebsiteUpdateRemark](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = s.websiteRepo.UpdateRemark(req.ID, req.Remark); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, nil)
}

func (s *WebsiteService) ResetConfig(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.ID](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = s.websiteRepo.ResetConfig(req.ID); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, nil)
}

func (s *WebsiteService) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.WebsiteUpdateStatus](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = s.websiteRepo.UpdateStatus(req.ID, req.Status); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, nil)
}

func (s *WebsiteService) UpdateExpireAt(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.WebsiteUpdateExpireAt](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	var expireAt *time.Time
	if req.ExpireAt != "" {
		t, err := time.Parse(time.DateTime, req.ExpireAt)
		if err != nil {
			Error(w, http.StatusUnprocessableEntity, "%v", err)
			return
		}
		expireAt = &t
	}

	if err = s.websiteRepo.UpdateExpireAt(req.ID, expireAt); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, nil)
}

func (s *WebsiteService) ObtainCert(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.WebsiteObtainCert](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = s.websiteRepo.ObtainCert(r.Context(), req.ID, req.DNSID); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	Success(w, nil)
}
