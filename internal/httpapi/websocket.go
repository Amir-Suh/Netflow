package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"netflow/internal/queue"
)

const (
	wsWriteTimeout = 10 * time.Second
	wsPongTimeout  = 60 * time.Second
	wsPingInterval = 30 * time.Second
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// CheckOrigin is permissive for the local proxy/sandbox setup.
	// In production this would validate against an allow-list.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Hub fans out worker events to authenticated browser WebSocket sessions.
// Connections are tracked per user; a single user may have multiple browser
// tabs open and each receives the same broadcast.
type Hub struct {
	mu     sync.RWMutex
	conns  map[int64]map[*websocket.Conn]struct{}
	logger *slog.Logger
}

func NewHub(logger *slog.Logger) *Hub {
	if logger == nil {
		logger = slog.Default()
	}
	return &Hub{
		conns:  make(map[int64]map[*websocket.Conn]struct{}),
		logger: logger,
	}
}

func (h *Hub) Register(userID int64, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	bucket, ok := h.conns[userID]
	if !ok {
		bucket = make(map[*websocket.Conn]struct{})
		h.conns[userID] = bucket
	}
	bucket[conn] = struct{}{}
}

func (h *Hub) Unregister(userID int64, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	bucket, ok := h.conns[userID]
	if !ok {
		return
	}
	delete(bucket, conn)
	if len(bucket) == 0 {
		delete(h.conns, userID)
	}
}

func (h *Hub) Broadcast(userID int64, event queue.WorkerEvent) {
	envelope := map[string]any{
		"event_type":     event.EventType,
		"transaction_id": event.TransactionID,
		"run_id":         event.RunID,
		"run_kind":       event.RunKind,
		"status":         event.Status,
		"occurred_at":    event.OccurredAt,
		"payload":        event.Payload,
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		h.logger.Warn("ws marshal failed", "error", err)
		return
	}

	h.mu.RLock()
	bucket := h.conns[userID]
	targets := make([]*websocket.Conn, 0, len(bucket))
	for conn := range bucket {
		targets = append(targets, conn)
	}
	h.mu.RUnlock()

	for _, conn := range targets {
		_ = conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
		if err := conn.WriteMessage(websocket.TextMessage, body); err != nil {
			h.logger.Warn("ws write failed", "user_id", userID, "error", err)
			h.Unregister(userID, conn)
			_ = conn.Close()
		}
	}
}

// StartConsumer drains decoded WorkerEvents from the consumer and forwards
// them to the matching browser sessions. It returns when ctx is canceled or
// when the underlying RabbitMQ channel closes.
func (h *Hub) StartConsumer(ctx context.Context, consumer *queue.Consumer) {
	events := make(chan queue.WorkerEvent, 64)
	go func() {
		if err := consumer.Consume(ctx, events); err != nil {
			h.logger.Error("ws consumer stopped", "error", err)
		}
		close(events)
	}()
	for event := range events {
		if event.UserID == 0 {
			continue
		}
		h.Broadcast(event.UserID, event)
	}
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	if s.hub == nil {
		writeError(w, http.StatusServiceUnavailable, "websocket hub not configured")
		return
	}
	userID := userIDFromContext(r.Context())
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Warn("ws upgrade failed", "error", err)
		return
	}
	s.hub.Register(userID, conn)
	s.logger.Info("ws connected", "user_id", userID)

	conn.SetReadLimit(1 << 16)
	_ = conn.SetReadDeadline(time.Now().Add(wsPongTimeout))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(wsPongTimeout))
	})

	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(wsPingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				_ = conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}()

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
	close(stop)
	s.hub.Unregister(userID, conn)
	_ = conn.Close()
	s.logger.Info("ws disconnected", "user_id", userID)
}
