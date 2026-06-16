package chain

import (
	"context"
	"testing"

	"pledgev2-rebackend/internal/store"
)

func TestSyncPoolsCopiesContractSnapshotsIntoStore(t *testing.T) {
	ctx := context.Background()
	repo := store.NewMemoryStore()

	if err := SyncPools(ctx, NewDemoReader(), repo, "97"); err != nil {
		t.Fatalf("sync pools: %v", err)
	}

	pools, err := repo.ListPoolBases(ctx, "97")
	if err != nil {
		t.Fatalf("list pool bases: %v", err)
	}
	if len(pools) != 1 {
		t.Fatalf("expected one pool, got %d", len(pools))
	}

	pool := pools[0]
	if pool.Key.PoolID != 1 {
		t.Fatalf("expected contract index 0 to become pool id 1, got %d", pool.Key.PoolID)
	}
	if pool.LendToken.Symbol != "BUSD" || pool.BorrowToken.Symbol != "BTC" {
		t.Fatalf("unexpected token snapshots: %+v %+v", pool.LendToken, pool.BorrowToken)
	}

	data, err := repo.GetPoolData(ctx, store.PoolKey{ChainID: "97", PoolID: 1})
	if err != nil {
		t.Fatalf("get pool data: %v", err)
	}
	if data.SettleAmountLend != "0" {
		t.Fatalf("unexpected settle amount: %s", data.SettleAmountLend)
	}
}

func TestSyncPoolsStoresTokenMetadata(t *testing.T) {
	ctx := context.Background()
	repo := store.NewMemoryStore()

	if err := SyncPools(ctx, NewDemoReader(), repo, "97"); err != nil {
		t.Fatalf("sync pools: %v", err)
	}

	tokens, err := repo.ListTokens(ctx, "97")
	if err != nil {
		t.Fatalf("list tokens: %v", err)
	}
	if len(tokens) != 2 {
		t.Fatalf("expected two tokens, got %d", len(tokens))
	}
}
