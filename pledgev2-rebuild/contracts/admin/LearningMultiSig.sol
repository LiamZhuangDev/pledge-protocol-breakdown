// SPDX-License-Identifier: MIT

pragma solidity ^0.8.24;

contract LearningMultiSig {

    // These define who can approve and how many approvals are required.
    address[] private owners;
    uint256 public threshold;
    mapping(address => bool) public isOwner;
    
    // These track the approval status of each transaction.
    mapping(bytes32 => uint256) public approvalCount;
    mapping(bytes32 => bool) public executed;
    mapping(bytes32 => mapping(address => bool)) public hasApproved;

    event TransactionApproved(bytes32 indexed txHash, address indexed owner, uint256 approvalCount);
    event TransactionExecuted(bytes32 indexed txHash, address indexed executor, address indexed target);

    constructor(address[] memory initialOwners, uint256 threshold_) {
        require(initialOwners.length > 0, "LearningMultiSig: no owners");
        require(threshold_ > 0, "LearningMultiSig: zero threshold");
        require(threshold_ <= initialOwners.length, "LearningMultiSig: threshold too high");

        threshold = threshold_;

        for (uint256 i = 0; i < initialOwners.length; i++) {
            address owner = initialOwners[i];
            require(owner != address(0), "LearningMultiSig: zero owner");
            require(!isOwner[owner], "LearningMultiSig: duplicate owner");

            isOwner[owner] = true;
            owners.push(owner);
        }
    }

    function ownerCount() external view returns (uint256) {
        return owners.length;
    }

    function getOwner(uint256 index) external view returns (address) {
        require(index < owners.length, "LearningMultiSig: owner index out of bounds");
        return owners[index];
    }

    function getTransactionHash(address target, uint256 value, bytes calldata data, uint256 nonce)
        public
        view
        returns (bytes32)
    {
        return keccak256(abi.encode(address(this), block.chainid, target, value, keccak256(data), nonce));
    }

    function approveTransaction(address target, uint256 value, bytes calldata data, uint256 nonce)
        external
        onlyOwner
        returns (bytes32 txHash)
    {
        require(target != address(0), "LearningMultiSig: zero target");

        txHash = getTransactionHash(target, value, data, nonce);
        require(!executed[txHash], "LearningMultiSig: already executed");
        require(!hasApproved[txHash][msg.sender], "LearningMultiSig: already approved");

        hasApproved[txHash][msg.sender] = true;
        approvalCount[txHash] += 1;

        emit TransactionApproved(txHash, msg.sender, approvalCount[txHash]);
    }

    function executeTransaction(address target, uint256 value, bytes calldata data, uint256 nonce)
        external
        onlyOwner
        returns (bytes memory result)
    {
        bytes32 txHash = getTransactionHash(target, value, data, nonce);
        require(!executed[txHash], "LearningMultiSig: already executed");
        require(approvalCount[txHash] >= threshold, "LearningMultiSig: not enough approvals");

        executed[txHash] = true;

        bool success;
        (success, result) = target.call{value: value}(data);
        require(success, "LearningMultiSig: transaction failed");

        emit TransactionExecuted(txHash, msg.sender, target);
    }

    receive() external payable {}

    modifier onlyOwner() {
        require(isOwner[msg.sender], "LearningMultiSig: caller is not owner");
        _;
    }
}
