package biz_test

import (
	"bytes"
	"io"
	"log/slog"
	"testing"

	"github.com/leonelquinteros/gotext"

	"github.com/acepanel/panel/v3/internal/biz"
	mockbiz "github.com/acepanel/panel/v3/mocks/biz"
)

func TestCertUsecase_ObtainPanel(t *testing.T) {
	t.Run("绑定域名时优先签发域名证书", func(t *testing.T) {
		t.Parallel()

		account := &biz.CertAccount{}
		domains := []string{"panel.example.com"}
		certRepo := mockbiz.NewCertRepo(t)
		settingRepo := mockbiz.NewSettingRepo(t)
		settingRepo.EXPECT().Get(biz.SettingKeyWebserver).Return("nginx", nil)
		certRepo.EXPECT().ObtainPanel(account, domains, "nginx").Return([]byte("cert"), []byte("key"), nil)

		uc, err := biz.NewCertUsecase(gotext.NewLocale("", "en"), slog.New(slog.NewTextHandler(io.Discard, nil)), certRepo, settingRepo)
		if err != nil {
			t.Fatal(err)
		}

		crt, key, err := uc.ObtainPanel(account, domains)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(crt, []byte("cert")) || !bytes.Equal(key, []byte("key")) {
			t.Fatalf("证书返回值不匹配: crt=%q key=%q", crt, key)
		}
	})

	t.Run("未绑定域名时回退签发IP证书", func(t *testing.T) {
		t.Parallel()

		account := &biz.CertAccount{}
		ips := []string{"192.0.2.1"}
		certRepo := mockbiz.NewCertRepo(t)
		settingRepo := mockbiz.NewSettingRepo(t)
		settingRepo.EXPECT().GetSlice(biz.SettingKeyPublicIPs).Return(ips, nil)
		settingRepo.EXPECT().Get(biz.SettingKeyWebserver).Return("apache", nil)
		certRepo.EXPECT().ObtainPanel(account, ips, "apache").Return([]byte("cert"), []byte("key"), nil)

		uc, err := biz.NewCertUsecase(gotext.NewLocale("", "en"), slog.New(slog.NewTextHandler(io.Discard, nil)), certRepo, settingRepo)
		if err != nil {
			t.Fatal(err)
		}

		if _, _, err = uc.ObtainPanel(account, nil); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("域名和公网IP均为空时返回错误", func(t *testing.T) {
		t.Parallel()

		settingRepo := mockbiz.NewSettingRepo(t)
		settingRepo.EXPECT().GetSlice(biz.SettingKeyPublicIPs).Return(nil, nil)

		uc, err := biz.NewCertUsecase(gotext.NewLocale("", "en"), slog.New(slog.NewTextHandler(io.Discard, nil)), mockbiz.NewCertRepo(t), settingRepo)
		if err != nil {
			t.Fatal(err)
		}

		if _, _, err = uc.ObtainPanel(&biz.CertAccount{}, nil); err == nil {
			t.Fatal("预期返回缺少面板公网 IP 的错误")
		}
	})
}
