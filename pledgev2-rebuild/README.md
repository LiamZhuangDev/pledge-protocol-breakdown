# pledgev2 rebuild

This is a clean Hardhat project for rebuilding the `pledgev2` contracts from scratch for learning.

The original folders are left alone. Each rebuild checkpoint should add only the contracts and tests needed for that step.

## Step 1

Contracts:

- `contracts/token/LearningDebtToken.sol`
- `contracts/oracle/LearningMockOracle.sol`

Run:

```bash
cd pledgev2-rebuild
npm install
npm run test:step1
```

Learning goal:

- Understand SP/JP-style receipt tokens as mintable/burnable ERC20 claim tokens.
- Understand a simple owner-controlled mock oracle before using price math inside a lending pool.

## Step 2

Contracts:

- `contracts/pool/LearningPledgePool.sol`

Run:

```bash
cd pledgev2-rebuild
npm run test:step2
```

Learning goal:

- Store the fixed-term pool configuration.
- Understand the pool lifecycle enum before adding deposits.
- Keep admin-only pool creation separate from user-facing lending and borrowing flows.

## Step 3

Contract changes:

- `LearningPledgePool.depositLend`
- `LearningPledgePool.userLendInfo`
- `LearningPledgePool.minLendAmount`

Run:

```bash
cd pledgev2-rebuild
npm run test:step3
```

Learning goal:

- Move lender funds into the pool with ERC20 `approve` + `transferFrom`.
- Track lender stake amount separately from total pool lend supply.
- Enforce the first user-facing gates: paused state, settlement time, min amount, and max pool capacity.

## Step 4

Contract changes:

- `LearningPledgePool.depositBorrow`
- `LearningPledgePool.userBorrowInfo`
- `LearningPledgePool.minBorrowAmount`

Run:

```bash
cd pledgev2-rebuild
npm run test:step4
```

Learning goal:

- Move borrower collateral into the pool with ERC20 `approve` + `transferFrom`.
- Track borrower collateral separately from lender stablecoin deposits.
- Understand that borrowers do not receive the loan yet; matched loan payout happens after settlement.

## Step 5

Contract changes:

- `LearningPledgePool.PoolDataInfo`
- `LearningPledgePool.getPoolData`
- `LearningPledgePool.settle`

Run:

```bash
cd pledgev2-rebuild
npm run test:step5
```

Learning goal:

- Convert borrower collateral value into lender-token value using oracle prices.
- Apply the mortgage/collateralization rate to find the maximum matched loan amount.
- Move pools from `MATCH` to `EXECUTION`, or to `UNDONE` when one side is empty.

## Step 6

Contract changes:

- `LearningPledgePool.refundLend`
- `LearningPledgePool.refundBorrow`

Run:

```bash
cd pledgev2-rebuild
npm run test:step6
```

Learning goal:

- Return unmatched lender stablecoin when borrower collateral cannot support the full lend supply.
- Return unmatched borrower collateral when lender demand does not need all collateral.
- Keep refund accounting separate from the later SP/JP claim flow.

## Step 7

Contract changes:

- `LearningPledgePool.claimLend`
- `LearningPledgePool.claimBorrow`

Run:

```bash
cd pledgev2-rebuild
npm run test:step7
```

Learning goal:

- Mint SP tokens to lenders for their matched stablecoin contribution.
- Mint JP tokens to borrowers for their matched collateral contribution.
- Send borrowers their matched stablecoin loan only after settlement.
