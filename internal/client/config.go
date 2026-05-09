package client

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

type Config struct {
	HTTPAddr   string
	APIBaseURL *url.URL
}

func LoadConfig() (Config, error) {
	apiBaseURL, err := parseAPIBaseURL(getenv("NETFLOW_API_BASE_URL", "http://localhost:8080"))
	if err != nil {
		return Config{}, err
	}
	return Config{
		HTTPAddr:   getenv("CLIENT_HTTP_ADDR", ":8081"),
		APIBaseURL: apiBaseURL,
	}, nil
}

func parseAPIBaseURL(value string) (*url.URL, error) {
	value = strings.TrimRight(strings.TrimSpace(value), "/")
	if value == "" {
		return nil, fmt.Errorf("NETFLOW_API_BASE_URL is required")
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return nil, fmt.Errorf("parse NETFLOW_API_BASE_URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("NETFLOW_API_BASE_URL must use http or https")
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("NETFLOW_API_BASE_URL must include a host")
	}
	return parsed, nil
}

func getenv(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}
