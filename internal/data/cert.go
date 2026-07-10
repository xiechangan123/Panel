package data

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/leonelquinteros/gotext"
	"github.com/samber/do/v2"
	"github.com/samber/lo"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/acme"
	pkgcert "github.com/acepanel/panel/v3/pkg/cert"
	"github.com/acepanel/panel/v3/pkg/io"
	"github.com/acepanel/panel/v3/pkg/shell"
	"github.com/acepanel/panel/v3/pkg/systemctl"
	"github.com/acepanel/panel/v3/pkg/types"
	"github.com/acepanel/panel/v3/pkg/webserver"
	webservertypes "github.com/acepanel/panel/v3/pkg/webserver/types"
)

type certRepo struct {
	t   *gotext.Locale
	db  *gorm.DB
	log *slog.Logger
}

func NewCertRepo(i do.Injector) (biz.CertRepo, error) {
	return &certRepo{
		t:   do.MustInvoke[*gotext.Locale](i),
		db:  do.MustInvoke[*gorm.DB](i),
		log: do.MustInvoke[*slog.Logger](i),
	}, nil
}

func (r *certRepo) List(page, limit uint) ([]*types.CertList, int64, error) {
	var certs []*biz.Cert
	var total int64
	err := r.db.Model(&biz.Cert{}).Preload("Website").Preload("Account").Preload("DNS").Order("id desc").Count(&total).Offset(int((page - 1) * limit)).Limit(int(limit)).Find(&certs).Error

	list := lo.Map(certs, func(cert *biz.Cert, _ int) *types.CertList {
		item := &types.CertList{
			ID:          cert.ID,
			AccountID:   cert.AccountID,
			WebsiteID:   cert.WebsiteID,
			DNSID:       cert.DNSID,
			Type:        cert.Type,
			Domains:     cert.Domains,
			Alias:       cert.Alias,
			AutoRenewal: cert.AutoRenewal,
			NextRenewal: cert.RenewalInfo.SelectedTime,
			Cert:        cert.Cert,
			Key:         cert.Key,
			CertURL:     cert.CertURL,
			Script:      cert.Script,
			CreatedAt:   cert.CreatedAt,
			UpdatedAt:   cert.UpdatedAt,
		}
		if decode, err := pkgcert.ParseCert([]byte(cert.Cert)); err == nil {
			item.NotBefore = decode.NotBefore
			item.NotAfter = decode.NotAfter
			item.Issuer = decode.Issuer.CommonName
			item.OCSPServer = decode.OCSPServer
			// 合并 DNSNames 和 IPAddresses
			item.DNSNames = append(decode.DNSNames, lo.Map(decode.IPAddresses, func(ip net.IP, _ int) string {
				return ip.String()
			})...)
		}
		return item
	})

	return list, total, err
}

func (r *certRepo) Get(id uint) (*biz.Cert, error) {
	cert := new(biz.Cert)
	err := r.db.Model(&biz.Cert{}).Preload("Website").Preload("Account").Preload("DNS").Where("id = ?", id).First(cert).Error
	return cert, err
}

func (r *certRepo) GetByWebsite(WebsiteID uint) (*biz.Cert, error) {
	cert := new(biz.Cert)
	err := r.db.Model(&biz.Cert{}).Preload("Website").Preload("Account").Preload("DNS").Where("website_id = ?", WebsiteID).First(cert).Error
	return cert, err
}

func (r *certRepo) Create(req *request.CertCreate) (*biz.Cert, error) {
	cert := &biz.Cert{
		AccountID:   req.AccountID,
		WebsiteID:   req.WebsiteID,
		DNSID:       req.DNSID,
		Type:        req.Type,
		Domains:     req.Domains,
		Alias:       req.Alias,
		AutoRenewal: req.AutoRenewal,
	}
	if err := r.db.Create(cert).Error; err != nil {
		return nil, err
	}

	return cert, nil
}

func (r *certRepo) CreateUploaded(cert *biz.Cert) error {
	return r.db.Create(cert).Error
}

func (r *certRepo) Update(req *request.CertUpdate) error {
	return r.db.Model(&biz.Cert{}).Where("id = ?", req.ID).Select("*").Updates(&biz.Cert{
		ID:          req.ID,
		AccountID:   req.AccountID,
		WebsiteID:   req.WebsiteID,
		DNSID:       req.DNSID,
		Type:        req.Type,
		Cert:        req.Cert,
		Key:         req.Key,
		Script:      req.Script,
		Domains:     req.Domains,
		Alias:       req.Alias,
		AutoRenewal: req.AutoRenewal,
	}).Error
}

func (r *certRepo) Delete(id uint) error {
	return r.db.Model(&biz.Cert{}).Where("id = ?", id).Delete(&biz.Cert{}).Error
}

func (r *certRepo) Save(cert *biz.Cert) error {
	return r.db.Save(cert).Error
}

func (r *certRepo) GenerateSelfSigned(domains []string) ([]byte, []byte, error) {
	return pkgcert.GenerateSelfSigned(domains)
}

func (r *certRepo) ObtainPanel(account *biz.CertAccount, ips []string, webServer string) ([]byte, []byte, error) {
	client, err := acme.NewPrivateKeyAccount(account.Email, account.PrivateKey, acme.CALetsEncrypt, nil, r.log)
	if err != nil {
		return nil, nil, err
	}

	confPath := filepath.Join(app.Root, "server/nginx/conf/acme.conf")
	if webServer == "apache" {
		confPath = filepath.Join(app.Root, "server/apache/conf/extra/acme.conf")
	}
	client.UsePanel(ips, confPath, webServer)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	ssl, err := client.ObtainIPCertificate(ctx, ips, acme.KeyEC256)
	if err != nil {
		return nil, nil, err
	}

	return ssl.ChainPEM, ssl.PrivateKey, nil
}

// LoadWebsite 根据 ID 加载网站
func (r *certRepo) LoadWebsite(WebsiteID uint) (*biz.Website, error) {
	website := new(biz.Website)
	if err := r.db.Where("id", WebsiteID).First(website).Error; err != nil {
		return nil, err
	}
	return website, nil
}

// WriteCertFiles 写入证书与私钥文件
func (r *certRepo) WriteCertFiles(cert *biz.Cert, certPath, keyPath string) error {
	if err := io.Write(certPath, cert.Cert, 0600); err != nil {
		return err
	}
	if err := io.Write(keyPath, cert.Key, 0600); err != nil {
		return err
	}
	return nil
}

// EnableWebsiteSSL 为网站开启 HTTPS
func (r *certRepo) EnableWebsiteSSL(website *biz.Website, certPath, keyPath, webServer string, tlsVersions []string) error {
	vhost, err := r.getVhost(website, webServer)
	if err != nil {
		return err
	}

	// 添加 443 监听
	listens := vhost.Listen()
	hasSSL := slices.ContainsFunc(listens, func(l webservertypes.Listen) bool {
		return slices.Contains(l.Args, "ssl")
	})
	if !hasSSL {
		args := []string{"ssl"}
		if webServer != "apache" {
			args = append(args, "quic")
		}
		listens = append(listens, webservertypes.Listen{Address: "443", Args: args})
		if err = vhost.SetListen(listens); err != nil {
			return err
		}
	}

	// 配置 SSL
	if err = vhost.SetSSLConfig(&webservertypes.SSLConfig{
		Cert:      certPath,
		Key:       keyPath,
		Protocols: tlsVersions,
	}); err != nil {
		return err
	}

	if err = vhost.Save(); err != nil {
		return err
	}

	website.SSL = true
	if err = r.db.Save(website).Error; err != nil {
		return err
	}

	return nil
}

// ReloadWebserver 重载 Web 服务器
func (r *certRepo) ReloadWebserver(webServer string) error {
	if webServer == "apache" {
		if err := systemctl.Reload("apache"); err != nil {
			_, err = shell.Execf("apachectl -t")
			return err
		}
	} else {
		if err := systemctl.Reload("nginx"); err != nil {
			_, err = shell.Execf("nginx -t")
			return err
		}
	}

	return nil
}

// getVhost 根据网站类型获取虚拟主机配置
func (r *certRepo) getVhost(website *biz.Website, webServer string) (webservertypes.Vhost, error) {
	configDir := filepath.Join(app.Root, "sites", website.Name, "config")
	var vhost webservertypes.Vhost
	var err error
	switch website.Type {
	case biz.WebsiteTypeProxy:
		vhost, err = webserver.NewProxyVhost(webserver.Type(webServer), configDir)
	case biz.WebsiteTypePHP:
		vhost, err = webserver.NewPHPVhost(webserver.Type(webServer), configDir)
	case biz.WebsiteTypeStatic:
		vhost, err = webserver.NewStaticVhost(webserver.Type(webServer), configDir)
	default:
		return nil, errors.New(r.t.Get("unsupported website type: %s", website.Type))
	}
	if err != nil {
		return nil, err
	}

	return vhost, nil
}

func (r *certRepo) RunScript(cert *biz.Cert) error {
	if cert.Script == "" {
		return nil
	}

	f, err := os.CreateTemp("", "cert-deploy-*.sh")
	if err != nil {
		return err
	}

	// 替换变量
	cert.Script = strings.ReplaceAll(cert.Script, "{cert}", cert.Cert)
	cert.Script = strings.ReplaceAll(cert.Script, "{key}", cert.Key)

	if _, err = f.WriteString(cert.Script); err != nil {
		return err
	}

	_ = f.Chmod(0755)
	_ = f.Close()
	defer func(name string) { _ = os.Remove(name) }(f.Name())

	_, err = shell.Execf("bash " + f.Name())
	return err
}

func (r *certRepo) GetClient(cert *biz.Cert) (*acme.Client, error) {
	if cert.Account == nil {
		return nil, errors.New(r.t.Get("this certificate is not associated with an ACME account and cannot be obtained"))
	}

	var ca string
	var eab *acme.EAB
	switch cert.Account.CA {
	case "googlecn":
		ca = acme.CAGoogleCN
		eab = &acme.EAB{KeyID: cert.Account.Kid, MACKey: cert.Account.HmacEncoded}
	case "google":
		ca = acme.CAGoogle
		eab = &acme.EAB{KeyID: cert.Account.Kid, MACKey: cert.Account.HmacEncoded}
	case "letsencrypt":
		ca = acme.CALetsEncrypt
	case "litessl":
		ca = acme.CALiteSSL
	case "zerossl":
		ca = acme.CAZeroSSL
		eab = &acme.EAB{KeyID: cert.Account.Kid, MACKey: cert.Account.HmacEncoded}
	case "sslcom":
		ca = acme.CASSLcom
		eab = &acme.EAB{KeyID: cert.Account.Kid, MACKey: cert.Account.HmacEncoded}
	}

	return acme.NewPrivateKeyAccount(cert.Account.Email, cert.Account.PrivateKey, ca, eab, r.log)
}
