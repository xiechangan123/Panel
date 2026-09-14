package acme

import (
	"context"
	"log/slog"
	"testing"

	"github.com/libtnb/assert/check"
	"github.com/libtnb/assert/must"
)

func TestObtainSSL(t *testing.T) {
	ctx := context.Background()
	client, err := NewRegisterAccount(ctx, "ci@haozi.net", CALetsEncryptStaging, nil, KeyEC256, slog.Default())
	must.NoError(t, err)

	client.UseDns(AliYun, DNSParam{
		AK: "123456",
		SK: "654321",
	})

	/*client.UseManualDns(2)

	resolves, err := client.GetDNSRecords(ctx, []string{"*.haozi.net", "haozi.net"}, KeyEC256)
	debug.Dump(resolves)
	check.Nil(t, err)
	check.NotNil(t, resolves)

	time.Sleep(2 * time.Minute)

	ssl, err := client.ObtainCertificateManual()*/
	ssl, err := client.ObtainCertificate(ctx, []string{"*.haozi.net", "haozi.net"}, KeyEC256)
	check.Error(t, err)
	check.Zero(t, ssl)
}
