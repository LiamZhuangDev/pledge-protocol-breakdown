package store

import "context"

func SeedDemoData(ctx context.Context, repo Repository) error {
	busd := TokenInfo{
		Key:      TokenKey{ChainID: "97", Address: "0xbusd"},
		Symbol:   "BUSD",
		LogoURL:  "/storage/img/BUSD.png",
		Price:    "100000000",
		Decimals: 18,
	}
	btc := TokenInfo{
		Key:      TokenKey{ChainID: "97", Address: "0xbtc"},
		Symbol:   "BTC",
		LogoURL:  "/storage/img/BTC.png",
		Price:    "4200000000000",
		Decimals: 18,
	}

	for _, token := range []TokenInfo{busd, btc} {
		if err := repo.UpsertToken(ctx, token); err != nil {
			return err
		}
	}

	key := PoolKey{ChainID: "97", PoolID: 1}
	if err := repo.UpsertPoolBase(ctx, PoolBase{
		Key:          key,
		SettleTime:   "1767225600",
		EndTime:      "1767830400",
		InterestRate: "10000000",
		MaxSupply:    "1000000000000000000000",
		LendSupply:   "500000000000000000000",
		BorrowSupply: "12500000000000000",
		MortgageRate: "200000000",
		LendToken: TokenSnapshot{
			Address:  busd.Key.Address,
			Symbol:   busd.Symbol,
			LogoURL:  busd.LogoURL,
			Price:    busd.Price,
			Fee:      "250000",
			Decimals: busd.Decimals,
		},
		BorrowToken: TokenSnapshot{
			Address:  btc.Key.Address,
			Symbol:   btc.Symbol,
			LogoURL:  btc.LogoURL,
			Price:    btc.Price,
			Fee:      "250000",
			Decimals: btc.Decimals,
		},
		State:                  PoolStateMatching,
		SPCoin:                 "0xsp",
		JPCoin:                 "0xjp",
		AutoLiquidateThreshold: "20000000",
	}); err != nil {
		return err
	}

	return repo.UpsertPoolData(ctx, PoolData{
		Key:                     key,
		SettleAmountLend:        "0",
		SettleAmountBorrow:      "0",
		FinishAmountLend:        "0",
		FinishAmountBorrow:      "0",
		LiquidationAmountLend:   "0",
		LiquidationAmountBorrow: "0",
	})
}
