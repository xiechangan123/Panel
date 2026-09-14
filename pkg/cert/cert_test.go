package cert

import (
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/libtnb/assert/check"
	"github.com/libtnb/assert/must"
)

func TestGenerateSelfSigned(t *testing.T) {
	tests := []struct {
		name     string
		gen      func([]string) ([]byte, []byte, error)
		keyBlock string
	}{
		{name: "ecdsa", gen: GenerateSelfSigned, keyBlock: "PRIVATE KEY"},
		{name: "rsa", gen: GenerateSelfSignedRSA, keyBlock: "RSA PRIVATE KEY"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			certPEM, keyPEM, err := test.gen([]string{"haozi.dev"})
			must.NoError(t, err)

			block, _ := pem.Decode(certPEM)
			must.NotNil(t, block)
			check.Equal(t, block.Type, "CERTIFICATE")

			parsed, err := x509.ParseCertificate(block.Bytes)
			must.NoError(t, err)
			check.DeepEqual(t, parsed.DNSNames, []string{"haozi.dev"})
			check.Equal(t, parsed.Subject.CommonName, "AcePanel")

			keyDecoded, _ := pem.Decode(keyPEM)
			must.NotNil(t, keyDecoded)
			check.Equal(t, keyDecoded.Type, test.keyBlock)
		})
	}
}
