package auth

import (
	"strconv"
	"testing"
	"time"
)

func TestPasswordHashAndVerify(t *testing.T) {
	hash, err := HashPassword("correct horse")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "correct horse" {
		t.Fatal("password hash must not contain plaintext")
	}
	if !VerifyPassword(hash, "correct horse") {
		t.Fatal("expected password to verify")
	}
	if VerifyPassword(hash, "wrong password") {
		t.Fatal("expected wrong password to fail")
	}
}

func TestJWTSignAndParse(t *testing.T) {
	secret := []byte("0123456789abcdef0123456789abcdef")
	token, err := NewJWT(secret, 42, "user@example.com", time.Hour)
	if err != nil {
		t.Fatalf("NewJWT: %v", err)
	}
	claims, err := ParseJWT(secret, token)
	if err != nil {
		t.Fatalf("ParseJWT: %v", err)
	}
	if claims.Subject != strconv.FormatInt(42, 10) {
		t.Fatalf("subject = %q", claims.Subject)
	}
	if claims.Email != "user@example.com" {
		t.Fatalf("email = %q", claims.Email)
	}
}

func TestJWTRejectsTampering(t *testing.T) {
	secret := []byte("0123456789abcdef0123456789abcdef")
	token, err := NewJWT(secret, 42, "user@example.com", time.Hour)
	if err != nil {
		t.Fatalf("NewJWT: %v", err)
	}
	token += "tampered"
	if _, err := ParseJWT(secret, token); err == nil {
		t.Fatal("expected tampered token to fail")
	}
}

