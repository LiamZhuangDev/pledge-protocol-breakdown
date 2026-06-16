package store

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("record not found")

type PoolRepository interface {
	UpsertPoolBase(ctx context.Context, pool PoolBase) error
	UpsertPoolData(ctx context.Context, data PoolData) error
	GetPoolBase(ctx context.Context, key PoolKey) (PoolBase, error)
	GetPoolData(ctx context.Context, key PoolKey) (PoolData, error)
	ListPoolBases(ctx context.Context, chainID string) ([]PoolBase, error)
	ListPoolData(ctx context.Context, chainID string) ([]PoolData, error)
}

type TokenRepository interface {
	UpsertToken(ctx context.Context, token TokenInfo) error
	GetToken(ctx context.Context, key TokenKey) (TokenInfo, error)
	ListTokens(ctx context.Context, chainID string) ([]TokenInfo, error)
}

type Repository interface {
	PoolRepository
	TokenRepository
}
