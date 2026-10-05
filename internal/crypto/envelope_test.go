package cryptoutil

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"testing"
)

func legacyCiphertext(t *testing.T, plain string, key []byte) []byte {
	t.Helper()
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	nonce := bytes.Repeat([]byte{7}, 12)
	return gcm.Seal(nonce, nonce, []byte(plain), nil)
}

func TestStoreEnvelopeAndLegacyCompatibility(t *testing.T) {
	key := bytes.Repeat([]byte{1}, 32)
	encrypted, err := EncryptData("secret", key)
	if err != nil {
		t.Fatal(err)
	}
	if !IsVersionedStore(encrypted) {
		t.Fatal("missing envelope")
	}
	for _, data := range [][]byte{encrypted, legacyCiphertext(t, "secret", key)} {
		got, err := DecryptData(data, key)
		if err != nil || got != "secret" {
			t.Fatalf("compatibility: %v", err)
		}
	}
	for _, index := range []int{0, 8, 9, 10, 11, 12, 24, len(encrypted) - 1} {
		data := bytes.Clone(encrypted)
		data[index] ^= 1
		if got, err := DecryptData(data, key); err == nil || got != "" {
			t.Fatalf("tamper at %d accepted", index)
		}
	}
	for n := 0; n < len(encrypted); n++ {
		if _, err := DecryptData(encrypted[:n], key); err == nil {
			t.Fatalf("truncation at %d accepted", n)
		}
	}
	if _, err := DecryptData(encrypted, bytes.Repeat([]byte{2}, 32)); err == nil {
		t.Fatal("wrong key accepted")
	}
}

func FuzzDecryptData(f *testing.F) {
	key := bytes.Repeat([]byte{1}, 32)
	valid, err := EncryptData("fixture", key)
	if err != nil {
		f.Fatal(err)
	}
	f.Add(valid)
	f.Add([]byte{})
	f.Add([]byte("SSHMSTOR"))
	f.Add(bytes.Repeat([]byte{0}, 64))
	f.Fuzz(func(t *testing.T, data []byte) {
		got, err := DecryptData(data, key)
		if err != nil && got != "" {
			t.Fatal("failed decryption returned partial plaintext")
		}
	})
}
