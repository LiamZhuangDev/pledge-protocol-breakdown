// SPDX-License-Identifier: MIT

pragma solidity ^0.8.24;

interface IERC20RouterLike {
    function transfer(address to, uint256 amount) external returns (bool);
    function transferFrom(address from, address to, uint256 amount) external returns (bool);
}

contract LearningDexRouter {
    uint256 private constant RATE_SCALE = 1e18;

    address public owner;
    mapping(address => mapping(address => uint256)) public rateOutPerTokenIn;

    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);
    event RateUpdated(address indexed tokenIn, address indexed tokenOut, uint256 rateOutPerTokenIn);
    event Swap(
        address indexed sender,
        address indexed tokenIn,
        address indexed tokenOut,
        uint256 amountIn,
        uint256 amountOut,
        address recipient
    );

    constructor() {
        owner = msg.sender;
        emit OwnershipTransferred(address(0), msg.sender);
    }

    function setRate(address tokenIn, address tokenOut, uint256 rate) external onlyOwner {
        require(tokenIn != address(0), "LearningDexRouter: zero token in");
        require(tokenOut != address(0), "LearningDexRouter: zero token out");
        require(rate > 0, "LearningDexRouter: zero rate");

        rateOutPerTokenIn[tokenIn][tokenOut] = rate;

        emit RateUpdated(tokenIn, tokenOut, rate);
    }

    function getAmountIn(address tokenIn, address tokenOut, uint256 amountOut) public view returns (uint256) {
        uint256 rate = rateOutPerTokenIn[tokenIn][tokenOut];
        require(rate > 0, "LearningDexRouter: rate not set");

        return (amountOut * RATE_SCALE + rate - 1) / rate;
    }

    function swapTokensForExactTokens(
        address tokenIn,
        address tokenOut,
        uint256 amountOut,
        uint256 amountInMax,
        address recipient
    ) external returns (uint256 amountIn) {
        require(recipient != address(0), "LearningDexRouter: zero recipient");

        amountIn = getAmountIn(tokenIn, tokenOut, amountOut);
        require(amountIn <= amountInMax, "LearningDexRouter: excessive input");

        bool pulled = IERC20RouterLike(tokenIn).transferFrom(msg.sender, address(this), amountIn);
        require(pulled, "LearningDexRouter: input transfer failed");

        bool paid = IERC20RouterLike(tokenOut).transfer(recipient, amountOut);
        require(paid, "LearningDexRouter: output transfer failed");

        emit Swap(msg.sender, tokenIn, tokenOut, amountIn, amountOut, recipient);
    }

    modifier onlyOwner() {
        require(msg.sender == owner, "LearningDexRouter: caller is not owner");
        _;
    }
}
