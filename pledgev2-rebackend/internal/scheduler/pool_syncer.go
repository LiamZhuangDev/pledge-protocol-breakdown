package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"pledgev2-rebackend/internal/chain"
	"pledgev2-rebackend/internal/store"
)

type PoolSyncer struct {
	reader  chain.Reader
	repo    store.Repository
	chainID string
	logger  *slog.Logger
}

func NewPoolSyncer(reader chain.Reader, repo store.Repository, chainID string, logger *slog.Logger) *PoolSyncer {
	return &PoolSyncer{
		reader:  reader,
		repo:    repo,
		chainID: chainID,
		logger:  logger,
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
