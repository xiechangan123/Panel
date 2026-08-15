package mysql

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/go-chi/chi/v5"
	"github.com/leonelquinteros/gotext"
	"github.com/spf13/cast"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/apps/confval"
	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/service"
	"github.com/acepanel/panel/v3/pkg/db"
	"github.com/acepanel/panel/v3/pkg/io"
	"github.com/acepanel/panel/v3/pkg/shell"
	"github.com/acepanel/panel/v3/pkg/systemctl"
	"github.com/acepanel/panel/v3/pkg/tools"
	"github.com/acepanel/panel/v3/pkg/types"
)

type App struct {
	t                  *gotext.Locale
	settingRepo        biz.SettingRepo
	databaseServerRepo biz.DatabaseServerRepo
}

func NewApp(t *gotext.Locale, databaseServerRepo biz.DatabaseServerRepo, settingRepo biz.SettingRepo) *App {
	return &App{
		t:                  t,
		settingRepo:        settingRepo,
		databaseServerRepo: databaseServerRepo,
	}
}

func (s *App) Route(r chi.Router) {
	r.Get("/load", s.Load)
	r.Get("/config", s.GetConfig)
	r.Post("/config", s.UpdateConfig)
	r.Get("/slow_log", s.SlowLog)
	r.Get("/root_password", s.GetRootPassword)
	r.Post("/root_password", s.SetRootPassword)
	r.Get("/config_tune", s.GetConfigTune)
	r.Post("/config_tune", s.UpdateConfigTune)
}

func (s *App) Status() string {
	ok, _ := systemctl.Status("mysqld")
	return types.AggregateAppStatus(ok)
}

// GetConfig 获取配置
func (s *App) GetConfig(w http.ResponseWriter, r *http.Request) {
	config, err := io.Read(app.Root + "/server/mysql/conf/my.cnf")
	if err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	service.Success(w, config)
}

// UpdateConfig 保存配置
func (s *App) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	req, err := service.Bind[UpdateConfig](r)
	if err != nil {
		service.Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = io.Write(app.Root+"/server/mysql/conf/my.cnf", req.Config, 0644); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	if err = systemctl.Restart("mysqld"); err != nil {
		service.Error(w, http.StatusInternalServerError, s.t.Get("failed to restart MySQL: %v", err))
		return
	}

	service.Success(w, nil)
}

// Load 获取负载
func (s *App) Load(w http.ResponseWriter, r *http.Request) {
	status, _ := systemctl.Status("mysqld")
	if !status {
		service.Success(w, []types.NV{})
		return
	}

	rootPassword, err := s.settingRepo.Get(biz.SettingKeyMySQLRootPassword)
	if err != nil {
		service.Error(w, http.StatusInternalServerError, s.t.Get("failed to load MySQL root password: %v", err))
		return
	}

	raw, err := shell.ExecfWithEnv([]string{"MYSQL_PWD=" + rootPassword}, `mysqladmin -u root extended-status`)
	if err != nil {
		service.Error(w, http.StatusInternalServerError, s.t.Get("failed to get MySQL status: %v", err))
		return
	}

	var load []map[string]string
	expressions := []struct {
		regex string
		name  string
	}{
		{`Uptime\s+\|\s+(\d+)\s+\|`, s.t.Get("Uptime")},
		{`Queries\s+\|\s+(\d+)\s+\|`, s.t.Get("Total Queries")},
		{`Connections\s+\|\s+(\d+)\s+\|`, s.t.Get("Total Connections")},
		{`Com_commit\s+\|\s+(\d+)\s+\|`, s.t.Get("Transactions per Second")},
		{`Com_rollback\s+\|\s+(\d+)\s+\|`, s.t.Get("Rollbacks per Second")},
		{`Bytes_sent\s+\|\s+(\d+)\s+\|`, s.t.Get("Bytes Sent")},
		{`Bytes_received\s+\|\s+(\d+)\s+\|`, s.t.Get("Bytes Received")},
		{`Threads_connected\s+\|\s+(\d+)\s+\|`, s.t.Get("Active Connections")},
		{`Max_used_connections\s+\|\s+(\d+)\s+\|`, s.t.Get("Peak Connections")},
		{`Key_read_requests\s+\|\s+(\d+)\s+\|`, s.t.Get("Index Hit Rate")},
		{`Innodb_buffer_pool_reads\s+\|\s+(\d+)\s+\|`, s.t.Get("Innodb Index Hit Rate")},
		{`Created_tmp_disk_tables\s+\|\s+(\d+)\s+\|`, s.t.Get("Temporary Tables Created on Disk")},
		{`Open_tables\s+\|\s+(\d+)\s+\|`, s.t.Get("Open Tables")},
		{`Select_full_join\s+\|\s+(\d+)\s+\|`, s.t.Get("Full Joins without Index")},
		{`Select_full_range_join\s+\|\s+(\d+)\s+\|`, s.t.Get("Full Range Joins without Index")},
		{`Select_range_check\s+\|\s+(\d+)\s+\|`, s.t.Get("Subqueries without Index")},
		{`Sort_merge_passes\s+\|\s+(\d+)\s+\|`, s.t.Get("Sort Merge Passes")},
		{`Table_locks_waited\s+\|\s+(\d+)\s+\|`, s.t.Get("Table Locks Waited")},
	}

	for _, expression := range expressions {
		re := regexp.MustCompile(expression.regex)
		matches := re.FindStringSubmatch(raw)
		if len(matches) > 1 {
			d := map[string]string{"name": expression.name, "value": matches[1]}
			if expression.name == s.t.Get("Bytes Sent") || expression.name == s.t.Get("Bytes Received") {
				d["value"] = tools.FormatBytes(cast.ToFloat64(matches[1]))
			}

			load = append(load, d)
		}
	}

	// 索引命中率
	readRequests := cast.ToFloat64(load[9]["value"])
	reads := cast.ToFloat64(load[10]["value"])
	load[9]["value"] = fmt.Sprintf("%.2f%%", readRequests/(reads+readRequests)*100)
	// Innodb 索引命中率
	bufferPoolReads := cast.ToFloat64(load[11]["value"])
	bufferPoolReadRequests := cast.ToFloat64(load[12]["value"])
	load[10]["value"] = fmt.Sprintf("%.2f%%", bufferPoolReadRequests/(bufferPoolReads+bufferPoolReadRequests)*100)

	// 查询缓存命中率
	// MySQL 8.0+ 删除了查询缓存功能
	qcacheHitsRe := regexp.MustCompile(`Qcache_hits\s+\|\s+(\d+)\s+\|`)
	qcacheHitsMatches := qcacheHitsRe.FindStringSubmatch(raw)
	qcacheInsertsRe := regexp.MustCompile(`Qcache_inserts\s+\|\s+(\d+)\s+\|`)
	qcacheInsertsMatches := qcacheInsertsRe.FindStringSubmatch(raw)
	qcacheNotCachedRe := regexp.MustCompile(`Qcache_not_cached\s+\|\s+(\d+)\s+\|`)
	qcacheNotCachedMatches := qcacheNotCachedRe.FindStringSubmatch(raw)
	if len(qcacheHitsMatches) > 1 && len(qcacheInsertsMatches) > 1 && len(qcacheNotCachedMatches) > 1 {
		qcacheHits := cast.ToFloat64(qcacheHitsMatches[1])
		qcacheInserts := cast.ToFloat64(qcacheInsertsMatches[1])
		qcacheNotCached := cast.ToFloat64(qcacheNotCachedMatches[1])
		var qcacheHitRate float64
		denominator := qcacheHits + qcacheInserts + qcacheNotCached
		if denominator > 0 {
			qcacheHitRate = qcacheHits / denominator * 100
		}
		load = append(load, map[string]string{
			"name":  s.t.Get("Query Cache Hits"),
			"value": qcacheHitsMatches[1],
		})
		load = append(load, map[string]string{
			"name":  s.t.Get("Query Cache Inserts"),
			"value": qcacheInsertsMatches[1],
		})
		load = append(load, map[string]string{
			"name":  s.t.Get("Query Cache Not Cached"),
			"value": qcacheNotCachedMatches[1],
		})
		load = append(load, map[string]string{
			"name":  s.t.Get("Query Cache Hit Rate"),
			"value": fmt.Sprintf("%.2f%%", qcacheHitRate),
		})
	}

	service.Success(w, load)
}

// SlowLog 获取慢查询日志
func (s *App) SlowLog(w http.ResponseWriter, r *http.Request) {
	service.Success(w, app.Root+"/server/mysql/mysql-slow.log")
}

// GetRootPassword 获取root密码
func (s *App) GetRootPassword(w http.ResponseWriter, r *http.Request) {
	rootPassword, err := s.settingRepo.Get(biz.SettingKeyMySQLRootPassword)
	if err != nil {
		service.Error(w, http.StatusInternalServerError, s.t.Get("failed to load MySQL root password: %v", err))
		return
	}

	service.Success(w, rootPassword)
}

// SetRootPassword 设置root密码
func (s *App) SetRootPassword(w http.ResponseWriter, r *http.Request) {
	req, err := service.Bind[SetRootPassword](r)
	if err != nil {
		service.Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	oldRootPassword, _ := s.settingRepo.Get(biz.SettingKeyMySQLRootPassword)
	mysql, err := db.NewMySQL(r.Context(), "root", oldRootPassword, db.MySQLSocket(app.Root), "unix")
	if err != nil {
		// 尝试安全模式直接改密
		if err = db.MySQLResetRootPassword(req.Password, app.Root); err != nil {
			service.Error(w, http.StatusInternalServerError, "%v", err)
			return
		}
	} else {
		defer mysql.Close()
		if err = mysql.UserPassword("root", req.Password, "localhost"); err != nil {
			service.Error(w, http.StatusInternalServerError, "%v", err)
			return
		}
	}

	if err = s.settingRepo.Set(biz.SettingKeyMySQLRootPassword, req.Password); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	_ = s.databaseServerRepo.UpdatePassword("local_mysql", req.Password)

	service.Success(w, nil)
}

// GetConfigTune 获取 MySQL 配置调整参数
func (s *App) GetConfigTune(w http.ResponseWriter, r *http.Request) {
	config, err := io.Read(app.Root + "/server/mysql/conf/my.cnf")
	if err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	tune := ConfigTune{
		// 常规设置
		Port:                 confval.INI.Get(config, "port"),
		MaxConnections:       confval.INI.Get(config, "max_connections"),
		MaxConnectErrors:     confval.INI.Get(config, "max_connect_errors"),
		DefaultStorageEngine: confval.INI.Get(config, "default_storage_engine"),
		TableOpenCache:       confval.INI.Get(config, "table_open_cache"),
		MaxAllowedPacket:     confval.INI.Get(config, "max_allowed_packet"),
		OpenFilesLimit:       confval.INI.Get(config, "open_files_limit"),
		// 性能调整
		KeyBufferSize:        confval.INI.Get(config, "key_buffer_size"),
		SortBufferSize:       confval.INI.Get(config, "sort_buffer_size"),
		ReadBufferSize:       confval.INI.Get(config, "read_buffer_size"),
		ReadRndBufferSize:    confval.INI.Get(config, "read_rnd_buffer_size"),
		JoinBufferSize:       confval.INI.Get(config, "join_buffer_size"),
		ThreadCacheSize:      confval.INI.Get(config, "thread_cache_size"),
		ThreadStack:          confval.INI.Get(config, "thread_stack"),
		TmpTableSize:         confval.INI.Get(config, "tmp_table_size"),
		MaxHeapTableSize:     confval.INI.Get(config, "max_heap_table_size"),
		MyisamSortBufferSize: confval.INI.Get(config, "myisam_sort_buffer_size"),
		// InnoDB
		InnodbBufferPoolSize:      confval.INI.Get(config, "innodb_buffer_pool_size"),
		InnodbLogBufferSize:       confval.INI.Get(config, "innodb_log_buffer_size"),
		InnodbFlushLogAtTrxCommit: confval.INI.Get(config, "innodb_flush_log_at_trx_commit"),
		InnodbLockWaitTimeout:     confval.INI.Get(config, "innodb_lock_wait_timeout"),
		InnodbMaxDirtyPagesPct:    confval.INI.Get(config, "innodb_max_dirty_pages_pct"),
		InnodbReadIoThreads:       confval.INI.Get(config, "innodb_read_io_threads"),
		InnodbWriteIoThreads:      confval.INI.Get(config, "innodb_write_io_threads"),
		// 日志
		SlowQueryLog:  confval.INI.Get(config, "slow_query_log"),
		LongQueryTime: confval.INI.Get(config, "long_query_time"),
	}

	service.Success(w, tune)
}

// UpdateConfigTune 更新 MySQL 配置调整参数
func (s *App) UpdateConfigTune(w http.ResponseWriter, r *http.Request) {
	req, err := service.Bind[ConfigTune](r)
	if err != nil {
		service.Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	confPath := app.Root + "/server/mysql/conf/my.cnf"
	config, err := io.Read(confPath)
	if err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	// 更新常规设置
	config = confval.INI.Set(config, "port", req.Port)
	config = confval.INI.Set(config, "max_connections", req.MaxConnections)
	config = confval.INI.Set(config, "max_connect_errors", req.MaxConnectErrors)
	config = confval.INI.Set(config, "default_storage_engine", req.DefaultStorageEngine)
	config = confval.INI.Set(config, "table_open_cache", req.TableOpenCache)
	config = confval.INI.Set(config, "max_allowed_packet", req.MaxAllowedPacket)
	config = confval.INI.Set(config, "open_files_limit", req.OpenFilesLimit)
	// 更新性能调整
	config = confval.INI.Set(config, "key_buffer_size", req.KeyBufferSize)
	config = confval.INI.Set(config, "sort_buffer_size", req.SortBufferSize)
	config = confval.INI.Set(config, "read_buffer_size", req.ReadBufferSize)
	config = confval.INI.Set(config, "read_rnd_buffer_size", req.ReadRndBufferSize)
	config = confval.INI.Set(config, "join_buffer_size", req.JoinBufferSize)
	config = confval.INI.Set(config, "thread_cache_size", req.ThreadCacheSize)
	config = confval.INI.Set(config, "thread_stack", req.ThreadStack)
	config = confval.INI.Set(config, "tmp_table_size", req.TmpTableSize)
	config = confval.INI.Set(config, "max_heap_table_size", req.MaxHeapTableSize)
	config = confval.INI.Set(config, "myisam_sort_buffer_size", req.MyisamSortBufferSize)
	// 更新 InnoDB
	config = confval.INI.Set(config, "innodb_buffer_pool_size", req.InnodbBufferPoolSize)
	config = confval.INI.Set(config, "innodb_log_buffer_size", req.InnodbLogBufferSize)
	config = confval.INI.Set(config, "innodb_flush_log_at_trx_commit", req.InnodbFlushLogAtTrxCommit)
	config = confval.INI.Set(config, "innodb_lock_wait_timeout", req.InnodbLockWaitTimeout)
	config = confval.INI.Set(config, "innodb_max_dirty_pages_pct", req.InnodbMaxDirtyPagesPct)
	config = confval.INI.Set(config, "innodb_read_io_threads", req.InnodbReadIoThreads)
	config = confval.INI.Set(config, "innodb_write_io_threads", req.InnodbWriteIoThreads)
	// 更新日志
	config = confval.INI.Set(config, "slow_query_log", req.SlowQueryLog)
	config = confval.INI.Set(config, "long_query_time", req.LongQueryTime)

	if err = io.Write(confPath, config, 0644); err != nil {
		service.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}

	service.Success(w, nil)
}
