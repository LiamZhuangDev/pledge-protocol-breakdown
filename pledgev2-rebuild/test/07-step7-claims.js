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

describe("Step 7: claim SP and JP tokens", function () {
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

    await spToken.addMinter(await pool.getAddress());
    await jpToken.addMinter(await pool.getAddress());
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

  it("lets lenders claim SP tokens for their matched lend amount", async function () {
    await createPoolAndApproveAll();

    await pool.connect(alice).depositLend(0, ethers.parseEther("60000"));
    await pool.connect(carol).depositLend(0, ethers.parseEther("40000"));
    await pool.connect(bob).depositBorrow(0, ethers.parseEther("2"));
    await moveToSettleAndSettle();

    await pool.connect(alice).claimLend(0);
    await pool.connect(carol).claimLend(0);

    const aliceInfo = await pool.userLendInfo(alice.address, 0);
    const carolInfo = await pool.userLendInfo(carol.address, 0);

    assert.equal(await spToken.balanceOf(alice.address), ethers.parseEther("30000"));
    assert.equal(await spToken.balanceOf(carol.address), ethers.parseEther("20000"));
    assert.equal(await spToken.totalSupply(), ethers.parseEther("50000"));
    assert.equal(aliceInfo.hasClaimed, true);
    assert.equal(carolInfo.hasClaimed, true);
  });

  it("lets borrowers claim JP tokens and matched stablecoin loan", async function () {
    await createPoolAndApproveAll();

    await pool.connect(alice).depositLend(0, ethers.parseEther("25000"));
    await pool.connect(bob).depositBorrow(0, ethers.parseEther("1.2"));
    await pool.setMinBorrowAmount(ethers.parseEther("0.1"));
    await pool.connect(carol).depositBorrow(0, ethers.parseEther("0.8"));
    await moveToSettleAndSettle();

    await pool.connect(bob).claimBorrow(0);
    await pool.connect(carol).claimBorrow(0);

    const bobInfo = await pool.userBorrowInfo(bob.address, 0);
    const carolInfo = await pool.userBorrowInfo(carol.address, 0);

    assert.equal(await jpToken.balanceOf(bob.address), ethers.parseEther("0.6"));
    assert.equal(await jpToken.balanceOf(carol.address), ethers.parseEther("0.4"));
    assert.equal(await jpToken.totalSupply(), ethers.parseEther("1"));
    assert.equal(await lendToken.balanceOf(bob.address), ethers.parseEther("15000"));
    assert.equal(await lendToken.balanceOf(carol.address), ethers.parseEther("110000"));
    assert.equal(bobInfo.hasClaimed, true);
    assert.equal(carolInfo.hasClaimed, true);
  });

  it("allows refunds and claims to happen in either order", async function () {
    await createPoolAndApproveAll();

    await pool.connect(alice).depositLend(0, ethers.parseEther("100000"));
    await pool.connect(bob).depositBorrow(0, ethers.parseEther("2"));
    await moveToSettleAndSettle();

    await pool.connect(alice).refundLend(0);
    await pool.connect(alice).claimLend(0);
    await pool.connect(bob).claimBorrow(0);

    assert.equal(await lendToken.balanceOf(alice.address), ethers.parseEther("50000"));
    assert.equal(await lendToken.balanceOf(bob.address), ethers.parseEther("50000"));
    assert.equal(await spToken.balanceOf(alice.address), ethers.parseEther("50000"));
    assert.equal(await jpToken.balanceOf(bob.address), ethers.parseEther("2"));
  });

  it("rejects claims before settlement, missing stakes, duplicate claims, and paused claims", async function () {
    await createPoolAndApproveAll();

    await pool.connect(alice).depositLend(0, ethers.parseEther("25000"));
    await pool.connect(bob).depositBorrow(0, ethers.parseEther("2"));

    await expectRevert(pool.connect(alice).claimLend(0), "LearningPledgePool: pool not execution");
    await expectRevert(pool.connect(bob).claimBorrow(0), "LearningPledgePool: pool not execution");

    await moveToSettleAndSettle();

    await expectRevert(pool.connect(carol).claimLend(0), "LearningPledgePool: no lend stake");
    await expectRevert(pool.connect(carol).claimBorrow(0), "LearningPledgePool: no borrow stake");

    await pool.connect(alice).claimLend(0);
    await pool.connect(bob).claimBorrow(0);

    await expectRevert(pool.connect(alice).claimLend(0), "LearningPledgePool: lend already claimed");
    await expectRevert(pool.connect(bob).claimBorrow(0), "LearningPledgePool: borrow already claimed");

    await pool.setPause(true);
    await expectRevert(pool.connect(carol).claimLend(0), "LearningPledgePool: paused");
  });

  it("requires the pool to be a minter for SP and JP tokens", async function () {
    const LearningPledgePool = await ethers.getContractFactory("LearningPledgePool");
    const poolWithoutMinterRole = await LearningPledgePool.deploy(await oracle.getAddress(), feeRecipient.address);
    await poolWithoutMinterRole.waitForDeployment();

    await poolWithoutMinterRole.createPool(await buildCreateParams());
    const poolAddress = await poolWithoutMinterRole.getAddress();
    await lendToken.connect(alice).approve(poolAddress, ethers.parseEther("25000"));
    await borrowToken.connect(bob).approve(poolAddress, ethers.parseEther("2"));
    await poolWithoutMinterRole.connect(alice).depositLend(0, ethers.parseEther("25000"));
    await poolWithoutMinterRole.connect(bob).depositBorrow(0, ethers.parseEther("2"));

    await ethers.provider.send("evm_increaseTime", [3601]);
    await ethers.provider.send("evm_mine");
    await poolWithoutMinterRole.settle(0);

    await expectRevert(
      poolWithoutMinterRole.connect(alice).claimLend(0),
      "LearningDebtToken: caller is not minter"
    );
    await expectRevert(
      poolWithoutMinterRole.connect(bob).claimBorrow(0),
      "LearningDebtToken: caller is not minter"
    );
  });
});
