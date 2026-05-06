package kafka

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

func (s *App) Status() string {
	ok, _ := systemctl.Status("kafka")
	return types.AggregateAppStatus(ok)
}

func (s *App) Load(w http.ResponseWriter, r *http.Request) {
	status, err := systemctl.Status("kafka")
	if err != nil {
		service.Error(w, http.StatusInternalServerError, s.t.Get("failed to get kafka status: %v", err))
		return
	}
	if !status {
		service.Success(w, []types.NV{})
		return
	}

	config, _ := io.Read(s.configPath())

	data := []types.NV{
		{Name: s.t.Get("Node ID"), Value: s.getPropertiesValue(config, "node.id")},
		{Name: s.t.Get("Listeners"), Value: s.getPropertiesValue(config, "listeners")},
		{Name: s.t.Get("Log Dirs"), Value: s.getPropertiesValue(config, "log.dirs")},
		{Name: s.t.Get("Num Partitions"), Value: s.getPropertiesValue(config, "num.partitions")},
		{Name: s.t.Get("Log Retention Hours"), Value: s.getPropertiesValue(config, "log.retention.hours")},
		{Name: s.t.Get("Log Segment Bytes"), Value: s.getPropertiesValue(config, "log.segment.bytes")},
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

	if err = systemctl.Restart("kafka"); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	service.Success(w, nil)
}

// GetConfigTune 获取 Kafka 配置调整参数
func (s *App) GetConfigTune(w http.ResponseWriter, r *http.Request) {
	config, _ := io.Read(s.configPath())

	heapRaw, _ := io.Read(s.heapEnvPath())
	heapInit, heapMax := s.parseHeapEnv(heapRaw)

	tune := ConfigTune{
		NodeID:          s.getPropertiesValue(config, "node.id"),
		Listeners:       s.getPropertiesValue(config, "listeners"),
		LogDirs:         s.getPropertiesValue(config, "log.dirs"),
		NumPartitions:   s.getPropertiesValue(config, "num.partitions"),
		RetentionHours:  s.getPropertiesValue(config, "log.retention.hours"),
		LogSegmentBytes: s.getPropertiesValue(config, "log.segment.bytes"),
		HeapInitSize:    heapInit,
		HeapMaxSize:     heapMax,
	}

	service.Success(w, tune)
}

// UpdateConfigTune 更新 Kafka 配置调整参数
func (s *App) UpdateConfigTune(w http.ResponseWriter, r *http.Request) {
	req, err := service.Bind[ConfigTune](r)
	if err != nil {
		service.Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	config, _ := io.Read(s.configPath())

	config = s.setPropertiesValue(config, "node.id", req.NodeID)
	config = s.setPropertiesValue(config, "listeners", req.Listeners)
	config = s.setPropertiesValue(config, "log.dirs", req.LogDirs)
	config = s.setPropertiesValue(config, "num.partitions", req.NumPartitions)
	config = s.setPropertiesValue(config, "log.retention.hours", req.RetentionHours)
	config = s.setPropertiesValue(config, "log.segment.bytes", req.LogSegmentBytes)

	if err = io.Write(s.configPath(), config, 0644); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	// 更新 JVM 堆内存
	if req.HeapInitSize != "" || req.HeapMaxSize != "" {
		heapRaw, _ := io.Read(s.heapEnvPath())
		heapRaw = s.setHeapEnv(heapRaw, req.HeapInitSize, req.HeapMaxSize)
		if err = io.Write(s.heapEnvPath(), heapRaw, 0644); err != nil {
			service.Error(w, http.StatusInternalServerError, "%v", err)
			return
		}
	}

	if err = systemctl.Restart("kafka"); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	service.Success(w, nil)
}

// configPath 返回配置文件路径
func (s *App) configPath() string {
	return fmt.Sprintf("%s/server/kafka/config/server.properties", app.Root)
}

// heapEnvPath 返回 JVM 堆内存配置文件路径
func (s *App) heapEnvPath() string {
	return fmt.Sprintf("%s/server/kafka/config/heap.env", app.Root)
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

// parseHeapEnv 从 heap.env 中提取堆内存配置
func (s *App) parseHeapEnv(content string) (initSize, maxSize string) {
	re := regexp.MustCompile(`KAFKA_HEAP_OPTS=(.+)`)
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
func (s *App) setHeapEnv(content string, initSize, maxSize string) string {
	// 读取已有值作为默认
	oldInit, oldMax := s.parseHeapEnv(content)
	if initSize == "" {
		initSize = oldInit
	}
	if maxSize == "" {
		maxSize = oldMax
	}
	if initSize == "" {
		initSize = "1g"
	}
	if maxSize == "" {
		maxSize = "1g"
	}
	return fmt.Sprintf("KAFKA_HEAP_OPTS=-Xms%s -Xmx%s\n", initSize, maxSize)
}
