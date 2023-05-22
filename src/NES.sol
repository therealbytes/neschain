// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

address constant NES_ADDRESS = address(0x80);

enum Button {
    A,
    B,
    Select,
    Start,
    Up,
    Down,
    Left,
    Right
}

struct Action {
    Button button;
    bool press;
    uint32 duration;
}

interface NES {
    function run(
        bytes32 staticHash,
        bytes32 dynHash,
        Action[] memory activity
    ) external returns (bytes32);

    function addPreimage(bytes memory preimage) external returns (bytes32);

    function getPreimageSize(bytes32 _hash) external view returns (uint256);

    function getPreimage(uint256 size, bytes32 _hash) external view returns (bytes memory);
}
