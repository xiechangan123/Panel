//go:build linux

package tamper

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"golang.org/x/sys/unix"
)

// fileEntry 一个受保护条目
type fileEntry struct {
	path  string
	dev   uint64
	inode uint64
	isDir bool
}

// fileKey 设备号+inode 唯一标识一个文件,用于事件回填路径
type fileKey struct {
	dev   uint64
	inode uint64
}

// engine 防篡改底层引擎(chattr / ebpf 各一实现)
type engine interface {
	apply(entries []fileEntry) error
	remove(entries []fileEntry) error
	events() <-chan Event
	close() error
}

// Manager 防篡改运行时管理器,负责规则扫描、保护应用、新建监控与事件汇聚
type Manager struct {
	cfg    Config
	log    *slog.Logger
	eng    engine
	isEBPF bool

	watcher *fsnotify.Watcher

	mu        sync.RWMutex
	entries   []fileEntry
	inodePath map[fileKey]string // eBPF 事件回填路径
	nFiles    int
	nDirs     int
	running   bool

	out    chan Event
	closed chan struct{}
	once   sync.Once
}

// NewManager 按配置创建管理器(尚未开始保护,需调用 Start)
func NewManager(cfg Config, log *slog.Logger) (*Manager, error) {
	m := &Manager{
		cfg:       cfg,
		log:       log,
		inodePath: make(map[fileKey]string),
		out:       make(chan Event, 256),
		closed:    make(chan struct{}),
	}

	switch cfg.Mode {
	case ModeEBPF:
		st := DetectEBPF()
		if !st.Available {
			return nil, fmt.Errorf("eBPF 模式不可用: %s", st.Reason)
		}
		eng, err := newEBPFEngine()
		if err != nil {
			return nil, err
		}
		m.eng = eng
		m.isEBPF = true
	case ModeChattr:
		m.eng = newChattrEngine()
	default:
		return nil, fmt.Errorf("未知防篡改模式: %s", cfg.Mode)
	}

	return m, nil
}

// matchExt 判断文件是否命中受保护后缀(exts 为空表示全部文件)
func matchExt(path string, exts []string) bool {
	if len(exts) == 0 {
		return true
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
	for _, e := range exts {
		if strings.ToLower(strings.TrimPrefix(e, ".")) == ext {
			return true
		}
	}
	return false
}

// isExcluded 判断路径是否落在排除项内(支持绝对路径前缀与路径段名)
func isExcluded(path string, excludes []string) bool {
	for _, ex := range excludes {
		ex = strings.TrimSpace(ex)
		if ex == "" {
			continue
		}
		if filepath.IsAbs(ex) {
			if path == ex || strings.HasPrefix(path, strings.TrimRight(ex, "/")+"/") {
				return true
			}
			continue
		}
		// 相对名:匹配任意路径段
		if slices.Contains(strings.Split(path, string(os.PathSeparator)), ex) {
			return true
		}
	}
	return false
}

// statOf 取路径的设备号(内核 dev_t 编码)与 inode 号
// stat 返回的 st_dev 为 glibc 编码,需转换为内核 s_dev 格式以便与 eBPF 读到的值一致
func statOf(path string) (dev, ino uint64) {
	var st syscall.Stat_t
	if err := syscall.Lstat(path, &st); err != nil {
		return 0, 0
	}
	major := unix.Major(uint64(st.Dev))
	minor := unix.Minor(uint64(st.Dev))
	return uint64(major)<<20 | uint64(minor)&0xfffff, st.Ino
}

// scan 遍历规则计算受保护条目
// exts 为空时保护整树(文件 +i、目录 +a);exts 非空时仅保护匹配文件
func (m *Manager) scan() []fileEntry {
	var entries []fileEntry
	seen := make(map[string]bool)

	for _, rule := range m.cfg.Rules {
		wholeTree := len(rule.Exts) == 0
		for _, root := range rule.Paths {
			_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				if isExcluded(path, rule.Excludes) {
					if d.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
				if seen[path] {
					return nil
				}
				if d.IsDir() {
					if wholeTree {
						seen[path] = true
						dev, ino := statOf(path)
						entries = append(entries, fileEntry{path: path, dev: dev, inode: ino, isDir: true})
					}
					return nil
				}
				if matchExt(path, rule.Exts) {
					seen[path] = true
					dev, ino := statOf(path)
					entries = append(entries, fileEntry{path: path, dev: dev, inode: ino, isDir: false})
				}
				return nil
			})
		}
	}

	return entries
}

// Start 扫描规则、应用保护、启动监控
func (m *Manager) Start() error {
	entries := m.scan()

	m.mu.Lock()
	m.entries = entries
	m.nFiles, m.nDirs = 0, 0
	for _, e := range entries {
		if e.isDir {
			m.nDirs++
		} else {
			m.nFiles++
			m.inodePath[fileKey{e.dev, e.inode}] = e.path
		}
	}
	m.running = true
	m.mu.Unlock()

	if err := m.eng.apply(entries); err != nil {
		return fmt.Errorf("应用保护失败: %w", err)
	}

	// eBPF 事件转发(回填路径)
	if ch := m.eng.events(); ch != nil {
		go m.forwardEvents(ch)
	}

	// fsnotify 监控受保护目录下的新建
	if err := m.startWatcher(); err != nil {
		m.log.Warn("防篡改文件监控启动失败,新建拦截不可用", slog.Any("err", err))
	}

	m.log.Info("防篡改已启用",
		slog.String("mode", string(m.cfg.Mode)),
		slog.Int("files", m.nFiles),
		slog.Int("dirs", m.nDirs))
	return nil
}

// startWatcher 对所有受保护目录(及规则根)建立 fsnotify 监控
func (m *Manager) startWatcher() error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	m.watcher = w

	dirs := make(map[string]bool)
	for _, rule := range m.cfg.Rules {
		for _, root := range rule.Paths {
			_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
				if err != nil || !d.IsDir() {
					return nil
				}
				if isExcluded(path, rule.Excludes) {
					return filepath.SkipDir
				}
				dirs[path] = true
				return nil
			})
		}
	}
	for dir := range dirs {
		_ = w.Add(dir)
	}

	go m.watchLoop()
	return nil
}

// watchLoop 处理文件系统事件
func (m *Manager) watchLoop() {
	for {
		select {
		case <-m.closed:
			return
		case ev, ok := <-m.watcher.Events:
			if !ok {
				return
			}
			if ev.Op&fsnotify.Create != 0 {
				m.onCreate(ev.Name)
			}
		case err, ok := <-m.watcher.Errors:
			if !ok {
				return
			}
			m.log.Warn("防篡改监控错误", slog.Any("err", err))
		}
	}
}

// onCreate 处理受保护目录下的新建条目
func (m *Manager) onCreate(path string) {
	info, err := os.Lstat(path)
	if err != nil {
		return
	}

	// 新建目录:纳入监控
	if info.IsDir() {
		if m.watcher != nil {
			_ = m.watcher.Add(path)
		}
		return
	}

	rule := m.ruleOf(path)
	if rule == nil || isExcluded(path, rule.Excludes) || !matchExt(path, rule.Exts) {
		return
	}

	dev, ino := statOf(path)
	ev := Event{
		Path:  path,
		Dev:   dev,
		Inode: ino,
		Op:    OpCreate,
		OpStr: OpCreate.String(),
		Time:  time.Now(),
	}

	if m.cfg.BlockNewFiles {
		// 拦截策略:删除新建的可疑文件
		_ = os.Remove(path)
	} else {
		// 冻结策略:锁定新文件防止后续修改
		m.mu.Lock()
		m.inodePath[fileKey{dev, ino}] = path
		m.entries = append(m.entries, fileEntry{path: path, dev: dev, inode: ino, isDir: false})
		m.nFiles++
		m.mu.Unlock()
		switch e := m.eng.(type) {
		case *chattrEngine:
			e.lockOne(path, false)
		case *ebpfEngine:
			_ = e.Add([]protKey{{Ino: ino, Dev: dev}})
		}
	}

	m.emit(ev)
}

// ruleOf 找出路径所属规则
func (m *Manager) ruleOf(path string) *Rule {
	for i := range m.cfg.Rules {
		for _, root := range m.cfg.Rules[i].Paths {
			if path == root || strings.HasPrefix(path, strings.TrimRight(root, "/")+"/") {
				return &m.cfg.Rules[i]
			}
		}
	}
	return nil
}

// forwardEvents 回填 eBPF 事件路径并转发
func (m *Manager) forwardEvents(ch <-chan Event) {
	for {
		select {
		case <-m.closed:
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			m.mu.RLock()
			ev.Path = m.inodePath[fileKey{ev.Dev, ev.Inode}]
			m.mu.RUnlock()
			ev.Time = time.Now()
			m.emit(ev)
		}
	}
}

// emit 向外投递事件(不阻塞)
func (m *Manager) emit(ev Event) {
	select {
	case m.out <- ev:
	case <-m.closed:
	default:
	}
}

// Events 拦截/告警事件通道
func (m *Manager) Events() <-chan Event {
	return m.out
}

// Stats 运行统计
func (m *Manager) Stats() Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return Stats{
		Mode:           m.cfg.Mode,
		Running:        m.running,
		ProtectedFiles: m.nFiles,
		ProtectedDirs:  m.nDirs,
	}
}

// Unlock 临时解除指定路径保护(供面板合法写入前调用)
func (m *Manager) Unlock(paths []string) {
	entries := make([]fileEntry, 0, len(paths))
	for _, p := range paths {
		info, err := os.Lstat(p)
		if err != nil {
			continue
		}
		dev, ino := statOf(p)
		entries = append(entries, fileEntry{path: p, dev: dev, inode: ino, isDir: info.IsDir()})
	}
	_ = m.eng.remove(entries)
}

// Relock 恢复指定路径保护
func (m *Manager) Relock(paths []string) {
	entries := make([]fileEntry, 0, len(paths))
	for _, p := range paths {
		info, err := os.Lstat(p)
		if err != nil {
			continue
		}
		dev, ino := statOf(p)
		entries = append(entries, fileEntry{path: p, dev: dev, inode: ino, isDir: info.IsDir()})
	}
	_ = m.eng.apply(entries)
}

// Stop 解除全部保护并停止(cfg.KeepLocked 场景由上层控制)
func (m *Manager) Stop() error {
	m.once.Do(func() { close(m.closed) })

	if m.watcher != nil {
		_ = m.watcher.Close()
	}

	m.mu.RLock()
	entries := m.entries
	m.mu.RUnlock()
	_ = m.eng.remove(entries)

	m.mu.Lock()
	m.running = false
	m.mu.Unlock()

	return m.eng.close()
}
