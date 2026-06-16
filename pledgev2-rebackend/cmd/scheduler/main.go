package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"pledgev2-rebackend/internal/chain"
	"pledgev2-rebackend/internal/config"
	"pledgev2-rebackend/internal/logging"
	"pledgev2-rebackend/internal/price"
	"pledgev2-rebackend/internal/scheduler"
	"pledgev2-rebackend/internal/store"
)

func main() {
	cfg := config.Load()
	logger := logging.New(cfg.Env)

	repo := store.NewMemoryStore()
	reader := chain.NewDemoReader()
	priceService := price.NewService(price.NewDemoProvider())
	syncer := scheduler.NewPoolSyncer(reader, repo, cfg.ChainID, logger, priceService, cfg.PriceSymbol)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.Info(
		"scheduler starting",
		slog.String("chainID", cfg.ChainID),
		slog.Duration("interval", cfg.SyncInterval),
		slog.String("priceSymbol", cfg.PriceSymbol),
	)

	if err := syncer.Run(ctx, cfg.SyncInterval); err != nil {
		logger.Error("scheduler stopped with error", slog.Any("error", err))
		os.Exit(1)
	}

	logger.Info("scheduler stopped")
}
