package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"pledgev2-rebackend/internal/multisig"
)

func TestMemoryStoreUpsertsAndListsPoolBases(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	store.now = fixedClock(time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC))

	err := store.UpsertPoolBase(ctx, PoolBase{
		Key:          PoolKey{ChainID: "97", PoolID: 2},
		LendSupply:   "200",
		BorrowSupply: "20",
		State:        PoolStateMatching,
	})
	if err != nil {
		t.Fatalf("upsert pool base: %v", err)
	}

	err = store.UpsertPoolBase(ctx, PoolBase{
		Key:          PoolKey{ChainID: "97", PoolID: 1},
		LendSupply:   "100",
		BorrowSupply: "10",
		State:        PoolStatePending,
	})
	if err != nil {
		t.Fatalf("upsert pool base: %v", err)
	}

	pools, err := store.ListPoolBases(ctx, "97")
	if err != nil {
		t.Fatalf("list pool bases: %v", err)
	}

	if len(pools) != 2 {
		t.Fatalf("expected 2 pools, got %d", len(pools))
	}
	if pools[0].Key.PoolID != 1 || pools[1].Key.PoolID != 2 {
		t.Fatalf("expected pools sorted by pool id, got %+v", pools)
	}
}

func TestMemoryStorePreservesCreatedAtOnPoolUpdate(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	firstTime := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	secondTime := firstTime.Add(time.Hour)

	store.now = fixedClock(firstTime)
	key := PoolKey{ChainID: "97", PoolID: 1}
	if err := store.UpsertPoolData(ctx, PoolData{Key: key, SettleAmountLend: "100"}); err != nil {
		t.Fatalf("upsert pool data: %v", err)
	}

	store.now = fixedClock(secondTime)
	if err := store.UpsertPoolData(ctx, PoolData{Key: key, SettleAmountLend: "200"}); err != nil {
		t.Fatalf("update pool data: %v", err)
	}

	data, err := store.GetPoolData(ctx, key)
	if err != nil {
		t.Fatalf("get pool data: %v", err)
	}

	if data.CreatedAt != firstTime {
		t.Fatalf("expected created_at to stay %s, got %s", firstTime, data.CreatedAt)
	}
	if data.UpdatedAt != secondTime {
		t.Fatalf("expected updated_at to become %s, got %s", secondTime, data.UpdatedAt)
	}
	if data.SettleAmountLend != "200" {
		t.Fatalf("expected updated settle amount, got %s", data.SettleAmountLend)
	}
}

func TestMemoryStoreReturnsErrNotFound(t *testing.T) {
	store := NewMemoryStore()

	_, err := store.GetToken(context.Background(), TokenKey{ChainID: "97", Address: "0xmissing"})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestMemoryStoreSavesMultiSignConfig(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	store.SetClockForTest(func() time.Time { return now })

	err := store.Save(ctx, multisig.Config{
		ChainID:          "97",
		SPName:           "SP",
		MultiSignAccount: []string{"0xowner1"},
	})
	if err != nil {
		t.Fatalf("save multisig: %v", err)
	}

	cfg, err := store.Get(ctx, "97")
	if err != nil {
		t.Fatalf("get multisig: %v", err)
	}
	if cfg.SPName != "SP" {
		t.Fatalf("expected SP, got %+v", cfg)
	}
	if cfg.CreatedAt != now || cfg.UpdatedAt != now {
		t.Fatalf("unexpected timestamps: %+v", cfg)
	}
}

func TestMemoryStoreReturnsMultiSignNotFound(t *testing.T) {
	_, err := NewMemoryStore().Get(context.Background(), "97")
	if !errors.Is(err, multisig.ErrNotFound) {
		t.Fatalf("expected multisig not found, got %v", err)
	}
}

func fixedClock(t time.Time) func() time.Time {
	return func() time.Time {
		return t
	}
}
