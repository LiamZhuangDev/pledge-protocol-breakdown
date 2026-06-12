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
