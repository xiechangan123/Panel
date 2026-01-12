package acme

import (
	"context"
	"crypto/x509"
	"sort"

	"github.com/libdns/libdns"
	"github.com/mholt/acmez/v3"
	"github.com/mholt/acmez/v3/acme"

	"github.com/acepanel/panel/pkg/cert"
)

type Certificate struct {
	PrivateKey []byte
	acme.Certificate
}

type Client struct {
	Account acme.Account
	zClient acmez.Client
	// 手动 DNS 所需的信号通道
	manualDNSSolver
}

// UseDns 使用 DNS 接口验证
func (c *Client) UseDns(dnsType DnsType, param DNSParam) {
	c.zClient.ChallengeSolvers = map[string]acmez.Solver{
		acme.ChallengeTypeDNS01: &dnsSolver{
			dns:     dnsType,
			param:   param,
			records: []libdns.Record{},
		},
	}
}

// UseManualDns 使用手动 DNS 验证
func (c *Client) UseManualDns(check ...bool) {
	c.controlChan = make(chan struct{})
	c.dnsChan = make(chan any)
	c.certChan = make(chan any)
	c.zClient.ChallengeSolvers = map[string]acmez.Solver{
		acme.ChallengeTypeDNS01: &manualDNSSolver{
			check:       len(check) > 0 && check[0],
			controlChan: c.controlChan,
			dnsChan:     c.dnsChan,
			certChan:    c.certChan,
			records:     []DNSRecord{},
		},
	}
}

// UseHTTP 使用 HTTP 验证
// conf 配置文件路径
// webServer web 服务器类型 ("nginx" 或 "apache")
func (c *Client) UseHTTP(conf string, webServer string) {
	c.zClient.ChallengeSolvers = map[string]acmez.Solver{
		acme.ChallengeTypeHTTP01: httpSolver{
			conf:      conf,
			webServer: webServer,
		},
	}
}

// UsePanel 使用面板 HTTP 验证
// ip 外网访问 IP 地址
// conf 配置文件路径
// webServer web 服务器类型 ("nginx" 或 "apache")
func (c *Client) UsePanel(ip []string, conf string, webServer string) {
	c.zClient.ChallengeSolvers = map[string]acmez.Solver{
		acme.ChallengeTypeHTTP01: &panelSolver{
			ip:        ip,
			conf:      conf,
			webServer: webServer,
		},
	}
}

// ObtainCertificate 签发 SSL 证书
func (c *Client) ObtainCertificate(ctx context.Context, sans []string, keyType KeyType) (Certificate, error) {
	certPrivateKey, err := generatePrivateKey(keyType)
	if err != nil {
		return Certificate{}, err
	}
	pemPrivateKey, err := cert.EncodeKey(certPrivateKey)
	if err != nil {
		return Certificate{}, err
	}

	certs, err := c.zClient.ObtainCertificateForSANs(ctx, c.Account, certPrivateKey, sans)
	if err != nil {
		return Certificate{}, err
	}

	crt := c.selectPreferredChain(certs)
	return Certificate{PrivateKey: pemPrivateKey, Certificate: crt}, nil
}

// ObtainIPCertificate 签发 IP SSL 证书
func (c *Client) ObtainIPCertificate(ctx context.Context, sans []string, keyType KeyType) (Certificate, error) {
	certPrivateKey, err := generatePrivateKey(keyType)
	if err != nil {
		return Certificate{}, err
	}
	pemPrivateKey, err := cert.EncodeKey(certPrivateKey)
	if err != nil {
		return Certificate{}, err
	}

	csr, err := acmez.NewCSR(certPrivateKey, sans)
	if err != nil {
		return Certificate{}, err
	}

	params, err := acmez.OrderParametersFromCSR(c.Account, csr)
	if err != nil {
		return Certificate{}, err
	}
	params.Profile = "shortlived"

	certs, err := c.zClient.ObtainCertificate(ctx, params)
	if err != nil {
		return Certificate{}, err
	}

	crt := c.selectPreferredChain(certs)
	return Certificate{PrivateKey: pemPrivateKey, Certificate: crt}, nil
}

// ObtainCertificateManual 手动验证 SSL 证书
func (c *Client) ObtainCertificateManual() (Certificate, error) {
	// 发送信号，开始验证
	c.controlChan <- struct{}{}
	// 等待验证完成
	certs := <-c.certChan

	if err, ok := certs.(error); ok {
		return Certificate{}, err
	}

	return certs.(Certificate), nil
}

// RenewCertificate 续签 SSL 证书
func (c *Client) RenewCertificate(ctx context.Context, certUrl string, domains []string, keyType KeyType) (Certificate, error) {
	_, err := c.zClient.GetCertificateChain(ctx, c.Account, certUrl)
	if err != nil {
		return Certificate{}, err
	}

	return c.ObtainCertificate(ctx, domains, keyType)
}

// GetDNSRecords 获取 DNS 解析（手动设置）
func (c *Client) GetDNSRecords(ctx context.Context, domains []string, keyType KeyType) ([]DNSRecord, error) {
	go func(ctx context.Context, domains []string, keyType KeyType) {
		certs, err := c.ObtainCertificate(ctx, domains, keyType)
		// 将证书和错误信息发送到 certChan
		if err != nil {
			c.certChan <- err
			return
		}
		c.certChan <- certs
	}(ctx, domains, keyType)

	// 这里要少一次循环，因为需要卡住最后一次的 dnsChan，等待手动 DNS 验证完成
	for i := 1; i < len(domains); i++ {
		<-c.dnsChan
		c.controlChan <- struct{}{}
	}

	// 因为上面少了一次循环，所以这里接收到的即为完整的 DNS 记录切片
	data := <-c.dnsChan
	if err, ok := data.(error); ok {
		return nil, err
	}

	return data.([]DNSRecord), nil
}

// GetRenewalInfo 获取续签建议
func (c *Client) GetRenewalInfo(ctx context.Context, cert x509.Certificate) (acme.RenewalInfo, error) {
	return c.zClient.GetRenewalInfo(ctx, &cert)
}

func (c *Client) selectPreferredChain(certChains []acme.Certificate) acme.Certificate {
	if len(certChains) == 1 {
		return certChains[0]
	}

	sort.Slice(certChains, func(i, j int) bool {
		return len(certChains[i].ChainPEM) < len(certChains[j].ChainPEM)
	})

	return certChains[0]
}
