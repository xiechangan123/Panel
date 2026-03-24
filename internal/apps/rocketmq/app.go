package rocketmq

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/leonelquinteros/gotext"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/service"
	"github.com/acepanel/panel/v3/pkg/io"
	"github.com/acepanel/panel/v3/pkg/systemctl"
	"github.com/acepanel/panel/v3/pkg/types"
)

type App struct {
	t *gotext.Locale
}

func NewApp(t *gotext.Locale) *App {
	return &App{t: t}
}

func (s *App) Route(r chi.Router) {
	r.Get("/load", s.Load)
	r.Get("/config", s.GetConfig)
	r.Post("/config", s.UpdateConfig)
	r.Get("/config_tune", s.GetConfigTune)
	r.Post("/config_tune", s.UpdateConfigTune)
}

func (s *App) Load(w http.ResponseWriter, r *http.Request) {
	namesrvStatus, _ := systemctl.Status("rocketmq-namesrv")
	brokerStatus, _ := systemctl.Status("rocketmq-broker")

	namesrvStr := "stopped"
	if namesrvStatus {
		namesrvStr = "running"
	}
	brokerStr := "stopped"
	if brokerStatus {
		brokerStr = "running"
	}

	if !namesrvStatus && !brokerStatus {
		service.Success(w, []types.NV{})
		return
	}

	config, _ := io.Read(s.configPath())

	data := []types.NV{
		{Name: s.t.Get("NameServer Status"), Value: namesrvStr},
		{Name: s.t.Get("Broker Status"), Value: brokerStr},
		{Name: s.t.Get("Broker Name"), Value: s.getPropertiesValue(config, "brokerName")},
		{Name: s.t.Get("Listen Port"), Value: s.getPropertiesValue(config, "listenPort")},
		{Name: s.t.Get("NameServer Address"), Value: s.getPropertiesValue(config, "namesrvAddr")},
		{Name: s.t.Get("Broker Role"), Value: s.getPropertiesValue(config, "brokerRole")},
		{Name: s.t.Get("Flush Disk Type"), Value: s.getPropertiesValue(config, "flushDiskType")},
	}

	service.Success(w, data)
}

func (s *App) GetConfig(w http.ResponseWriter, r *http.Request) {
	conf, _ := io.Read(s.configPath())
	service.Success(w, conf)
}

func (s *App) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	req, err := service.Bind[UpdateConfig](r)
	if err != nil {
		service.Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = io.Write(s.configPath(), req.Config, 0644); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	if err = s.restartServices(); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	service.Success(w, nil)
}

// GetConfigTune 获取 RocketMQ 配置调整参数
func (s *App) GetConfigTune(w http.ResponseWriter, r *http.Request) {
	config, _ := io.Read(s.configPath())
	heapRaw, _ := io.Read(s.heapEnvPath())

	namesrvInit, namesrvMax := s.parseHeapLine(heapRaw, "ROCKETMQ_NAMESRV_HEAP")
	brokerInit, brokerMax := s.parseHeapLine(heapRaw, "ROCKETMQ_BROKER_HEAP")

	tune := ConfigTune{
		BrokerName:          s.getPropertiesValue(config, "brokerName"),
		ListenPort:          s.getPropertiesValue(config, "listenPort"),
		NamesrvAddr:         s.getPropertiesValue(config, "namesrvAddr"),
		BrokerRole:          s.getPropertiesValue(config, "brokerRole"),
		FlushDiskType:       s.getPropertiesValue(config, "flushDiskType"),
		StorePathRootDir:    s.getPropertiesValue(config, "storePathRootDir"),
		StorePathCommitLog:  s.getPropertiesValue(config, "storePathCommitLog"),
		MaxMessageSize:      s.getPropertiesValue(config, "maxMessageSize"),
		NamesrvHeapInitSize: namesrvInit,
		NamesrvHeapMaxSize:  namesrvMax,
		BrokerHeapInitSize:  brokerInit,
		BrokerHeapMaxSize:   brokerMax,
	}

	service.Success(w, tune)
}

// UpdateConfigTune 更新 RocketMQ 配置调整参数
func (s *App) UpdateConfigTune(w http.ResponseWriter, r *http.Request) {
	req, err := service.Bind[ConfigTune](r)
	if err != nil {
		service.Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	config, _ := io.Read(s.configPath())

	config = s.setPropertiesValue(config, "brokerName", req.BrokerName)
	config = s.setPropertiesValue(config, "listenPort", req.ListenPort)
	config = s.setPropertiesValue(config, "namesrvAddr", req.NamesrvAddr)
	config = s.setPropertiesValue(config, "brokerRole", req.BrokerRole)
	config = s.setPropertiesValue(config, "flushDiskType", req.FlushDiskType)
	config = s.setPropertiesValue(config, "storePathRootDir", req.StorePathRootDir)
	config = s.setPropertiesValue(config, "storePathCommitLog", req.StorePathCommitLog)
	config = s.setPropertiesValue(config, "maxMessageSize", req.MaxMessageSize)

	if err = io.Write(s.configPath(), config, 0644); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	// 更新 JVM 堆内存
	if req.NamesrvHeapInitSize != "" || req.NamesrvHeapMaxSize != "" || req.BrokerHeapInitSize != "" || req.BrokerHeapMaxSize != "" {
		heapRaw, _ := io.Read(s.heapEnvPath())
		heapRaw = s.setHeapEnv(heapRaw, *req)
		if err = io.Write(s.heapEnvPath(), heapRaw, 0644); err != nil {
			service.Error(w, http.StatusInternalServerError, "%v", err)
			return
		}
	}

	if err = s.restartServices(); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	service.Success(w, nil)
}

// restartServices 重启 NameServer 和 Broker 服务
func (s *App) restartServices() error {
	if err := systemctl.Restart("rocketmq-namesrv"); err != nil {
		return err
	}
	return systemctl.Restart("rocketmq-broker")
}

// configPath 返回 broker 配置文件路径
func (s *App) configPath() string {
	return fmt.Sprintf("%s/server/rocketmq/conf/broker.conf", app.Root)
}

// heapEnvPath 返回 JVM 堆内存配置文件路径
func (s *App) heapEnvPath() string {
	return fmt.Sprintf("%s/server/rocketmq/conf/heap.env", app.Root)
}

// getPropertiesValue 从 properties 内容中获取指定键的值
func (s *App) getPropertiesValue(content string, key string) string {
	for line := range strings.SplitSeq(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		k, v, ok := strings.Cut(trimmed, "=")
		if ok && strings.TrimSpace(k) == key {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// setPropertiesValue 在 properties 内容中设置指定键的值
func (s *App) setPropertiesValue(content string, key string, value string) string {
	value = strings.ReplaceAll(value, "\n", "")
	value = strings.ReplaceAll(value, "\r", "")

	lines := strings.Split(content, "\n")
	result := make([]string, 0, len(lines))
	found := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			result = append(result, line)
			continue
		}
		checkLine := trimmed
		if strings.HasPrefix(checkLine, "#") {
			checkLine = strings.TrimSpace(checkLine[1:])
		}
		k, _, ok := strings.Cut(checkLine, "=")
		if ok && strings.TrimSpace(k) == key {
			if found {
				continue
			}
			found = true
			if value == "" {
				if !strings.HasPrefix(trimmed, "#") {
					result = append(result, "#"+trimmed)
				} else {
					result = append(result, line)
				}
				continue
			}
			result = append(result, key+"="+value)
		} else {
			result = append(result, line)
		}
	}
	if !found && value != "" {
		result = append(result, key+"="+value)
	}
	return strings.Join(result, "\n")
}

// parseHeapLine 从 heap.env 中提取指定环境变量的堆内存配置
func (s *App) parseHeapLine(content string, envKey string) (initSize, maxSize string) {
	re := regexp.MustCompile(envKey + `=(.+)`)
	m := re.FindStringSubmatch(content)
	if len(m) != 2 {
		return
	}
	opts := m[1]
	if mi := regexp.MustCompile(`-Xms(\S+)`).FindStringSubmatch(opts); len(mi) == 2 {
		initSize = mi[1]
	}
	if mx := regexp.MustCompile(`-Xmx(\S+)`).FindStringSubmatch(opts); len(mx) == 2 {
		maxSize = mx[1]
	}
	return
}

// setHeapEnv 设置 heap.env 中的堆内存配置
func (s *App) setHeapEnv(content string, req ConfigTune) string {
	// 读取已有值作为默认
	oldNsInit, oldNsMax := s.parseHeapLine(content, "ROCKETMQ_NAMESRV_HEAP")
	oldBrInit, oldBrMax := s.parseHeapLine(content, "ROCKETMQ_BROKER_HEAP")

	nsInit := req.NamesrvHeapInitSize
	if nsInit == "" {
		nsInit = oldNsInit
	}
	if nsInit == "" {
		nsInit = "512m"
	}
	nsMax := req.NamesrvHeapMaxSize
	if nsMax == "" {
		nsMax = oldNsMax
	}
	if nsMax == "" {
		nsMax = "512m"
	}

	brInit := req.BrokerHeapInitSize
	if brInit == "" {
		brInit = oldBrInit
	}
	if brInit == "" {
		brInit = "1g"
	}
	brMax := req.BrokerHeapMaxSize
	if brMax == "" {
		brMax = oldBrMax
	}
	if brMax == "" {
		brMax = "1g"
	}

	return fmt.Sprintf("ROCKETMQ_NAMESRV_HEAP=-Xms%s -Xmx%s\nROCKETMQ_BROKER_HEAP=-Xms%s -Xmx%s\n", nsInit, nsMax, brInit, brMax)
}
