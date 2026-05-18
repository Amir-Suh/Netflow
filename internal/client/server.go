package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"
	"time"
)

type Server struct {
	cfg        Config
	httpClient *http.Client
	proxy      *httputil.ReverseProxy
	logger     *slog.Logger
}

func NewServer(cfg Config, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	s := &Server{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
	s.proxy = s.newReverseProxy()
	return s
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.index)
	mux.HandleFunc("GET /login", s.login)
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

func (s *Server) login(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = io.WriteString(w, loginHTML)
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

// newReverseProxy returns an httputil.ReverseProxy that strips the "/api"
// prefix and forwards everything else to the upstream API. ReverseProxy
// natively handles WebSocket upgrades (since Go 1.12), which the previous
// manual proxy did not.
func (s *Server) newReverseProxy() *httputil.ReverseProxy {
	target := s.cfg.APIBaseURL
	return &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
			apiPath := strings.TrimPrefix(pr.In.URL.Path, "/api")
			pr.Out.URL.Path = joinURLPath(target.Path, apiPath)
			pr.Out.URL.RawQuery = pr.In.URL.RawQuery
			pr.Out.Host = target.Host
			pr.Out.Header.Set("X-NetFlow-Client", "docker-go-client")
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			s.logger.Warn("api proxy failed", "error", err, "path", r.URL.Path)
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "api is not reachable"})
		},
	}
}

func (s *Server) proxyAPI(w http.ResponseWriter, r *http.Request) {
	s.proxy.ServeHTTP(w, r)
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

// Hijack lets httputil.ReverseProxy proxy WebSocket upgrades through this
// middleware. The embedded ResponseWriter implements http.Hijacker, but our
// wrapper type hides it from interface assertions unless we forward.
func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := r.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, errors.New("response writer does not support hijacking")
}

func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
