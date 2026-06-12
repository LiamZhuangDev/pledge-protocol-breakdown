const assert = require("assert/strict");
const { ethers } = require("hardhat");

async function expectRevert(action, message) {
  try {
    await action;
  } catch (error) {
    assert.ok(
      error.message.includes(message),
      `Expected revert message "${message}", got "${error.message}"`
    );
    return;
  }

  assert.fail(`Expected transaction to revert with "${message}"`);
}

describe("Step 2: create pool", function () {
  const INTEREST_RATE = 1_000_000;
  const MAX_SUPPLY = ethers.parseEther("100000");
  const MORTGAGE_RATE = 200_000_000;
  const AUTO_LIQUIDATE_THRESHOLD = 20_000_000;

  let owner;
  let alice;
  let feeRecipient;
  let lendToken;
  let borrowToken;
  let oracle;
  let spToken;
  let jpToken;
  let pool;

  beforeEach(async function () {
    [owner, alice, feeRecipient, lendToken, borrowToken] = await ethers.getSigners();

    const LearningMockOracle = await ethers.getContractFactory("LearningMockOracle");
    oracle = await LearningMockOracle.deploy();
    await oracle.waitForDeployment();

    const LearningDebtToken = await ethers.getContractFactory("LearningDebtToken");
    spToken = await LearningDebtToken.deploy("Senior Pool BUSD", "spBUSD");
    jpToken = await LearningDebtToken.deploy("Junior Pool BTC", "jpBTC");
    await spToken.waitForDeployment();
    await jpToken.waitForDeployment();

    const LearningPledgePool = await ethers.getContractFactory("LearningPledgePool");
    pool = await LearningPledgePool.deploy(await oracle.getAddress(), feeRecipient.address);
    await pool.waitForDeployment();
  });

  async function buildCreateParams(overrides = {}) {
    const latestBlock = await ethers.provider.getBlock("latest");
    const settleTime = latestBlock.timestamp + 3600;
    const params = {
      settleTime,
      endTime: settleTime + 7 * 24 * 60 * 60,
      interestRate: INTEREST_RATE,
      maxSupply: MAX_SUPPLY,
      mortgageRate: MORTGAGE_RATE,
      lendToken: lendToken.address,
      borrowToken: borrowToken.address,
      spToken: await spToken.getAddress(),
      jpToken: await jpToken.getAddress(),
      autoLiquidateThreshold: AUTO_LIQUIDATE_THRESHOLD
    };

    return { ...params, ...overrides };
  }

  it("deploys with owner, oracle, fee address, and no pools", async function () {
    assert.equal(await pool.owner(), owner.address);
    assert.equal(await pool.oracle(), await oracle.getAddress());
    assert.equal(await pool.feeAddress(), feeRecipient.address);
    assert.equal(await pool.globalPaused(), false);
    assert.equal(await pool.poolLength(), 0n);
  });

  it("creates a pool in MATCH state", async function () {
    const params = await buildCreateParams();

    await pool.createPool(params);

    assert.equal(await pool.poolLength(), 1n);
    assert.equal(await pool.getPoolState(0), 0n);
    assert.equal(await pool.isBeforeSettle(0), true);

    const createdPool = await pool.getPool(0);
    assert.equal(createdPool.settleTime, BigInt(params.settleTime));
    assert.equal(createdPool.endTime, BigInt(params.endTime));
    assert.equal(createdPool.interestRate, BigInt(params.interestRate));
    assert.equal(createdPool.maxSupply, params.maxSupply);
    assert.equal(createdPool.lendSupply, 0n);
    assert.equal(createdPool.borrowSupply, 0n);
    assert.equal(createdPool.mortgageRate, BigInt(params.mortgageRate));
    assert.equal(createdPool.lendToken, params.lendToken);
    assert.equal(createdPool.borrowToken, params.borrowToken);
    assert.equal(createdPool.state, 0n);
    assert.equal(createdPool.spToken, params.spToken);
    assert.equal(createdPool.jpToken, params.jpToken);
    assert.equal(createdPool.autoLiquidateThreshold, BigInt(params.autoLiquidateThreshold));
  });

  it("creates multiple pools with increasing ids", async function () {
    await pool.createPool(await buildCreateParams());
    await pool.createPool(await buildCreateParams({ maxSupply: ethers.parseEther("50000") }));

    assert.equal(await pool.poolLength(), 2n);

    const secondPool = await pool.getPool(1);
    assert.equal(secondPool.maxSupply, ethers.parseEther("50000"));
  });

  it("blocks non-owner pool creation and admin updates", async function () {
    await expectRevert(
      pool.connect(alice).createPool(await buildCreateParams()),
      "LearningPledgePool: caller is not owner"
    );
    await expectRevert(
      pool.connect(alice).setPause(true),
      "LearningPledgePool: caller is not owner"
    );
    await expectRevert(
      pool.connect(alice).setFeeAddress(alice.address),
      "LearningPledgePool: caller is not owner"
    );
  });

  it("validates required create pool fields", async function () {
    const params = await buildCreateParams();

    await expectRevert(
      pool.createPool({ ...params, settleTime: 1 }),
      "LearningPledgePool: settle time not future"
    );
    await expectRevert(
      pool.createPool({ ...params, endTime: params.settleTime }),
      "LearningPledgePool: end before settle"
    );
    await expectRevert(
      pool.createPool({ ...params, maxSupply: 0 }),
      "LearningPledgePool: zero max supply"
    );
    await expectRevert(
      pool.createPool({ ...params, interestRate: 0 }),
      "LearningPledgePool: zero interest rate"
    );
    await expectRevert(
      pool.createPool({ ...params, mortgageRate: 0 }),
      "LearningPledgePool: zero mortgage rate"
    );
    await expectRevert(
      pool.createPool({ ...params, lendToken: ethers.ZeroAddress }),
      "LearningPledgePool: zero lend token"
    );
    await expectRevert(
      pool.createPool({ ...params, borrowToken: ethers.ZeroAddress }),
      "LearningPledgePool: zero borrow token"
    );
    await expectRevert(
      pool.createPool({ ...params, borrowToken: params.lendToken }),
      "LearningPledgePool: same pool tokens"
    );
    await expectRevert(
      pool.createPool({ ...params, spToken: ethers.ZeroAddress }),
      "LearningPledgePool: zero sp token"
    );
    await expectRevert(
      pool.createPool({ ...params, jpToken: ethers.ZeroAddress }),
      "LearningPledgePool: zero jp token"
    );
    await expectRevert(
      pool.createPool({ ...params, jpToken: params.spToken }),
      "LearningPledgePool: same debt tokens"
    );
  });

  it("validates pool reads and simple admin setters", async function () {
    await expectRevert(pool.getPool(0), "LearningPledgePool: pool not found");
    await expectRevert(pool.getPoolState(0), "LearningPledgePool: pool not found");
    await expectRevert(pool.isBeforeSettle(0), "LearningPledgePool: pool not found");

    await pool.setPause(true);
    assert.equal(await pool.globalPaused(), true);

    await pool.setFeeAddress(alice.address);
    assert.equal(await pool.feeAddress(), alice.address);
  });
});
