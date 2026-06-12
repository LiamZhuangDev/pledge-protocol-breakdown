// SPDX-License-Identifier: MIT

pragma solidity ^0.8.24;

contract LearningDebtToken {
    string public name;
    string public symbol;
    uint8 public constant decimals = 18;

    address public owner;
    uint256 public totalSupply;

    mapping(address => uint256) public balanceOf;
    mapping(address => mapping(address => uint256)) public allowance;

    address[] private minters;
    mapping(address => uint256) private minterIndexPlusOne;

    event Approval(address indexed owner, address indexed spender, uint256 value);
    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);
    event MinterAdded(address indexed account);
    event MinterRemoved(address indexed account);
    event Transfer(address indexed from, address indexed to, uint256 value);

    constructor(string memory name_, string memory symbol_) {
        name = name_;
        symbol = symbol_;
        owner = msg.sender;

        emit OwnershipTransferred(address(0), msg.sender);
    }

    function transfer(address to, uint256 amount) external returns (bool) {
        _transfer(msg.sender, to, amount);
        return true;
    }

    function approve(address spender, uint256 amount) external returns (bool) {
        require(spender != address(0), "LearningDebtToken: zero spender");

        allowance[msg.sender][spender] = amount;
        emit Approval(msg.sender, spender, amount);
        return true;
    }

    function transferFrom(address from, address to, uint256 amount) external returns (bool) {
        uint256 currentAllowance = allowance[from][msg.sender];
        require(currentAllowance >= amount, "LearningDebtToken: insufficient allowance");

        allowance[from][msg.sender] = currentAllowance - amount;
        emit Approval(from, msg.sender, allowance[from][msg.sender]);

        _transfer(from, to, amount);
        return true;
    }

    function transferOwnership(address newOwner) external onlyOwner {
        require(newOwner != address(0), "LearningDebtToken: zero owner");

        emit OwnershipTransferred(owner, newOwner);
        owner = newOwner;
    }

    function addMinter(address account) external onlyOwner returns (bool) {
        require(account != address(0), "LearningDebtToken: zero minter");

        if (isMinter(account)) {
            return false;
        }

        minters.push(account);
        minterIndexPlusOne[account] = minters.length;

        emit MinterAdded(account);
        return true;
    }

    function removeMinter(address account) external onlyOwner returns (bool) {
        require(account != address(0), "LearningDebtToken: zero minter");

        uint256 indexPlusOne = minterIndexPlusOne[account];
        if (indexPlusOne == 0) {
            return false;
        }

        uint256 index = indexPlusOne - 1;
        uint256 lastIndex = minters.length - 1;

        if (index != lastIndex) {
            address lastMinter = minters[lastIndex];
            minters[index] = lastMinter;
            minterIndexPlusOne[lastMinter] = indexPlusOne;
        }

        minters.pop();
        delete minterIndexPlusOne[account];

        emit MinterRemoved(account);
        return true;
    }

    function mint(address to, uint256 amount) external onlyMinter returns (bool) {
        require(to != address(0), "LearningDebtToken: mint to zero");

        totalSupply += amount;
        balanceOf[to] += amount;

        emit Transfer(address(0), to, amount);
        return true;
    }

    function burn(address from, uint256 amount) external onlyMinter returns (bool) {
        require(from != address(0), "LearningDebtToken: burn from zero");
        require(balanceOf[from] >= amount, "LearningDebtToken: burn exceeds balance");

        balanceOf[from] -= amount;
        totalSupply -= amount;

        emit Transfer(from, address(0), amount);
        return true;
    }

    function isMinter(address account) public view returns (bool) {
        return minterIndexPlusOne[account] != 0;
    }

    function getMinterLength() external view returns (uint256) {
        return minters.length;
    }

    function getMinter(uint256 index) external view returns (address) {
        require(index < minters.length, "LearningDebtToken: index out of bounds");
        return minters[index];
    }

    function _transfer(address from, address to, uint256 amount) internal {
        require(to != address(0), "LearningDebtToken: transfer to zero");
        require(balanceOf[from] >= amount, "LearningDebtToken: transfer exceeds balance");

        balanceOf[from] -= amount;
        balanceOf[to] += amount;

        emit Transfer(from, to, amount);
    }

    modifier onlyOwner() {
        require(msg.sender == owner, "LearningDebtToken: caller is not owner");
        _;
    }

    modifier onlyMinter() {
        require(isMinter(msg.sender), "LearningDebtToken: caller is not minter");
        _;
    }
}
