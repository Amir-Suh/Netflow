package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

const AlgorithmAES256GCM = "AES-256-GCM"

type EncryptedValue struct {
	Ciphertext string
	Nonce      string
	KeyID      string
	Algorithm  string
}

type AESGCM struct {
	gcm   cipher.AEAD
	keyID string
}

func NewAESGCM(key []byte, keyID string) (*AESGCM, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("AES-256-GCM requires a 32-byte key, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}
	return &AESGCM{gcm: gcm, keyID: keyID}, nil
}

func (a *AESGCM) EncryptString(plaintext string) (EncryptedValue, error) {
	nonce := make([]byte, a.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return EncryptedValue{}, fmt.Errorf("generate nonce: %w", err)
	}
	ciphertext := a.gcm.Seal(nil, nonce, []byte(plaintext), nil)
	return EncryptedValue{
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		KeyID:      a.keyID,
		Algorithm:  AlgorithmAES256GCM,
	}, nil
}

func (a *AESGCM) DecryptString(value EncryptedValue) (string, error) {
	if value.Algorithm != "" && value.Algorithm != AlgorithmAES256GCM {
		return "", fmt.Errorf("unsupported encryption algorithm %q", value.Algorithm)
	}
	nonce, err := base64.StdEncoding.DecodeString(value.Nonce)
	if err != nil {
		return "", fmt.Errorf("decode nonce: %w", err)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(value.Ciphertext)
	if err != nil {
		return "", fmt.Errorf("decode ciphertext: %w", err)
	}
	plaintext, err := a.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt value: %w", err)
	}
	return string(plaintext), nil
}

