// SPDX-License-Identifier: MIT

pragma solidity ^0.8.24;

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

    address public owner;
    address public oracle;
    address payable public feeAddress;
    bool public globalPaused;

    PoolBaseInfo[] private pools;

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
    event PauseUpdated(bool paused);

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

    function getPool(uint256 poolId) external view returns (PoolBaseInfo memory) {
        require(poolId < pools.length, "LearningPledgePool: pool not found");
        return pools[poolId];
    }

    function getPoolState(uint256 poolId) external view returns (PoolState) {
        require(poolId < pools.length, "LearningPledgePool: pool not found");
        return pools[poolId].state;
    }

    function isBeforeSettle(uint256 poolId) external view returns (bool) {
        require(poolId < pools.length, "LearningPledgePool: pool not found");
        return block.timestamp < pools[poolId].settleTime;
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

    function setPause(bool paused) external onlyOwner {
        globalPaused = paused;
        emit PauseUpdated(paused);
    }

    modifier onlyOwner() {
        require(msg.sender == owner, "LearningPledgePool: caller is not owner");
        _;
    }
}
