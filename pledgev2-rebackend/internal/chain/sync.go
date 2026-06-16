package chain

import (
	"context"
	"fmt"

	"pledgev2-rebackend/internal/store"
)

func SyncPools(ctx context.Context, reader Reader, repo store.Repository, chainID string) error {
	poolLength, err := reader.PoolLength(ctx, chainID)
	if err != nil {
		return fmt.Errorf("read pool length: %w", err)
	}

	for contractIndex := int64(0); contractIndex < poolLength; contractIndex++ {
		rawBase, err := reader.PoolBaseInfo(ctx, chainID, contractIndex)
		if err != nil {
			return fmt.Errorf("read pool base at index %d: %w", contractIndex, err)
		}

		rawData, err := reader.PoolDataInfo(ctx, chainID, contractIndex)
		if err != nil {
			return fmt.Errorf("read pool data at index %d: %w", contractIndex, err)
		}

		lendToken, err := reader.TokenInfo(ctx, chainID, rawBase.LendTokenAddress)
		if err != nil {
			return fmt.Errorf("read lend token for index %d: %w", contractIndex, err)
		}

		borrowToken, err := reader.TokenInfo(ctx, chainID, rawBase.BorrowTokenAddress)
		if err != nil {
			return fmt.Errorf("read borrow token for index %d: %w", contractIndex, err)
		}

		for _, token := range []ContractToken{lendToken, borrowToken} {
			if err := repo.UpsertToken(ctx, toTokenInfo(chainID, token)); err != nil {
				return fmt.Errorf("save token %s: %w", token.Address, err)
			}
		}

		poolID := contractIndex + 1
		key := store.PoolKey{ChainID: chainID, PoolID: poolID}
		if err := repo.UpsertPoolBase(ctx, toPoolBase(key, rawBase, lendToken, borrowToken)); err != nil {
			return fmt.Errorf("save pool base %d: %w", poolID, err)
		}
		if err := repo.UpsertPoolData(ctx, toPoolData(key, rawData)); err != nil {
			return fmt.Errorf("save pool data %d: %w", poolID, err)
		}
	}

	return nil
}

func toPoolBase(key store.PoolKey, raw ContractPoolBase, lendToken ContractToken, borrowToken ContractToken) store.PoolBase {
	return store.PoolBase{
		Key:                    key,
		SettleTime:             raw.SettleTime,
		EndTime:                raw.EndTime,
		InterestRate:           raw.InterestRate,
		MaxSupply:              raw.MaxSupply,
		LendSupply:             raw.LendSupply,
		BorrowSupply:           raw.BorrowSupply,
		MortgageRate:           raw.MortgageRate,
		LendToken:              toTokenSnapshot(lendToken),
		BorrowToken:            toTokenSnapshot(borrowToken),
		State:                  store.PoolState(raw.State),
		SPCoin:                 raw.SPCoin,
		JPCoin:                 raw.JPCoin,
		AutoLiquidateThreshold: raw.AutoLiquidateThreshold,
	}
}

func toPoolData(key store.PoolKey, raw ContractPoolData) store.PoolData {
	return store.PoolData{
		Key:                     key,
		SettleAmountLend:        raw.SettleAmountLend,
		SettleAmountBorrow:      raw.SettleAmountBorrow,
		FinishAmountLend:        raw.FinishAmountLend,
		FinishAmountBorrow:      raw.FinishAmountBorrow,
		LiquidationAmountLend:   raw.LiquidationAmountLend,
		LiquidationAmountBorrow: raw.LiquidationAmountBorrow,
	}
}

func toTokenInfo(chainID string, token ContractToken) store.TokenInfo {
	return store.TokenInfo{
		Key:      store.TokenKey{ChainID: chainID, Address: token.Address},
		Symbol:   token.Symbol,
		LogoURL:  token.LogoURL,
		Price:    token.Price,
		Decimals: token.Decimals,
	}
}

func toTokenSnapshot(token ContractToken) store.TokenSnapshot {
	return store.TokenSnapshot{
		Address:  token.Address,
		Symbol:   token.Symbol,
		LogoURL:  token.LogoURL,
		Price:    token.Price,
		Fee:      token.Fee,
		Decimals: token.Decimals,
	}
}
