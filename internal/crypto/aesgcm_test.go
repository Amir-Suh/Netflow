package crypto

import (
	"encoding/base64"
	"testing"
)

func TestAESGCMEncryptDecrypt(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	encryptor, err := NewAESGCM(key, "test-key")
	if err != nil {
		t.Fatalf("NewAESGCM: %v", err)
	}

	encrypted, err := encryptor.EncryptString("Coffee Shop")
	if err != nil {
		t.Fatalf("EncryptString: %v", err)
	}
	if encrypted.Ciphertext == "Coffee Shop" {
		t.Fatal("ciphertext must not contain plaintext")
	}
	if encrypted.Algorithm != AlgorithmAES256GCM {
		t.Fatalf("algorithm = %q", encrypted.Algorithm)
	}

	decrypted, err := encryptor.DecryptString(encrypted)
	if err != nil {
		t.Fatalf("DecryptString: %v", err)
	}
	if decrypted != "Coffee Shop" {
		t.Fatalf("decrypted = %q", decrypted)
	}
}

func TestAESGCMUsesFreshNonce(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	encryptor, err := NewAESGCM(key, "test-key")
	if err != nil {
		t.Fatalf("NewAESGCM: %v", err)
	}

	first, err := encryptor.EncryptString("same plaintext")
	if err != nil {
		t.Fatalf("EncryptString first: %v", err)
	}
	second, err := encryptor.EncryptString("same plaintext")
	if err != nil {
		t.Fatalf("EncryptString second: %v", err)
	}
	if first.Nonce == second.Nonce {
		t.Fatal("expected a fresh nonce for each encryption")
	}
	if first.Ciphertext == second.Ciphertext {
		t.Fatal("expected different ciphertext for identical plaintext")
	}
}

func TestAESGCMRejectsTamperedCiphertext(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	encryptor, err := NewAESGCM(key, "test-key")
	if err != nil {
		t.Fatalf("NewAESGCM: %v", err)
	}

	encrypted, err := encryptor.EncryptString("Coffee Shop")
	if err != nil {
		t.Fatalf("EncryptString: %v", err)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted.Ciphertext)
	if err != nil {
		t.Fatalf("decode ciphertext: %v", err)
	}
	ciphertext[0] ^= 0xff
	encrypted.Ciphertext = base64.StdEncoding.EncodeToString(ciphertext)

	if _, err := encryptor.DecryptString(encrypted); err == nil {
		t.Fatal("expected tampered ciphertext to fail authentication")
	}
}

func TestAESGCMRequires32ByteKey(t *testing.T) {
	if _, err := NewAESGCM([]byte("short"), "bad-key"); err == nil {
		t.Fatal("expected short key to fail")
	}
}

