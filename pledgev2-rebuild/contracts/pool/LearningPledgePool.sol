// SPDX-License-Identifier: MIT

pragma solidity ^0.8.24;

interface IERC20Like {
    function transferFrom(address from, address to, uint256 amount) external returns (bool);
}

contract LearningPledgePool {
    enum PoolState {
        MATCH,
        EXECUTION,
        FINISH,
        LIQUIDATION,
        UNDONE
    }

    struct CreatePoolParams {
        uint256 settleTime;
        uint256 endTime;
        uint256 interestRate;
        uint256 maxSupply;
        uint256 mortgageRate;
        address lendToken;
        address borrowToken;
        address spToken;
        address jpToken;
        uint256 autoLiquidateThreshold;
    }

    struct PoolBaseInfo {
        uint256 settleTime;
        uint256 endTime;
        uint256 interestRate;
        uint256 maxSupply;
        uint256 lendSupply;
        uint256 borrowSupply;
        uint256 mortgageRate;
        address lendToken;
        address borrowToken;
        PoolState state;
        address spToken;
        address jpToken;
        uint256 autoLiquidateThreshold;
    }

    struct LendInfo {
        uint256 stakeAmount;
        uint256 refundAmount;
        bool hasRefunded;
        bool hasClaimed;
    }

    address public owner;
    address public oracle;
    address payable public feeAddress;
    bool public globalPaused;
    uint256 public minLendAmount = 100 ether;

    PoolBaseInfo[] private pools;
    mapping(address => mapping(uint256 => LendInfo)) public userLendInfo;

    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);
    event PoolCreated(
        uint256 indexed poolId,
        address indexed lendToken,
        address indexed borrowToken,
        address spToken,
        address jpToken,
        uint256 settleTime,
        uint256 endTime
    );
    event FeeAddressUpdated(address indexed previousFeeAddress, address indexed newFeeAddress);
    event MinLendAmountUpdated(uint256 previousMinAmount, uint256 newMinAmount);
    event PauseUpdated(bool paused);
    event DepositLend(address indexed lender, uint256 indexed poolId, address indexed token, uint256 amount);

    constructor(address oracle_, address payable feeAddress_) {
        require(oracle_ != address(0), "LearningPledgePool: zero oracle");
        require(feeAddress_ != address(0), "LearningPledgePool: zero fee address");

        owner = msg.sender;
        oracle = oracle_;
        feeAddress = feeAddress_;

        emit OwnershipTransferred(address(0), msg.sender);
    }

    function createPool(CreatePoolParams calldata params) external onlyOwner returns (uint256 poolId) {
        require(params.settleTime > block.timestamp, "LearningPledgePool: settle time not future");
        require(params.endTime > params.settleTime, "LearningPledgePool: end before settle");
        require(params.maxSupply > 0, "LearningPledgePool: zero max supply");
        require(params.interestRate > 0, "LearningPledgePool: zero interest rate");
        require(params.mortgageRate > 0, "LearningPledgePool: zero mortgage rate");
        require(params.lendToken != address(0), "LearningPledgePool: zero lend token");
        require(params.borrowToken != address(0), "LearningPledgePool: zero borrow token");
        require(params.lendToken != params.borrowToken, "LearningPledgePool: same pool tokens");
        require(params.spToken != address(0), "LearningPledgePool: zero sp token");
        require(params.jpToken != address(0), "LearningPledgePool: zero jp token");
        require(params.spToken != params.jpToken, "LearningPledgePool: same debt tokens");

        poolId = pools.length;

        pools.push(
            PoolBaseInfo({
                settleTime: params.settleTime,
                endTime: params.endTime,
                interestRate: params.interestRate,
                maxSupply: params.maxSupply,
                lendSupply: 0,
                borrowSupply: 0,
                mortgageRate: params.mortgageRate,
                lendToken: params.lendToken,
                borrowToken: params.borrowToken,
                state: PoolState.MATCH,
                spToken: params.spToken,
                jpToken: params.jpToken,
                autoLiquidateThreshold: params.autoLiquidateThreshold
            })
        );

        emit PoolCreated(
            poolId,
            params.lendToken,
            params.borrowToken,
            params.spToken,
            params.jpToken,
            params.settleTime,
            params.endTime
        );
    }

    function poolLength() external view returns (uint256) {
        return pools.length;
    }

    function getPool(uint256 poolId) external view poolExists(poolId) returns (PoolBaseInfo memory) {
        return pools[poolId];
    }

    function getPoolState(uint256 poolId) external view poolExists(poolId) returns (PoolState) {
        return pools[poolId].state;
    }

    function isBeforeSettle(uint256 poolId) external view poolExists(poolId) returns (bool) {
        return block.timestamp < pools[poolId].settleTime;
    }

    function depositLend(uint256 poolId, uint256 amount)
        external
        whenNotPaused
        poolExists(poolId)
        stateMatch(poolId)
        beforeSettle(poolId)
    {
        PoolBaseInfo storage pool = pools[poolId];
        LendInfo storage lendInfo = userLendInfo[msg.sender][poolId];

        require(amount >= minLendAmount, "LearningPledgePool: lend amount too small");
        require(pool.lendSupply + amount <= pool.maxSupply, "LearningPledgePool: lend supply exceeded");

        bool success = IERC20Like(pool.lendToken).transferFrom(msg.sender, address(this), amount);
        require(success, "LearningPledgePool: lend transfer failed");

        lendInfo.stakeAmount += amount;
        lendInfo.hasRefunded = false;
        lendInfo.hasClaimed = false;
        pool.lendSupply += amount;

        emit DepositLend(msg.sender, poolId, pool.lendToken, amount);
    }

    function transferOwnership(address newOwner) external onlyOwner {
        require(newOwner != address(0), "LearningPledgePool: zero owner");

        emit OwnershipTransferred(owner, newOwner);
        owner = newOwner;
    }

    function setFeeAddress(address payable newFeeAddress) external onlyOwner {
        require(newFeeAddress != address(0), "LearningPledgePool: zero fee address");

        emit FeeAddressUpdated(feeAddress, newFeeAddress);
        feeAddress = newFeeAddress;
    }

    function setMinLendAmount(uint256 newMinAmount) external onlyOwner {
        emit MinLendAmountUpdated(minLendAmount, newMinAmount);
        minLendAmount = newMinAmount;
    }

    function setPause(bool paused) external onlyOwner {
        globalPaused = paused;
        emit PauseUpdated(paused);
    }

    modifier whenNotPaused() {
        require(!globalPaused, "LearningPledgePool: paused");
        _;
    }

    modifier poolExists(uint256 poolId) {
        require(poolId < pools.length, "LearningPledgePool: pool not found");
        _;
    }

    modifier stateMatch(uint256 poolId) {
        require(pools[poolId].state == PoolState.MATCH, "LearningPledgePool: pool not match");
        _;
    }

    modifier beforeSettle(uint256 poolId) {
        require(block.timestamp < pools[poolId].settleTime, "LearningPledgePool: settle time passed");
        _;
    }

    modifier onlyOwner() {
        require(msg.sender == owner, "LearningPledgePool: caller is not owner");
        _;
    }
}
