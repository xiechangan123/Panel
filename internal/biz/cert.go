package biz

import (
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/leonelquinteros/gotext"
	mholtacme "github.com/mholt/acmez/v3/acme"
	"github.com/samber/lo"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/acme"
	pkgcert "github.com/acepanel/panel/v3/pkg/cert"
	"github.com/acepanel/panel/v3/pkg/types"
)

type Cert struct {
	ID          uint                  `gorm:"primaryKey" json:"id"`
	AccountID   uint                  `gorm:"not null;default:0" json:"account_id"`                      // 关联的 ACME 账户 ID
	DNSID       uint                  `gorm:"not null;default:0" json:"dns_id"`                          // 关联的 DNS ID
	Type        string                `gorm:"not null;default:''" json:"type"`                           // 证书类型 (P256, P384, 2048, 3072, 4096)
	Domains     []string              `gorm:"not null;default:'[]';serializer:json" json:"domains"`      // 域名
	Alias       map[string]string     `gorm:"not null;default:'{}';serializer:json" json:"alias"`        // DNS 验证别名
	AutoRenewal bool                  `gorm:"not null;default:false" json:"auto_renewal"`                // 自动续签
	RenewalInfo mholtacme.RenewalInfo `gorm:"not null;default:'{}';serializer:json" json:"renewal_info"` // 续签信息
	CertURL     string                `gorm:"not null;default:''" json:"cert_url"`                       // 证书 URL (续签时使用)
	Cert        string                `gorm:"not null;default:''" json:"cert"`                           // 证书内容
	Key         string                `gorm:"not null;default:''" json:"key"`                            // 私钥内容
	Script      string                `gorm:"not null;default:''" json:"script"`                         // 部署脚本
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`

	Websites []*Website   `gorm:"foreignKey:CertID" json:"websites"` // 部署的网站
	Account  *CertAccount `gorm:"foreignKey:AccountID" json:"account"`
	DNS      *CertDNS     `gorm:"foreignKey:DNSID" json:"dns"`
}

// WebsiteIDs 取证书关联的网站 ID 列表
func (r *Cert) WebsiteIDs() []uint {
	return lo.Map(r.Websites, func(website *Website, _ int) uint {
		return website.ID
	})
}

type CertRepo interface {
	List(page, limit uint) ([]*types.CertList, int64, error)
	Get(id uint) (*Cert, error)
	GetByWebsite(WebsiteID uint) (*Cert, error)
	Create(req *request.CertCreate) (*Cert, error)
	CreateUploaded(cert *Cert) error
	Update(req *request.CertUpdate) error
	Delete(id uint) error
	Save(cert *Cert) error
	GetClient(cert *Cert) (*acme.Client, error)
	GenerateSelfSigned(domains []string) ([]byte, []byte, error)
	RunScript(cert *Cert) error
	ObtainPanel(account *CertAccount, names []string, webServer string) ([]byte, []byte, error)
	LoadWebsites(websiteIDs []uint) ([]*Website, error)
	BindWebsites(certID uint, websiteIDs []uint) error
	HTTPConfs(cert *Cert, webServer string) (map[string]string, []string)
	WriteCertFiles(cert *Cert, certPath, keyPath string) error
	EnableWebsiteSSL(website *Website, certPath, keyPath, webServer string, tlsVersions []string, listenIPv6 bool) error
	ReloadWebserver(webServer string) error
}

type CertUsecase struct {
	repo    CertRepo
	setting SettingRepo
	t       *gotext.Locale
	log     *slog.Logger
}

func NewCertUsecase(t *gotext.Locale, log *slog.Logger, certRepo CertRepo, settingRepo SettingRepo) *CertUsecase {
	return &CertUsecase{
		repo:    certRepo,
		setting: settingRepo,
		t:       t,
		log:     log,
	}
}

func (uc *CertUsecase) List(page, limit uint) ([]*types.CertList, int64, error) {
	return uc.repo.List(page, limit)
}

func (uc *CertUsecase) Get(id uint) (*Cert, error) {
	return uc.repo.Get(id)
}

func (uc *CertUsecase) GetByWebsite(websiteID uint) (*Cert, error) {
	return uc.repo.GetByWebsite(websiteID)
}

func (uc *CertUsecase) Upload(ctx context.Context, req *request.CertUpload) (*Cert, error) {
	info, err := pkgcert.ParseCert([]byte(req.Cert))
	if err != nil {
		return nil, errors.New(uc.t.Get("failed to parse certificate: %v", err))
	}
	if _, err = pkgcert.ParseKey([]byte(req.Key)); err != nil {
		return nil, errors.New(uc.t.Get("failed to parse private key: %v", err))
	}

	// 合并 DNSNames 和 IPAddresses
	domains := info.DNSNames
	for _, ip := range info.IPAddresses {
		domains = append(domains, ip.String())
	}

	cert := &Cert{
		Type:    "upload",
		Domains: domains,
		Cert:    req.Cert,
		Key:     req.Key,
	}
	if err = uc.repo.CreateUploaded(cert); err != nil {
		return nil, err
	}

	// 记录日志
	uc.log.Info("cert uploaded", slog.String("type", OperationTypeCert), slog.Uint64("operator_id", operatorID(ctx)), slog.Uint64("id", uint64(cert.ID)))

	return cert, nil
}

func (uc *CertUsecase) Create(ctx context.Context, req *request.CertCreate) (*Cert, error) {
	cert, err := uc.repo.Create(req)
	if err != nil {
		return nil, err
	}

	// 记录日志
	uc.log.Info("cert created", slog.String("type", OperationTypeCert), slog.Uint64("operator_id", operatorID(ctx)), slog.Uint64("id", uint64(cert.ID)), slog.String("cert_type", req.Type))

	return cert, nil
}

func (uc *CertUsecase) Update(ctx context.Context, req *request.CertUpdate) error {
	info, err := pkgcert.ParseCert([]byte(req.Cert))
	if err == nil && req.Type == "upload" {
		// 合并 DNSNames 和 IPAddresses
		req.Domains = info.DNSNames
		for _, ip := range info.IPAddresses {
			req.Domains = append(req.Domains, ip.String())
		}
	}
	if req.Type == "upload" && req.AutoRenewal {
		return errors.New(uc.t.Get("upload certificate cannot be set to auto renewal"))
	}

	if err = uc.repo.Update(req); err != nil {
		return err
	}

	// 记录日志
	uc.log.Info("cert updated", slog.String("type", OperationTypeCert), slog.Uint64("operator_id", operatorID(ctx)), slog.Uint64("id", uint64(req.ID)))

	return nil
}

func (uc *CertUsecase) Delete(ctx context.Context, id uint) error {
	if err := uc.repo.Delete(id); err != nil {
		return err
	}

	// 记录日志
	uc.log.Info("cert deleted", slog.String("type", OperationTypeCert), slog.Uint64("operator_id", operatorID(ctx)), slog.Uint64("id", uint64(id)))

	return nil
}

func (uc *CertUsecase) ObtainAuto(id uint) (*acme.Certificate, error) {
	return uc.ObtainAutoWithProgressCallback(context.Background(), id, nil)
}

func (uc *CertUsecase) ObtainAutoWithProgressCallback(ctx context.Context, id uint, progressCallback func(string)) (*acme.Certificate, error) {
	report := func(msg string) {
		if progressCallback != nil {
			progressCallback(msg)
		}
	}

	report(uc.t.Get("initializing ACME client"))
	cert, err := uc.repo.Get(id)
	if err != nil {
		return nil, err
	}

	client, err := uc.repo.GetClient(cert)
	if err != nil {
		return nil, err
	}

	webServer, _ := uc.setting.Get(SettingKeyWebserver)

	if cert.DNS != nil {
		client.UseDns(cert.DNS.Type, cert.DNS.Data, acme.DnsOption{
			Alias:            cert.Alias,
			DnsServer:        cert.DNS.Data.DnsServer,
			SkipVerify:       cert.DNS.Data.SkipVerify,
			ProgressCallback: progressCallback,
		})
	} else {
		if len(cert.Websites) == 0 {
			return nil, errors.New(uc.t.Get("this certificate is not associated with a website and cannot be obtained. You can try to obtain it manually"))
		}
		hasWildcard := slices.ContainsFunc(cert.Domains, func(d string) bool {
			return strings.Contains(d, "*")
		})
		if hasWildcard {
			return nil, errors.New(uc.t.Get("wildcard domains cannot use HTTP verification"))
		}
		confs, fallback := uc.repo.HTTPConfs(cert, webServer)
		client.UseHTTP(confs, fallback, webServer)
	}

	report(uc.t.Get("issuing certificate, domains: %s", strings.Join(cert.Domains, ", ")))
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	ssl, err := client.ObtainCertificate(ctx, cert.Domains, acme.KeyType(cert.Type))
	if err != nil {
		return nil, err
	}

	report(uc.t.Get("obtaining and saving certificate"))
	cert.RenewalInfo = *ssl.RenewalInfo
	cert.CertURL = ssl.URL
	cert.Cert = string(ssl.ChainPEM)
	cert.Key = string(ssl.PrivateKey)
	if err = uc.repo.Save(cert); err != nil {
		return nil, err
	}

	if len(cert.Websites) > 0 {
		report(uc.t.Get("deploying certificate to website"))
		return &ssl, uc.Deploy(cert.ID, cert.WebsiteIDs(), false)
	}

	if err = uc.repo.RunScript(cert); err != nil {
		return nil, err
	}

	return &ssl, nil
}

func (uc *CertUsecase) ObtainPanel(account *CertAccount, domains []string) ([]byte, []byte, error) {
	names := domains
	if len(names) == 0 {
		var err error
		names, err = uc.setting.GetSlice(SettingKeyPublicIPs)
		if err != nil {
			return nil, nil, err
		}
		if len(names) == 0 {
			return nil, nil, errors.New(uc.t.Get("Please set the panel IP in settings first for ACME certificate generation"))
		}
	}

	webServer, _ := uc.setting.Get(SettingKeyWebserver)
	return uc.repo.ObtainPanel(account, names, webServer)
}

func (uc *CertUsecase) ObtainSelfSigned(id uint) error {
	cert, err := uc.repo.Get(id)
	if err != nil {
		return err
	}

	crt, key, err := uc.repo.GenerateSelfSigned(cert.Domains)
	if err != nil {
		return err
	}

	cert.Cert = string(crt)
	cert.Key = string(key)
	if err = uc.repo.Save(cert); err != nil {
		return err
	}

	if len(cert.Websites) > 0 {
		return uc.Deploy(cert.ID, cert.WebsiteIDs(), false)
	}

	if err = uc.repo.RunScript(cert); err != nil {
		return err
	}

	return nil
}

func (uc *CertUsecase) Renew(id uint) (*acme.Certificate, error) {
	return uc.RenewWithProgressCallback(context.Background(), id, nil)
}

func (uc *CertUsecase) RenewWithProgressCallback(ctx context.Context, id uint, progressCallback func(string)) (*acme.Certificate, error) {
	report := func(msg string) {
		if progressCallback != nil {
			progressCallback(msg)
		}
	}

	report(uc.t.Get("preparing renewal"))
	cert, err := uc.repo.Get(id)
	if err != nil {
		return nil, err
	}

	client, err := uc.repo.GetClient(cert)
	if err != nil {
		return nil, err
	}

	if cert.CertURL == "" {
		return nil, errors.New(uc.t.Get("this certificate has not been obtained successfully and cannot be renewed"))
	}

	webServer, _ := uc.setting.Get(SettingKeyWebserver)

	if cert.DNS != nil {
		client.UseDns(cert.DNS.Type, cert.DNS.Data, acme.DnsOption{
			Alias:            cert.Alias,
			DnsServer:        cert.DNS.Data.DnsServer,
			SkipVerify:       cert.DNS.Data.SkipVerify,
			ProgressCallback: progressCallback,
		})
	} else {
		if len(cert.Websites) == 0 {
			return nil, errors.New(uc.t.Get("this certificate is not associated with a website and cannot be obtained. You can try to obtain it manually"))
		}
		for _, domain := range cert.Domains {
			if strings.Contains(domain, "*") {
				return nil, errors.New(uc.t.Get("wildcard domains cannot use HTTP verification"))
			}
		}
		confs, fallback := uc.repo.HTTPConfs(cert, webServer)
		client.UseHTTP(confs, fallback, webServer)
	}

	report(uc.t.Get("renewing certificate, domains: %s", strings.Join(cert.Domains, ", ")))
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	ssl, err := client.RenewCertificate(ctx, cert.CertURL, cert.Domains, acme.KeyType(cert.Type))
	if err != nil {
		// 续签失败，尝试重签
		report(uc.t.Get("renewal failed, attempting re-issuance"))
		ssl, err = client.ObtainCertificate(ctx, cert.Domains, acme.KeyType(cert.Type))
		if err != nil {
			return nil, err
		}
	}

	report(uc.t.Get("obtaining and saving certificate"))
	cert.RenewalInfo = *ssl.RenewalInfo
	cert.CertURL = ssl.URL
	cert.Cert = string(ssl.ChainPEM)
	cert.Key = string(ssl.PrivateKey)
	if err = uc.repo.Save(cert); err != nil {
		return nil, err
	}

	if len(cert.Websites) > 0 {
		report(uc.t.Get("deploying certificate to website"))
		return &ssl, uc.Deploy(cert.ID, cert.WebsiteIDs(), false)
	}

	return &ssl, nil
}

func (uc *CertUsecase) RefreshRenewalInfo(id uint) (mholtacme.RenewalInfo, error) {
	cert, err := uc.repo.Get(id)
	if err != nil {
		return mholtacme.RenewalInfo{}, err
	}
	client, err := uc.repo.GetClient(cert)
	if err != nil {
		return mholtacme.RenewalInfo{}, err
	}

	crt, err := pkgcert.ParseCert([]byte(cert.Cert))
	if err != nil {
		return mholtacme.RenewalInfo{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	renewInfo, err := client.GetRenewalInfo(ctx, crt)
	if err != nil {
		return mholtacme.RenewalInfo{}, err
	}

	cert.RenewalInfo = renewInfo
	if err = uc.repo.Save(cert); err != nil {
		return mholtacme.RenewalInfo{}, err
	}

	return renewInfo, nil
}

func (uc *CertUsecase) Deploy(id uint, websiteIDs []uint, enableHTTPS bool) error {
	cert, err := uc.repo.Get(id)
	if err != nil {
		return err
	}

	if cert.Cert == "" || cert.Key == "" {
		return errors.New(uc.t.Get("this certificate has not been obtained successfully and cannot be deployed"))
	}

	websites, err := uc.repo.LoadWebsites(websiteIDs)
	if err != nil {
		return err
	}

	// 建立关联，使续签后能自动部署到这些网站
	if err = uc.repo.BindWebsites(cert.ID, websiteIDs); err != nil {
		return err
	}

	webServer, webServerErr := uc.setting.Get(SettingKeyWebserver)
	tlsVersions, _ := uc.setting.GetSlice(SettingKeyWebsiteTLSVersions)
	listenIPv6, err := uc.setting.GetBool(SettingKeyWebsiteListenIPv6, false)
	if err != nil {
		return err
	}

	for _, website := range websites {
		configDir := filepath.Join(app.Root, "sites", website.Name, "config")
		certPath := filepath.Join(configDir, "fullchain.pem")
		keyPath := filepath.Join(configDir, "private.key")
		if err = uc.repo.WriteCertFiles(cert, certPath, keyPath); err != nil {
			return err
		}

		// 开启 HTTPS
		if enableHTTPS && !website.SSL {
			if webServerErr != nil {
				return webServerErr
			}
			if err = uc.repo.EnableWebsiteSSL(website, certPath, keyPath, webServer, tlsVersions, listenIPv6); err != nil {
				return err
			}
		}
	}

	return uc.repo.ReloadWebserver(webServer)
}

func (uc *CertUsecase) Save(cert *Cert) error {
	return uc.repo.Save(cert)
}
