package pgadmin

import (
	"encoding/json"
	"errors"
	"fmt"
	stdio "io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/leonelquinteros/gotext"
	"github.com/libtnb/chix/v2"
	"github.com/samber/do/v2"
	"github.com/spf13/cast"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/service"
	"github.com/acepanel/panel/v3/pkg/firewall"
	"github.com/acepanel/panel/v3/pkg/io"
	"github.com/acepanel/panel/v3/pkg/shell"
	"github.com/acepanel/panel/v3/pkg/systemctl"
	"github.com/acepanel/panel/v3/pkg/types"
)

type App struct {
	t                  *gotext.Locale
	databaseServerRepo biz.DatabaseServerRepo
}

func NewApp(i do.Injector) (*App, error) {
	return &App{
		t:                  do.MustInvoke[*gotext.Locale](i),
		databaseServerRepo: do.MustInvoke[biz.DatabaseServerRepo](i),
	}, nil
}

func (s *App) Route(r chi.Router) {
	r.Get("/info", s.Info)
	r.Post("/port", s.UpdatePort)
	r.Post("/login", s.Login)
	r.Post("/reset_password", s.ResetPassword)
}

func (s *App) Status() string {
	ok, _ := systemctl.Status("pgadmin")
	return types.AggregateAppStatus(ok)
}

func (s *App) path() string {
	return fmt.Sprintf("%s/server/pgadmin", app.Root)
}

// port 从 systemd 环境文件中解析监听端口
func (s *App) port() (uint, error) {
	conf, err := io.Read(fmt.Sprintf("%s/pgadmin.conf", s.path()))
	if err != nil {
		return 0, err
	}
	match := regexp.MustCompile(`PGADMIN_LISTEN=.+:(\d+)`).FindStringSubmatch(conf)
	if len(match) < 2 {
		return 0, errors.New(s.t.Get("pgAdmin port not found"))
	}

	return cast.ToUint(match[1]), nil
}

// credential 读取安装时生成的初始凭据(邮箱与密码)
func (s *App) credential() (string, string) {
	raw, err := io.Read(fmt.Sprintf("%s/credential", s.path()))
	if err != nil {
		return "", ""
	}
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	if len(lines) < 2 {
		return strings.TrimSpace(lines[0]), ""
	}

	return strings.TrimSpace(lines[0]), strings.TrimSpace(lines[1])
}

func (s *App) Info(w http.ResponseWriter, r *http.Request) {
	port, err := s.port()
	if err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}
	email, password := s.credential()

	service.Success(w, chix.M{
		"port":     port,
		"email":    email,
		"password": password,
	})
}

func (s *App) UpdatePort(w http.ResponseWriter, r *http.Request) {
	req, err := service.Bind[UpdatePort](r)
	if err != nil {
		service.Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	conf := fmt.Sprintf("%s/pgadmin.conf", s.path())
	content, err := io.Read(conf)
	if err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}
	content = regexp.MustCompile(`PGADMIN_LISTEN=(.+):\d+`).ReplaceAllString(content, "PGADMIN_LISTEN=${1}:"+cast.ToString(req.Port))
	if err = io.Write(conf, content, 0600); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	fw := firewall.NewFirewall()
	err = fw.Port(firewall.FireInfo{
		Type:      firewall.TypeNormal,
		PortStart: req.Port,
		PortEnd:   req.Port,
		Strategy:  firewall.StrategyAccept,
		Direction: firewall.DirectionIn,
	}, firewall.OperationAdd)
	if err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	if err = systemctl.Restart("pgadmin"); err != nil {
		service.Error(w, http.StatusInternalServerError, s.t.Get("failed to restart pgAdmin: %v", err))
		return
	}

	service.Success(w, nil)
}

// serversFile pgAdmin dump-servers/load-servers 的 JSON 结构
type serversFile struct {
	Servers map[string]serverEntry `json:"Servers"`
}

type serverEntry struct {
	Name          string `json:"Name"`
	Group         string `json:"Group"`
	Host          string `json:"Host"`
	Port          int    `json:"Port"`
	MaintenanceDB string `json:"MaintenanceDB"`
	Username      string `json:"Username"`
	SSLMode       string `json:"SSLMode,omitempty"`
	PassFile      string `json:"PassFile,omitempty"`
}

// escapePgpass 转义 pgpass 字段中的反斜杠与冒号
func escapePgpass(s string) string {
	return strings.NewReplacer(`\`, `\\`, `:`, `\:`).Replace(s)
}

// syncServer 将面板中的 PostgreSQL 服务器同步注册到 pgAdmin,凭据写入 pgpass 实现免密
func (s *App) syncServer(server *biz.DatabaseServer, email string) error {
	// pgAdmin server 模式下 PassFile 以用户 storage 目录为根,目录名为邮箱 @ 转 _
	storageDir := fmt.Sprintf("%s/data/storage/%s", s.path(), strings.ReplaceAll(email, "@", "_"))
	pgpass := filepath.Join(storageDir, "pgpass")

	// 更新 pgpass 中该服务器的凭据行
	entryPrefix := fmt.Sprintf("%s:%d:*:%s:", escapePgpass(server.Host), server.Port, escapePgpass(server.Username))
	var lines []string
	if raw, err := io.Read(pgpass); err == nil {
		for line := range strings.SplitSeq(strings.TrimSpace(raw), "\n") {
			if line != "" && !strings.HasPrefix(line, entryPrefix) {
				lines = append(lines, line)
			}
		}
	}
	lines = append(lines, entryPrefix+escapePgpass(server.Password))
	if err := os.MkdirAll(storageDir, 0700); err != nil {
		return err
	}
	if err := io.Write(pgpass, strings.Join(lines, "\n")+"\n", 0600); err != nil {
		return err
	}

	// 已注册的服务器无需重复导入
	dump := filepath.Join(os.TempDir(), "pgadmin-servers.json")
	defer func() { _ = io.Remove(dump) }()
	_, _ = shell.Execf("%s/cli dump-servers '%s' --user '%s'", s.path(), dump, email)
	exists := false
	if raw, err := io.Read(dump); err == nil {
		var dumped serversFile
		if err = json.Unmarshal([]byte(raw), &dumped); err == nil {
			for _, item := range dumped.Servers {
				if item.Host == server.Host && item.Port == int(server.Port) && item.Username == server.Username {
					exists = true
					break
				}
			}
		}
	}

	if !exists {
		load := filepath.Join(os.TempDir(), "pgadmin-servers-add.json")
		defer func() { _ = io.Remove(load) }()
		payload, err := json.Marshal(serversFile{Servers: map[string]serverEntry{
			"1": {
				Name:          server.Name,
				Group:         "AcePanel",
				Host:          server.Host,
				Port:          int(server.Port),
				MaintenanceDB: "postgres",
				Username:      server.Username,
				SSLMode:       "prefer",
				PassFile:      "/pgpass",
			},
		}})
		if err != nil {
			return err
		}
		if err = io.Write(load, string(payload), 0600); err != nil {
			return err
		}
		if out, err := shell.Execf("%s/cli load-servers '%s' --user '%s'", s.path(), load, email); err != nil {
			return errors.Join(err, errors.New(out))
		}
	}

	// CLI 以 root 运行,修正数据目录属主避免服务写入失败
	if _, err := shell.Execf("chown -R www:www %s/data", s.path()); err != nil {
		return err
	}

	return nil
}

// Login 同步服务器后代理登录 pgAdmin 并将会话 Cookie 转发给浏览器
// 面板与 pgAdmin 同主机不同端口,Cookie 按主机共享,浏览器凭转发的 Cookie 即为已登录态
func (s *App) Login(w http.ResponseWriter, r *http.Request) {
	req, err := service.Bind[Login](r)
	if err != nil {
		service.Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	server, err := s.databaseServerRepo.Get(req.ServerID)
	if err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}
	if server.Type != biz.DatabaseTypePostgresql {
		service.Error(w, http.StatusUnprocessableEntity, s.t.Get("server %s is not a PostgreSQL server", server.Name))
		return
	}

	port, err := s.port()
	if err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}
	email, password := s.credential()
	if email == "" || password == "" {
		service.Error(w, http.StatusInternalServerError, s.t.Get("pgAdmin credential file not found"))
		return
	}

	if err = s.syncServer(server, email); err != nil {
		service.Error(w, http.StatusInternalServerError, s.t.Get("failed to sync server to pgAdmin: %v", err))
		return
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// 获取登录页以取得会话 Cookie 与 CSRF token
	loginURL := fmt.Sprintf("http://127.0.0.1:%d/login", port)
	pageReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, loginURL, nil)
	if err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}
	pageResp, err := client.Do(pageReq)
	if err != nil {
		service.Error(w, http.StatusInternalServerError, s.t.Get("failed to request pgAdmin: %v", err))
		return
	}
	defer pageResp.Body.Close()
	page, err := stdio.ReadAll(stdio.LimitReader(pageResp.Body, 4<<20))
	if err != nil {
		service.Error(w, http.StatusInternalServerError, s.t.Get("failed to request pgAdmin: %v", err))
		return
	}

	csrf := regexp.MustCompile(`name="csrf_token"[^>]*value="([^"]+)"`).FindStringSubmatch(string(page))
	if len(csrf) < 2 {
		service.Error(w, http.StatusInternalServerError, s.t.Get("failed to parse pgAdmin login page"))
		return
	}

	form := url.Values{}
	form.Set("csrf_token", csrf[1])
	form.Set("email", email)
	form.Set("password", password)

	loginReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, fmt.Sprintf("http://127.0.0.1:%d/authenticate/login", port), strings.NewReader(form.Encode()))
	if err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, cookie := range pageResp.Cookies() {
		loginReq.AddCookie(cookie)
	}

	loginResp, err := client.Do(loginReq)
	if err != nil {
		service.Error(w, http.StatusInternalServerError, s.t.Get("failed to request pgAdmin: %v", err))
		return
	}
	defer loginResp.Body.Close()
	_, _ = stdio.Copy(stdio.Discard, stdio.LimitReader(loginResp.Body, 4<<20))

	// 登录成功时 pgAdmin 返回 302 且跳转目标不是登录页
	location := loginResp.Header.Get("Location")
	if loginResp.StatusCode != http.StatusFound || strings.Contains(location, "/login") {
		service.Error(w, http.StatusInternalServerError, s.t.Get("failed to login pgAdmin, please check the credential of pgAdmin"))
		return
	}

	// 改写 SameSite 以兼容面板 https 跳转 http 的场景
	for _, cookie := range append(pageResp.Cookies(), loginResp.Cookies()...) {
		cookie.SameSite = http.SameSiteLaxMode
		cookie.Secure = false
		http.SetCookie(w, cookie)
	}

	service.Success(w, chix.M{
		"port": port,
	})
}

// ResetPassword 通过 CLI 重置管理员密码并同步凭据文件
func (s *App) ResetPassword(w http.ResponseWriter, r *http.Request) {
	req, err := service.Bind[ResetPassword](r)
	if err != nil {
		service.Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	email, _ := s.credential()
	if email == "" {
		service.Error(w, http.StatusInternalServerError, s.t.Get("pgAdmin credential file not found"))
		return
	}

	// cli 为安装脚本生成的稳定入口,屏蔽上游命令名随大版本变化
	if out, err := shell.Execf("%s/cli update-user '%s' --password '%s'", s.path(), email, req.Password); err != nil {
		service.Error(w, http.StatusInternalServerError, s.t.Get("failed to reset password: %v", errors.Join(err, errors.New(out))))
		return
	}
	// CLI 以 root 运行,修正数据目录属主避免服务写入失败
	if _, err = shell.Execf("chown -R www:www %s/data", s.path()); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	if err = io.Write(fmt.Sprintf("%s/credential", s.path()), email+"\n"+req.Password+"\n", 0600); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	service.Success(w, nil)
}
