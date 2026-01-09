package acme

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/libdns/alidns"
	"github.com/libdns/cloudflare"
	"github.com/libdns/cloudns"
	"github.com/libdns/gcore"
	"github.com/libdns/huaweicloud"
	"github.com/libdns/libdns"
	"github.com/libdns/namesilo"
	"github.com/libdns/porkbun"
	"github.com/libdns/tencentcloud"
	"github.com/libdns/westcn"
	"github.com/mholt/acmez/v3/acme"
	"golang.org/x/net/publicsuffix"

	pkgos "github.com/acepanel/panel/pkg/os"
	"github.com/acepanel/panel/pkg/shell"
	"github.com/acepanel/panel/pkg/systemctl"
)

var panelSolverGlobal sync.Mutex

type panelSolver struct {
	ip     []string
	conf   string
	server *http.Server
	// tokens 存储所有待验证的 challenge，key 为路径，value 为 token
	tokens map[string]string
	// presentCount Present 调用计数
	presentCount int
	// cleanupCount CleanUp 调用计数
	cleanupCount int
	// useBuiltin 标记是否使用内置 HTTP 服务器
	useBuiltin bool
}

func (s *panelSolver) Present(_ context.Context, challenge acme.Challenge) error {
	if s.presentCount == 0 {
		panelSolverGlobal.Lock()
	}

	path := challenge.HTTP01ResourcePath()
	token := challenge.KeyAuthorization

	// 初始化 tokens map
	if s.tokens == nil {
		s.tokens = make(map[string]string)
	}

	// 收集所有域名的 token
	s.tokens[path] = token
	s.presentCount++
	if s.presentCount < len(s.ip) {
		return nil
	}

	// 如果 80 端口没有被占用，则使用内置的 HTTP 服务器
	if !pkgos.TCPPortInUse(80) {
		s.useBuiltin = true
		return s.startServer()
	}

	// 否则使用 nginx 配置
	s.useBuiltin = false
	return s.writeNginxConfig()
}

func (s *panelSolver) startServer() error {
	s.server = &http.Server{
		Addr: ":80",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := s.tokens[r.URL.Path]
			if !ok {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte(token))
		}),
	}

	errChan := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
		close(errChan)
	}()

	// 等待一小段时间确保服务器启动成功
	select {
	case err := <-errChan:
		s.server = nil
		return fmt.Errorf("failed to start HTTP server: %w", err)
	case <-time.After(100 * time.Millisecond):
		return nil
	}
}

func (s *panelSolver) writeNginxConfig() error {
	var conf strings.Builder
	conf.WriteString(fmt.Sprintf("server {\n    listen 80;\n    server_name %s;\n", strings.Join(s.ip, " ")))
	for path, token := range s.tokens {
		conf.WriteString(fmt.Sprintf("    location = %s {\n        default_type text/plain;\n        return 200 %q;\n    }\n", path, token))
	}
	conf.WriteString("}\n")

	if err := os.WriteFile(s.conf, []byte(conf.String()), 0600); err != nil {
		return fmt.Errorf("failed to write nginx config %q: %w", s.conf, err)
	}

	if err := systemctl.Reload("nginx"); err != nil {
		_, err = shell.Execf("nginx -t")
		return fmt.Errorf("failed to reload nginx: %w", err)
	}

	return nil
}

// CleanUp cleans up the HTTP server on last call.
func (s *panelSolver) CleanUp(ctx context.Context, _ acme.Challenge) error {
	s.cleanupCount++

	// 等待最后一次 CleanUp
	if s.cleanupCount < len(s.ip) {
		return nil
	}

	defer panelSolverGlobal.Unlock()

	if s.useBuiltin && s.server != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := s.server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("failed to shutdown HTTP server: %w", err)
		}
		s.server = nil
		return nil
	}

	// 清理 nginx 配置
	if err := os.WriteFile(s.conf, []byte(""), 0600); err != nil {
		return fmt.Errorf("failed to write to nginx config %q: %w", s.conf, err)
	}

	if err := systemctl.Reload("nginx"); err != nil {
		_, _ = shell.Execf("nginx -t")
		return fmt.Errorf("failed to reload nginx: %w", err)
	}

	return nil
}

type httpSolver struct {
	conf string
}

func (s httpSolver) Present(_ context.Context, challenge acme.Challenge) error {
	conf := fmt.Sprintf(`location = %s {
    default_type text/plain;
    return 200 %q;
}
`, challenge.HTTP01ResourcePath(), challenge.KeyAuthorization)

	file, err := os.OpenFile(s.conf, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open nginx config %q: %w", s.conf, err)
	}
	defer func(file *os.File) { _ = file.Close() }(file)

	if _, err = file.Write([]byte(conf)); err != nil {
		return fmt.Errorf("failed to write to nginx config %q: %w", s.conf, err)
	}

	if err = systemctl.Reload("nginx"); err != nil {
		_, err = shell.Execf("nginx -t")
		return fmt.Errorf("failed to reload nginx: %w", err)
	}

	return nil
}

// CleanUp cleans up the HTTP server if it is the last one to finish.
func (s httpSolver) CleanUp(_ context.Context, challenge acme.Challenge) error {
	conf, err := os.ReadFile(s.conf)
	if err != nil {
		return fmt.Errorf("failed to read nginx config %q: %w", s.conf, err)
	}

	target := fmt.Sprintf(`location = %s {
    default_type text/plain;
    return 200 %q;
}
`, challenge.HTTP01ResourcePath(), challenge.KeyAuthorization)

	newConf := strings.ReplaceAll(string(conf), target, "")
	if err = os.WriteFile(s.conf, []byte(newConf), 0600); err != nil {
		return fmt.Errorf("failed to write to nginx config %q: %w", s.conf, err)
	}

	if err = systemctl.Reload("nginx"); err != nil {
		_, err = shell.Execf("nginx -t")
		return fmt.Errorf("failed to reload nginx: %w", err)
	}

	return nil
}

type dnsSolver struct {
	dns     DnsType
	param   DNSParam
	records []libdns.Record
}

func (s *dnsSolver) Present(ctx context.Context, challenge acme.Challenge) error {
	dnsName := challenge.DNS01TXTRecordName()
	keyAuth := challenge.DNS01KeyAuthorization()
	provider, err := s.getDNSProvider()
	if err != nil {
		return fmt.Errorf("failed to get DNS provider: %w", err)
	}
	zone, err := publicsuffix.EffectiveTLDPlusOne(dnsName)
	if err != nil {
		return fmt.Errorf("failed to get the effective TLD+1 for %q: %w", dnsName, err)
	}

	rec := libdns.TXT{
		Name: libdns.RelativeName(dnsName+".", zone+"."),
		Text: keyAuth,
		TTL:  10 * time.Minute,
	}

	results, err := provider.SetRecords(ctx, zone+".", []libdns.Record{rec})
	if err != nil {
		return fmt.Errorf("failed to set DNS record %q for %q: %w", dnsName, zone, err)
	}
	if len(results) != 1 {
		return fmt.Errorf("expected to add 1 record, but actually added %d records", len(results))
	}

	s.records = results
	return nil
}

func (s *dnsSolver) CleanUp(ctx context.Context, challenge acme.Challenge) error {
	dnsName := challenge.DNS01TXTRecordName()
	provider, err := s.getDNSProvider()
	if err != nil {
		return fmt.Errorf("failed to get DNS provider: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	zone, err := publicsuffix.EffectiveTLDPlusOne(dnsName)
	if err != nil {
		return fmt.Errorf("failed to get the effective TLD+1 for %q: %w", dnsName, err)
	}

	_, _ = provider.DeleteRecords(ctx, zone+".", s.records)
	return nil
}

func (s *dnsSolver) getDNSProvider() (DNSProvider, error) {
	var dns DNSProvider

	switch s.dns {
	case AliYun:
		dns = &alidns.Provider{
			CredentialInfo: alidns.CredentialInfo{
				AccessKeyID:     s.param.AK,
				AccessKeySecret: s.param.SK,
			},
		}
	case Tencent:
		dns = &tencentcloud.Provider{
			SecretId:  s.param.AK,
			SecretKey: s.param.SK,
		}
	case Huawei:
		dns = &huaweicloud.Provider{
			AccessKeyId:     s.param.AK,
			SecretAccessKey: s.param.SK,
		}
	case Westcn:
		dns = &westcn.Provider{
			Username:    s.param.SK,
			APIPassword: s.param.AK,
		}
	case CloudFlare:
		dns = &cloudflare.Provider{
			APIToken: s.param.AK,
		}
	case Gcore:
		dns = &gcore.Provider{
			APIKey: s.param.AK,
		}
	case Porkbun:
		dns = &porkbun.Provider{
			APIKey:       s.param.AK,
			APISecretKey: s.param.SK,
		}
	case NameSilo:
		dns = &namesilo.Provider{
			APIToken: s.param.AK,
		}
	case ClouDNS:
		if strings.HasPrefix(s.param.AK, "sub-") {
			dns = &cloudns.Provider{
				SubAuthId:    strings.TrimPrefix(s.param.AK, "sub-"),
				AuthPassword: s.param.SK,
			}
		} else {
			dns = &cloudns.Provider{
				AuthId:       s.param.AK,
				AuthPassword: s.param.SK,
			}
		}
	default:
		return nil, fmt.Errorf("unsupported DNS provider: %s", s.dns)
	}

	return dns, nil
}

type DnsType string

const (
	AliYun     DnsType = "aliyun"
	Tencent    DnsType = "tencent"
	Huawei     DnsType = "huawei"
	Westcn     DnsType = "westcn"
	CloudFlare DnsType = "cloudflare"
	Gcore      DnsType = "gcore"
	Porkbun    DnsType = "porkbun"
	NameSilo   DnsType = "namesilo"
	ClouDNS    DnsType = "cloudns"
)

type DNSParam struct {
	AK string `form:"ak" json:"ak"`
	SK string `form:"sk" json:"sk"`
}

type DNSProvider interface {
	libdns.RecordSetter
	libdns.RecordDeleter
}

type manualDNSSolver struct {
	check       bool // 是否检查 DNS 解析，目前没写
	controlChan chan struct{}
	dnsChan     chan any
	certChan    chan any
	records     []DNSRecord
}

func (s *manualDNSSolver) Present(ctx context.Context, challenge acme.Challenge) error {
	full := challenge.DNS01TXTRecordName()
	keyAuth := challenge.DNS01KeyAuthorization()
	domain, err := publicsuffix.EffectiveTLDPlusOne(full)
	if err != nil {
		return fmt.Errorf("failed to get the effective TLD+1 for %q: %w", full, err)
	}

	s.records = append(s.records, DNSRecord{
		Name:   strings.TrimSuffix(full, "."+domain),
		Domain: domain,
		Value:  keyAuth,
	})
	s.dnsChan <- s.records

	select {
	case <-s.controlChan:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *manualDNSSolver) CleanUp(_ context.Context, _ acme.Challenge) error {
	defer func() { _ = recover() }()
	close(s.controlChan)
	close(s.dnsChan)
	close(s.certChan)
	return nil
}

type DNSRecord struct {
	Name   string `json:"name"`
	Domain string `json:"domain"`
	Value  string `json:"value"`
}
