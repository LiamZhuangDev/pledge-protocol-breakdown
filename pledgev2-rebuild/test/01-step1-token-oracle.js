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

describe("Step 1: token and oracle", function () {
  let owner;
  let alice;
  let bob;
  let busd;
  let btc;
  let spToken;
  let jpToken;
  let oracle;

  beforeEach(async function () {
    [owner, alice, bob, busd, btc] = await ethers.getSigners();

    const LearningDebtToken = await ethers.getContractFactory("LearningDebtToken");
    spToken = await LearningDebtToken.deploy("Senior Pool BUSD", "spBUSD");
    jpToken = await LearningDebtToken.deploy("Junior Pool BTC", "jpBTC");
    await spToken.waitForDeployment();
    await jpToken.waitForDeployment();

    const LearningMockOracle = await ethers.getContractFactory("LearningMockOracle");
    oracle = await LearningMockOracle.deploy();
    await oracle.waitForDeployment();
  });

  describe("LearningDebtToken", function () {
    it("starts with ERC20 metadata and zero supply", async function () {
      assert.equal(await spToken.name(), "Senior Pool BUSD");
      assert.equal(await spToken.symbol(), "spBUSD");
      assert.equal(await spToken.decimals(), 18n);
      assert.equal(await spToken.totalSupply(), 0n);
    });

    it("lets the owner add and remove minters", async function () {
      assert.equal(await spToken.isMinter(alice.address), false);

      await spToken.addMinter(alice.address);

      assert.equal(await spToken.isMinter(alice.address), true);
      assert.equal(await spToken.getMinterLength(), 1n);
      assert.equal(await spToken.getMinter(0), alice.address);

      await spToken.removeMinter(alice.address);

      assert.equal(await spToken.isMinter(alice.address), false);
      assert.equal(await spToken.getMinterLength(), 0n);
    });

    it("blocks non-owners from managing minters", async function () {
      await expectRevert(
        spToken.connect(alice).addMinter(alice.address),
        "LearningDebtToken: caller is not owner"
      );
      await expectRevert(
        spToken.connect(alice).removeMinter(owner.address),
        "LearningDebtToken: caller is not owner"
      );
    });

    it("lets minters mint and burn receipt tokens", async function () {
      await spToken.addMinter(alice.address);

      await spToken.connect(alice).mint(bob.address, 1000);
      assert.equal(await spToken.balanceOf(bob.address), 1000n);
      assert.equal(await spToken.totalSupply(), 1000n);

      await spToken.connect(alice).burn(bob.address, 400);
      assert.equal(await spToken.balanceOf(bob.address), 600n);
      assert.equal(await spToken.totalSupply(), 600n);
    });

    it("blocks accounts that are not minters from minting or burning", async function () {
      await expectRevert(
        jpToken.connect(alice).mint(bob.address, 1000),
        "LearningDebtToken: caller is not minter"
      );
      await expectRevert(
        jpToken.connect(alice).burn(bob.address, 1000),
        "LearningDebtToken: caller is not minter"
      );
    });

    it("supports normal ERC20 transfer and allowance behavior", async function () {
      await spToken.addMinter(owner.address);
      await spToken.mint(alice.address, 1000);

      await spToken.connect(alice).transfer(bob.address, 250);
      assert.equal(await spToken.balanceOf(alice.address), 750n);
      assert.equal(await spToken.balanceOf(bob.address), 250n);

      await spToken.connect(alice).approve(owner.address, 300);
      await spToken.transferFrom(alice.address, bob.address, 300);

      assert.equal(await spToken.balanceOf(alice.address), 450n);
      assert.equal(await spToken.balanceOf(bob.address), 550n);
      assert.equal(await spToken.allowance(alice.address, owner.address), 0n);
    });
  });

  describe("LearningMockOracle", function () {
    it("lets the owner set and read one asset price", async function () {
      await oracle.setPrice(busd.address, 100000000);

      assert.equal(await oracle.getPrice(busd.address), 100000000n);
      assert.equal(await oracle.getPrice(btc.address), 0n);
    });

    it("sets batch prices by asset address", async function () {
      await oracle.setPrices([busd.address, btc.address], [100000000, 5000000000000]);

      assert.equal(await oracle.getPrice(busd.address), 100000000n);
      assert.equal(await oracle.getPrice(btc.address), 5000000000000n);

      const prices = await oracle.getPrices([busd.address, btc.address]);
      assert.equal(prices[0], 100000000n);
      assert.equal(prices[1], 5000000000000n);
    });

    it("blocks non-owners and invalid prices", async function () {
      await expectRevert(
        oracle.connect(alice).setPrice(busd.address, 100000000),
        "LearningMockOracle: caller is not owner"
      );
      await expectRevert(
        oracle.setPrice(ethers.ZeroAddress, 100000000),
        "LearningMockOracle: zero asset"
      );
      await expectRevert(oracle.setPrice(busd.address, 0), "LearningMockOracle: zero price");
      await expectRevert(
        oracle.setPrices([busd.address], [100000000, 5000000000000]),
        "LearningMockOracle: length mismatch"
      );
    });
  });
});
