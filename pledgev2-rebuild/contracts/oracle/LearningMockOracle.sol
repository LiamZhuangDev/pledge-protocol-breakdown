// SPDX-License-Identifier: MIT

pragma solidity ^0.8.24;

contract LearningMockOracle {
    address public owner;

    mapping(address => uint256) private prices;

    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);
    event PriceUpdated(address indexed asset, uint256 price);

    constructor() {
        owner = msg.sender;

        emit OwnershipTransferred(address(0), msg.sender);
    }

    function transferOwnership(address newOwner) external onlyOwner {
        require(newOwner != address(0), "LearningMockOracle: zero owner");

        emit OwnershipTransferred(owner, newOwner);
        owner = newOwner;
    }

    function setPrice(address asset, uint256 price) external onlyOwner {
        _setPrice(asset, price);
    }

    function setPrices(address[] calldata assets, uint256[] calldata newPrices) external onlyOwner {
        require(assets.length == newPrices.length, "LearningMockOracle: length mismatch");

        for (uint256 i = 0; i < assets.length; i++) {
            _setPrice(assets[i], newPrices[i]);
        }
    }

    function getPrice(address asset) external view returns (uint256) {
        return prices[asset];
    }

    function getPrices(address[] calldata assets) external view returns (uint256[] memory result) {
        result = new uint256[](assets.length);

        for (uint256 i = 0; i < assets.length; i++) {
            result[i] = prices[assets[i]];
        }
    }

    function _setPrice(address asset, uint256 price) internal {
        require(asset != address(0), "LearningMockOracle: zero asset");
        require(price > 0, "LearningMockOracle: zero price");

        prices[asset] = price;

        emit PriceUpdated(asset, price);
    }

    modifier onlyOwner() {
        require(msg.sender == owner, "LearningMockOracle: caller is not owner");
        _;
    }
}
