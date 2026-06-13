// SPDX-License-Identifier: MIT

pragma solidity ^0.8.24;

interface IERC20Like {
    function transfer(address to, uint256 amount) external returns (bool);
    function transferFrom(address from, address to, uint256 amount) external returns (bool);
}

interface IOracleLike {
    function getPrice(address asset) external view returns (uint256);
}

contract LearningPledgePool {
    uint256 private constant RATE_BASE = 1e8;
    uint256 private constant PRICE_SCALE = 1e18;

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

    struct PoolDataInfo {
        uint256 settleAmountLend; // required lend amount
        uint256 settleAmountBorrow; // required collateral amount
        uint256 finishAmountLend;
        uint256 finishAmountBorrow;
        uint256 liquidationAmountLend;
        uint256 liquidationAmountBorrow;
    }

    struct LendInfo {
        uint256 stakeAmount;
        uint256 refundAmount;
        bool hasRefunded;
        bool hasClaimed;
    }

    struct BorrowInfo {
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
    uint256 public minBorrowAmount = 1 ether;

    PoolBaseInfo[] private pools;
    PoolDataInfo[] private poolData;
    mapping(address => mapping(uint256 => LendInfo)) public userLendInfo;
    mapping(address => mapping(uint256 => BorrowInfo)) public userBorrowInfo;

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
    event MinBorrowAmountUpdated(uint256 previousMinAmount, uint256 newMinAmount);
    event PauseUpdated(bool paused);
    event DepositLend(address indexed lender, uint256 indexed poolId, address indexed token, uint256 amount);
    event DepositBorrow(address indexed borrower, uint256 indexed poolId, address indexed token, uint256 amount);
    event RefundLend(address indexed lender, uint256 indexed poolId, address indexed token, uint256 amount);
    event RefundBorrow(address indexed borrower, uint256 indexed poolId, address indexed token, uint256 amount);
    event StateChanged(uint256 indexed poolId, PoolState previousState, PoolState newState);

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
        poolData.push(
            PoolDataInfo({
                settleAmountLend: 0,
                settleAmountBorrow: 0,
                finishAmountLend: 0,
                finishAmountBorrow: 0,
                liquidationAmountLend: 0,
                liquidationAmountBorrow: 0
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

    function getPoolData(uint256 poolId) external view poolExists(poolId) returns (PoolDataInfo memory) {
        return poolData[poolId];
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

    function depositBorrow(uint256 poolId, uint256 amount)
        external
        whenNotPaused
        poolExists(poolId)
        stateMatch(poolId)
        beforeSettle(poolId)
    {
        PoolBaseInfo storage pool = pools[poolId];
        BorrowInfo storage borrowInfo = userBorrowInfo[msg.sender][poolId];

        require(amount >= minBorrowAmount, "LearningPledgePool: borrow amount too small");

        bool success = IERC20Like(pool.borrowToken).transferFrom(msg.sender, address(this), amount);
        require(success, "LearningPledgePool: borrow transfer failed");

        borrowInfo.stakeAmount += amount;
        borrowInfo.hasRefunded = false;
        borrowInfo.hasClaimed = false;
        pool.borrowSupply += amount;

        emit DepositBorrow(msg.sender, poolId, pool.borrowToken, amount);
    }

    function settle(uint256 poolId) external onlyOwner poolExists(poolId) stateMatch(poolId) afterSettle(poolId) {
        PoolBaseInfo storage pool = pools[poolId];
        PoolDataInfo storage data = poolData[poolId];

        if (pool.lendSupply == 0 || pool.borrowSupply == 0) {
            data.settleAmountLend = pool.lendSupply;
            data.settleAmountBorrow = pool.borrowSupply;
            _setPoolState(poolId, PoolState.UNDONE);
            return;
        }

        uint256 lendPrice = IOracleLike(oracle).getPrice(pool.lendToken);
        uint256 borrowPrice = IOracleLike(oracle).getPrice(pool.borrowToken);
        require(lendPrice > 0, "LearningPledgePool: missing lend price");
        require(borrowPrice > 0, "LearningPledgePool: missing borrow price");

        uint256 borrowToLendRatio = (borrowPrice * PRICE_SCALE) / lendPrice;
        uint256 collateralValueInLend = (pool.borrowSupply * borrowToLendRatio) / PRICE_SCALE;
        uint256 maxSettleLend = (collateralValueInLend * RATE_BASE) / pool.mortgageRate;

        if (pool.lendSupply > maxSettleLend) {
            data.settleAmountLend = maxSettleLend;
            data.settleAmountBorrow = pool.borrowSupply;
        } else {
            data.settleAmountLend = pool.lendSupply;
            data.settleAmountBorrow = (pool.lendSupply * pool.mortgageRate * lendPrice) / (borrowPrice * RATE_BASE);
        }

        _setPoolState(poolId, PoolState.EXECUTION);
    }

    function refundLend(uint256 poolId) external whenNotPaused poolExists(poolId) stateExecution(poolId) {
        PoolBaseInfo storage pool = pools[poolId];
        PoolDataInfo storage data = poolData[poolId];
        LendInfo storage lendInfo = userLendInfo[msg.sender][poolId];

        require(lendInfo.stakeAmount > 0, "LearningPledgePool: no lend stake");
        require(!lendInfo.hasRefunded, "LearningPledgePool: lend already refunded");

        uint256 unmatchedAmount = pool.lendSupply - data.settleAmountLend;
        require(unmatchedAmount > 0, "LearningPledgePool: no lend refund");

        // unmatchedAmount       = TOTAL lender money that was NOT used in settlement
        // lendInfo.stakeAmount  = THIS lender's original deposit
        // pool.lendSupply       = TOTAL deposited by ALL lenders
        // refundAmount          = unmatchedAmount * (THIS lender's share of the pool)
        uint256 refundAmount = (unmatchedAmount * lendInfo.stakeAmount) / pool.lendSupply;
        lendInfo.refundAmount += refundAmount;
        lendInfo.hasRefunded = true;

        bool success = IERC20Like(pool.lendToken).transfer(msg.sender, refundAmount);
        require(success, "LearningPledgePool: lend refund transfer failed");

        emit RefundLend(msg.sender, poolId, pool.lendToken, refundAmount);
    }

    function refundBorrow(uint256 poolId) external whenNotPaused poolExists(poolId) stateExecution(poolId) {
        PoolBaseInfo storage pool = pools[poolId];
        PoolDataInfo storage data = poolData[poolId];
        BorrowInfo storage borrowInfo = userBorrowInfo[msg.sender][poolId];

        require(borrowInfo.stakeAmount > 0, "LearningPledgePool: no borrow stake");
        require(!borrowInfo.hasRefunded, "LearningPledgePool: borrow already refunded");

        uint256 unmatchedAmount = pool.borrowSupply - data.settleAmountBorrow;
        require(unmatchedAmount > 0, "LearningPledgePool: no borrow refund");

        uint256 refundAmount = (unmatchedAmount * borrowInfo.stakeAmount) / pool.borrowSupply;
        borrowInfo.refundAmount += refundAmount;
        borrowInfo.hasRefunded = true;

        bool success = IERC20Like(pool.borrowToken).transfer(msg.sender, refundAmount);
        require(success, "LearningPledgePool: borrow refund transfer failed");

        emit RefundBorrow(msg.sender, poolId, pool.borrowToken, refundAmount);
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

    function setMinBorrowAmount(uint256 newMinAmount) external onlyOwner {
        emit MinBorrowAmountUpdated(minBorrowAmount, newMinAmount);
        minBorrowAmount = newMinAmount;
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

    modifier stateExecution(uint256 poolId) {
        require(pools[poolId].state == PoolState.EXECUTION, "LearningPledgePool: pool not execution");
        _;
    }

    modifier beforeSettle(uint256 poolId) {
        require(block.timestamp < pools[poolId].settleTime, "LearningPledgePool: settle time passed");
        _;
    }

    modifier afterSettle(uint256 poolId) {
        require(block.timestamp >= pools[poolId].settleTime, "LearningPledgePool: before settle time");
        _;
    }

    modifier onlyOwner() {
        require(msg.sender == owner, "LearningPledgePool: caller is not owner");
        _;
    }

    function _setPoolState(uint256 poolId, PoolState newState) internal {
        PoolState previousState = pools[poolId].state;
        pools[poolId].state = newState;
        emit StateChanged(poolId, previousState, newState);
    }
}
