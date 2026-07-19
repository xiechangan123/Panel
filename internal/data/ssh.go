package data

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/leonelquinteros/gotext"
	"github.com/pkg/sftp"
	"github.com/samber/do/v2"
	cryptossh "golang.org/x/crypto/ssh"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/request"
	pkgssh "github.com/acepanel/panel/v3/pkg/ssh"
)

// sftpConn 缓存的 SFTP 连接
type sftpConn struct {
	ssh      *cryptossh.Client
	sftp     *sftp.Client
	lastUsed time.Time
}

type sshRepo struct {
	t     *gotext.Locale
	db    *gorm.DB
	mu    sync.Mutex
	conns map[uint]*sftpConn
}

func NewSSHRepo(i do.Injector) (biz.SSHRepo, error) {
	return &sshRepo{
		t:     do.MustInvoke[*gotext.Locale](i),
		db:    do.MustInvoke[*gorm.DB](i),
		conns: make(map[uint]*sftpConn),
	}, nil
}

func (r *sshRepo) List(page, limit uint) ([]*biz.SSH, int64, error) {
	ssh := make([]*biz.SSH, 0)
	var total int64
	err := r.db.Model(&biz.SSH{}).Omit("Hosts").Order("id desc").Count(&total).Offset(int((page - 1) * limit)).Limit(int(limit)).Find(&ssh).Error
	return ssh, total, err
}

func (r *sshRepo) Get(id uint) (*biz.SSH, error) {
	ssh := new(biz.SSH)
	if err := r.db.Where("id = ?", id).First(ssh).Error; err != nil {
		return nil, err
	}

	return ssh, nil
}

func (r *sshRepo) Create(req *request.SSHCreate) error {
	conf := pkgssh.ClientConfig{
		AuthMethod: pkgssh.AuthMethod(req.AuthMethod),
		Host:       fmt.Sprintf("%s:%d", req.Host, req.Port),
		User:       req.User,
		Password:   req.Password,
		Key:        req.Key,
		Passphrase: req.Passphrase,
	}
	client, err := pkgssh.NewSSHClient(conf)
	if err != nil {
		return errors.New(r.t.Get("failed to check ssh connection: %v", err))
	}
	defer func(client *cryptossh.Client) { _ = client.Close() }(client)

	ssh := &biz.SSH{
		Name:   req.Name,
		Host:   req.Host,
		Port:   req.Port,
		Config: conf,
		Remark: req.Remark,
	}

	if err = r.db.Create(ssh).Error; err != nil {
		return err
	}

	return nil
}

func (r *sshRepo) Update(req *request.SSHUpdate) error {
	conf := pkgssh.ClientConfig{
		AuthMethod: pkgssh.AuthMethod(req.AuthMethod),
		Host:       fmt.Sprintf("%s:%d", req.Host, req.Port),
		User:       req.User,
		Password:   req.Password,
		Key:        req.Key,
		Passphrase: req.Passphrase,
	}
	client, err := pkgssh.NewSSHClient(conf)
	if err != nil {
		return errors.New(r.t.Get("failed to check ssh connection: %v", err))
	}
	defer func(client *cryptossh.Client) { _ = client.Close() }(client)

	ssh := &biz.SSH{
		ID:     req.ID,
		Name:   req.Name,
		Host:   req.Host,
		Port:   req.Port,
		Config: conf,
		Remark: req.Remark,
	}

	if err = r.db.Model(ssh).Where("id = ?", req.ID).Select("*").Updates(ssh).Error; err != nil {
		return err
	}

	return nil
}

func (r *sshRepo) Delete(id uint) error {
	if err := r.db.Delete(&biz.SSH{}, id).Error; err != nil {
		return err
	}

	return nil
}

// dial 与主机建立 SSH 和 SFTP 连接
func (r *sshRepo) dial(hostID uint) (*cryptossh.Client, *sftp.Client, error) {
	info, err := r.Get(hostID)
	if err != nil {
		return nil, nil, err
	}
	sshClient, err := pkgssh.NewSSHClient(info.Config)
	if err != nil {
		return nil, nil, errors.New(r.t.Get("failed to connect to %s: %v", info.Name, err))
	}
	sftpClient, err := sftp.NewClient(sshClient, sftp.UseConcurrentWrites(true))
	if err != nil {
		_ = sshClient.Close()
		return nil, nil, errors.New(r.t.Get("failed to open sftp session on %s: %v", info.Name, err))
	}

	return sshClient, sftpClient, nil
}

// getSftp 获取缓存的 SFTP 连接,失效时重建,顺带清理闲置连接
func (r *sshRepo) getSftp(hostID uint) (*sftp.Client, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, conn := range r.conns {
		if id != hostID && time.Since(conn.lastUsed) > 10*time.Minute {
			_ = conn.sftp.Close()
			_ = conn.ssh.Close()
			delete(r.conns, id)
		}
	}

	if conn, ok := r.conns[hostID]; ok {
		if _, err := conn.sftp.Getwd(); err == nil {
			conn.lastUsed = time.Now()
			return conn.sftp, nil
		}
		_ = conn.sftp.Close()
		_ = conn.ssh.Close()
		delete(r.conns, hostID)
	}

	sshClient, sftpClient, err := r.dial(hostID)
	if err != nil {
		return nil, err
	}
	r.conns[hostID] = &sftpConn{ssh: sshClient, sftp: sftpClient, lastUsed: time.Now()}

	return sftpClient, nil
}

func (r *sshRepo) ListFiles(hostID uint, path string) ([]*biz.SSHFileInfo, error) {
	var infos []os.FileInfo
	if hostID == 0 {
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			infos = append(infos, info)
		}
	} else {
		client, err := r.getSftp(hostID)
		if err != nil {
			return nil, err
		}
		if infos, err = client.ReadDir(path); err != nil {
			return nil, err
		}
	}

	files := make([]*biz.SSHFileInfo, 0, len(infos))
	for _, info := range infos {
		files = append(files, &biz.SSHFileInfo{
			Name:    info.Name(),
			Size:    info.Size(),
			Mode:    info.Mode().String(),
			ModTime: info.ModTime().Unix(),
			IsDir:   info.IsDir(),
			IsLink:  info.Mode()&os.ModeSymlink != 0,
		})
	}
	// 目录在前,名称升序
	slices.SortFunc(files, func(a, b *biz.SSHFileInfo) int {
		if a.IsDir != b.IsDir {
			if a.IsDir {
				return -1
			}
			return 1
		}
		return strings.Compare(a.Name, b.Name)
	})

	return files, nil
}

func (r *sshRepo) Mkdir(hostID uint, path string) error {
	if hostID == 0 {
		return os.MkdirAll(path, 0755)
	}
	client, err := r.getSftp(hostID)
	if err != nil {
		return err
	}

	return client.MkdirAll(path)
}

// transferProbe 在读写路径上统计进度并响应取消
type transferProbe struct {
	ctx         context.Context
	r           io.Reader
	w           io.Writer
	transferred int64
	total       int64
	progress    func(transferred, total int64)
}

func (t *transferProbe) advance(n int) {
	if n > 0 {
		t.transferred += int64(n)
		t.progress(t.transferred, t.total)
	}
}

func (t *transferProbe) Read(p []byte) (int, error) {
	if err := t.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := t.r.Read(p)
	t.advance(n)
	return n, err
}

func (t *transferProbe) Write(p []byte) (int, error) {
	if err := t.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := t.w.Write(p)
	t.advance(n)
	return n, err
}

func (r *sshRepo) TransferFile(ctx context.Context, srcID uint, srcPath string, dstID uint, dstPath string, progress func(transferred, total int64)) error {
	// 传输不复用缓存连接,独立建连以支持长时间占用与随时取消
	var srcSftp, dstSftp *sftp.Client
	if srcID != 0 {
		sshClient, sftpClient, err := r.dial(srcID)
		if err != nil {
			return err
		}
		defer func() { _ = sftpClient.Close(); _ = sshClient.Close() }()
		srcSftp = sftpClient
	}
	if dstID != 0 {
		sshClient, sftpClient, err := r.dial(dstID)
		if err != nil {
			return err
		}
		defer func() { _ = sftpClient.Close(); _ = sshClient.Close() }()
		dstSftp = sftpClient
	}

	// 源文件信息
	var stat os.FileInfo
	var err error
	if srcSftp == nil {
		stat, err = os.Stat(srcPath)
	} else {
		stat, err = srcSftp.Stat(srcPath)
	}
	if err != nil {
		return err
	}
	if stat.IsDir() {
		return errors.New(r.t.Get("transferring a directory is not supported"))
	}

	var reader io.ReadCloser
	if srcSftp == nil {
		reader, err = os.Open(srcPath)
	} else {
		reader, err = srcSftp.Open(srcPath)
	}
	if err != nil {
		return err
	}
	defer func() { _ = reader.Close() }()

	var writer io.WriteCloser
	if dstSftp == nil {
		writer, err = os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, stat.Mode().Perm())
	} else {
		writer, err = dstSftp.Create(dstPath)
	}
	if err != nil {
		return err
	}

	// 探针包在本机一侧,保留 sftp 一侧 WriteTo/ReadFrom 的并发传输优化
	if srcSftp != nil {
		probe := &transferProbe{ctx: ctx, w: writer, total: stat.Size(), progress: progress}
		_, err = io.Copy(probe, reader)
	} else {
		probe := &transferProbe{ctx: ctx, r: reader, total: stat.Size(), progress: progress}
		_, err = io.Copy(writer, probe)
	}
	if err != nil {
		_ = writer.Close()
		return err
	}
	if err = writer.Close(); err != nil {
		return err
	}
	if dstSftp != nil {
		_ = dstSftp.Chmod(dstPath, stat.Mode().Perm())
	}

	return nil
}
