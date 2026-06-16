package multisig

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	ErrInvalidConfig = errors.New("invalid multisig config")
	ErrNotFound      = errors.New("multisig config not found")
)

type Config struct {
	ChainID          string    `json:"chain_id"`
	SPName           string    `json:"sp_name"`
	SPToken          string    `json:"_spToken"`
	JPName           string    `json:"jp_name"`
	JPToken          string    `json:"_jpToken"`
	SPAddress        string    `json:"sp_address"`
	JPAddress        string    `json:"jp_address"`
	SPHash           string    `json:"spHash"`
	JPHash           string    `json:"jpHash"`
	MultiSignAccount []string  `json:"multi_sign_account"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type MultiSigStore interface {
	Save(ctx context.Context, cfg Config) error
	Get(ctx context.Context, chainID string) (Config, error)
}

type Service struct {
	store MultiSigStore
}

func NewService(store MultiSigStore) *Service {
	return &Service{store: store}
}

func (s *Service) Set(ctx context.Context, cfg Config) error {
	if cfg.ChainID == "" {
		return fmt.Errorf("%w: chain_id is required", ErrInvalidConfig)
	}
	if cfg.SPName == "" {
		return fmt.Errorf("%w: sp_name is required", ErrInvalidConfig)
	}
	if len(cfg.MultiSignAccount) == 0 {
		return fmt.Errorf("%w: multi_sign_account is required", ErrInvalidConfig)
	}
	return s.store.Save(ctx, cfg)
}

func (s *Service) Get(ctx context.Context, chainID string) (Config, error) {
	if chainID == "" {
		return Config{}, fmt.Errorf("%w: chain_id is required", ErrInvalidConfig)
	}
	return s.store.Get(ctx, chainID)
}
