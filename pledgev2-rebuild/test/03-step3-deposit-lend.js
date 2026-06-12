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

describe("Step 3: deposit lend", function () {
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
    lendToken = await LearningDebtToken.deploy("Mock BUSD", "mBUSD");
    borrowToken = await LearningDebtToken.deploy("Mock BTC", "mBTC");
    spToken = await LearningDebtToken.deploy("Senior Pool BUSD", "spBUSD");
    jpToken = await LearningDebtToken.deploy("Junior Pool BTC", "jpBTC");
    await lendToken.waitForDeployment();
    await borrowToken.waitForDeployment();
    await spToken.waitForDeployment();
    await jpToken.waitForDeployment();

    await lendToken.addMinter(owner.address);
    await lendToken.mint(alice.address, ethers.parseEther("1000"));
    await lendToken.mint(bob.address, ethers.parseEther("1000"));

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

  it("accepts a lender deposit before settlement", async function () {
    const depositAmount = ethers.parseEther("200");
    const poolAddress = await pool.getAddress();

    await lendToken.connect(alice).approve(poolAddress, depositAmount);
    await pool.connect(alice).depositLend(0, depositAmount);

    const createdPool = await pool.getPool(0);
    const aliceInfo = await pool.userLendInfo(alice.address, 0);

    assert.equal(createdPool.lendSupply, depositAmount);
    assert.equal(aliceInfo.stakeAmount, depositAmount);
    assert.equal(aliceInfo.refundAmount, 0n);
    assert.equal(aliceInfo.hasRefunded, false);
    assert.equal(aliceInfo.hasClaimed, false);
    assert.equal(await lendToken.balanceOf(alice.address), ethers.parseEther("800"));
    assert.equal(await lendToken.balanceOf(poolAddress), depositAmount);
  });

  it("aggregates multiple deposits from one lender", async function () {
    const poolAddress = await pool.getAddress();

    await lendToken.connect(alice).approve(poolAddress, ethers.parseEther("500"));
    await pool.connect(alice).depositLend(0, ethers.parseEther("200"));
    await pool.connect(alice).depositLend(0, ethers.parseEther("300"));

    const createdPool = await pool.getPool(0);
    const aliceInfo = await pool.userLendInfo(alice.address, 0);

    assert.equal(createdPool.lendSupply, ethers.parseEther("500"));
    assert.equal(aliceInfo.stakeAmount, ethers.parseEther("500"));
    assert.equal(await lendToken.balanceOf(poolAddress), ethers.parseEther("500"));
  });

  it("tracks deposits from different lenders separately", async function () {
    const poolAddress = await pool.getAddress();

    await lendToken.connect(alice).approve(poolAddress, ethers.parseEther("200"));
    await lendToken.connect(bob).approve(poolAddress, ethers.parseEther("300"));

    await pool.connect(alice).depositLend(0, ethers.parseEther("200"));
    await pool.connect(bob).depositLend(0, ethers.parseEther("300"));

    const createdPool = await pool.getPool(0);
    const aliceInfo = await pool.userLendInfo(alice.address, 0);
    const bobInfo = await pool.userLendInfo(bob.address, 0);

    assert.equal(createdPool.lendSupply, ethers.parseEther("500"));
    assert.equal(aliceInfo.stakeAmount, ethers.parseEther("200"));
    assert.equal(bobInfo.stakeAmount, ethers.parseEther("300"));
  });

  it("rejects deposits that are too small or above pool capacity", async function () {
    const poolAddress = await pool.getAddress();

    await lendToken.connect(alice).approve(poolAddress, ethers.parseEther("2000"));

    await expectRevert(
      pool.connect(alice).depositLend(0, ethers.parseEther("99")),
      "LearningPledgePool: lend amount too small"
    );

    await pool.connect(alice).depositLend(0, ethers.parseEther("900"));

    await expectRevert(
      pool.connect(alice).depositLend(0, ethers.parseEther("101")),
      "LearningPledgePool: lend supply exceeded"
    );
  });

  it("rejects deposits without enough approval", async function () {
    await expectRevert(
      pool.connect(alice).depositLend(0, ethers.parseEther("200")),
      "LearningDebtToken: insufficient allowance"
    );
  });

  it("rejects deposits while paused", async function () {
    const poolAddress = await pool.getAddress();

    await lendToken.connect(alice).approve(poolAddress, ethers.parseEther("200"));
    await pool.setPause(true);

    await expectRevert(
      pool.connect(alice).depositLend(0, ethers.parseEther("200")),
      "LearningPledgePool: paused"
    );
  });

  it("rejects deposits after settlement time", async function () {
    const poolAddress = await pool.getAddress();

    await lendToken.connect(alice).approve(poolAddress, ethers.parseEther("200"));
    await ethers.provider.send("evm_increaseTime", [3601]);
    await ethers.provider.send("evm_mine");

    await expectRevert(
      pool.connect(alice).depositLend(0, ethers.parseEther("200")),
      "LearningPledgePool: settle time passed"
    );
  });

  it("lets owner change the minimum lend amount", async function () {
    const poolAddress = await pool.getAddress();

    await pool.setMinLendAmount(ethers.parseEther("10"));
    await lendToken.connect(alice).approve(poolAddress, ethers.parseEther("10"));
    await pool.connect(alice).depositLend(0, ethers.parseEther("10"));

    const aliceInfo = await pool.userLendInfo(alice.address, 0);
    assert.equal(aliceInfo.stakeAmount, ethers.parseEther("10"));
  });

  it("blocks non-owner minimum lend amount updates and missing pools", async function () {
    await expectRevert(
      pool.connect(alice).setMinLendAmount(ethers.parseEther("10")),
      "LearningPledgePool: caller is not owner"
    );
    await expectRevert(
      pool.connect(alice).depositLend(1, ethers.parseEther("200")),
      "LearningPledgePool: pool not found"
    );
  });
});
