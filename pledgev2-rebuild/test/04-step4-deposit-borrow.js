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

describe("Step 4: deposit borrow collateral", function () {
  const INTEREST_RATE = 1_000_000;
  const MAX_SUPPLY = ethers.parseEther("1000");
  const MORTGAGE_RATE = 200_000_000;
  const AUTO_LIQUIDATE_THRESHOLD = 20_000_000;

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

    await borrowToken.addMinter(owner.address);
    await borrowToken.mint(alice.address, ethers.parseEther("10"));
    await borrowToken.mint(bob.address, ethers.parseEther("10"));

    const LearningPledgePool = await ethers.getContractFactory("LearningPledgePool");
    pool = await LearningPledgePool.deploy(await oracle.getAddress(), feeRecipient.address);
    await pool.waitForDeployment();

    await pool.createPool(await buildCreateParams());
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

  it("accepts borrower collateral before settlement", async function () {
    const depositAmount = ethers.parseEther("2");
    const poolAddress = await pool.getAddress();

    await borrowToken.connect(alice).approve(poolAddress, depositAmount);
    await pool.connect(alice).depositBorrow(0, depositAmount);

    const createdPool = await pool.getPool(0);
    const aliceInfo = await pool.userBorrowInfo(alice.address, 0);

    assert.equal(createdPool.borrowSupply, depositAmount);
    assert.equal(createdPool.lendSupply, 0n);
    assert.equal(aliceInfo.stakeAmount, depositAmount);
    assert.equal(aliceInfo.refundAmount, 0n);
    assert.equal(aliceInfo.hasRefunded, false);
    assert.equal(aliceInfo.hasClaimed, false);
    assert.equal(await borrowToken.balanceOf(alice.address), ethers.parseEther("8"));
    assert.equal(await borrowToken.balanceOf(poolAddress), depositAmount);
  });

  it("aggregates multiple collateral deposits from one borrower", async function () {
    const poolAddress = await pool.getAddress();

    await borrowToken.connect(alice).approve(poolAddress, ethers.parseEther("5"));
    await pool.connect(alice).depositBorrow(0, ethers.parseEther("2"));
    await pool.connect(alice).depositBorrow(0, ethers.parseEther("3"));

    const createdPool = await pool.getPool(0);
    const aliceInfo = await pool.userBorrowInfo(alice.address, 0);

    assert.equal(createdPool.borrowSupply, ethers.parseEther("5"));
    assert.equal(aliceInfo.stakeAmount, ethers.parseEther("5"));
    assert.equal(await borrowToken.balanceOf(poolAddress), ethers.parseEther("5"));
  });

  it("tracks collateral deposits from different borrowers separately", async function () {
    const poolAddress = await pool.getAddress();

    await borrowToken.connect(alice).approve(poolAddress, ethers.parseEther("2"));
    await borrowToken.connect(bob).approve(poolAddress, ethers.parseEther("3"));

    await pool.connect(alice).depositBorrow(0, ethers.parseEther("2"));
    await pool.connect(bob).depositBorrow(0, ethers.parseEther("3"));

    const createdPool = await pool.getPool(0);
    const aliceInfo = await pool.userBorrowInfo(alice.address, 0);
    const bobInfo = await pool.userBorrowInfo(bob.address, 0);

    assert.equal(createdPool.borrowSupply, ethers.parseEther("5"));
    assert.equal(aliceInfo.stakeAmount, ethers.parseEther("2"));
    assert.equal(bobInfo.stakeAmount, ethers.parseEther("3"));
  });

  it("rejects collateral deposits that are too small", async function () {
    const poolAddress = await pool.getAddress();

    await borrowToken.connect(alice).approve(poolAddress, ethers.parseEther("0.9"));

    await expectRevert(
      pool.connect(alice).depositBorrow(0, ethers.parseEther("0.9")),
      "LearningPledgePool: borrow amount too small"
    );
  });

  it("rejects collateral deposits without enough approval", async function () {
    await expectRevert(
      pool.connect(alice).depositBorrow(0, ethers.parseEther("2")),
      "LearningDebtToken: insufficient allowance"
    );
  });

  it("rejects collateral deposits while paused", async function () {
    const poolAddress = await pool.getAddress();

    await borrowToken.connect(alice).approve(poolAddress, ethers.parseEther("2"));
    await pool.setPause(true);

    await expectRevert(
      pool.connect(alice).depositBorrow(0, ethers.parseEther("2")),
      "LearningPledgePool: paused"
    );
  });

  it("rejects collateral deposits after settlement time", async function () {
    const poolAddress = await pool.getAddress();

    await borrowToken.connect(alice).approve(poolAddress, ethers.parseEther("2"));
    await ethers.provider.send("evm_increaseTime", [3601]);
    await ethers.provider.send("evm_mine");

    await expectRevert(
      pool.connect(alice).depositBorrow(0, ethers.parseEther("2")),
      "LearningPledgePool: settle time passed"
    );
  });

  it("lets owner change the minimum borrow collateral amount", async function () {
    const poolAddress = await pool.getAddress();

    await pool.setMinBorrowAmount(ethers.parseEther("0.1"));
    await borrowToken.connect(alice).approve(poolAddress, ethers.parseEther("0.1"));
    await pool.connect(alice).depositBorrow(0, ethers.parseEther("0.1"));

    const aliceInfo = await pool.userBorrowInfo(alice.address, 0);
    assert.equal(aliceInfo.stakeAmount, ethers.parseEther("0.1"));
  });

  it("blocks non-owner minimum borrow amount updates and missing pools", async function () {
    await expectRevert(
      pool.connect(alice).setMinBorrowAmount(ethers.parseEther("0.1")),
      "LearningPledgePool: caller is not owner"
    );
    await expectRevert(
      pool.connect(alice).depositBorrow(1, ethers.parseEther("2")),
      "LearningPledgePool: pool not found"
    );
  });
});
