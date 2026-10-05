package cryptoutil

import (
	"bytes"
	"testing"
)

func TestBackupEncryption(t *testing.T) {
	plaintext := []byte("password: target-secret\nproxyJumpPassword: jump-secret\n")
	encrypted, err := EncryptBackup(plaintext, " long backup passphrase ")
	if err != nil {
		t.Fatal(err)
	}
	if !IsEncryptedBackup(encrypted) || bytes.Contains(encrypted, []byte("target-secret")) {
		t.Fatal("backup is not confidential")
	}
	another, err := EncryptBackup(plaintext, " long backup passphrase ")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(encrypted, another) {
		t.Fatal("encryption reused salt/nonce")
	}
	decrypted, err := DecryptBackup(encrypted, " long backup passphrase ")
	if err != nil || !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("round trip failed: %v", err)
	}
	clear(decrypted)
	if _, err := DecryptBackup(encrypted, "wrong"); err == nil {
		t.Fatal("wrong passphrase accepted")
	}
	for _, offset := range []int{0, 8, 9, 10, 11, 12, 28, 40, len(encrypted) - 1} {
		damaged := bytes.Clone(encrypted)
		damaged[offset] ^= 1
		if data, err := DecryptBackup(damaged, " long backup passphrase "); err == nil || len(data) != 0 {
			t.Fatalf("tamper at %d accepted", offset)
		}
	}
	for _, size := range []int{0, 7, 12, 39, 40, 55, len(encrypted) - 1} {
		if _, err := DecryptBackup(encrypted[:size], " long backup passphrase "); err == nil {
			t.Fatalf("truncation at %d accepted", size)
		}
	}
	if _, err := EncryptBackup(plaintext, ""); err == nil {
		t.Fatal("empty passphrase accepted")
	}
}
