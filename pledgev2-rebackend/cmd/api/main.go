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

	"pledgev2-rebackend/internal/auth"
	"pledgev2-rebackend/internal/chain"
	"pledgev2-rebackend/internal/config"
	"pledgev2-rebackend/internal/httpserver"
	"pledgev2-rebackend/internal/logging"
	"pledgev2-rebackend/internal/store"
)

func main() {
	cfg := config.Load()
	logger := logging.New(cfg.Env)

	repo := store.NewMemoryStore()
	reader := chain.NewDemoReader()
	if err := chain.SyncPools(context.Background(), reader, repo, cfg.ChainID); err != nil {
		logger.Error("sync demo contract data failed", slog.Any("error", err))
		os.Exit(1)
	}

	authService := auth.NewService(auth.Config{
		AdminUsername: cfg.AdminUsername,
		AdminPassword: cfg.AdminPassword,
		TokenSecret:   cfg.TokenSecret,
		TokenTTL:      cfg.TokenTTL,
	})

	server := httpserver.New(cfg, logger, repo, authService)

	go func() {
		logger.Info("api server starting", slog.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("api server failed", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	waitForShutdown(server, logger)
}

func waitForShutdown(server *http.Server, logger *slog.Logger) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logger.Info("api server shutting down")
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("api server shutdown failed", slog.Any("error", err))
	}
}
