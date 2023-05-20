// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

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
    ) external;

    function addPreimage(bytes32 preimage) external;
}
