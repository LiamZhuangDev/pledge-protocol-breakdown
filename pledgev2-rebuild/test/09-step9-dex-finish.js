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

describe("Step 9: DEX finish flow", function () {
  const INTEREST_RATE = 1_000_000;
  const MAX_SUPPLY = ethers.parseEther("100000");
  const MORTGAGE_RATE = 200_000_000;
  const AUTO_LIQUIDATE_THRESHOLD = 20_000_000;
  const USDT_PRICE = 100_000_000;
  const WBTC_PRICE = 5_000_000_000_000;
  const WBTC_TO_USDT_RATE = ethers.parseEther("50000");

  let owner;
  let alice;
  let bob;
  let feeRecipient;
  let oracle;
  let dexRouter;
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

    const LearningDexRouter = await ethers.getContractFactory("LearningDexRouter");
    dexRouter = await LearningDexRouter.deploy();
    await dexRouter.waitForDeployment();

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
    await lendToken.mint(await dexRouter.getAddress(), ethers.parseEther("100000"));
    await borrowToken.mint(bob.address, ethers.parseEther("10"));

    const LearningPledgePool = await ethers.getContractFactory("LearningPledgePool");
    pool = await LearningPledgePool.deploy(await oracle.getAddress(), feeRecipient.address);
    await pool.waitForDeployment();

    await pool.setDexRouter(await dexRouter.getAddress());
    await spToken.addMinter(await pool.getAddress());
    await jpToken.addMinter(await pool.getAddress());
    await oracle.setPrice(await lendToken.getAddress(), USDT_PRICE);
    await oracle.setPrice(await borrowToken.getAddress(), WBTC_PRICE);
    await dexRouter.setRate(await borrowToken.getAddress(), await lendToken.getAddress(), WBTC_TO_USDT_RATE);
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

  async function createPoolAndMoveToExecution() {
    await pool.createPool(await buildCreateParams());
    const poolAddress = await pool.getAddress();

    await lendToken.connect(alice).approve(poolAddress, ethers.parseEther("25000"));
    await borrowToken.connect(bob).approve(poolAddress, ethers.parseEther("2"));
    await pool.connect(alice).depositLend(0, ethers.parseEther("25000"));
    await pool.connect(bob).depositBorrow(0, ethers.parseEther("2"));

    await ethers.provider.send("evm_increaseTime", [3601]);
    await ethers.provider.send("evm_mine");
    await pool.settle(0);
  }

  async function moveToEnd() {
    await ethers.provider.send("evm_increaseTime", [7 * 24 * 60 * 60 + 1]);
    await ethers.provider.send("evm_mine");
  }

  it("finishes by swapping matched collateral through the DEX", async function () {
    await createPoolAndMoveToExecution();

    await pool.connect(bob).refundBorrow(0);
    await pool.connect(alice).claimLend(0);
    await pool.connect(bob).claimBorrow(0);

    await moveToEnd();

    const requiredRepayment = await pool.getRequiredRepayment(0);
    const collateralToSell = await dexRouter.getAmountIn(
      await borrowToken.getAddress(),
      await lendToken.getAddress(),
      requiredRepayment
    );

    await pool.finishWithDex(0, collateralToSell);

    const data = await pool.getPoolData(0);

    assert.equal(await pool.getPoolState(0), 2n);
    assert.equal(data.finishAmountLend, requiredRepayment);
    assert.equal(data.finishAmountBorrow, ethers.parseEther("1") - collateralToSell);
    assert.equal(await borrowToken.balanceOf(await dexRouter.getAddress()), collateralToSell);
    assert.equal(await lendToken.balanceOf(await pool.getAddress()), requiredRepayment);
  });

  it("lets SP and JP holders withdraw after DEX finish", async function () {
    await createPoolAndMoveToExecution();

    await pool.connect(bob).refundBorrow(0);
    await pool.connect(alice).claimLend(0);
    await pool.connect(bob).claimBorrow(0);

    await moveToEnd();

    const requiredRepayment = await pool.getRequiredRepayment(0);
    const collateralToSell = await dexRouter.getAmountIn(
      await borrowToken.getAddress(),
      await lendToken.getAddress(),
      requiredRepayment
    );

    await pool.finishWithDex(0, collateralToSell);
    await pool.connect(alice).withdrawLend(0, await spToken.balanceOf(alice.address));
    await pool.connect(bob).withdrawBorrow(0, await jpToken.balanceOf(bob.address));

    assert.equal(await lendToken.balanceOf(alice.address), ethers.parseEther("75000") + requiredRepayment);
    assert.equal(await borrowToken.balanceOf(bob.address), ethers.parseEther("9") + (ethers.parseEther("1") - collateralToSell));
  });

  it("rejects DEX finish without router, before end, or when slippage exceeds max collateral", async function () {
    await createPoolAndMoveToExecution();

    await expectRevert(pool.finishWithDex(0, ethers.parseEther("1")), "LearningPledgePool: before end time");

    await moveToEnd();

    const requiredRepayment = await pool.getRequiredRepayment(0);
    const collateralToSell = await dexRouter.getAmountIn(
      await borrowToken.getAddress(),
      await lendToken.getAddress(),
      requiredRepayment
    );

    await expectRevert(
      pool.finishWithDex(0, collateralToSell - 1n),
      "LearningPledgePool: dex slippage too high"
    );

    const LearningPledgePool = await ethers.getContractFactory("LearningPledgePool");
    const poolWithoutRouter = await LearningPledgePool.deploy(await oracle.getAddress(), feeRecipient.address);
    await poolWithoutRouter.waitForDeployment();
    await poolWithoutRouter.createPool(await buildCreateParams());

    const poolWithoutRouterAddress = await poolWithoutRouter.getAddress();
    await lendToken.connect(alice).approve(poolWithoutRouterAddress, ethers.parseEther("25000"));
    await borrowToken.connect(bob).approve(poolWithoutRouterAddress, ethers.parseEther("2"));
    await poolWithoutRouter.connect(alice).depositLend(0, ethers.parseEther("25000"));
    await poolWithoutRouter.connect(bob).depositBorrow(0, ethers.parseEther("2"));

    await ethers.provider.send("evm_increaseTime", [3601]);
    await ethers.provider.send("evm_mine");
    await poolWithoutRouter.settle(0);
    await moveToEnd();

    await expectRevert(
      poolWithoutRouter.finishWithDex(0, ethers.parseEther("1")),
      "LearningPledgePool: dex router not set"
    );
  });

  it("rejects DEX finish when collateral cannot cover required repayment", async function () {
    await createPoolAndMoveToExecution();
    await dexRouter.setRate(await borrowToken.getAddress(), await lendToken.getAddress(), ethers.parseEther("1000"));

    await moveToEnd();

    await expectRevert(
      pool.finishWithDex(0, ethers.parseEther("100")),
      "LearningPledgePool: insufficient collateral"
    );
  });
});
