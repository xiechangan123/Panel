package frp

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/leonelquinteros/gotext"
	"github.com/libtnb/chix/v2"

	"github.com/acepanel/panel/v3/internal/service"
	"github.com/acepanel/panel/v3/pkg/io"
	"github.com/acepanel/panel/v3/pkg/systemctl"
	"github.com/acepanel/panel/v3/pkg/types"
)

type App struct {
	t *gotext.Locale
}

func NewApp(t *gotext.Locale) *App {
	return &App{
		t: t,
	}
}

func (s *App) Route(r chi.Router) {
	r.Get("/config", s.GetConfig)
	r.Post("/config", s.UpdateConfig)
	r.Get("/user", s.GetUser)
	r.Post("/user", s.UpdateUser)
	r.Get("/server", s.GetServer)
	r.Post("/server", s.UpdateServer)
	r.Get("/client", s.GetClient)
	r.Post("/client", s.UpdateClient)
	r.Get("/proxies", s.Proxies)
	r.Post("/proxies", s.CreateProxy)
	r.Get("/proxies/{name}", s.GetProxy)
	r.Post("/proxies/{name}", s.UpdateProxy)
	r.Delete("/proxies/{name}", s.DeleteProxy)
	r.Get("/visitors", s.Visitors)
	r.Post("/visitors", s.CreateVisitor)
	r.Get("/visitors/{name}", s.GetVisitor)
	r.Post("/visitors/{name}", s.UpdateVisitor)
	r.Delete("/visitors/{name}", s.DeleteVisitor)
}

func (s *App) Status() string {
	frps, _ := systemctl.Status("frps")
	frpc, _ := systemctl.Status("frpc")
	return types.AggregateAppStatus(frps, frpc)
}

func (s *App) GetConfig(w http.ResponseWriter, r *http.Request) {
	req, err := service.Bind[Name](r)
	if err != nil {
		service.Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	config, err := io.Read(confPath(req.Name))
	if err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	service.Success(w, config)
}

func (s *App) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	req, err := service.Bind[UpdateConfig](r)
	if err != nil {
		service.Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = io.Write(confPath(req.Name), req.Config, 0644); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	if err = systemctl.Restart(req.Name); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	service.Success(w, nil)
}

func (s *App) GetUser(w http.ResponseWriter, r *http.Request) {
	req, err := service.Bind[Name](r)
	if err != nil {
		service.Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	servicePath := fmt.Sprintf("/etc/systemd/system/%s.service", req.Name)
	content, err := io.Read(servicePath)
	if err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	userInfo := UserInfo{}

	// 解析 User 和 Group
	if matches := userCaptureRegex.FindStringSubmatch(content); len(matches) > 1 {
		userInfo.User = matches[1]
	}
	if matches := groupCaptureRegex.FindStringSubmatch(content); len(matches) > 1 {
		userInfo.Group = matches[1]
	}

	service.Success(w, userInfo)
}

func (s *App) UpdateUser(w http.ResponseWriter, r *http.Request) {
	req, err := service.Bind[UpdateUser](r)
	if err != nil {
		service.Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	servicePath := fmt.Sprintf("/etc/systemd/system/%s.service", req.Name)
	content, err := io.Read(servicePath)
	if err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	// 检查 User 和 Group 是否存在
	hasUser := userRegex.MatchString(content)
	hasGroup := groupRegex.MatchString(content)

	// 替换或添加 User 和 Group 配置
	switch {
	case hasUser && hasGroup:
		// 两者都存在，分别替换
		content = userRegex.ReplaceAllString(content, "User="+req.User)
		content = groupRegex.ReplaceAllString(content, "Group="+req.Group)
	case hasUser:
		// 只有 User，替换 User 并添加 Group
		content = userRegex.ReplaceAllString(content, fmt.Sprintf("User=%s\nGroup=%s", req.User, req.Group))
	case hasGroup:
		// 只有 Group，添加 User 并替换 Group
		content = serviceRegex.ReplaceAllString(content, "[Service]\nUser="+req.User)
		content = groupRegex.ReplaceAllString(content, "Group="+req.Group)
	default:
		// 两者都不存在，在 [Service] 后添加两者
		content = serviceRegex.ReplaceAllString(content, fmt.Sprintf("[Service]\nUser=%s\nGroup=%s", req.User, req.Group))
	}

	if err = io.Write(servicePath, content, 0644); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	if err = systemctl.DaemonReload(); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}
	if err = systemctl.Restart(req.Name); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	service.Success(w, nil)
}

// GetServer 获取 frps 可视化参数
func (s *App) GetServer(w http.ResponseWriter, r *http.Request) {
	config, err := io.Read(confPath("frps"))
	if err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	tune := new(ServerTune)
	readTune(config, serverFields(tune))

	service.Success(w, tune)
}

// UpdateServer 更新 frps 可视化参数
func (s *App) UpdateServer(w http.ResponseWriter, r *http.Request) {
	req, err := service.Bind[ServerTune](r)
	if err != nil {
		service.Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	path := confPath("frps")
	config, err := io.Read(path)
	if err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	if err = io.Write(path, writeTune(config, serverFields(req)), 0644); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	if err = systemctl.Restart("frps"); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	service.Success(w, nil)
}

// GetClient 获取 frpc 可视化公共参数
func (s *App) GetClient(w http.ResponseWriter, r *http.Request) {
	config, err := io.Read(confPath("frpc"))
	if err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	tune := new(ClientTune)
	readTune(config, clientFields(tune))

	service.Success(w, tune)
}

// UpdateClient 更新 frpc 可视化公共参数
func (s *App) UpdateClient(w http.ResponseWriter, r *http.Request) {
	req, err := service.Bind[ClientTune](r)
	if err != nil {
		service.Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	path := confPath("frpc")
	config, err := io.Read(path)
	if err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	if err = io.Write(path, writeTune(config, clientFields(req)), 0644); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	if err = systemctl.Restart("frpc"); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	service.Success(w, nil)
}

// Proxies 代理列表
func (s *App) Proxies(w http.ResponseWriter, r *http.Request) {
	confD, err := listConfD()
	if err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	paged, total := service.Paginate(r, confD.Proxies)

	service.Success(w, chix.M{
		"total": total,
		"items": paged,
	})
}

// CreateProxy 新增代理
func (s *App) CreateProxy(w http.ResponseWriter, r *http.Request) {
	req, err := service.Bind[Proxy](r)
	if err != nil {
		service.Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if io.Exists(itemPath(proxyPrefix, req.Name)) {
		service.Error(w, http.StatusUnprocessableEntity, s.t.Get("proxy %s already exists", req.Name))
		return
	}

	s.save(w, proxyPrefix, req.Name, ConfD{Proxies: []Proxy{*req}})
}

// GetProxy 获取单个代理
func (s *App) GetProxy(w http.ResponseWriter, r *http.Request) {
	req, err := service.Bind[ItemName](r)
	if err != nil {
		service.Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	confD, err := readConfD(itemPath(proxyPrefix, req.Name))
	if err != nil || len(confD.Proxies) == 0 {
		service.Error(w, http.StatusUnprocessableEntity, s.t.Get("proxy %s does not exist", req.Name))
		return
	}

	service.Success(w, confD.Proxies[0])
}

// UpdateProxy 更新代理
func (s *App) UpdateProxy(w http.ResponseWriter, r *http.Request) {
	req, err := service.Bind[Proxy](r)
	if err != nil {
		service.Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if !io.Exists(itemPath(proxyPrefix, req.Name)) {
		service.Error(w, http.StatusUnprocessableEntity, s.t.Get("proxy %s does not exist", req.Name))
		return
	}

	s.save(w, proxyPrefix, req.Name, ConfD{Proxies: []Proxy{*req}})
}

// DeleteProxy 删除代理
func (s *App) DeleteProxy(w http.ResponseWriter, r *http.Request) {
	req, err := service.Bind[ItemName](r)
	if err != nil {
		service.Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = io.Remove(itemPath(proxyPrefix, req.Name)); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	if err = systemctl.Restart("frpc"); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	service.Success(w, nil)
}

// Visitors 访问者列表
func (s *App) Visitors(w http.ResponseWriter, r *http.Request) {
	confD, err := listConfD()
	if err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	paged, total := service.Paginate(r, confD.Visitors)

	service.Success(w, chix.M{
		"total": total,
		"items": paged,
	})
}

// CreateVisitor 新增访问者
func (s *App) CreateVisitor(w http.ResponseWriter, r *http.Request) {
	req, err := service.Bind[Visitor](r)
	if err != nil {
		service.Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if io.Exists(itemPath(visitorPrefix, req.Name)) {
		service.Error(w, http.StatusUnprocessableEntity, s.t.Get("visitor %s already exists", req.Name))
		return
	}

	s.save(w, visitorPrefix, req.Name, ConfD{Visitors: []Visitor{*req}})
}

// GetVisitor 获取单个访问者
func (s *App) GetVisitor(w http.ResponseWriter, r *http.Request) {
	req, err := service.Bind[ItemName](r)
	if err != nil {
		service.Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	confD, err := readConfD(itemPath(visitorPrefix, req.Name))
	if err != nil || len(confD.Visitors) == 0 {
		service.Error(w, http.StatusUnprocessableEntity, s.t.Get("visitor %s does not exist", req.Name))
		return
	}

	service.Success(w, confD.Visitors[0])
}

// UpdateVisitor 更新访问者
func (s *App) UpdateVisitor(w http.ResponseWriter, r *http.Request) {
	req, err := service.Bind[Visitor](r)
	if err != nil {
		service.Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if !io.Exists(itemPath(visitorPrefix, req.Name)) {
		service.Error(w, http.StatusUnprocessableEntity, s.t.Get("visitor %s does not exist", req.Name))
		return
	}

	s.save(w, visitorPrefix, req.Name, ConfD{Visitors: []Visitor{*req}})
}

// DeleteVisitor 删除访问者
func (s *App) DeleteVisitor(w http.ResponseWriter, r *http.Request) {
	req, err := service.Bind[ItemName](r)
	if err != nil {
		service.Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = io.Remove(itemPath(visitorPrefix, req.Name)); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	if err = systemctl.Restart("frpc"); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	service.Success(w, nil)
}

func (s *App) save(w http.ResponseWriter, prefix, name string, confD ConfD) {
	if err := writeConfD(prefix, name, confD); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	if err := systemctl.Restart("frpc"); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	service.Success(w, nil)
}
