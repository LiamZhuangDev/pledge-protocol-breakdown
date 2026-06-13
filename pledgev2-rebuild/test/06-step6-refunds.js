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

describe("Step 6: refund unmatched deposits", function () {
  const INTEREST_RATE = 1_000_000;
  const MAX_SUPPLY = ethers.parseEther("100000");
  const MORTGAGE_RATE = 200_000_000;
  const AUTO_LIQUIDATE_THRESHOLD = 20_000_000;
  const USDT_PRICE = 100_000_000;
  const WBTC_PRICE = 5_000_000_000_000;

  let owner;
  let alice;
  let bob;
  let carol;
  let feeRecipient;
  let oracle;
  let lendToken;
  let borrowToken;
  let spToken;
  let jpToken;
  let pool;

  beforeEach(async function () {
    [owner, alice, bob, carol, feeRecipient] = await ethers.getSigners();

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
    await lendToken.mint(carol.address, ethers.parseEther("100000"));
    await borrowToken.mint(bob.address, ethers.parseEther("10"));
    await borrowToken.mint(carol.address, ethers.parseEther("10"));

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

  async function createPoolAndApproveAll() {
    await pool.createPool(await buildCreateParams());
    const poolAddress = await pool.getAddress();

    for (const account of [alice, carol]) {
      await lendToken.connect(account).approve(poolAddress, ethers.parseEther("100000"));
    }
    for (const account of [bob, carol]) {
      await borrowToken.connect(account).approve(poolAddress, ethers.parseEther("10"));
    }
  }

  async function moveToSettleAndSettle() {
    await ethers.provider.send("evm_increaseTime", [3601]);
    await ethers.provider.send("evm_mine");
    await pool.settle(0);
  }

  it("refunds unmatched lender stablecoin after collateral-limited settlement", async function () {
    await createPoolAndApproveAll();

    await pool.connect(alice).depositLend(0, ethers.parseEther("100000"));
    await pool.connect(bob).depositBorrow(0, ethers.parseEther("2"));
    await moveToSettleAndSettle();

    await pool.connect(alice).refundLend(0);

    const aliceInfo = await pool.userLendInfo(alice.address, 0);

    assert.equal(aliceInfo.refundAmount, ethers.parseEther("50000"));
    assert.equal(aliceInfo.hasRefunded, true);
    assert.equal(await lendToken.balanceOf(alice.address), ethers.parseEther("50000"));
    assert.equal(await lendToken.balanceOf(await pool.getAddress()), ethers.parseEther("50000"));
  });

  it("refunds unmatched lender stablecoin proportionally across lenders", async function () {
    await createPoolAndApproveAll();

    await pool.connect(alice).depositLend(0, ethers.parseEther("60000"));
    await pool.connect(carol).depositLend(0, ethers.parseEther("40000"));
    await pool.connect(bob).depositBorrow(0, ethers.parseEther("2"));
    await moveToSettleAndSettle();

    await pool.connect(alice).refundLend(0);
    await pool.connect(carol).refundLend(0);

    const aliceInfo = await pool.userLendInfo(alice.address, 0);
    const carolInfo = await pool.userLendInfo(carol.address, 0);

    assert.equal(aliceInfo.refundAmount, ethers.parseEther("30000"));
    assert.equal(carolInfo.refundAmount, ethers.parseEther("20000"));
  });

  it("refunds unmatched borrower collateral after lender-demand-limited settlement", async function () {
    await createPoolAndApproveAll();

    await pool.connect(alice).depositLend(0, ethers.parseEther("25000"));
    await pool.connect(bob).depositBorrow(0, ethers.parseEther("2"));
    await moveToSettleAndSettle();

    await pool.connect(bob).refundBorrow(0);

    const bobInfo = await pool.userBorrowInfo(bob.address, 0);

    assert.equal(bobInfo.refundAmount, ethers.parseEther("1"));
    assert.equal(bobInfo.hasRefunded, true);
    assert.equal(await borrowToken.balanceOf(bob.address), ethers.parseEther("9"));
    assert.equal(await borrowToken.balanceOf(await pool.getAddress()), ethers.parseEther("1"));
  });

  it("refunds unmatched borrower collateral proportionally across borrowers", async function () {
    await createPoolAndApproveAll();

    await pool.connect(alice).depositLend(0, ethers.parseEther("25000"));
    await pool.setMinBorrowAmount(ethers.parseEther("0.1"));
    await pool.connect(bob).depositBorrow(0, ethers.parseEther("1.2"));
    await pool.connect(carol).depositBorrow(0, ethers.parseEther("0.8"));
    await moveToSettleAndSettle();

    await pool.connect(bob).refundBorrow(0);
    await pool.connect(carol).refundBorrow(0);

    const bobInfo = await pool.userBorrowInfo(bob.address, 0);
    const carolInfo = await pool.userBorrowInfo(carol.address, 0);

    assert.equal(bobInfo.refundAmount, ethers.parseEther("0.6"));
    assert.equal(carolInfo.refundAmount, ethers.parseEther("0.4"));
  });

  it("rejects duplicate refunds, missing stakes, and no-refund sides", async function () {
    await createPoolAndApproveAll();

    await pool.connect(alice).depositLend(0, ethers.parseEther("25000"));
    await pool.connect(bob).depositBorrow(0, ethers.parseEther("2"));
    await moveToSettleAndSettle();

    await expectRevert(pool.connect(alice).refundLend(0), "LearningPledgePool: no lend refund");
    await expectRevert(pool.connect(carol).refundBorrow(0), "LearningPledgePool: no borrow stake");

    await pool.connect(bob).refundBorrow(0);
    await expectRevert(pool.connect(bob).refundBorrow(0), "LearningPledgePool: borrow already refunded");
  });

  it("rejects refunds before settlement, after UNDONE settlement, or while paused", async function () {
    await createPoolAndApproveAll();

    await pool.connect(alice).depositLend(0, ethers.parseEther("100000"));
    await pool.connect(bob).depositBorrow(0, ethers.parseEther("2"));

    await expectRevert(pool.connect(alice).refundLend(0), "LearningPledgePool: pool not execution");

    await moveToSettleAndSettle();
    await pool.setPause(true);
    await expectRevert(pool.connect(alice).refundLend(0), "LearningPledgePool: paused");

    const LearningPledgePool = await ethers.getContractFactory("LearningPledgePool");
    const undonePool = await LearningPledgePool.deploy(await oracle.getAddress(), feeRecipient.address);
    await undonePool.waitForDeployment();
    await undonePool.createPool(await buildCreateParams());

    const undonePoolAddress = await undonePool.getAddress();
    await lendToken.connect(carol).approve(undonePoolAddress, ethers.parseEther("1000"));
    await undonePool.connect(carol).depositLend(0, ethers.parseEther("1000"));
    await ethers.provider.send("evm_increaseTime", [3601]);
    await ethers.provider.send("evm_mine");
    await undonePool.settle(0);

    await expectRevert(undonePool.connect(carol).refundLend(0), "LearningPledgePool: pool not execution");
  });
});
