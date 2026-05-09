package client

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestLoadConfigValidatesAPIBaseURL(t *testing.T) {
	t.Setenv("NETFLOW_API_BASE_URL", "ftp://api:8080")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected invalid API base URL to fail")
	}
}

func TestProxyStripsAPIPrefixAndForwardsHeaders(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/readyz" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.URL.RawQuery != "probe=1" {
			t.Fatalf("raw query = %q", r.URL.RawQuery)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer token" {
			t.Fatalf("Authorization = %q", got)
		}
		if got := r.Header.Get("X-NetFlow-Client"); got != "docker-go-client" {
			t.Fatalf("X-NetFlow-Client = %q", got)
		}
		w.Header().Set("X-Backend", "ok")
		_, _ = io.WriteString(w, `{"status":"ready"}`)
	}))
	defer backend.Close()

	handler := newTestServer(t, backend.URL).Handler()
	req := httptest.NewRequest(http.MethodGet, "/api/readyz?probe=1", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("X-Backend"); got != "ok" {
		t.Fatalf("X-Backend = %q", got)
	}
}

func TestReadyReflectsBackendReadiness(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/readyz" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, `{"status":"waiting"}`)
	}))
	defer backend.Close()

	handler := newTestServer(t, backend.URL).Handler()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "unavailable" {
		t.Fatalf("status body = %v", body)
	}
}

func newTestServer(t *testing.T, rawBaseURL string) *Server {
	t.Helper()
	parsed, err := url.Parse(rawBaseURL)
	if err != nil {
		t.Fatal(err)
	}
	return NewServer(Config{
		HTTPAddr:   ":0",
		APIBaseURL: parsed,
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
}
