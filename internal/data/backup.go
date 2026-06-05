package data

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/leonelquinteros/gotext"
	"github.com/samber/lo"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/pkg/config"
	"github.com/acepanel/panel/v3/pkg/db"
	"github.com/acepanel/panel/v3/pkg/io"
	"github.com/acepanel/panel/v3/pkg/shell"
	"github.com/acepanel/panel/v3/pkg/storage"
	"github.com/acepanel/panel/v3/pkg/systemctl"
	"github.com/acepanel/panel/v3/pkg/tools"
	"github.com/acepanel/panel/v3/pkg/types"
)

type backupRepo struct {
	hr      string
	t       *gotext.Locale
	conf    *config.Config
	db      *gorm.DB
	log     *slog.Logger
	setting biz.SettingRepo
	website biz.WebsiteRepo
}

func NewBackupRepo(t *gotext.Locale, conf *config.Config, db *gorm.DB, log *slog.Logger, setting biz.SettingRepo, website biz.WebsiteRepo) biz.BackupRepo {
	return &backupRepo{
		hr:      "+----------------------------------------------------",
		t:       t,
		conf:    conf,
		db:      db,
		log:     log,
		setting: setting,
		website: website,
	}
}

// List 备份列表
func (r *backupRepo) List(typ biz.BackupType) ([]*types.BackupFile, error) {
	path := r.GetDefaultPath(typ)
	files, err := os.ReadDir(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return make([]*types.BackupFile, 0), nil
		}
		return nil, err
	}

	list := make([]*types.BackupFile, 0)
	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			continue
		}
		list = append(list, &types.BackupFile{
			Name: file.Name(),
			Path: filepath.Join(path, file.Name()),
			Size: tools.FormatBytes(float64(info.Size())),
			Time: info.ModTime(),
		})
	}

	return list, nil
}

// Create 创建备份
// typ 备份类型
// target 目标名称
// storage 备份存储ID
func (r *backupRepo) Create(ctx context.Context, typ biz.BackupType, target string, storage uint) error {
	// 取备份存储，0 为本地备份
	backupStorage := new(biz.BackupStorage)
	if storage != 0 {
		if err := r.db.First(backupStorage, storage).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New(r.t.Get("backup storage not found"))
			}
			return err
		}
	} else {
		backupStorage = &biz.BackupStorage{
			Name: r.t.Get("Local Storage"),
			Type: biz.BackupStorageTypeLocal,
			Info: types.BackupStorageInfo{
				Path: filepath.Dir(r.GetDefaultPath(typ)), // 需要取根目录
			},
		}
	}

	client, err := r.getStorage(*backupStorage)
	if err != nil {
		return err
	}

	start := time.Now()
	namePrefix := target
	if typ == biz.BackupTypePath {
		namePrefix = filepath.Base(target)
	}
	name := fmt.Sprintf("%s_%s", namePrefix, start.Format("20060102150405"))
	if app.IsCli {
		fmt.Println(r.hr)
		fmt.Println(r.t.Get("★ Start backup [%s]", start.Format(time.DateTime)))
		fmt.Println(r.hr)
		fmt.Println(r.t.Get("|-Backup type: %s", string(typ)))
		fmt.Println(r.t.Get("|-Backup storage: %s", backupStorage.Name))
		fmt.Println(r.t.Get("|-Backup target: %s", target))
	}

	switch typ {
	case biz.BackupTypeWebsite:
		err = r.createWebsite(name, client, target)
	case biz.BackupTypeMySQL:
		err = r.createMySQL(name, client, target)
	case biz.BackupTypePostgres:
		err = r.createPostgres(name, client, target)
	case biz.BackupTypeClickHouse:
		err = r.createClickHouse(name, client, target)
	case biz.BackupTypeRedis:
		err = r.createRedisLike(name, client, "redis")
	case biz.BackupTypeValkey:
		err = r.createRedisLike(name, client, "valkey")
	case biz.BackupTypePath:
		err = r.createPath(name, client, target)
	default:
		return errors.New(r.t.Get("unknown backup type"))
	}

	if app.IsCli {
		fmt.Println(r.t.Get("|-Backup time: %s", time.Since(start).String()))
		fmt.Println(r.hr)
	}
	if err != nil {
		r.log.Warn("backup failed",
			slog.String("type", biz.OperationTypeBackup),
			slog.Uint64("operator_id", getOperatorID(ctx)),
			slog.String("backup_type", string(typ)),
			slog.String("target", target),
		)
		if app.IsCli {
			fmt.Println(r.t.Get("☆ Backup failed: %v [%s]", err, time.Now().Format(time.DateTime)))
		}
	} else {
		r.log.Info("backup created",
			slog.String("type", biz.OperationTypeBackup),
			slog.Uint64("operator_id", getOperatorID(ctx)),
			slog.String("backup_type", string(typ)),
			slog.String("target", target),
		)
		if app.IsCli {
			fmt.Println(r.t.Get("☆ Backup completed [%s]", time.Now().Format(time.DateTime)))
		}
	}

	if app.IsCli {
		fmt.Println(r.hr)
	}

	return err
}

// CreatePanel 创建面板备份
// 面板备份始终保存在本地
func (r *backupRepo) CreatePanel() error {
	start := time.Now()

	backup := filepath.Join(r.GetDefaultPath(biz.BackupTypePanel), fmt.Sprintf("panel_%s%s", time.Now().Format("20060102150405"), r.backupExt()))

	temp, err := os.MkdirTemp("", "ace-backup-*")
	if err != nil {
		return err
	}
	defer func(path string) { _ = os.RemoveAll(path) }(temp)

	if err = io.Cp(filepath.Join(app.Root, "panel"), temp); err != nil {
		return err
	}
	if err = io.Cp("/usr/local/sbin/acepanel", temp); err != nil {
		return err
	}

	_ = io.Chmod(temp, 0600)
	if err = io.Compress(temp, nil, backup); err != nil {
		return err
	}
	if err = io.Chmod(backup, 0600); err != nil {
		return err
	}

	if app.IsCli {
		fmt.Println(r.t.Get("|-Backup time: %s", time.Since(start).String()))
		fmt.Println(r.t.Get("|-Backup file: %s", filepath.Base(backup)))
	}

	return nil
}

// Delete 删除备份
func (r *backupRepo) Delete(ctx context.Context, typ biz.BackupType, name string) error {
	path := r.GetDefaultPath(typ)

	file := filepath.Join(path, name)
	if err := io.Remove(file); err != nil {
		return err
	}

	// 记录日志
	r.log.Info("backup deleted", slog.String("type", biz.OperationTypeBackup), slog.Uint64("operator_id", getOperatorID(ctx)), slog.String("backup_type", string(typ)), slog.String("name", name))

	return nil
}

// Restore 恢复备份
// typ 备份类型
// backup 备份压缩包，可以是绝对路径或者相对路径
// target 目标名称
func (r *backupRepo) Restore(ctx context.Context, typ biz.BackupType, backup, target string) error {
	if !io.Exists(backup) {
		backup = filepath.Join(r.GetDefaultPath(typ), backup)
	}
	if !io.Exists(backup) {
		return errors.New(r.t.Get("backup file not exists"))
	}

	var err error
	switch typ {
	case biz.BackupTypeWebsite:
		err = r.restoreWebsite(backup, target)
	case biz.BackupTypeMySQL:
		err = r.restoreMySQL(backup, target)
	case biz.BackupTypePostgres:
		err = r.restorePostgres(backup, target)
	case biz.BackupTypeClickHouse:
		err = r.restoreClickHouse(backup, target)
	case biz.BackupTypeRedis:
		err = r.restoreRedisLike(backup, "redis")
	case biz.BackupTypeValkey:
		err = r.restoreRedisLike(backup, "valkey")
	default:
		return errors.New(r.t.Get("unknown backup type"))
	}

	if err != nil {
		return err
	}

	// 记录日志
	r.log.Info("backup restored",
		slog.String("type", biz.OperationTypeBackup),
		slog.Uint64("operator_id", getOperatorID(ctx)),
		slog.String("backup_type", string(typ)),
		slog.String("target", target),
	)

	return nil
}

// GetDefaultPath 获取默认备份路径
func (r *backupRepo) GetDefaultPath(typ biz.BackupType) string {
	backupPath, err := r.setting.Get(biz.SettingKeyBackupPath)
	if err != nil {
		backupPath = filepath.Join(app.Root, "backup")
	}
	path := filepath.Join(backupPath, string(typ))
	_ = os.MkdirAll(path, 0755)
	return path
}

// CutoffLog 切割日志
// path 保存目录绝对路径
// target 待切割日志文件绝对路径
func (r *backupRepo) CutoffLog(path, target string) (string, error) {
	if !io.Exists(target) {
		return "", errors.New(r.t.Get("log file %s not exists", target))
	}

	name := strings.TrimSuffix(filepath.Base(target), filepath.Ext(target))
	to := filepath.Join(path, fmt.Sprintf("%s_%s%s", name, time.Now().Format("20060102150405"), r.backupExt()))
	if err := io.Compress(filepath.Dir(target), []string{filepath.Base(target)}, to); err != nil {
		return "", err
	}

	// 原文件不能直接删除，直接删的话仍会占用空间直到重启相关的应用
	if _, err := shell.Execf("cat /dev/null > '%s'", target); err != nil {
		return "", err
	}

	return to, nil
}

// CutoffUpload 将指定的切割日志文件上传到远程存储
func (r *backupRepo) CutoffUpload(account uint, typ biz.BackupType, name string, files []string) error {
	backupStorage := new(biz.BackupStorage)
	if err := r.db.First(backupStorage, account).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New(r.t.Get("backup storage not found"))
		}
		return err
	}

	client, err := r.getStorage(*backupStorage)
	if err != nil {
		return err
	}

	for _, localPath := range files {
		file, err := os.Open(localPath)
		if err != nil {
			return err
		}
		remotePath := filepath.Join("cutoff", string(typ), name, filepath.Base(localPath))
		if putErr := client.Put(remotePath, file); putErr != nil {
			_ = file.Close()
			return putErr
		}
		_ = file.Close()
	}

	return nil
}

// ClearExpired 清理过期备份
// path 备份目录绝对路径
// prefix 目标文件前缀
// save 保存份数
func (r *backupRepo) ClearExpired(path, prefix string, save uint) error {
	files, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	filtered := lo.FilterMap(files, func(file os.DirEntry, _ int) (os.FileInfo, bool) {
		if !strings.HasPrefix(file.Name(), prefix) || !r.isBackupArchive(file.Name()) {
			return nil, false
		}
		info, err := os.Stat(filepath.Join(path, file.Name()))
		if err != nil {
			return nil, false
		}
		return info, true
	})

	// 排序所有备份文件，从新到旧
	slices.SortFunc(filtered, func(a, b os.FileInfo) int {
		if a.ModTime().After(b.ModTime()) {
			return -1
		}
		if a.ModTime().Before(b.ModTime()) {
			return 1
		}
		return 0
	})
	if uint(len(filtered)) <= save {
		return nil
	}

	// 切片保留 save 份，删除剩余
	toDelete := filtered[save:]
	for _, file := range toDelete {
		filePath := filepath.Join(path, file.Name())
		if app.IsCli {
			fmt.Println(r.t.Get("|-Cleaning expired file: %s", filePath))
		}
		if err = os.Remove(filePath); err != nil {
			return errors.New(r.t.Get("Cleanup failed: %v", err))
		}
	}

	return nil
}

// ClearStorageExpired 清理备份账号过期备份
func (r *backupRepo) ClearStorageExpired(storage uint, typ biz.BackupType, prefix string, save uint) error {
	backupStorage := new(biz.BackupStorage)
	if err := r.db.First(backupStorage, storage).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New(r.t.Get("backup storage not found"))
		}
		return err
	}

	client, err := r.getStorage(*backupStorage)
	if err != nil {
		return err
	}

	files, err := client.List(string(typ))
	if err != nil {
		return err
	}

	type fileInfo struct {
		name    string
		modTime time.Time
	}
	var filtered []fileInfo
	for _, file := range files {
		if strings.HasPrefix(file, prefix) && r.isBackupArchive(file) {
			lastModified, modErr := client.LastModified(filepath.Join(string(typ), file))
			if modErr != nil {
				continue
			}
			filtered = append(filtered, fileInfo{name: file, modTime: lastModified})
		}
	}

	// 排序所有备份文件，从新到旧
	slices.SortFunc(filtered, func(a, b fileInfo) int {
		if a.modTime.After(b.modTime) {
			return -1
		}
		if a.modTime.Before(b.modTime) {
			return 1
		}
		return 0
	})
	if uint(len(filtered)) <= save {
		return nil
	}

	// 切片保留 save 份，删除剩余
	toDelete := filtered[save:]
	for _, file := range toDelete {
		filePath := filepath.Join(string(typ), file.name)
		if app.IsCli {
			fmt.Println(r.t.Get("|-Cleaning expired file: %s", filePath))
		}
		if err = client.Delete(filePath); err != nil {
			return errors.New(r.t.Get("Cleanup failed: %v", err))
		}
	}

	return nil
}

// getStorage 获取存储器
func (r *backupRepo) getStorage(backupStorage biz.BackupStorage) (storage.Storage, error) {
	switch backupStorage.Type {
	case biz.BackupStorageTypeLocal:
		return storage.NewLocal(backupStorage.Info.Path)
	case biz.BackupStorageTypeS3:
		return storage.NewS3(storage.S3Config{
			Region:          backupStorage.Info.Region,
			Bucket:          backupStorage.Info.Bucket,
			AccessKey:       backupStorage.Info.AccessKey,
			SecretKey:       backupStorage.Info.SecretKey,
			Endpoint:        backupStorage.Info.Endpoint,
			Scheme:          backupStorage.Info.Scheme,
			BasePath:        backupStorage.Info.Path,
			AddressingStyle: storage.S3AddressingStyle(backupStorage.Info.Style),
		})
	case biz.BackupStorageTypeSFTP:
		return storage.NewSFTP(storage.SFTPConfig{
			Host:       backupStorage.Info.Host,
			Port:       backupStorage.Info.Port,
			Username:   backupStorage.Info.Username,
			Password:   backupStorage.Info.Password,
			PrivateKey: backupStorage.Info.PrivateKey,
			BasePath:   backupStorage.Info.Path,
		})
	case biz.BackupStorageTypeWebDAV:
		return storage.NewWebDav(storage.WebDavConfig{
			URL:      backupStorage.Info.URL,
			Username: backupStorage.Info.Username,
			Password: backupStorage.Info.Password,
			BasePath: backupStorage.Info.Path,
		})
	default:
		return nil, errors.New(r.t.Get("unknown storage type"))
	}
}

// createWebsite 创建网站备份
func (r *backupRepo) createWebsite(name string, storage storage.Storage, target string) error {
	website, err := r.website.GetByName(target)
	if err != nil {
		return err
	}

	// 创建用于压缩的临时目录
	tmpDir, err := os.MkdirTemp("", "ace-backup-*")
	if err != nil {
		return err
	}
	defer func(path string) { _ = os.RemoveAll(path) }(tmpDir)

	if app.IsCli {
		fmt.Println(r.t.Get("|-Temporary directory: %s", tmpDir))
	}

	// 压缩网站
	name = name + r.backupExt()
	if err = io.Compress(website.Path, nil, filepath.Join(tmpDir, name)); err != nil {
		return err
	}

	// 上传备份文件到存储器
	file, err := os.Open(filepath.Join(tmpDir, name))
	if err != nil {
		return err
	}
	defer func(file *os.File) { _ = file.Close() }(file)

	if err = storage.Put(filepath.Join(string(biz.BackupTypeWebsite), name), file); err != nil {
		return err
	}

	if app.IsCli {
		fmt.Println(r.t.Get("|-Backup file: %s", name))
	}

	return nil
}

// createMySQL 创建 MySQL 备份
func (r *backupRepo) createMySQL(name string, storage storage.Storage, target string) error {
	rootPassword, err := r.setting.Get(biz.SettingKeyMySQLRootPassword)
	if err != nil {
		return err
	}
	mysql, err := db.NewMySQL("root", rootPassword, "/tmp/mysql.sock", "unix")
	if err != nil {
		return err
	}
	defer mysql.Close()
	if exist, _ := mysql.DatabaseExists(target); !exist {
		return errors.New(r.t.Get("database does not exist: %s", target))
	}

	// 创建用于压缩的临时目录
	tmpDir, err := os.MkdirTemp("", "ace-backup-*")
	if err != nil {
		return err
	}
	defer func(path string) { _ = os.RemoveAll(path) }(tmpDir)

	if app.IsCli {
		fmt.Println(r.t.Get("|-Temporary directory: %s", tmpDir))
	}

	// 导出数据库
	name = name + ".sql"
	_ = os.Setenv("MYSQL_PWD", rootPassword)
	if _, err = shell.Execf(`mysqldump -u root --single-transaction --quick '%s' > '%s'`, target, filepath.Join(tmpDir, name)); err != nil {
		return err
	}
	_ = os.Unsetenv("MYSQL_PWD")

	// 压缩备份文件
	if err = io.Compress(tmpDir, []string{name}, filepath.Join(tmpDir, name+r.backupExt())); err != nil {
		return err
	}

	// 上传备份文件到存储器
	name = name + r.backupExt()
	file, err := os.Open(filepath.Join(tmpDir, name))
	if err != nil {
		return err
	}
	defer func(file *os.File) { _ = file.Close() }(file)

	if err = storage.Put(filepath.Join(string(biz.BackupTypeMySQL), name), file); err != nil {
		return err
	}

	if app.IsCli {
		fmt.Println(r.t.Get("|-Backup file: %s", name))
	}

	return nil
}

// createPostgres 创建 PostgreSQL 备份
func (r *backupRepo) createPostgres(name string, storage storage.Storage, target string) error {
	postgresPassword, err := r.setting.Get(biz.SettingKeyPostgresPassword)
	if err != nil {
		return err
	}
	postgres, err := db.NewPostgres("postgres", postgresPassword, "127.0.0.1", 5432)
	if err != nil {
		return err
	}
	defer postgres.Close()
	if exist, _ := postgres.DatabaseExists(target); !exist {
		return errors.New(r.t.Get("database does not exist: %s", target))
	}

	// 创建用于压缩的临时目录
	tmpDir, err := os.MkdirTemp("", "ace-backup-*")
	if err != nil {
		return err
	}
	defer func(path string) { _ = os.RemoveAll(path) }(tmpDir)

	if app.IsCli {
		fmt.Println(r.t.Get("|-Temporary directory: %s", tmpDir))
	}

	// 导出数据库
	name = name + ".sql"
	_ = os.Setenv("PGPASSWORD", postgresPassword)
	if _, err = shell.Execf(`pg_dump -h 127.0.0.1 -U postgres '%s' > '%s'`, target, filepath.Join(tmpDir, name)); err != nil {
		return err
	}
	_ = os.Unsetenv("PGPASSWORD")

	// 压缩备份文件
	if err = io.Compress(tmpDir, []string{name}, filepath.Join(tmpDir, name+r.backupExt())); err != nil {
		return err
	}

	// 上传备份文件到存储器
	name = name + r.backupExt()
	file, err := os.Open(filepath.Join(tmpDir, name))
	if err != nil {
		return err
	}
	defer func(file *os.File) { _ = file.Close() }(file)

	if err = storage.Put(filepath.Join(string(biz.BackupTypePostgres), name), file); err != nil {
		return err
	}

	if app.IsCli {
		fmt.Println(r.t.Get("|-Backup file: %s", name))
	}

	return nil
}

// createClickHouse 创建 ClickHouse 备份
func (r *backupRepo) createClickHouse(name string, storage storage.Storage, target string) error {
	password, err := r.setting.Get(biz.SettingKeyClickHouseDefaultPassword)
	if err != nil {
		return err
	}
	// clickhouse-client 走 native 9000 端口，本地 default 用户
	conn := fmt.Sprintf("--host 127.0.0.1 --port 9000 --user default --password '%s'", password)

	// 校验数据库是否存在
	exist, err := shell.Execf("clickhouse-client %s --query \"SELECT count() FROM system.databases WHERE name = '%s'\"", conn, target)
	if err != nil {
		return err
	}
	if strings.TrimSpace(exist) == "0" {
		return errors.New(r.t.Get("database does not exist: %s", target))
	}

	// 创建用于压缩的临时目录
	tmpDir, err := os.MkdirTemp("", "ace-backup-*")
	if err != nil {
		return err
	}
	defer func(path string) { _ = os.RemoveAll(path) }(tmpDir)

	if app.IsCli {
		fmt.Println(r.t.Get("|-Temporary directory: %s", tmpDir))
	}

	// 数据表（含数据）在前，视图（仅结构）在后
	dataTables, err := r.clickHouseTables(conn, target, false)
	if err != nil {
		return err
	}
	views, err := r.clickHouseTables(conn, target, true)
	if err != nil {
		return err
	}

	// 导出结构到 schema.sql，去掉库名限定（恢复时由 --database 指定目标库）
	objects := make([]string, 0, len(dataTables)+len(views))
	objects = append(objects, dataTables...)
	objects = append(objects, views...)
	var schema strings.Builder
	for _, tbl := range objects {
		create, err := shell.Execf("clickhouse-client %s --query \"SELECT create_table_query FROM system.tables WHERE database = '%s' AND name = '%s' FORMAT TabSeparatedRaw\"", conn, target, tbl)
		if err != nil {
			return err
		}
		stmt := strings.TrimSpace(create)
		stmt = strings.ReplaceAll(stmt, fmt.Sprintf("`%s`.", target), "")
		stmt = strings.ReplaceAll(stmt, fmt.Sprintf("%s.", target), "")
		schema.WriteString(stmt)
		schema.WriteString(";\n")
	}
	if err = io.Write(filepath.Join(tmpDir, "schema.sql"), schema.String(), 0644); err != nil {
		return err
	}

	// 导出数据（仅数据表，Native 格式）
	files := []string{"schema.sql"}
	for _, tbl := range dataTables {
		dataFile := tbl + ".native"
		if _, err = shell.Execf("clickhouse-client %s --query 'SELECT * FROM `%s`.`%s` FORMAT Native' > '%s'", conn, target, tbl, filepath.Join(tmpDir, dataFile)); err != nil {
			return err
		}
		files = append(files, dataFile)
	}

	// 压缩备份文件
	name = name + r.backupExt()
	if err = io.Compress(tmpDir, files, filepath.Join(tmpDir, name)); err != nil {
		return err
	}

	// 上传备份文件到存储器
	file, err := os.Open(filepath.Join(tmpDir, name))
	if err != nil {
		return err
	}
	defer func(file *os.File) { _ = file.Close() }(file)

	if err = storage.Put(filepath.Join(string(biz.BackupTypeClickHouse), name), file); err != nil {
		return err
	}

	if app.IsCli {
		fmt.Println(r.t.Get("|-Backup file: %s", name))
	}

	return nil
}

// clickHouseTables 列出库中对象名，onlyView 为 true 时仅返回视图，否则返回非视图数据表
func (r *backupRepo) clickHouseTables(conn, database string, onlyView bool) ([]string, error) {
	op := "NOT LIKE"
	if onlyView {
		op = "LIKE"
	}
	out, err := shell.Execf("clickhouse-client %s --query \"SELECT name FROM system.tables WHERE database = '%s' AND NOT is_temporary AND engine %s '%%View%%' ORDER BY name FORMAT TabSeparated\"", conn, database, op)
	if err != nil {
		return nil, err
	}

	var tables []string
	for line := range strings.SplitSeq(out, "\n") {
		if l := strings.TrimSpace(line); l != "" {
			tables = append(tables, l)
		}
	}
	return tables, nil
}

// createPath 创建目录备份
func (r *backupRepo) createPath(name string, storage storage.Storage, target string) error {
	if !io.Exists(target) {
		return errors.New(r.t.Get("path does not exist: %s", target))
	}
	if !io.IsDir(target) {
		return errors.New(r.t.Get("path is not a directory: %s", target))
	}

	// 创建用于压缩的临时目录
	tmpDir, err := os.MkdirTemp("", "ace-backup-*")
	if err != nil {
		return err
	}
	defer func(path string) { _ = os.RemoveAll(path) }(tmpDir)

	if app.IsCli {
		fmt.Println(r.t.Get("|-Temporary directory: %s", tmpDir))
	}

	// 压缩目录
	name = name + r.backupExt()
	if err = io.Compress(target, nil, filepath.Join(tmpDir, name)); err != nil {
		return err
	}

	// 上传备份文件到存储器
	file, err := os.Open(filepath.Join(tmpDir, name))
	if err != nil {
		return err
	}
	defer func(file *os.File) { _ = file.Close() }(file)

	if err = storage.Put(filepath.Join(string(biz.BackupTypePath), name), file); err != nil {
		return err
	}

	if app.IsCli {
		fmt.Println(r.t.Get("|-Backup file: %s", name))
	}

	return nil
}

// restoreWebsite 恢复网站备份
func (r *backupRepo) restoreWebsite(backup, target string) error {
	website, err := r.website.GetByName(target)
	if err != nil {
		return err
	}

	if err = io.Remove(website.Path); err != nil {
		return err
	}
	if err = io.UnCompress(backup, website.Path); err != nil {
		return err
	}
	if err = io.Chmod(website.Path, 0755); err != nil {
		return err
	}
	if err = io.Chown(website.Path, "www", "www"); err != nil {
		return err
	}

	return nil
}

// restoreMySQL 恢复 MySQL 备份
func (r *backupRepo) restoreMySQL(backup, target string) error {
	rootPassword, err := r.setting.Get(biz.SettingKeyMySQLRootPassword)
	if err != nil {
		return err
	}
	mysql, err := db.NewMySQL("root", rootPassword, "/tmp/mysql.sock", "unix")
	if err != nil {
		return err
	}
	defer mysql.Close()
	if exist, _ := mysql.DatabaseExists(target); !exist {
		return errors.New(r.t.Get("database does not exist: %s", target))
	}

	clean := false
	if !strings.HasSuffix(backup, ".sql") {
		backup, err = r.autoUnCompressSQL(backup)
		if err != nil {
			return err
		}
		clean = true
	}

	_ = os.Setenv("MYSQL_PWD", rootPassword)
	if _, err = shell.Execf(`mysql -u root '%s' < '%s'`, target, backup); err != nil {
		return err
	}
	_ = os.Unsetenv("MYSQL_PWD")
	if clean {
		_ = io.Remove(filepath.Dir(backup))
	}

	return nil
}

// restorePostgres 恢复 PostgreSQL 备份
func (r *backupRepo) restorePostgres(backup, target string) error {
	postgresPassword, err := r.setting.Get(biz.SettingKeyPostgresPassword)
	if err != nil {
		return err
	}
	postgres, err := db.NewPostgres("postgres", postgresPassword, "127.0.0.1", 5432)
	if err != nil {
		return err
	}
	defer postgres.Close()
	if exist, _ := postgres.DatabaseExists(target); !exist {
		return errors.New(r.t.Get("database does not exist: %s", target))
	}

	clean := false
	if !strings.HasSuffix(backup, ".sql") {
		backup, err = r.autoUnCompressSQL(backup)
		if err != nil {
			return err
		}
		clean = true
	}

	_ = os.Setenv("PGPASSWORD", postgresPassword)
	if _, err = shell.Execf(`psql -h 127.0.0.1 -U postgres '%s' < '%s'`, target, backup); err != nil {
		return err
	}
	_ = os.Unsetenv("PGPASSWORD")
	if clean {
		_ = io.Remove(filepath.Dir(backup))
	}

	return nil
}

// restoreClickHouse 恢复 ClickHouse 备份
func (r *backupRepo) restoreClickHouse(backup, target string) error {
	password, err := r.setting.Get(biz.SettingKeyClickHouseDefaultPassword)
	if err != nil {
		return err
	}
	conn := fmt.Sprintf("--host 127.0.0.1 --port 9000 --user default --password '%s'", password)

	// 校验目标数据库是否存在
	exist, err := shell.Execf("clickhouse-client %s --query \"SELECT count() FROM system.databases WHERE name = '%s'\"", conn, target)
	if err != nil {
		return err
	}
	if strings.TrimSpace(exist) == "0" {
		return errors.New(r.t.Get("database does not exist: %s", target))
	}

	// 纯 SQL 文件直接执行
	if strings.HasSuffix(backup, ".sql") {
		_, err = shell.Execf("clickhouse-client %s --database '%s' --multiquery < '%s'", conn, target, backup)
		return err
	}

	// 解压到临时目录
	tmpDir, err := os.MkdirTemp("", "acepanel-ch-*")
	if err != nil {
		return err
	}
	defer func(path string) { _ = os.RemoveAll(path) }(tmpDir)

	if err = io.UnCompress(backup, tmpDir); err != nil {
		return err
	}

	// 先恢复结构（视图已排在末尾，源表先建好，规避物化视图依赖顺序问题）
	schemaPath := filepath.Join(tmpDir, "schema.sql")
	if io.Exists(schemaPath) {
		if _, err = shell.Execf("clickhouse-client %s --database '%s' --multiquery < '%s'", conn, target, schemaPath); err != nil {
			return err
		}
	}

	// 再导入数据
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".native") {
			continue
		}
		tbl := strings.TrimSuffix(entry.Name(), ".native")
		if _, err = shell.Execf("clickhouse-client %s --database '%s' --query 'INSERT INTO `%s` FORMAT Native' < '%s'", conn, target, tbl, filepath.Join(tmpDir, entry.Name())); err != nil {
			return err
		}
	}

	return nil
}

// redisLikeConf redis/valkey 的连接与持久化参数
type redisLikeConf struct {
	kind       string // redis / valkey
	cli        string // redis-cli / valkey-cli
	authEnv    string // REDISCLI_AUTH / VALKEYCLI_AUTH
	dataDir    string // {app.Root}/server/{kind}
	confPath   string
	port       string
	password   string
	appendonly bool
}

// loadRedisLikeConf 从 {kind}.conf 读取连接与持久化参数
func (r *backupRepo) loadRedisLikeConf(kind string) (*redisLikeConf, error) {
	dataDir := fmt.Sprintf("%s/server/%s", app.Root, kind)
	confPath := fmt.Sprintf("%s/%s.conf", dataDir, kind)
	content, err := io.Read(confPath)
	if err != nil {
		return nil, err
	}

	port := redisLikeValue(content, "port")
	if port == "" {
		port = "6379"
	}
	authEnv := "REDISCLI_AUTH"
	if kind == "valkey" {
		authEnv = "VALKEYCLI_AUTH"
	}

	return &redisLikeConf{
		kind:       kind,
		cli:        kind + "-cli",
		authEnv:    authEnv,
		dataDir:    dataDir,
		confPath:   confPath,
		port:       port,
		password:   redisLikeValue(content, "requirepass"),
		appendonly: redisLikeValue(content, "appendonly") == "yes",
	}, nil
}

// redisLikeValue 从 redis/valkey 配置内容中读取指定键的有效值（忽略注释行）
func redisLikeValue(content, key string) string {
	for line := range strings.SplitSeq(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) >= 2 && fields[0] == key {
			return strings.Join(fields[1:], " ")
		}
	}
	return ""
}

// createRedisLike 创建 Redis/Valkey 整实例备份（{kind}-cli --rdb 导出快照）
// redis 与 valkey 共用本实现，kind 为 "redis" 或 "valkey"
func (r *backupRepo) createRedisLike(name string, storage storage.Storage, kind string) error {
	conf, err := r.loadRedisLikeConf(kind)
	if err != nil {
		return err
	}
	// 用环境变量传密码，避免密码出现在命令行/进程列表
	if conf.password != "" {
		_ = os.Setenv(conf.authEnv, conf.password)
		defer func() { _ = os.Unsetenv(conf.authEnv) }()
	}

	// 创建用于压缩的临时目录
	tmpDir, err := os.MkdirTemp("", "ace-backup-*")
	if err != nil {
		return err
	}
	defer func(path string) { _ = os.RemoveAll(path) }(tmpDir)

	if app.IsCli {
		fmt.Println(r.t.Get("|-Temporary directory: %s", tmpDir))
	}

	// 通过复制协议拉取整实例 RDB 快照到本地文件
	rdb := filepath.Join(tmpDir, "dump.rdb")
	if _, err = shell.Execf("%s -h 127.0.0.1 -p %s --rdb '%s'", conf.cli, conf.port, rdb); err != nil {
		return err
	}
	if !io.Exists(rdb) {
		return errors.New(r.t.Get("failed to export RDB snapshot"))
	}

	// 压缩备份文件
	name = name + r.backupExt()
	if err = io.Compress(tmpDir, []string{"dump.rdb"}, filepath.Join(tmpDir, name)); err != nil {
		return err
	}

	// 上传备份文件到存储器
	file, err := os.Open(filepath.Join(tmpDir, name))
	if err != nil {
		return err
	}
	defer func(file *os.File) { _ = file.Close() }(file)

	if err = storage.Put(filepath.Join(kind, name), file); err != nil {
		return err
	}

	if app.IsCli {
		fmt.Println(r.t.Get("|-Backup file: %s", name))
	}

	return nil
}

// restoreRedisLike 恢复 Redis/Valkey 整实例备份
// 停服务 → 替换 dump.rdb → 启动；妥善处理 AOF 优先级陷阱（appendonly=yes 时 AOF 会盖过 RDB）
func (r *backupRepo) restoreRedisLike(backup, kind string) error {
	conf, err := r.loadRedisLikeConf(kind)
	if err != nil {
		return err
	}

	// 准备 dump.rdb：裸 .rdb 直接用，否则解压取包内的 dump.rdb
	rdb := backup
	if !strings.HasSuffix(backup, ".rdb") {
		tmpDir, err := os.MkdirTemp("", "acepanel-rdb-*")
		if err != nil {
			return err
		}
		defer func(path string) { _ = os.RemoveAll(path) }(tmpDir)
		if err = io.UnCompress(backup, tmpDir); err != nil {
			return err
		}
		rdb = filepath.Join(tmpDir, "dump.rdb")
		if !io.Exists(rdb) {
			return errors.New(r.t.Get("dump.rdb not found in backup file"))
		}
	}

	// 停止服务
	if err = systemctl.Stop(conf.kind); err != nil {
		return err
	}

	// 清理旧 AOF（多部件目录与旧式单文件），避免 AOF 优先于 RDB 被加载
	_ = io.Remove(filepath.Join(conf.dataDir, "appendonlydir"))
	_ = io.Remove(filepath.Join(conf.dataDir, "appendonly.aof"))

	// 覆盖 dump.rdb
	target := filepath.Join(conf.dataDir, "dump.rdb")
	if err = io.Cp(rdb, target); err != nil {
		_ = systemctl.Start(conf.kind) // 尽力恢复服务
		return err
	}
	_ = io.Chown(target, kind, kind)
	_ = io.Chmod(target, 0640)

	// 若原本开启 AOF，必须先以 appendonly no 启动加载 RDB，否则会建空 AOF 以空库覆盖
	if conf.appendonly {
		if err = r.disableAppendonly(conf.confPath); err != nil {
			_ = systemctl.Start(conf.kind)
			return err
		}
	}

	// 启动服务（Type=notify，返回即已加载 RDB）
	if err = systemctl.Start(conf.kind); err != nil {
		return err
	}

	// 原本开启 AOF 的，在线转回并持久化配置
	if conf.appendonly {
		if conf.password != "" {
			_ = os.Setenv(conf.authEnv, conf.password)
			defer func() { _ = os.Unsetenv(conf.authEnv) }()
		}
		_, _ = shell.Execf("%s -h 127.0.0.1 -p %s config set appendonly yes", conf.cli, conf.port)
		_, _ = shell.Execf("%s -h 127.0.0.1 -p %s config rewrite", conf.cli, conf.port)
	}

	return nil
}

// disableAppendonly 将 redis/valkey 配置中的 appendonly 临时改为 no
func (r *backupRepo) disableAppendonly(confPath string) error {
	content, err := io.Read(confPath)
	if err != nil {
		return err
	}

	lines := strings.Split(content, "\n")
	found := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if fields := strings.Fields(trimmed); len(fields) >= 1 && fields[0] == "appendonly" {
			lines[i] = "appendonly no"
			found = true
			break
		}
	}
	if !found {
		lines = append(lines, "appendonly no")
	}

	return io.Write(confPath, strings.Join(lines, "\n"), 0644)
}

// autoUnCompressSQL 自动处理压缩文件
func (r *backupRepo) autoUnCompressSQL(backup string) (string, error) {
	temp, err := os.MkdirTemp("", "acepanel-sql-*")
	if err != nil {
		return "", err
	}

	if err = io.UnCompress(backup, temp); err != nil {
		return "", err
	}

	backup = "" // 置空，防止干扰后续判断
	if files, err := os.ReadDir(temp); err == nil {
		if len(files) != 1 {
			return "", errors.New(r.t.Get("The number of files contained in the compressed file is not 1, actual %d", len(files)))
		}
		if strings.HasSuffix(files[0].Name(), ".sql") {
			backup = filepath.Join(temp, files[0].Name())
		}
	}

	if backup == "" {
		return "", errors.New(r.t.Get("could not find .sql backup file"))
	}

	return backup, nil
}

func (r *backupRepo) FixPanel() error {
	if app.IsCli {
		fmt.Println(r.t.Get("|-Start fixing the panel..."))
	}

	// 检查关键文件是否正常
	panelBroken := !io.Exists(filepath.Join(app.Root, "panel", "ace")) ||
		!io.Exists(filepath.Join(app.Root, "panel", "storage", "config.yml")) ||
		!io.Exists(filepath.Join(app.Root, "panel", "storage", "panel.db")) ||
		io.Exists("/tmp/panel-storage.zip")
	// 检查主数据库连接
	if err := r.db.Exec("VACUUM").Error; err != nil {
		panelBroken = true
	}
	if err := r.db.Exec("PRAGMA wal_checkpoint(TRUNCATE);").Error; err != nil {
		panelBroken = true
	}

	// 检查辅助数据库是否异常
	var brokenAuxDBs []string
	for _, name := range []string{"stat", "scan"} {
		auxDB, err := openDB(name)
		if err == nil {
			if sqlDB, dbErr := auxDB.DB(); dbErr == nil {
				_ = sqlDB.Close()
			}
			continue
		}
		brokenAuxDBs = append(brokenAuxDBs, name)
	}

	// 一切正常，无需修复
	if !panelBroken && len(brokenAuxDBs) == 0 {
		return errors.New(r.t.Get("Files are normal and do not need to be repaired, please run acepanel update to update the panel"))
	}

	// 有异常，先停止面板
	tools.StopPanel()

	// 删除损坏的辅助数据库（会自动重建）
	for _, name := range brokenAuxDBs {
		dbPath := filepath.Join(app.Root, "panel", "storage", name+".db")
		if removeErr := io.Remove(dbPath); removeErr != nil {
			return errors.New(r.t.Get("Failed to remove %s.db: %v", name, removeErr))
		}
		if app.IsCli {
			fmt.Println(r.t.Get("|-Found %s.db is abnormal, removed it", name))
		}
	}

	// 仅辅助数据库异常，重启即可恢复
	if !panelBroken {
		if app.IsCli {
			fmt.Println(r.t.Get("|-Fix completed"))
		}
		tools.RestartPanel()
		return nil
	}

	// 再次确认是否需要修复
	if io.Exists("/tmp/panel-storage.zip") {
		// 文件齐全情况下只移除临时文件
		if io.Exists(filepath.Join(app.Root, "panel", "ace")) &&
			io.Exists(filepath.Join(app.Root, "panel", "storage", "config.yml")) &&
			io.Exists(filepath.Join(app.Root, "panel", "storage", "panel.db")) {
			if err := io.Remove("/tmp/panel-storage.zip"); err != nil {
				return errors.New(r.t.Get("failed to clean temporary files: %v", err))
			}
			if app.IsCli {
				fmt.Println(r.t.Get("|-Cleaned up temporary files, please run acepanel update to update the panel"))
			}
			return nil
		}
	}

	// 从备份目录中找最新的备份文件
	files, err := os.ReadDir(r.GetDefaultPath(biz.BackupTypePanel))
	if err != nil {
		return err
	}
	var list []os.FileInfo
	for _, file := range files {
		if !r.isBackupArchive(file.Name()) {
			continue
		}
		info, infoErr := file.Info()
		if infoErr != nil {
			continue
		}
		list = append(list, info)
	}
	slices.SortFunc(list, func(a os.FileInfo, b os.FileInfo) int {
		return int(b.ModTime().Unix() - a.ModTime().Unix())
	})
	if len(list) == 0 {
		return errors.New(r.t.Get("No backup file found, unable to automatically repair"))
	}
	latest := list[0]
	latestPath := filepath.Join(r.GetDefaultPath(biz.BackupTypePanel), latest.Name())
	if app.IsCli {
		fmt.Println(r.t.Get("|-Backup file used: %s", latest.Name()))
	}

	// 解压备份文件
	if app.IsCli {
		fmt.Println(r.t.Get("|-Unzip backup file..."))
	}
	if err = io.Remove("/tmp/panel-fix"); err != nil {
		return errors.New(r.t.Get("Cleaning temporary directory failed: %v", err))
	}
	if err = io.UnCompress(latestPath, "/tmp/panel-fix"); err != nil {
		return errors.New(r.t.Get("Unzip backup file failed: %v", err))
	}

	// 移动文件到对应位置
	if app.IsCli {
		fmt.Println(r.t.Get("|-Move backup file..."))
	}
	if io.Exists(filepath.Join("/tmp/panel-fix", "panel")) && io.IsDir(filepath.Join("/tmp/panel-fix", "panel")) {
		if err = io.Remove(filepath.Join(app.Root, "panel")); err != nil {
			return errors.New(r.t.Get("Remove panel file failed: %v", err))
		}
		if err = io.Mv(filepath.Join("/tmp/panel-fix", "panel"), filepath.Join(app.Root)); err != nil {
			return errors.New(r.t.Get("Move panel file failed: %v", err))
		}
	}
	if io.Exists(filepath.Join("/tmp/panel-fix", "acepanel")) {
		if err = io.Mv(filepath.Join("/tmp/panel-fix", "acepanel"), "/usr/local/sbin/acepanel"); err != nil {
			return errors.New(r.t.Get("Move acepanel file failed: %v", err))
		}
	}

	// tmp 目录下如果有 storage 备份，则解压回去
	if app.IsCli {
		fmt.Println(r.t.Get("|-Restore panel data..."))
	}
	if io.Exists("/tmp/panel-storage.zip") {
		if err = io.UnCompress("/tmp/panel-storage.zip", filepath.Join(app.Root, "panel")); err != nil {
			return errors.New(r.t.Get("Unzip panel data failed: %v", err))
		}
		if err = io.Remove("/tmp/panel-storage.zip"); err != nil {
			return errors.New(r.t.Get("Cleaning temporary file failed: %v", err))
		}
	}

	// 下载服务文件
	if !io.Exists("/etc/systemd/system/acepanel.service") {
		if _, err = shell.Execf(`wget -O /etc/systemd/system/acepanel.service https://%s/acepanel.service && sed -i "s|/opt/ace|%s|g" /etc/systemd/system/acepanel.service`, r.conf.App.DownloadEndpoint, app.Root); err != nil {
			return err
		}
	}

	// 处理权限
	if app.IsCli {
		fmt.Println(r.t.Get("|-Set key file permissions..."))
	}
	if err = io.Chmod(filepath.Join(app.Root, "panel", "storage", "config.yml"), 0600); err != nil {
		return err
	}
	if err = io.Chmod(filepath.Join(app.Root, "panel", "storage", "panel.db"), 0600); err != nil {
		return err
	}
	if err = io.Chmod("/etc/systemd/system/acepanel.service", 0644); err != nil {
		return err
	}
	if err = io.Chmod("/usr/local/sbin/acepanel", 0700); err != nil {
		return err
	}
	if err = io.Chmod(filepath.Join(app.Root, "panel"), 0700); err != nil {
		return err
	}

	if err = io.Remove("/tmp/panel-fix"); err != nil {
		return err
	}

	if app.IsCli {
		fmt.Println(r.t.Get("|-Fix completed"))
	}

	tools.RestartPanel()
	return nil
}

func (r *backupRepo) UpdatePanel(version, url, checksum string) error {
	// 预先优化数据库
	if err := r.db.Exec("VACUUM").Error; err != nil {
		return err
	}
	if err := r.db.Exec("PRAGMA wal_checkpoint(TRUNCATE);").Error; err != nil {
		return err
	}

	name := filepath.Base(url)
	if app.IsCli {
		fmt.Println(r.t.Get("|-Target version: %s", version))
		fmt.Println(r.t.Get("|-Download link: %s", url))
		fmt.Println(r.t.Get("|-File name: %s", name))
	}

	if app.IsCli {
		fmt.Println(r.t.Get("|-Downloading..."))
	}
	if _, err := shell.Execf("aria2c -c --file-allocation=falloc --allow-overwrite=true --auto-file-renaming=false --retry-wait=5 --max-tries=5 -x 16 -s 16 -k 1M -d /tmp -o %s %s", name, url); err != nil {
		return errors.New(r.t.Get("Download failed: %v", err))
	}
	if _, err := shell.Execf("aria2c -c --file-allocation=falloc --allow-overwrite=true --auto-file-renaming=false --retry-wait=5 --max-tries=5 -x 1 -s 1 -k 1M -d /tmp -o %s %s", name+".sha256", checksum); err != nil {
		return errors.New(r.t.Get("Download failed: %v", err))
	}
	if !io.Exists(filepath.Join("/tmp", name)) || !io.Exists(filepath.Join("/tmp", name+".sha256")) {
		return errors.New(r.t.Get("Download file check failed"))
	}

	if app.IsCli {
		fmt.Println(r.t.Get("|-Verify download file..."))
	}
	if check, err := shell.Execf("cd /tmp && sha256sum -c %s --ignore-missing", name+".sha256"); check != name+": OK" || err != nil {
		return errors.New(r.t.Get("Verify download file failed: %v", err))
	}
	if err := io.Remove(filepath.Join("/tmp", name+".sha256")); err != nil {
		return errors.New(r.t.Get("|-Clean up verification file failed: %v", err))
	}

	if io.Exists("/tmp/panel-storage.zip") {
		return errors.New(r.t.Get("Temporary file detected in /tmp, this may be caused by the last update failure, please run acepanel fix to repair and try again"))
	}

	if app.IsCli {
		fmt.Println(r.t.Get("|-Backup panel data..."))
	}
	// 备份面板
	if err := r.CreatePanel(); err != nil {
		return errors.New(r.t.Get("|-Backup panel data failed: %v", err))
	}
	if err := io.Compress(filepath.Join(app.Root, "panel/storage"), nil, "/tmp/panel-storage.zip"); err != nil {
		return errors.New(r.t.Get("|-Backup panel data failed: %v", err))
	}
	if !io.Exists("/tmp/panel-storage.zip") {
		return errors.New(r.t.Get("|-Backup panel data failed, missing file"))
	}

	if app.IsCli {
		fmt.Println(r.t.Get("|-Cleaning old version..."))
	}
	if _, err := shell.Execf("rm -rf %s/panel/*", app.Root); err != nil {
		return errors.New(r.t.Get("|-Cleaning old version failed: %v", err))
	}

	if app.IsCli {
		fmt.Println(r.t.Get("|-Unzip new version..."))
	}
	if err := io.UnCompress(filepath.Join("/tmp", name), filepath.Join(app.Root, "panel")); err != nil {
		return errors.New(r.t.Get("|-Unzip new version failed: %v", err))
	}
	if !io.Exists(filepath.Join(app.Root, "panel", "ace")) {
		return errors.New(r.t.Get("|-Unzip new version failed, missing file"))
	}
	if err := io.Remove(filepath.Join("/tmp", name)); err != nil {
		return errors.New(r.t.Get("|-Clean up temporary file failed: %v", err))
	}

	if app.IsCli {
		fmt.Println(r.t.Get("|-Restore panel data..."))
	}
	if err := io.UnCompress("/tmp/panel-storage.zip", filepath.Join(app.Root, "panel", "storage")); err != nil {
		return errors.New(r.t.Get("|-Restore panel data failed: %v", err))
	}
	if !io.Exists(filepath.Join(app.Root, "panel/storage/panel.db")) {
		return errors.New(r.t.Get("|-Restore panel data failed, missing file"))
	}

	if app.IsCli {
		fmt.Println(r.t.Get("|-Run post-update script..."))
	}
	if _, err := shell.Execf("curl -sSLm 10 https://%s/auto_update.sh | bash", r.conf.App.DownloadEndpoint); err != nil {
		return errors.New(r.t.Get("|-Run post-update script failed: %v", err))
	}
	if _, err := shell.Execf(
		`wget -O /etc/systemd/system/acepanel.service https://%s/acepanel.service && sed -i "s|/www|%s|g" /etc/systemd/system/acepanel.service`,
		r.conf.App.DownloadEndpoint, app.Root,
	); err != nil {
		return errors.New(r.t.Get("|-Download panel service file failed: %v", err))
	}
	if _, err := shell.Execf("acepanel setting write version %s", version); err != nil {
		return errors.New(r.t.Get("|-Write new panel version failed: %v", err))
	}
	if err := io.Mv(filepath.Join(app.Root, "panel/cli"), "/usr/local/sbin/acepanel"); err != nil {
		return errors.New(r.t.Get("|-Move acepanel tool failed: %v", err))
	}

	if app.IsCli {
		fmt.Println(r.t.Get("|-Set key file permissions..."))
	}
	_ = io.Chmod("/usr/local/sbin/acepanel", 0700)
	_ = io.Chmod("/etc/systemd/system/acepanel.service", 0644)
	_ = io.Chmod(filepath.Join(app.Root, "panel"), 0700)

	if app.IsCli {
		fmt.Println(r.t.Get("|-Update completed"))
	}

	r.log.Info("panel updated", slog.String("version", version))

	_, _ = shell.Execf("systemctl daemon-reload")
	_ = io.Remove("/tmp/panel-storage.zip")
	_ = io.Remove(filepath.Join(app.Root, "panel/config.example.yml"))
	if sqlDB, err := r.db.DB(); err == nil {
		_ = sqlDB.Close()
	}
	tools.RestartPanel()

	return nil
}

// backupExt 根据全局设置返回备份文件扩展名
func (r *backupRepo) backupExt() string {
	format, _ := r.setting.Get(biz.SettingKeyBackupFormat, "tar.xz")
	return "." + format
}

// isBackupArchive 判断文件名是否是已知的备份压缩包后缀
func (r *backupRepo) isBackupArchive(name string) bool {
	for _, ext := range []string{".tar.xz", ".tar.gz", ".tar.zst", ".zip", ".7z"} {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	return false
}
