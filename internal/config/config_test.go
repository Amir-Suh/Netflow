package config

import (
	"strings"
	"testing"
)

func TestLoadRejectsNonSandboxPlaidEnv(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("PLAID_ENV", "production")

	_, err := Load()
	if err == nil {
		t.Fatal("expected non-sandbox Plaid env to fail")
	}
	if !strings.Contains(err.Error(), "PLAID_ENV must be exactly sandbox") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadAcceptsSandboxPlaidEnv(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("PLAID_ENV", "sandbox")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Plaid.BaseURL != plaidSandboxBaseURL {
		t.Fatalf("Plaid base url = %q", cfg.Plaid.BaseURL)
	}
	if len(cfg.Crypto.Key) != 32 {
		t.Fatalf("crypto key length = %d", len(cfg.Crypto.Key))
	}
}

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://netflow:netflow@localhost:5432/netflow?sslmode=disable")
	t.Setenv("RABBITMQ_URL", "amqp://netflow:netflow@localhost:5672/")
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("AES_256_GCM_KEY_BASE64", "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=")
	t.Setenv("PLAID_CLIENT_ID", "sandbox-client")
	t.Setenv("PLAID_SECRET", "sandbox-secret")
	t.Setenv("PLAID_WEBHOOK_URL", "http://localhost:8080/plaid/webhook")
}

