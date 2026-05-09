package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
)

const plaidSandboxBaseURL = "https://sandbox.plaid.com"

type Config struct {
	HTTPAddr           string
	PublicBaseURL      string
	DatabaseURL        string
	ReplicaDatabaseURL string
	RabbitMQ           RabbitMQConfig
	Auth               AuthConfig
	Crypto             CryptoConfig
	Plaid              PlaidConfig
}

type RabbitMQConfig struct {
	URL      string
	Exchange string
	Queue    string
}

type AuthConfig struct {
	JWTSecret []byte
	JWTTTL    time.Duration
}

type CryptoConfig struct {
	Key   []byte
	KeyID string
}

type PlaidConfig struct {
	Env        string
	BaseURL    string
	ClientID   string
	Secret     string
	WebhookURL string
}

func Load() (Config, error) {
	var cfg Config

	cfg.HTTPAddr = getenv("HTTP_ADDR", ":8080")
	cfg.PublicBaseURL = strings.TrimRight(getenv("PUBLIC_BASE_URL", "http://localhost:8080"), "/")
	cfg.DatabaseURL = os.Getenv("DATABASE_URL")
	cfg.ReplicaDatabaseURL = os.Getenv("DATABASE_REPLICA_URL")
	cfg.RabbitMQ = RabbitMQConfig{
		URL:      os.Getenv("RABBITMQ_URL"),
		Exchange: getenv("RABBITMQ_EXCHANGE", "netflow.events"),
		Queue:    getenv("RABBITMQ_QUEUE", "transactions.ingested"),
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if len(jwtSecret) < 32 {
		return Config{}, errors.New("JWT_SECRET must be at least 32 bytes")
	}
	ttl, err := time.ParseDuration(getenv("JWT_TTL", "24h"))
	if err != nil {
		return Config{}, fmt.Errorf("parse JWT_TTL: %w", err)
	}
	cfg.Auth = AuthConfig{JWTSecret: []byte(jwtSecret), JWTTTL: ttl}

	key, err := decodeAESKey(os.Getenv("AES_256_GCM_KEY_BASE64"))
	if err != nil {
		return Config{}, err
	}
	cfg.Crypto = CryptoConfig{Key: key, KeyID: getenv("AES_KEY_ID", "local-sandbox-v1")}

	plaidEnv := os.Getenv("PLAID_ENV")
	if plaidEnv != "sandbox" {
		return Config{}, fmt.Errorf("PLAID_ENV must be exactly sandbox for Cycle 1, got %q", plaidEnv)
	}
	cfg.Plaid = PlaidConfig{
		Env:        plaidEnv,
		BaseURL:    plaidSandboxBaseURL,
		ClientID:   os.Getenv("PLAID_CLIENT_ID"),
		Secret:     os.Getenv("PLAID_SECRET"),
		WebhookURL: os.Getenv("PLAID_WEBHOOK_URL"),
	}
	if cfg.Plaid.WebhookURL == "" {
		cfg.Plaid.WebhookURL = cfg.PublicBaseURL + "/plaid/webhook"
	}

	if err := validate(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func decodeAESKey(value string) ([]byte, error) {
	if value == "" {
		return nil, errors.New("AES_256_GCM_KEY_BASE64 is required")
	}
	key, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("decode AES_256_GCM_KEY_BASE64: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("AES_256_GCM_KEY_BASE64 must decode to 32 bytes, got %d", len(key))
	}
	return key, nil
}

func validate(cfg Config) error {
	required := map[string]string{
		"DATABASE_URL":    cfg.DatabaseURL,
		"RABBITMQ_URL":    cfg.RabbitMQ.URL,
		"PLAID_CLIENT_ID": cfg.Plaid.ClientID,
		"PLAID_SECRET":    cfg.Plaid.Secret,
	}
	for name, value := range required {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	if _, err := url.ParseRequestURI(cfg.Plaid.WebhookURL); err != nil {
		return fmt.Errorf("PLAID_WEBHOOK_URL must be a valid URL: %w", err)
	}
	return nil
}

func getenv(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}

