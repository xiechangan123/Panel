package ssh

import (
	"errors"
	"time"

	"golang.org/x/crypto/ssh"
)

type AuthMethod string

const (
	PASSWORD  AuthMethod = "password"
	PUBLICKEY AuthMethod = "publickey"
)

type ClientConfig struct {
	AuthMethod AuthMethod    `json:"auth_method"`
	Host       string        `json:"host"`
	User       string        `json:"user"`
	Password   string        `json:"password"`
	Key        string        `json:"key"`
	Passphrase string        `json:"passphrase"`
	Timeout    time.Duration `json:"timeout"`
}

func ClientConfigPassword(host, user, password string) *ClientConfig {
	return &ClientConfig{
		Timeout:    10 * time.Second,
		AuthMethod: PASSWORD,
		Host:       host,
		User:       user,
		Password:   password,
	}
}

func ClientConfigPublicKey(host, user, key, passphrase string) *ClientConfig {
	return &ClientConfig{
		Timeout:    10 * time.Second,
		AuthMethod: PUBLICKEY,
		Host:       host,
		User:       user,
		Key:        key,
		Passphrase: passphrase,
	}
}

func NewSSHClient(conf ClientConfig) (*ssh.Client, error) {
	if conf.Timeout == 0 {
		conf.Timeout = 10 * time.Second
	}

	config := &ssh.ClientConfig{}
	config.SetDefaults()
	config.Timeout = conf.Timeout
	config.User = conf.User
	config.HostKeyCallback = ssh.InsecureIgnoreHostKey()

	switch conf.AuthMethod {
	case PASSWORD:
		config.Auth = []ssh.AuthMethod{ssh.Password(conf.Password)}
	case PUBLICKEY:
		signer, err := parseKey(conf.Key, conf.Passphrase)
		if err != nil {
			return nil, err
		}
		config.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	}
	c, err := ssh.Dial("tcp", conf.Host, config) // TODO support ipv6
	if err != nil {
		return nil, err
	}

	return c, nil
}

// parseKey 解析私钥
func parseKey(key, passphrase string) (ssh.Signer, error) {
	keyBytes := []byte(key)

	if passphrase != "" {
		return ssh.ParsePrivateKeyWithPassphrase(keyBytes, []byte(passphrase))
	}

	signer, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		// 密钥被加密
		if passphraseMissingError, ok := errors.AsType[*ssh.PassphraseMissingError](err); ok {
			return nil, passphraseMissingError
		}
		return nil, err
	}

	return signer, nil
}
