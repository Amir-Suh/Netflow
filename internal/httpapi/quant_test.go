package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"netflow/internal/config"
)

func TestQuantRoutesRequireAuthCookie(t *testing.T) {
	s := NewServer(
		config.Config{Auth: config.AuthConfig{JWTSecret: []byte("0123456789abcdef0123456789abcdef")}},
		nil,
		nil,
		nil,
		nil,
		nil,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	req := httptest.NewRequest(http.MethodPost, "/quant/debt/optimize", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()

	s.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "authentication cookie is required") {
		t.Fatalf("unexpected body = %s", rec.Body.String())
	}
}

func TestQuantDefaultsNormalizeSafely(t *testing.T) {
	if got := normalizeAssumptionMethod("live_api"); got != "demo_static" {
		t.Fatalf("assumption method = %q", got)
	}
	if got := normalizeDebtPreference("snowball"); got != "fastest_payoff" {
		t.Fatalf("debt preference = %q", got)
	}
	if got := normalizeAssetClass("", "VTI", ""); got != "us_equity" {
		t.Fatalf("asset class = %q", got)
	}
}
