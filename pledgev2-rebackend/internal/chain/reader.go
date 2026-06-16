package chain

import "context"

type Reader interface {
	PoolLength(ctx context.Context, chainID string) (int64, error)
	PoolBaseInfo(ctx context.Context, chainID string, contractIndex int64) (ContractPoolBase, error)
	PoolDataInfo(ctx context.Context, chainID string, contractIndex int64) (ContractPoolData, error)
	TokenInfo(ctx context.Context, chainID string, tokenAddress string) (ContractToken, error)
}

type ContractPoolBase struct {
	SettleTime             string
	EndTime                string
	InterestRate           string
	MaxSupply              string
	LendSupply             string
	BorrowSupply           string
	MortgageRate           string
	LendTokenAddress       string
	BorrowTokenAddress     string
	State                  string
	SPCoin                 string
	JPCoin                 string
	AutoLiquidateThreshold string
}

type ContractPoolData struct {
	SettleAmountLend        string
	SettleAmountBorrow      string
	FinishAmountLend        string
	FinishAmountBorrow      string
	LiquidationAmountLend   string
	LiquidationAmountBorrow string
}

type ContractToken struct {
	Address  string
	Symbol   string
	LogoURL  string
	Price    string
	Fee      string
	Decimals int
}
