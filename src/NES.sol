// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

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
        bytes32 staticRoot,
        bytes32 dynRoot,
        Action[] memory activity
    ) external returns (bytes32);
}
