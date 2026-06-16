package scheduler

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"pledgev2-rebackend/internal/chain"
	"pledgev2-rebackend/internal/price"
	"pledgev2-rebackend/internal/store"
)

func TestPoolSyncerRunOnce(t *testing.T) {
	ctx := context.Background()
	repo := store.NewMemoryStore()
	syncer := NewPoolSyncer(
		chain.NewDemoReader(),
		repo,
		"97",
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		price.NewService(price.NewDemoProvider()),
		"PLGR",
	)

	if err := syncer.RunOnce(ctx); err != nil {
		t.Fatalf("run once: %v", err)
	}

	pools, err := repo.ListPoolBases(ctx, "97")
	if err != nil {
		t.Fatalf("list pool bases: %v", err)
	}
	if len(pools) != 1 {
		t.Fatalf("expected one synced pool, got %d", len(pools))
	}

	tokens, err := repo.ListTokens(ctx, "97")
	if err != nil {
		t.Fatalf("list tokens: %v", err)
	}
	if len(tokens) != 2 {
		t.Fatalf("expected two synced tokens, got %d", len(tokens))
	}
}
