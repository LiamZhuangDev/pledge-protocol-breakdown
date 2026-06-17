package price

import (
	"context"
	"errors"
	"testing"
	"time"

	"pledgev2-rebackend/internal/cache"
)

func TestCachedProviderUsesCache(t *testing.T) {
	ctx := context.Background()
	cacheStore := newFakeCache()
	provider := NewDemoProvider()
	start := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	provider.now = func() time.Time { return start }

	cached := NewCachedProvider(provider, cacheStore, time.Minute)

	first, err := cached.Latest(ctx, "PLGR")
	if err != nil {
		t.Fatalf("first latest: %v", err)
	}

	provider.quotes["PLGR"] = Quote{
		Symbol:   "PLGR",
		Currency: "USDT",
		Price:    "999",
		Source:   "demo",
	}

	second, err := cached.Latest(ctx, "PLGR")
	if err != nil {
		t.Fatalf("second latest: %v", err)
	}

	if first.Price != second.Price {
		t.Fatalf("expected cached price %s, got %s", first.Price, second.Price)
	}
}

type fakeCache struct {
	records map[string][]byte
}

func newFakeCache() *fakeCache {
	return &fakeCache{records: make(map[string][]byte)}
}

func (c *fakeCache) Get(_ context.Context, key string) ([]byte, error) {
	value, ok := c.records[key]
	if !ok {
		return nil, cache.ErrMiss
	}
	return append([]byte(nil), value...), nil
}

func (c *fakeCache) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	c.records[key] = append([]byte(nil), value...)
	return nil
}

func (c *fakeCache) Delete(_ context.Context, key string) error {
	if _, ok := c.records[key]; !ok {
		return errors.New("missing key")
	}
	delete(c.records, key)
	return nil
}
