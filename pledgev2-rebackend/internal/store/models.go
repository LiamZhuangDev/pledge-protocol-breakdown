package store

import "time"

type PoolKey struct {
	ChainID string
	PoolID  int64
}

type TokenKey struct {
	ChainID string
	Address string
}

type PoolState string

const (
	PoolStatePending     PoolState = "0"
	PoolStateMatching    PoolState = "1"
	PoolStateExecution   PoolState = "2"
	PoolStateLiquidating PoolState = "3"
	PoolStateUndone      PoolState = "4"
)

type TokenSnapshot struct {
	Address  string `json:"address"`
	Symbol   string `json:"symbol"`
	LogoURL  string `json:"logoUrl"`
	Price    string `json:"price"`
	Fee      string `json:"fee"`
	Decimals int    `json:"decimals"`
}

type PoolBase struct {
	Key                    PoolKey       `json:"key"`
	SettleTime             string        `json:"settleTime"`
	EndTime                string        `json:"endTime"`
	InterestRate           string        `json:"interestRate"`
	MaxSupply              string        `json:"maxSupply"`
	LendSupply             string        `json:"lendSupply"`
	BorrowSupply           string        `json:"borrowSupply"`
	MortgageRate           string        `json:"mortgageRate"`
	LendToken              TokenSnapshot `json:"lendToken"`
	BorrowToken            TokenSnapshot `json:"borrowToken"`
	State                  PoolState     `json:"state"`
	SPCoin                 string        `json:"spCoin"`
	JPCoin                 string        `json:"jpCoin"`
	AutoLiquidateThreshold string        `json:"autoLiquidateThreshold"`
	CreatedAt              time.Time     `json:"createdAt"`
	UpdatedAt              time.Time     `json:"updatedAt"`
}

type PoolData struct {
	Key                     PoolKey   `json:"key"`
	SettleAmountLend        string    `json:"settleAmountLend"`
	SettleAmountBorrow      string    `json:"settleAmountBorrow"`
	FinishAmountLend        string    `json:"finishAmountLend"`
	FinishAmountBorrow      string    `json:"finishAmountBorrow"`
	LiquidationAmountLend   string    `json:"liquidationAmountLend"`
	LiquidationAmountBorrow string    `json:"liquidationAmountBorrow"`
	CreatedAt               time.Time `json:"createdAt"`
	UpdatedAt               time.Time `json:"updatedAt"`
}

type TokenInfo struct {
	Key       TokenKey  `json:"key"`
	Symbol    string    `json:"symbol"`
	LogoURL   string    `json:"logoUrl"`
	Price     string    `json:"price"`
	Decimals  int       `json:"decimals"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type PoolSnapshot struct {
	Base PoolBase `json:"base"`
	Data PoolData `json:"data"`
}
