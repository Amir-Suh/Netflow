package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"netflow/internal/auth"
	"netflow/internal/config"
	"netflow/internal/db"
	nfcrypto "netflow/internal/crypto"
	"netflow/internal/plaid"
	"netflow/internal/queue"
)

type transactionPublisher interface {
	PublishTransactionIngested(context.Context, queue.TransactionEvent) error
}

type Server struct {
	cfg       config.Config
	store     *db.Store
	encryptor *nfcrypto.AESGCM
	plaid     *plaid.Client
	publisher transactionPublisher
	hub       *Hub
	logger    *slog.Logger
}

func NewServer(cfg config.Config, store *db.Store, encryptor *nfcrypto.AESGCM, plaidClient *plaid.Client, publisher transactionPublisher, hub *Hub, logger *slog.Logger) *Server {
	return &Server{
		cfg:       cfg,
		store:     store,
		encryptor: encryptor,
		plaid:     plaidClient,
		publisher: publisher,
		hub:       hub,
		logger:    logger,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /readyz", s.ready)
	mux.HandleFunc("POST /auth/register", s.register)
	mux.HandleFunc("POST /auth/login", s.login)
	mux.HandleFunc("POST /auth/logout", s.logout)
	mux.Handle("POST /plaid/link-token", s.requireAuth(http.HandlerFunc(s.createLinkToken)))
	mux.Handle("POST /plaid/exchange-public-token", s.requireAuth(http.HandlerFunc(s.exchangePublicToken)))
	mux.Handle("POST /plaid/sync", s.requireAuth(http.HandlerFunc(s.syncPlaidItem)))
	mux.HandleFunc("POST /plaid/webhook", s.plaidWebhook)
	mux.Handle("GET /ws", s.requireAuth(http.HandlerFunc(s.handleWebSocket)))
	return s.recoverPanic(s.logRequests(mux))
}

func (s *Server) setAuthCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(s.cfg.Auth.JWTTTL.Seconds()),
	})
}

func (s *Server) clearAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func (s *Server) logout(w http.ResponseWriter, _ *http.Request) {
	s.clearAuthCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.store.Ping(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database is not ready")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if !strings.Contains(req.Email, "@") {
		writeError(w, http.StatusBadRequest, "valid email is required")
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	user, err := s.store.CreateUser(r.Context(), req.Email, hash)
	if err != nil {
		s.logger.Warn("register failed", "error", err)
		writeError(w, http.StatusConflict, "user could not be created")
		return
	}
	token, err := auth.NewJWT(s.cfg.Auth.JWTSecret, user.ID, user.Email, s.cfg.Auth.JWTTTL)
	if err != nil {
		s.logger.Error("jwt creation failed", "error", err)
		writeError(w, http.StatusInternalServerError, "token creation failed")
		return
	}
	s.setAuthCookie(w, token)
	writeJSON(w, http.StatusCreated, map[string]any{"user_id": user.ID, "email": user.Email})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	user, err := s.store.GetUserByEmail(r.Context(), req.Email)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	if err != nil {
		s.logger.Error("login lookup failed", "error", err)
		writeError(w, http.StatusInternalServerError, "login failed")
		return
	}
	if !auth.VerifyPassword(user.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	token, err := auth.NewJWT(s.cfg.Auth.JWTSecret, user.ID, user.Email, s.cfg.Auth.JWTTTL)
	if err != nil {
		s.logger.Error("jwt creation failed", "error", err)
		writeError(w, http.StatusInternalServerError, "token creation failed")
		return
	}
	s.setAuthCookie(w, token)
	writeJSON(w, http.StatusOK, map[string]any{"user_id": user.ID, "email": user.Email})
}

func (s *Server) createLinkToken(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r.Context())
	resp, err := s.plaid.CreateLinkToken(r.Context(), strconv.FormatInt(userID, 10), s.cfg.Plaid.WebhookURL)
	if err != nil {
		s.logger.Error("create plaid sandbox link token failed", "error", err)
		writeError(w, http.StatusBadGateway, "plaid sandbox link token creation failed")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

type exchangePublicTokenRequest struct {
	PublicToken string `json:"public_token"`
}

func (s *Server) exchangePublicToken(w http.ResponseWriter, r *http.Request) {
	var req exchangePublicTokenRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.PublicToken) == "" {
		writeError(w, http.StatusBadRequest, "public_token is required")
		return
	}

	resp, err := s.plaid.ExchangePublicToken(r.Context(), req.PublicToken)
	if err != nil {
		s.logger.Error("exchange plaid sandbox public token failed", "error", err)
		writeError(w, http.StatusBadGateway, "plaid sandbox token exchange failed")
		return
	}
	encryptedAccessToken, err := s.encryptor.EncryptString(resp.AccessToken)
	if err != nil {
		s.logger.Error("encrypt plaid access token failed", "error", err)
		writeError(w, http.StatusInternalServerError, "access token encryption failed")
		return
	}
	item, err := s.store.StorePlaidItem(r.Context(), db.PlaidItem{
		UserID:                userIDFromContext(r.Context()),
		ItemID:                resp.ItemID,
		AccessTokenCiphertext: encryptedAccessToken.Ciphertext,
		AccessTokenNonce:      encryptedAccessToken.Nonce,
		AccessTokenKeyID:      encryptedAccessToken.KeyID,
		AccessTokenAlgorithm:  encryptedAccessToken.Algorithm,
	})
	if err != nil {
		s.logger.Error("store plaid sandbox item failed", "error", err)
		writeError(w, http.StatusInternalServerError, "plaid item storage failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"item_id": item.ItemID, "plaid_item_id": item.ID})
}

type syncRequest struct {
	ItemID string `json:"item_id"`
}

func (s *Server) syncPlaidItem(w http.ResponseWriter, r *http.Request) {
	var req syncRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := s.store.GetPlaidItemByItemID(r.Context(), req.ItemID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "plaid item not found")
		return
	}
	if err != nil {
		s.logger.Error("lookup plaid item failed", "error", err)
		writeError(w, http.StatusInternalServerError, "plaid item lookup failed")
		return
	}
	if item.UserID != userIDFromContext(r.Context()) {
		writeError(w, http.StatusForbidden, "plaid item does not belong to this user")
		return
	}
	count, err := s.syncItemTransactions(r.Context(), item)
	if err != nil {
		s.logger.Error("sync plaid sandbox item failed", "error", err)
		writeError(w, http.StatusBadGateway, "plaid sandbox sync failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"item_id": item.ItemID, "transactions_ingested": count})
}

type plaidWebhookPayload struct {
	WebhookType string `json:"webhook_type"`
	WebhookCode string `json:"webhook_code"`
	ItemID      string `json:"item_id"`
	Environment string `json:"environment"`
}

func (s *Server) plaidWebhook(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid webhook body")
		return
	}
	var payload plaidWebhookPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid webhook json")
		return
	}
	if payload.Environment != "" && payload.Environment != "sandbox" {
		writeError(w, http.StatusBadRequest, "only Plaid Sandbox webhooks are accepted")
		return
	}

	sum := sha256.Sum256(raw)
	eventKey := hex.EncodeToString(sum[:])
	inserted, err := s.store.RecordWebhookEvent(r.Context(), db.WebhookInput{
		EventKey:    eventKey,
		ItemID:      payload.ItemID,
		WebhookType: payload.WebhookType,
		WebhookCode: payload.WebhookCode,
		Payload:     raw,
	})
	if err != nil {
		s.logger.Error("store plaid webhook failed", "error", err)
		writeError(w, http.StatusInternalServerError, "webhook storage failed")
		return
	}
	if !inserted {
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "duplicate"})
		return
	}
	if payload.WebhookType != "TRANSACTIONS" || payload.ItemID == "" {
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
		return
	}

	item, err := s.store.GetPlaidItemByItemID(r.Context(), payload.ItemID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted_unknown_item"})
		return
	}
	if err != nil {
		s.logger.Error("lookup webhook plaid item failed", "error", err)
		writeError(w, http.StatusInternalServerError, "plaid item lookup failed")
		return
	}
	count, err := s.syncItemTransactions(r.Context(), item)
	if err != nil {
		s.logger.Error("webhook plaid sandbox sync failed", "error", err)
		writeError(w, http.StatusBadGateway, "plaid sandbox sync failed")
		return
	}
	if err := s.store.MarkWebhookProcessed(r.Context(), eventKey); err != nil {
		s.logger.Warn("mark webhook processed failed", "error", err)
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"status": "processed", "transactions_ingested": count})
}

func (s *Server) syncItemTransactions(ctx context.Context, item db.PlaidItem) (int, error) {
	accessToken, err := s.encryptor.DecryptString(nfcrypto.EncryptedValue{
		Ciphertext: item.AccessTokenCiphertext,
		Nonce:      item.AccessTokenNonce,
		KeyID:      item.AccessTokenKeyID,
		Algorithm:  item.AccessTokenAlgorithm,
	})
	if err != nil {
		return 0, fmt.Errorf("decrypt plaid access token: %w", err)
	}

	cursor := item.TransactionsCursor
	nextCursor := cursor
	total := 0
	for {
		resp, err := s.plaid.SyncTransactions(ctx, accessToken, cursor)
		if err != nil {
			return total, err
		}
		accountIDs := make(map[string]int64, len(resp.Accounts))
		for _, account := range resp.Accounts {
			id, err := s.store.UpsertAccount(ctx, db.AccountInput{
				UserID:         item.UserID,
				PlaidItemID:    item.ID,
				PlaidAccountID: account.AccountID,
				Name:           account.Name,
				OfficialName:   account.OfficialName,
				Type:           account.Type,
				Subtype:        account.Subtype,
				Mask:           account.Mask,
			})
			if err != nil {
				return total, err
			}
			accountIDs[account.AccountID] = id
		}
		transactions := append(resp.Added, resp.Modified...)
		for _, tx := range transactions {
			accountID, ok := accountIDs[tx.AccountID]
			if !ok {
				accountID, err = s.store.GetAccountIDByPlaidAccountID(ctx, tx.AccountID)
				if err != nil {
					return total, err
				}
			}
			transactionDate, err := time.Parse("2006-01-02", tx.Date)
			if err != nil {
				return total, fmt.Errorf("parse transaction date %q: %w", tx.Date, err)
			}
			encryptedDescription, err := s.encryptor.EncryptString(tx.Name)
			if err != nil {
				return total, fmt.Errorf("encrypt transaction description: %w", err)
			}
			stored, err := s.store.UpsertTransaction(ctx, db.TransactionInput{
				UserID:                item.UserID,
				BankAccountID:        accountID,
				PlaidTransactionID:    tx.TransactionID,
				Amount:                tx.Amount,
				ISOCurrencyCode:       tx.ISOCurrencyCode,
				TransactionDate:       transactionDate,
				Pending:               tx.Pending,
				DescriptionCiphertext:  encryptedDescription.Ciphertext,
				DescriptionNonce:       encryptedDescription.Nonce,
				DescriptionKeyID:       encryptedDescription.KeyID,
				DescriptionAlgorithm:   encryptedDescription.Algorithm,
				MerchantName:          tx.MerchantName,
				SourceEnvironment:     "sandbox",
			})
			if err != nil {
				return total, err
			}
			if err := s.publisher.PublishTransactionIngested(ctx, queue.TransactionEvent{
				EventType:          queue.TransactionIngestedRoutingKey,
				TransactionID:     stored.ID,
				UserID:            stored.UserID,
				AccountID:         stored.BankAccountID,
				SourceEnvironment: "sandbox",
				OccurredAt:        time.Now().UTC(),
			}); err != nil {
				return total, fmt.Errorf("publish transaction event: %w", err)
			}
			total++
		}
		nextCursor = resp.NextCursor
		if !resp.HasMore {
			break
		}
		cursor = resp.NextCursor
	}
	if nextCursor != "" {
		if err := s.store.UpdatePlaidCursor(ctx, item.ID, nextCursor); err != nil {
			return total, err
		}
	}
	return total, nil
}
