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

describe("Step 5: settle pool", function () {
  const INTEREST_RATE = 1_000_000;
  const MAX_SUPPLY = ethers.parseEther("100000");
  const MORTGAGE_RATE = 200_000_000;
  const AUTO_LIQUIDATE_THRESHOLD = 20_000_000;
  const USDT_PRICE = 100_000_000;
  const WBTC_PRICE = 5_000_000_000_000;

  let owner;
  let alice;
  let bob;
  let feeRecipient;
  let oracle;
  let lendToken;
  let borrowToken;
  let spToken;
  let jpToken;
  let pool;

  beforeEach(async function () {
    [owner, alice, bob, feeRecipient] = await ethers.getSigners();

    const LearningMockOracle = await ethers.getContractFactory("LearningMockOracle");
    oracle = await LearningMockOracle.deploy();
    await oracle.waitForDeployment();

    const LearningDebtToken = await ethers.getContractFactory("LearningDebtToken");
    lendToken = await LearningDebtToken.deploy("Mock USDT", "mUSDT");
    borrowToken = await LearningDebtToken.deploy("Mock WBTC", "mWBTC");
    spToken = await LearningDebtToken.deploy("Senior Pool USDT", "spUSDT");
    jpToken = await LearningDebtToken.deploy("Junior Pool WBTC", "jpWBTC");
    await lendToken.waitForDeployment();
    await borrowToken.waitForDeployment();
    await spToken.waitForDeployment();
    await jpToken.waitForDeployment();

    await lendToken.addMinter(owner.address);
    await borrowToken.addMinter(owner.address);
    await lendToken.mint(alice.address, ethers.parseEther("100000"));
    await borrowToken.mint(bob.address, ethers.parseEther("10"));

    const LearningPledgePool = await ethers.getContractFactory("LearningPledgePool");
    pool = await LearningPledgePool.deploy(await oracle.getAddress(), feeRecipient.address);
    await pool.waitForDeployment();

    await oracle.setPrice(await lendToken.getAddress(), USDT_PRICE);
    await oracle.setPrice(await borrowToken.getAddress(), WBTC_PRICE);
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
      lendToken: await lendToken.getAddress(),
      borrowToken: await borrowToken.getAddress(),
      spToken: await spToken.getAddress(),
      jpToken: await jpToken.getAddress(),
      autoLiquidateThreshold: AUTO_LIQUIDATE_THRESHOLD
    };

    return { ...params, ...overrides };
  }

  async function createPoolAndApprove() {
    await pool.createPool(await buildCreateParams());
    const poolAddress = await pool.getAddress();
    await lendToken.connect(alice).approve(poolAddress, ethers.parseEther("100000"));
    await borrowToken.connect(bob).approve(poolAddress, ethers.parseEther("10"));
  }

  async function moveToSettleTime() {
    await ethers.provider.send("evm_increaseTime", [3601]);
    await ethers.provider.send("evm_mine");
  }

  it("settles to EXECUTION when borrower collateral limits lender demand", async function () {
    await createPoolAndApprove();

    await pool.connect(alice).depositLend(0, ethers.parseEther("100000"));
    await pool.connect(bob).depositBorrow(0, ethers.parseEther("2"));
    await moveToSettleTime();
    await pool.settle(0);

    const createdPool = await pool.getPool(0);
    const data = await pool.getPoolData(0);

    assert.equal(createdPool.state, 1n);
    assert.equal(data.settleAmountLend, ethers.parseEther("50000"));
    assert.equal(data.settleAmountBorrow, ethers.parseEther("2"));
  });

  it("settles to EXECUTION when lender demand limits borrower collateral", async function () {
    await createPoolAndApprove();

    await pool.connect(alice).depositLend(0, ethers.parseEther("25000"));
    await pool.connect(bob).depositBorrow(0, ethers.parseEther("2"));
    await moveToSettleTime();
    await pool.settle(0);

    const data = await pool.getPoolData(0);

    assert.equal(await pool.getPoolState(0), 1n);
    assert.equal(data.settleAmountLend, ethers.parseEther("25000"));
    assert.equal(data.settleAmountBorrow, ethers.parseEther("1"));
  });

  it("moves to UNDONE when either side is empty", async function () {
    await pool.createPool(await buildCreateParams());
    const poolAddress = await pool.getAddress();

    await lendToken.connect(alice).approve(poolAddress, ethers.parseEther("1000"));
    await pool.connect(alice).depositLend(0, ethers.parseEther("1000"));

    await moveToSettleTime();
    await pool.settle(0);

    const data = await pool.getPoolData(0);

    assert.equal(await pool.getPoolState(0), 4n);
    assert.equal(data.settleAmountLend, ethers.parseEther("1000"));
    assert.equal(data.settleAmountBorrow, 0n);
  });

  it("rejects settlement before settle time, by non-owner, or twice", async function () {
    await createPoolAndApprove();

    await pool.connect(alice).depositLend(0, ethers.parseEther("1000"));
    await pool.connect(bob).depositBorrow(0, ethers.parseEther("1"));

    await expectRevert(pool.settle(0), "LearningPledgePool: before settle time");

    await moveToSettleTime();
    await expectRevert(pool.connect(alice).settle(0), "LearningPledgePool: caller is not owner");

    await pool.settle(0);
    await expectRevert(pool.settle(0), "LearningPledgePool: pool not match");
  });

  it("rejects settlement when oracle prices are missing", async function () {
    await createPoolAndApprove();

    await pool.connect(alice).depositLend(0, ethers.parseEther("1000"));
    await pool.connect(bob).depositBorrow(0, ethers.parseEther("1"));
    await oracle.setPrice(await lendToken.getAddress(), 1);

    const NewOracle = await ethers.getContractFactory("LearningMockOracle");
    const emptyOracle = await NewOracle.deploy();
    await emptyOracle.waitForDeployment();

    const LearningPledgePool = await ethers.getContractFactory("LearningPledgePool");
    const secondPool = await LearningPledgePool.deploy(await emptyOracle.getAddress(), feeRecipient.address);
    await secondPool.waitForDeployment();
    await secondPool.createPool(await buildCreateParams());

    const secondPoolAddress = await secondPool.getAddress();
    await lendToken.connect(alice).approve(secondPoolAddress, ethers.parseEther("1000"));
    await borrowToken.connect(bob).approve(secondPoolAddress, ethers.parseEther("1"));
    await secondPool.connect(alice).depositLend(0, ethers.parseEther("1000"));
    await secondPool.connect(bob).depositBorrow(0, ethers.parseEther("1"));

    await moveToSettleTime();
    await expectRevert(secondPool.settle(0), "LearningPledgePool: missing lend price");
  });
});
