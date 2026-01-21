package storage

import (
	"fmt"
	"io"
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
	PrivateKey string        // SSH 私钥
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
	config.BasePath = strings.TrimSuffix(config.BasePath, "/")

	if config.Username == "" || (config.Password == "" && config.PrivateKey == "") {
		return nil, fmt.Errorf("username and either password or private key must be provided")
	}

	return &SFTP{config: config}, nil
}

// connect 建立 SFTP 连接，返回 client 和 cleanup 函数
func (s *SFTP) connect() (*sftp.Client, func(), error) {
	var auth []ssh.AuthMethod
	// 密码认证
	if s.config.Password != "" {
		auth = append(auth, ssh.Password(s.config.Password))
	}
	// 私钥认证
	if s.config.PrivateKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(s.config.PrivateKey))
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
		return nil, nil, fmt.Errorf("failed to connect to SSH server: %w", err)
	}

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		_ = sshClient.Close()
		return nil, nil, fmt.Errorf("failed to create SFTP client: %w", err)
	}

	cleanup := func() {
		_ = sftpClient.Close()
		_ = sshClient.Close()
	}

	return sftpClient, cleanup, nil
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
		if err = client.Remove(remotePath); err != nil {
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

// List 列出目录下的所有文件
func (s *SFTP) List(path string) ([]string, error) {
	client, cleanup, err := s.connect()
	if err != nil {
		return nil, err
	}
	defer cleanup()

	// 确保基础路径存在
	if s.config.BasePath != "" {
		_ = client.MkdirAll(s.config.BasePath)
	}

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

// Put 写入文件内容
func (s *SFTP) Put(file string, content io.Reader) error {
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

	// 确保基础路径存在
	if s.config.BasePath != "" {
		_ = client.MkdirAll(s.config.BasePath)
	}

	remoteFile, err := client.Create(remotePath)
	if err != nil {
		return err
	}
	defer func() { _ = remoteFile.Close() }()

	_, err = io.Copy(remoteFile, content)
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
