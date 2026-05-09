package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"netflow/internal/config"
	nfcrypto "netflow/internal/crypto"
	"netflow/internal/db"
	"netflow/internal/httpapi"
	"netflow/internal/plaid"
	"netflow/internal/queue"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration failed", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	if err := store.RunMigrations(ctx); err != nil {
		logger.Error("database migrations failed", "error", err)
		os.Exit(1)
	}

	encryptor, err := nfcrypto.NewAESGCM(cfg.Crypto.Key, cfg.Crypto.KeyID)
	if err != nil {
		logger.Error("encryption setup failed", "error", err)
		os.Exit(1)
	}

	publisher, err := queue.NewPublisherWithRetry(ctx, cfg.RabbitMQ.URL, cfg.RabbitMQ.Exchange, cfg.RabbitMQ.Queue, time.Second, 30)
	if err != nil {
		logger.Error("rabbitmq setup failed", "error", err)
		os.Exit(1)
	}
	defer publisher.Close()

	plaidClient := plaid.NewClient(cfg.Plaid.BaseURL, cfg.Plaid.ClientID, cfg.Plaid.Secret)
	api := httpapi.NewServer(cfg, store, encryptor, plaidClient, publisher, logger)
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("netflow api listening", "addr", cfg.HTTPAddr, "plaid_env", cfg.Plaid.Env)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("http shutdown failed", "error", err)
	}
}
