package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

type Server struct {
	cfg        Config
	httpClient *http.Client
	logger     *slog.Logger
}

func NewServer(cfg Config, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.index)
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /readyz", s.ready)
	mux.HandleFunc("/api/", s.proxyAPI)
	return s.logRequests(mux)
}

func (s *Server) index(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = io.WriteString(w, indexHTML)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	resp, err := s.callAPI(r.Context(), http.MethodGet, "/readyz", nil)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "unavailable",
			"error":  "api is not reachable",
		})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"status":     "unavailable",
			"api_status": resp.StatusCode,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) proxyAPI(w http.ResponseWriter, r *http.Request) {
	apiPath := strings.TrimPrefix(r.URL.Path, "/api")
	if apiPath == "" {
		apiPath = "/"
	}
	target := s.apiURL(apiPath, r.URL.RawQuery)
	req, err := http.NewRequestWithContext(r.Context(), r.Method, target.String(), r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "api request could not be created"})
		return
	}
	copyHeaders(req.Header, r.Header)
	req.Header.Set("X-NetFlow-Client", "docker-go-client")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Warn("api proxy failed", "error", err, "path", apiPath)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "api is not reachable"})
		return
	}
	defer resp.Body.Close()

	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		s.logger.Warn("api proxy response copy failed", "error", err, "path", apiPath)
	}
}

func (s *Server) callAPI(ctx context.Context, method, apiPath string, body []byte) (*http.Response, error) {
	target := s.apiURL(apiPath, "")
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, target.String(), reader)
	if err != nil {
		return nil, err
	}
	return s.httpClient.Do(req)
}

func (s *Server) apiURL(apiPath, rawQuery string) url.URL {
	target := *s.cfg.APIBaseURL
	target.Path = joinURLPath(s.cfg.APIBaseURL.Path, apiPath)
	target.RawQuery = rawQuery
	return target
}

func joinURLPath(basePath, apiPath string) string {
	if basePath == "" {
		basePath = "/"
	}
	if apiPath == "" {
		apiPath = "/"
	}
	joined := path.Join(basePath, apiPath)
	if strings.HasSuffix(apiPath, "/") && !strings.HasSuffix(joined, "/") {
		joined += "/"
	}
	if !strings.HasPrefix(joined, "/") {
		joined = "/" + joined
	}
	return joined
}

func copyHeaders(dst, src http.Header) {
	for name, values := range src {
		if isHopByHopHeader(name) {
			continue
		}
		for _, value := range values {
			dst.Add(name, value)
		}
	}
}

func isHopByHopHeader(name string) bool {
	switch strings.ToLower(name) {
	case "connection", "keep-alive", "proxy-authenticate", "proxy-authorization",
		"te", "trailer", "transfer-encoding", "upgrade":
		return true
	default:
		return false
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		s.logger.Info("client request completed",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", recorder.status),
			slog.String("duration", fmt.Sprint(time.Since(start))),
		)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
