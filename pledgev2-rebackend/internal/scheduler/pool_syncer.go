package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"pledgev2-rebackend/internal/chain"
	"pledgev2-rebackend/internal/price"
	"pledgev2-rebackend/internal/store"
)

type PoolSyncer struct {
	reader  chain.Reader
	repo    store.Repository
	chainID string
	logger  *slog.Logger
	prices  *price.Service
	symbol  string
}

func NewPoolSyncer(reader chain.Reader, repo store.Repository, chainID string, logger *slog.Logger, prices *price.Service, symbol string) *PoolSyncer {
	return &PoolSyncer{
		reader:  reader,
		repo:    repo,
		chainID: chainID,
		logger:  logger,
		prices:  prices,
		symbol:  symbol,
	}
}

func (s *PoolSyncer) RunOnce(ctx context.Context) error {
	if err := chain.SyncPools(ctx, s.reader, s.repo, s.chainID); err != nil {
		return err
	}

	pools, err := s.repo.ListPoolBases(ctx, s.chainID)
	if err != nil {
		return fmt.Errorf("count synced pools: %w", err)
	}
	tokens, err := s.repo.ListTokens(ctx, s.chainID)
	if err != nil {
		return fmt.Errorf("count synced tokens: %w", err)
	}

	s.logger.Info(
		"pool sync completed",
		slog.String("chainID", s.chainID),
		slog.Int("pools", len(pools)),
		slog.Int("tokens", len(tokens)),
	)

	if s.prices != nil && s.symbol != "" {
		quote, err := s.prices.Latest(ctx, s.symbol)
		if err != nil {
			return fmt.Errorf("refresh price %s: %w", s.symbol, err)
		}
		s.logger.Info(
			"price refresh completed",
			slog.String("symbol", quote.Symbol),
			slog.String("currency", quote.Currency),
			slog.String("price", quote.Price),
			slog.String("source", quote.Source),
		)
	}

	return nil
}

func (s *PoolSyncer) Run(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		return fmt.Errorf("sync interval must be positive")
	}

	if err := s.RunOnce(ctx); err != nil {
		s.logger.Error("initial pool sync failed", slog.Any("error", err))
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := s.RunOnce(ctx); err != nil {
				s.logger.Error("scheduled pool sync failed", slog.Any("error", err))
			}
		}
	}
}
