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

describe("Step 8: finish flow", function () {
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
    await lendToken.mint(owner.address, ethers.parseEther("100000"));
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

    for (const account of [owner, alice, carol]) {
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

  async function moveToEnd() {
    await ethers.provider.send("evm_increaseTime", [7 * 24 * 60 * 60 + 1]);
    await ethers.provider.send("evm_mine");
  }

  it("finishes the pool after repayment and lets SP/JP holders withdraw", async function () {
    await createPoolAndApproveAll();

    await pool.connect(alice).depositLend(0, ethers.parseEther("25000"));
    await pool.connect(bob).depositBorrow(0, ethers.parseEther("2"));
    await moveToSettleAndSettle();

    await pool.connect(bob).refundBorrow(0);
    await pool.connect(alice).claimLend(0);
    await pool.connect(bob).claimBorrow(0);

    await moveToEnd();

    const requiredRepayment = await pool.getRequiredRepayment(0);
    await pool.finish(0, requiredRepayment);

    const data = await pool.getPoolData(0);

    assert.equal(await pool.getPoolState(0), 2n);
    assert.equal(data.finishAmountLend, requiredRepayment);
    assert.equal(data.finishAmountBorrow, ethers.parseEther("1"));

    await pool.connect(alice).withdrawLend(0, await spToken.balanceOf(alice.address));
    await pool.connect(bob).withdrawBorrow(0, await jpToken.balanceOf(bob.address));

    assert.equal(await spToken.balanceOf(alice.address), 0n);
    assert.equal(await jpToken.balanceOf(bob.address), 0n);
    assert.equal(await lendToken.balanceOf(alice.address), ethers.parseEther("75000") + requiredRepayment);
    assert.equal(await borrowToken.balanceOf(bob.address), ethers.parseEther("10"));
  });

  it("withdraws lender repayment proportionally across SP holders", async function () {
    await createPoolAndApproveAll();

    await pool.connect(alice).depositLend(0, ethers.parseEther("60000"));
    await pool.connect(carol).depositLend(0, ethers.parseEther("40000"));
    await pool.connect(bob).depositBorrow(0, ethers.parseEther("2"));
    await moveToSettleAndSettle();

    await pool.connect(alice).refundLend(0);
    await pool.connect(carol).refundLend(0);
    await pool.connect(alice).claimLend(0);
    await pool.connect(carol).claimLend(0);
    await pool.connect(bob).claimBorrow(0);

    await moveToEnd();

    const requiredRepayment = await pool.getRequiredRepayment(0);
    await pool.finish(0, requiredRepayment);

    await pool.connect(alice).withdrawLend(0, await spToken.balanceOf(alice.address));
    await pool.connect(carol).withdrawLend(0, await spToken.balanceOf(carol.address));

    assert.equal(await lendToken.balanceOf(alice.address), ethers.parseEther("70000") + (requiredRepayment * 3n) / 5n);
    assert.equal(await lendToken.balanceOf(carol.address), ethers.parseEther("80000") + (requiredRepayment * 2n) / 5n);
  });

  it("rejects finish before end time, by non-owner, or with too little repayment", async function () {
    await createPoolAndApproveAll();

    await pool.connect(alice).depositLend(0, ethers.parseEther("25000"));
    await pool.connect(bob).depositBorrow(0, ethers.parseEther("2"));
    await moveToSettleAndSettle();

    await expectRevert(pool.finish(0, ethers.parseEther("25000")), "LearningPledgePool: before end time");

    await moveToEnd();

    const requiredRepayment = await pool.getRequiredRepayment(0);

    await expectRevert(
      pool.connect(alice).finish(0, requiredRepayment),
      "LearningPledgePool: caller is not owner"
    );
    await expectRevert(
      pool.finish(0, requiredRepayment - 1n),
      "LearningPledgePool: insufficient repayment"
    );
  });

  it("rejects withdrawals before finish, with zero amount, or without receipt tokens", async function () {
    await createPoolAndApproveAll();

    await pool.connect(alice).depositLend(0, ethers.parseEther("25000"));
    await pool.connect(bob).depositBorrow(0, ethers.parseEther("2"));
    await moveToSettleAndSettle();

    await pool.connect(alice).claimLend(0);
    await pool.connect(bob).claimBorrow(0);

    await expectRevert(
      pool.connect(alice).withdrawLend(0, ethers.parseEther("1")),
      "LearningPledgePool: pool not finish"
    );
    await expectRevert(
      pool.connect(bob).withdrawBorrow(0, ethers.parseEther("1")),
      "LearningPledgePool: pool not finish"
    );

    await moveToEnd();
    await pool.finish(0, await pool.getRequiredRepayment(0));

    await expectRevert(pool.connect(alice).withdrawLend(0, 0), "LearningPledgePool: zero sp amount");
    await expectRevert(pool.connect(bob).withdrawBorrow(0, 0), "LearningPledgePool: zero jp amount");
    await expectRevert(
      pool.connect(carol).withdrawLend(0, ethers.parseEther("1")),
      "LearningDebtToken: burn exceeds balance"
    );
    await expectRevert(
      pool.connect(carol).withdrawBorrow(0, ethers.parseEther("1")),
      "LearningDebtToken: burn exceeds balance"
    );
  });
});
