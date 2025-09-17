package storage

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type SFTPConfig struct {
	Host       string        // SFTP 服务器地址
	Port       int           // SFTP 端口，默认 22
	Username   string        // 用户名
	Password   string        // 密码
	PrivateKey string        // SSH 私钥路径或内容
	BasePath   string        // 基础路径
	Timeout    time.Duration // 连接超时时间
}

type SFTP struct {
	config SFTPConfig
}

func NewSFTP(config SFTPConfig) (Storage, error) {
	if config.Port == 0 {
		config.Port = 22
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	config.BasePath = strings.Trim(config.BasePath, "/")

	s := &SFTP{
		config: config,
	}

	if err := s.ensureBasePath(); err != nil {
		return nil, fmt.Errorf("failed to ensure base path: %w", err)
	}

	return s, nil
}

// connect 建立 SFTP 连接
func (s *SFTP) connect() (*sftp.Client, func(), error) {
	var auth []ssh.AuthMethod

	// 密码认证
	if s.config.Password != "" {
		auth = append(auth, ssh.Password(s.config.Password))
	}

	// 私钥认证
	if s.config.PrivateKey != "" {
		var signer ssh.Signer
		var err error

		if _, statErr := os.Stat(s.config.PrivateKey); statErr == nil {
			// 私钥文件路径
			keyBytes, err := os.ReadFile(s.config.PrivateKey)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to read private key file: %w", err)
			}
			signer, err = ssh.ParsePrivateKey(keyBytes)
		} else {
			// 私钥内容
			signer, err = ssh.ParsePrivateKey([]byte(s.config.PrivateKey))
		}

		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		auth = append(auth, ssh.PublicKeys(signer))
	}

	clientConfig := &ssh.ClientConfig{
		User:            s.config.Username,
		Auth:            auth,
		Timeout:         s.config.Timeout,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	sshClient, err := ssh.Dial("tcp", addr, clientConfig)
	if err != nil {
		return nil, nil, err
	}

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		_ = sshClient.Close()
		return nil, nil, err
	}

	cleanup := func() {
		_ = sftpClient.Close()
		_ = sshClient.Close()
	}

	return sftpClient, cleanup, nil
}

// ensureBasePath 确保基础路径存在
func (s *SFTP) ensureBasePath() error {
	if s.config.BasePath == "" {
		return nil
	}

	client, cleanup, err := s.connect()
	if err != nil {
		return err
	}
	defer cleanup()

	return client.MkdirAll(s.config.BasePath)
}

// getRemotePath 获取远程路径
func (s *SFTP) getRemotePath(path string) string {
	path = strings.TrimPrefix(path, "/")
	if s.config.BasePath == "" {
		return path
	}
	if path == "" {
		return s.config.BasePath
	}
	return filepath.Join(s.config.BasePath, path)
}

// MakeDirectory 创建目录
func (s *SFTP) MakeDirectory(directory string) error {
	client, cleanup, err := s.connect()
	if err != nil {
		return err
	}
	defer cleanup()

	remotePath := s.getRemotePath(directory)
	return client.MkdirAll(remotePath)
}

// DeleteDirectory 删除目录
func (s *SFTP) DeleteDirectory(directory string) error {
	client, cleanup, err := s.connect()
	if err != nil {
		return err
	}
	defer cleanup()

	remotePath := s.getRemotePath(directory)
	return client.RemoveDirectory(remotePath)
}

// Copy 复制文件到新位置
func (s *SFTP) Copy(oldFile, newFile string) error {
	// SFTP 不支持直接复制，需要读取再写入
	data, err := s.Get(oldFile)
	if err != nil {
		return err
	}
	return s.Put(newFile, string(data))
}

// Delete 删除文件
func (s *SFTP) Delete(files ...string) error {
	client, cleanup, err := s.connect()
	if err != nil {
		return err
	}
	defer cleanup()

	for _, file := range files {
		remotePath := s.getRemotePath(file)
		if err := client.Remove(remotePath); err != nil {
			return err
		}
	}
	return nil
}

// Exists 检查文件是否存在
func (s *SFTP) Exists(file string) bool {
	client, cleanup, err := s.connect()
	if err != nil {
		return false
	}
	defer cleanup()

	remotePath := s.getRemotePath(file)
	_, err = client.Stat(remotePath)
	return err == nil
}

// Files 获取目录下的所有文件
func (s *SFTP) Files(path string) ([]string, error) {
	client, cleanup, err := s.connect()
	if err != nil {
		return nil, err
	}
	defer cleanup()

	remotePath := s.getRemotePath(path)
	entries, err := client.ReadDir(remotePath)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}

	return files, nil
}

// Get 读取文件内容
func (s *SFTP) Get(file string) ([]byte, error) {
	client, cleanup, err := s.connect()
	if err != nil {
		return nil, err
	}
	defer cleanup()

	remotePath := s.getRemotePath(file)
	remoteFile, err := client.Open(remotePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = remoteFile.Close() }()

	return io.ReadAll(remoteFile)
}

// LastModified 获取文件最后修改时间
func (s *SFTP) LastModified(file string) (time.Time, error) {
	client, cleanup, err := s.connect()
	if err != nil {
		return time.Time{}, err
	}
	defer cleanup()

	remotePath := s.getRemotePath(file)
	stat, err := client.Stat(remotePath)
	if err != nil {
		return time.Time{}, err
	}

	return stat.ModTime(), nil
}

// MimeType 获取文件的 MIME 类型
func (s *SFTP) MimeType(file string) (string, error) {
	ext := filepath.Ext(file)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		return "application/octet-stream", nil
	}
	return mimeType, nil
}

// Missing 检查文件是否不存在
func (s *SFTP) Missing(file string) bool {
	return !s.Exists(file)
}

// Move 移动文件到新位置
func (s *SFTP) Move(oldFile, newFile string) error {
	client, cleanup, err := s.connect()
	if err != nil {
		return err
	}
	defer cleanup()

	oldPath := s.getRemotePath(oldFile)
	newPath := s.getRemotePath(newFile)

	// 确保目标目录存在
	newDir := filepath.Dir(newPath)
	if newDir != "." {
		_ = client.MkdirAll(newDir)
	}

	return client.Rename(oldPath, newPath)
}

// Path 获取文件的完整路径
func (s *SFTP) Path(file string) string {
	return fmt.Sprintf("sftp://%s:%d/%s", s.config.Host, s.config.Port, s.getRemotePath(file))
}

// Put 写入文件内容
func (s *SFTP) Put(file, content string) error {
	client, cleanup, err := s.connect()
	if err != nil {
		return err
	}
	defer cleanup()

	remotePath := s.getRemotePath(file)

	// 确保目录存在
	remoteDir := filepath.Dir(remotePath)
	if remoteDir != "." {
		_ = client.MkdirAll(remoteDir)
	}

	remoteFile, err := client.Create(remotePath)
	if err != nil {
		return err
	}
	defer func() { _ = remoteFile.Close() }()

	_, err = io.Copy(remoteFile, bytes.NewReader([]byte(content)))
	return err
}

// Size 获取文件大小
func (s *SFTP) Size(file string) (int64, error) {
	client, cleanup, err := s.connect()
	if err != nil {
		return 0, err
	}
	defer cleanup()

	remotePath := s.getRemotePath(file)
	stat, err := client.Stat(remotePath)
	if err != nil {
		return 0, err
	}

	return stat.Size(), nil
}
