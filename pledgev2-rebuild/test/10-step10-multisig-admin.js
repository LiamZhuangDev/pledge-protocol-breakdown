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

describe("Step 10: multisig admin", function () {
  const INTEREST_RATE = 1_000_000;
  const MAX_SUPPLY = ethers.parseEther("100000");
  const MORTGAGE_RATE = 200_000_000;
  const AUTO_LIQUIDATE_THRESHOLD = 20_000_000;

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
  let multiSig;

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

    const LearningPledgePool = await ethers.getContractFactory("LearningPledgePool");
    pool = await LearningPledgePool.deploy(await oracle.getAddress(), feeRecipient.address);
    await pool.waitForDeployment();

    const LearningMultiSig = await ethers.getContractFactory("LearningMultiSig");
    multiSig = await LearningMultiSig.deploy([owner.address, alice.address, bob.address], 2);
    await multiSig.waitForDeployment();

    await pool.transferOwnership(await multiSig.getAddress());
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

  async function approveAndExecute(signerA, signerB, target, value, data, nonce) {
    await multiSig.connect(signerA).approveTransaction(target, value, data, nonce);
    await multiSig.connect(signerB).approveTransaction(target, value, data, nonce);
    await multiSig.connect(signerB).executeTransaction(target, value, data, nonce);
  }

  it("stores owners and threshold", async function () {
    assert.equal(await multiSig.threshold(), 2n);
    assert.equal(await multiSig.ownerCount(), 3n);
    assert.equal(await multiSig.getOwner(0), owner.address);
    assert.equal(await multiSig.getOwner(1), alice.address);
    assert.equal(await multiSig.getOwner(2), bob.address);
    assert.equal(await multiSig.isOwner(carol.address), false);
  });

  it("executes pool admin calls only after enough approvals", async function () {
    const poolAddress = await pool.getAddress();
    const data = pool.interface.encodeFunctionData("setPause", [true]);
    const nonce = 1;

    await expectRevert(pool.setPause(true), "LearningPledgePool: caller is not owner");

    await multiSig.connect(owner).approveTransaction(poolAddress, 0, data, nonce);
    await expectRevert(
      multiSig.connect(owner).executeTransaction(poolAddress, 0, data, nonce),
      "LearningMultiSig: not enough approvals"
    );

    await multiSig.connect(alice).approveTransaction(poolAddress, 0, data, nonce);
    await multiSig.connect(bob).executeTransaction(poolAddress, 0, data, nonce);

    assert.equal(await pool.globalPaused(), true);
  });

  it("binds approvals to exact target, calldata, value, chain, and nonce", async function () {
    const poolAddress = await pool.getAddress();
    const pauseData = pool.interface.encodeFunctionData("setPause", [true]);
    const minAmountData = pool.interface.encodeFunctionData("setMinLendAmount", [ethers.parseEther("10")]);
    const nonce = 2;

    await multiSig.connect(owner).approveTransaction(poolAddress, 0, pauseData, nonce);
    await multiSig.connect(alice).approveTransaction(poolAddress, 0, pauseData, nonce);

    await expectRevert(
      multiSig.connect(bob).executeTransaction(poolAddress, 0, minAmountData, nonce),
      "LearningMultiSig: not enough approvals"
    );

    await multiSig.connect(owner).executeTransaction(poolAddress, 0, pauseData, nonce);

    assert.equal(await pool.globalPaused(), true);
  });

  it("rejects duplicate approvals, non-owner approvals, and replay execution", async function () {
    const poolAddress = await pool.getAddress();
    const data = pool.interface.encodeFunctionData("setMinBorrowAmount", [ethers.parseEther("0.1")]);
    const nonce = 3;

    await multiSig.connect(owner).approveTransaction(poolAddress, 0, data, nonce);

    await expectRevert(
      multiSig.connect(owner).approveTransaction(poolAddress, 0, data, nonce),
      "LearningMultiSig: already approved"
    );
    await expectRevert(
      multiSig.connect(carol).approveTransaction(poolAddress, 0, data, nonce),
      "LearningMultiSig: caller is not owner"
    );

    await multiSig.connect(alice).approveTransaction(poolAddress, 0, data, nonce);
    await multiSig.connect(alice).executeTransaction(poolAddress, 0, data, nonce);

    assert.equal(await pool.minBorrowAmount(), ethers.parseEther("0.1"));

    await expectRevert(
      multiSig.connect(bob).executeTransaction(poolAddress, 0, data, nonce),
      "LearningMultiSig: already executed"
    );
  });

  it("can create a pool through multisig ownership", async function () {
    const poolAddress = await pool.getAddress();
    const data = pool.interface.encodeFunctionData("createPool", [await buildCreateParams()]);

    await approveAndExecute(owner, alice, poolAddress, 0, data, 4);

    assert.equal(await pool.poolLength(), 1n);
    assert.equal(await pool.getPoolState(0), 0n);
  });
});
