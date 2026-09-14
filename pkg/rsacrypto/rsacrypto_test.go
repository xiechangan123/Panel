package rsacrypto

import (
	"testing"

	"github.com/libtnb/assert/check"
	"github.com/libtnb/assert/must"
)

func TestRSA(t *testing.T) {
	privateKey, err := GenerateKey()
	must.NoError(t, err)
	check.Equal(t, privateKey.Size(), keySize/8)

	privateStr, err := PrivateKeyToString(privateKey)
	check.NoError(t, err)
	check.Contains(t, privateStr, "-----BEGIN RSA PRIVATE KEY-----")
	publicStr, err := PublicKeyToString(&privateKey.PublicKey)
	check.NoError(t, err)
	check.Contains(t, publicStr, "-----BEGIN PUBLIC KEY-----")

	message := []byte("AcePanel")

	ciphertext, err := EncryptData(&privateKey.PublicKey, message)
	must.NoError(t, err)
	check.NotEmpty(t, ciphertext)

	decrypted, err := DecryptData(privateKey, ciphertext)
	must.NoError(t, err)
	check.Equal(t, string(decrypted), string(message))
}
