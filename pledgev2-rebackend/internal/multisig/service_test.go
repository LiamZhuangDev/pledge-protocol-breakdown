package multisig

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestSetAndGet(t *testing.T) {
	ctx := context.Background()
	memoryStore := newFakeStore()
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	memoryStore.now = func() time.Time { return now }
	service := NewService(memoryStore)

	err := service.Set(ctx, Config{
		ChainID:          "97",
		SPName:           "SP",
		SPToken:          "SP",
		JPName:           "JP",
		JPToken:          "JP",
		SPAddress:        "0xsp",
		JPAddress:        "0xjp",
		SPHash:           "0xsphash",
		JPHash:           "0xjphash",
		MultiSignAccount: []string{"0xowner1", "0xowner2"},
	})
	if err != nil {
		t.Fatalf("set multisig: %v", err)
	}

	cfg, err := service.Get(ctx, "97")
	if err != nil {
		t.Fatalf("get multisig: %v", err)
	}
	if cfg.SPName != "SP" {
		t.Fatalf("expected SP name, got %s", cfg.SPName)
	}
	if len(cfg.MultiSignAccount) != 2 {
		t.Fatalf("expected two multisig accounts, got %d", len(cfg.MultiSignAccount))
	}
	if cfg.CreatedAt != now || cfg.UpdatedAt != now {
		t.Fatalf("unexpected timestamps: %+v", cfg)
	}
}

func TestSetRequiresSPName(t *testing.T) {
	service := NewService(newFakeStore())

	err := service.Set(context.Background(), Config{
		ChainID:          "97",
		MultiSignAccount: []string{"0xowner1"},
	})
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("expected invalid config, got %v", err)
	}
}

func TestGetReturnsNotFound(t *testing.T) {
	service := NewService(newFakeStore())

	_, err := service.Get(context.Background(), "97")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

type fakeStore struct {
	records map[string]Config
	now     func() time.Time
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		records: make(map[string]Config),
		now:     time.Now,
	}
}

func (s *fakeStore) Save(_ context.Context, cfg Config) error {
	now := s.now().UTC()
	if existing, ok := s.records[cfg.ChainID]; ok {
		cfg.CreatedAt = existing.CreatedAt
	} else {
		cfg.CreatedAt = now
	}
	cfg.UpdatedAt = now
	s.records[cfg.ChainID] = cfg
	return nil
}

func (s *fakeStore) Get(_ context.Context, chainID string) (Config, error) {
	cfg, ok := s.records[chainID]
	if !ok {
		return Config{}, ErrNotFound
	}
	return cfg, nil
}
