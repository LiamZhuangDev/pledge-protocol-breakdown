package store

import (
	"context"
	"sort"
	"sync"
	"time"
)

type MemoryStore struct {
	mu       sync.RWMutex
	poolBase map[PoolKey]PoolBase
	poolData map[PoolKey]PoolData
	tokens   map[TokenKey]TokenInfo
	now      func() time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		poolBase: make(map[PoolKey]PoolBase),
		poolData: make(map[PoolKey]PoolData),
		tokens:   make(map[TokenKey]TokenInfo),
		now:      time.Now,
	}
}

func (s *MemoryStore) UpsertPoolBase(_ context.Context, pool PoolBase) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now().UTC()
	if existing, ok := s.poolBase[pool.Key]; ok {
		pool.CreatedAt = existing.CreatedAt
	} else if pool.CreatedAt.IsZero() {
		pool.CreatedAt = now
	}
	pool.UpdatedAt = now
	s.poolBase[pool.Key] = pool
	return nil
}

func (s *MemoryStore) UpsertPoolData(_ context.Context, data PoolData) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now().UTC()
	if existing, ok := s.poolData[data.Key]; ok {
		data.CreatedAt = existing.CreatedAt
	} else if data.CreatedAt.IsZero() {
		data.CreatedAt = now
	}
	data.UpdatedAt = now
	s.poolData[data.Key] = data
	return nil
}

func (s *MemoryStore) GetPoolBase(_ context.Context, key PoolKey) (PoolBase, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pool, ok := s.poolBase[key]
	if !ok {
		return PoolBase{}, ErrNotFound
	}
	return pool, nil
}

func (s *MemoryStore) GetPoolData(_ context.Context, key PoolKey) (PoolData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, ok := s.poolData[key]
	if !ok {
		return PoolData{}, ErrNotFound
	}
	return data, nil
}

func (s *MemoryStore) ListPoolBases(_ context.Context, chainID string) ([]PoolBase, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pools := make([]PoolBase, 0)
	for _, pool := range s.poolBase {
		if pool.Key.ChainID == chainID {
			pools = append(pools, pool)
		}
	}

	sort.Slice(pools, func(i, j int) bool {
		return pools[i].Key.PoolID < pools[j].Key.PoolID
	})

	return pools, nil
}

func (s *MemoryStore) ListPoolData(_ context.Context, chainID string) ([]PoolData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pools := make([]PoolData, 0)
	for _, data := range s.poolData {
		if data.Key.ChainID == chainID {
			pools = append(pools, data)
		}
	}

	sort.Slice(pools, func(i, j int) bool {
		return pools[i].Key.PoolID < pools[j].Key.PoolID
	})

	return pools, nil
}

func (s *MemoryStore) UpsertToken(_ context.Context, token TokenInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now().UTC()
	if existing, ok := s.tokens[token.Key]; ok {
		token.CreatedAt = existing.CreatedAt
	} else if token.CreatedAt.IsZero() {
		token.CreatedAt = now
	}
	token.UpdatedAt = now
	s.tokens[token.Key] = token
	return nil
}

func (s *MemoryStore) GetToken(_ context.Context, key TokenKey) (TokenInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	token, ok := s.tokens[key]
	if !ok {
		return TokenInfo{}, ErrNotFound
	}
	return token, nil
}

func (s *MemoryStore) ListTokens(_ context.Context, chainID string) ([]TokenInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tokens := make([]TokenInfo, 0)
	for _, token := range s.tokens {
		if token.Key.ChainID == chainID {
			tokens = append(tokens, token)
		}
	}

	sort.Slice(tokens, func(i, j int) bool {
		return tokens[i].Symbol < tokens[j].Symbol
	})

	return tokens, nil
}
