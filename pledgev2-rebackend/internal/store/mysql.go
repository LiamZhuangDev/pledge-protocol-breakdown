package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"pledgev2-rebackend/internal/multisig"
)

type MySQLStore struct {
	db  *sql.DB
	now func() time.Time
}

func OpenMySQL(ctx context.Context, dsn string) (*MySQLStore, error) {
	if dsn == "" {
		return nil, fmt.Errorf("mysql dsn is required")
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	store := &MySQLStore{
		db:  db,
		now: time.Now,
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := store.Migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *MySQLStore) Close() error {
	return s.db.Close()
}

func (s *MySQLStore) Migrate(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS poolbases (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			chain_id VARCHAR(32) NOT NULL,
			pool_id BIGINT NOT NULL,
			settle_time VARCHAR(100) NOT NULL DEFAULT '',
			end_time VARCHAR(100) NOT NULL DEFAULT '',
			interest_rate VARCHAR(100) NOT NULL DEFAULT '',
			max_supply VARCHAR(100) NOT NULL DEFAULT '',
			lend_supply VARCHAR(100) NOT NULL DEFAULT '',
			borrow_supply VARCHAR(100) NOT NULL DEFAULT '',
			mortgage_rate VARCHAR(100) NOT NULL DEFAULT '',
			lend_token_info JSON NOT NULL,
			borrow_token_info JSON NOT NULL,
			state VARCHAR(32) NOT NULL DEFAULT '',
			sp_coin VARCHAR(128) NOT NULL DEFAULT '',
			jp_coin VARCHAR(128) NOT NULL DEFAULT '',
			auto_liquidate_threshold VARCHAR(100) NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			UNIQUE KEY uniq_poolbases_chain_pool (chain_id, pool_id)
		)`,
		`CREATE TABLE IF NOT EXISTS pooldata (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			chain_id VARCHAR(32) NOT NULL,
			pool_id BIGINT NOT NULL,
			settle_amount_lend VARCHAR(100) NOT NULL DEFAULT '',
			settle_amount_borrow VARCHAR(100) NOT NULL DEFAULT '',
			finish_amount_lend VARCHAR(100) NOT NULL DEFAULT '',
			finish_amount_borrow VARCHAR(100) NOT NULL DEFAULT '',
			liquidation_amount_lend VARCHAR(100) NOT NULL DEFAULT '',
			liquidation_amount_borrow VARCHAR(100) NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			UNIQUE KEY uniq_pooldata_chain_pool (chain_id, pool_id)
		)`,
		`CREATE TABLE IF NOT EXISTS token_info (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			chain_id VARCHAR(32) NOT NULL,
			token VARCHAR(128) NOT NULL,
			symbol VARCHAR(64) NOT NULL DEFAULT '',
			logo VARCHAR(512) NOT NULL DEFAULT '',
			price VARCHAR(100) NOT NULL DEFAULT '',
			decimals INT NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			UNIQUE KEY uniq_token_info_chain_token (chain_id, token)
		)`,
		`CREATE TABLE IF NOT EXISTS multi_sign (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			chain_id VARCHAR(32) NOT NULL,
			sp_name VARCHAR(255) NOT NULL DEFAULT '',
			sp_token VARCHAR(255) NOT NULL DEFAULT '',
			jp_name VARCHAR(255) NOT NULL DEFAULT '',
			jp_token VARCHAR(255) NOT NULL DEFAULT '',
			sp_address VARCHAR(255) NOT NULL DEFAULT '',
			jp_address VARCHAR(255) NOT NULL DEFAULT '',
			sp_hash VARCHAR(255) NOT NULL DEFAULT '',
			jp_hash VARCHAR(255) NOT NULL DEFAULT '',
			multi_sign_account JSON NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			UNIQUE KEY uniq_multi_sign_chain (chain_id)
		)`,
	}

	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func (s *MySQLStore) UpsertPoolBase(ctx context.Context, pool PoolBase) error {
	now := s.now().UTC()
	lendToken, err := json.Marshal(pool.LendToken)
	if err != nil {
		return err
	}
	borrowToken, err := json.Marshal(pool.BorrowToken)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `INSERT INTO poolbases (
		chain_id, pool_id, settle_time, end_time, interest_rate, max_supply,
		lend_supply, borrow_supply, mortgage_rate, lend_token_info, borrow_token_info,
		state, sp_coin, jp_coin, auto_liquidate_threshold, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON DUPLICATE KEY UPDATE
		settle_time=VALUES(settle_time),
		end_time=VALUES(end_time),
		interest_rate=VALUES(interest_rate),
		max_supply=VALUES(max_supply),
		lend_supply=VALUES(lend_supply),
		borrow_supply=VALUES(borrow_supply),
		mortgage_rate=VALUES(mortgage_rate),
		lend_token_info=VALUES(lend_token_info),
		borrow_token_info=VALUES(borrow_token_info),
		state=VALUES(state),
		sp_coin=VALUES(sp_coin),
		jp_coin=VALUES(jp_coin),
		auto_liquidate_threshold=VALUES(auto_liquidate_threshold),
		updated_at=VALUES(updated_at)`,
		pool.Key.ChainID, pool.Key.PoolID, pool.SettleTime, pool.EndTime, pool.InterestRate, pool.MaxSupply,
		pool.LendSupply, pool.BorrowSupply, pool.MortgageRate, string(lendToken), string(borrowToken),
		string(pool.State), pool.SPCoin, pool.JPCoin, pool.AutoLiquidateThreshold, now, now,
	)
	return err
}

func (s *MySQLStore) UpsertPoolData(ctx context.Context, data PoolData) error {
	now := s.now().UTC()
	_, err := s.db.ExecContext(ctx, `INSERT INTO pooldata (
		chain_id, pool_id, settle_amount_lend, settle_amount_borrow,
		finish_amount_lend, finish_amount_borrow,
		liquidation_amount_lend, liquidation_amount_borrow, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON DUPLICATE KEY UPDATE
		settle_amount_lend=VALUES(settle_amount_lend),
		settle_amount_borrow=VALUES(settle_amount_borrow),
		finish_amount_lend=VALUES(finish_amount_lend),
		finish_amount_borrow=VALUES(finish_amount_borrow),
		liquidation_amount_lend=VALUES(liquidation_amount_lend),
		liquidation_amount_borrow=VALUES(liquidation_amount_borrow),
		updated_at=VALUES(updated_at)`,
		data.Key.ChainID, data.Key.PoolID, data.SettleAmountLend, data.SettleAmountBorrow,
		data.FinishAmountLend, data.FinishAmountBorrow, data.LiquidationAmountLend,
		data.LiquidationAmountBorrow, now, now,
	)
	return err
}

func (s *MySQLStore) GetPoolBase(ctx context.Context, key PoolKey) (PoolBase, error) {
	row := s.db.QueryRowContext(ctx, `SELECT chain_id, pool_id, settle_time, end_time, interest_rate,
		max_supply, lend_supply, borrow_supply, mortgage_rate, lend_token_info, borrow_token_info,
		state, sp_coin, jp_coin, auto_liquidate_threshold, created_at, updated_at
		FROM poolbases WHERE chain_id=? AND pool_id=?`, key.ChainID, key.PoolID)
	return scanPoolBase(row)
}

func (s *MySQLStore) GetPoolData(ctx context.Context, key PoolKey) (PoolData, error) {
	row := s.db.QueryRowContext(ctx, `SELECT chain_id, pool_id, settle_amount_lend, settle_amount_borrow,
		finish_amount_lend, finish_amount_borrow, liquidation_amount_lend, liquidation_amount_borrow,
		created_at, updated_at FROM pooldata WHERE chain_id=? AND pool_id=?`, key.ChainID, key.PoolID)
	return scanPoolData(row)
}

func (s *MySQLStore) ListPoolBases(ctx context.Context, chainID string) ([]PoolBase, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT chain_id, pool_id, settle_time, end_time, interest_rate,
		max_supply, lend_supply, borrow_supply, mortgage_rate, lend_token_info, borrow_token_info,
		state, sp_coin, jp_coin, auto_liquidate_threshold, created_at, updated_at
		FROM poolbases WHERE chain_id=? ORDER BY pool_id ASC`, chainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pools := make([]PoolBase, 0)
	for rows.Next() {
		pool, err := scanPoolBase(rows)
		if err != nil {
			return nil, err
		}
		pools = append(pools, pool)
	}
	return pools, rows.Err()
}

func (s *MySQLStore) ListPoolData(ctx context.Context, chainID string) ([]PoolData, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT chain_id, pool_id, settle_amount_lend, settle_amount_borrow,
		finish_amount_lend, finish_amount_borrow, liquidation_amount_lend, liquidation_amount_borrow,
		created_at, updated_at FROM pooldata WHERE chain_id=? ORDER BY pool_id ASC`, chainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pools := make([]PoolData, 0)
	for rows.Next() {
		data, err := scanPoolData(rows)
		if err != nil {
			return nil, err
		}
		pools = append(pools, data)
	}
	return pools, rows.Err()
}

func (s *MySQLStore) UpsertToken(ctx context.Context, token TokenInfo) error {
	now := s.now().UTC()
	_, err := s.db.ExecContext(ctx, `INSERT INTO token_info (
		chain_id, token, symbol, logo, price, decimals, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	ON DUPLICATE KEY UPDATE
		symbol=VALUES(symbol),
		logo=VALUES(logo),
		price=VALUES(price),
		decimals=VALUES(decimals),
		updated_at=VALUES(updated_at)`,
		token.Key.ChainID, token.Key.Address, token.Symbol, token.LogoURL, token.Price,
		token.Decimals, now, now,
	)
	return err
}

func (s *MySQLStore) GetToken(ctx context.Context, key TokenKey) (TokenInfo, error) {
	row := s.db.QueryRowContext(ctx, `SELECT chain_id, token, symbol, logo, price, decimals, created_at, updated_at
		FROM token_info WHERE chain_id=? AND token=?`, key.ChainID, key.Address)
	return scanToken(row)
}

func (s *MySQLStore) ListTokens(ctx context.Context, chainID string) ([]TokenInfo, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT chain_id, token, symbol, logo, price, decimals, created_at, updated_at
		FROM token_info WHERE chain_id=? ORDER BY symbol ASC`, chainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tokens := make([]TokenInfo, 0)
	for rows.Next() {
		token, err := scanToken(rows)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}
	return tokens, rows.Err()
}

func (s *MySQLStore) Save(ctx context.Context, cfg multisig.Config) error {
	now := s.now().UTC()
	accounts, err := json.Marshal(cfg.MultiSignAccount)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `INSERT INTO multi_sign (
		chain_id, sp_name, sp_token, jp_name, jp_token, sp_address, jp_address,
		sp_hash, jp_hash, multi_sign_account, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON DUPLICATE KEY UPDATE
		sp_name=VALUES(sp_name),
		sp_token=VALUES(sp_token),
		jp_name=VALUES(jp_name),
		jp_token=VALUES(jp_token),
		sp_address=VALUES(sp_address),
		jp_address=VALUES(jp_address),
		sp_hash=VALUES(sp_hash),
		jp_hash=VALUES(jp_hash),
		multi_sign_account=VALUES(multi_sign_account),
		updated_at=VALUES(updated_at)`,
		cfg.ChainID, cfg.SPName, cfg.SPToken, cfg.JPName, cfg.JPToken, cfg.SPAddress, cfg.JPAddress,
		cfg.SPHash, cfg.JPHash, string(accounts), now, now,
	)
	return err
}

func (s *MySQLStore) Get(ctx context.Context, chainID string) (multisig.Config, error) {
	row := s.db.QueryRowContext(ctx, `SELECT chain_id, sp_name, sp_token, jp_name, jp_token, sp_address,
		jp_address, sp_hash, jp_hash, multi_sign_account, created_at, updated_at
		FROM multi_sign WHERE chain_id=?`, chainID)

	cfg := multisig.Config{}
	accountsJSON := ""
	err := row.Scan(&cfg.ChainID, &cfg.SPName, &cfg.SPToken, &cfg.JPName, &cfg.JPToken,
		&cfg.SPAddress, &cfg.JPAddress, &cfg.SPHash, &cfg.JPHash, &accountsJSON,
		&cfg.CreatedAt, &cfg.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return multisig.Config{}, multisig.ErrNotFound
	}
	if err != nil {
		return multisig.Config{}, err
	}
	if err := json.Unmarshal([]byte(accountsJSON), &cfg.MultiSignAccount); err != nil {
		return multisig.Config{}, err
	}
	return cfg, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanPoolBase(row rowScanner) (PoolBase, error) {
	pool := PoolBase{}
	lendTokenJSON := ""
	borrowTokenJSON := ""
	err := row.Scan(&pool.Key.ChainID, &pool.Key.PoolID, &pool.SettleTime, &pool.EndTime,
		&pool.InterestRate, &pool.MaxSupply, &pool.LendSupply, &pool.BorrowSupply,
		&pool.MortgageRate, &lendTokenJSON, &borrowTokenJSON, &pool.State, &pool.SPCoin,
		&pool.JPCoin, &pool.AutoLiquidateThreshold, &pool.CreatedAt, &pool.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return PoolBase{}, ErrNotFound
	}
	if err != nil {
		return PoolBase{}, err
	}
	if err := json.Unmarshal([]byte(lendTokenJSON), &pool.LendToken); err != nil {
		return PoolBase{}, err
	}
	if err := json.Unmarshal([]byte(borrowTokenJSON), &pool.BorrowToken); err != nil {
		return PoolBase{}, err
	}
	return pool, nil
}

func scanPoolData(row rowScanner) (PoolData, error) {
	data := PoolData{}
	err := row.Scan(&data.Key.ChainID, &data.Key.PoolID, &data.SettleAmountLend,
		&data.SettleAmountBorrow, &data.FinishAmountLend, &data.FinishAmountBorrow,
		&data.LiquidationAmountLend, &data.LiquidationAmountBorrow, &data.CreatedAt,
		&data.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return PoolData{}, ErrNotFound
	}
	return data, err
}

func scanToken(row rowScanner) (TokenInfo, error) {
	token := TokenInfo{}
	err := row.Scan(&token.Key.ChainID, &token.Key.Address, &token.Symbol, &token.LogoURL,
		&token.Price, &token.Decimals, &token.CreatedAt, &token.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return TokenInfo{}, ErrNotFound
	}
	return token, err
}
