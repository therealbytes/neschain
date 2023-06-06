```
├── engine
│   ├── main.go             # Concrete app-chain setup
│   ├── main_test.go        # Concrete app-chain setup minimal test
│   └── pcs
│       ├── abi             # Precompile ABI
│       │   └── NES.json
│       ├── nes.go          # Precompile implementation
│       ├── nes_test.go     # Precompile test
│       └── testdata        # Test static and dynamic NES states as GOB binaries
│           ├── mario.dyn
│           └── mario.static
└── src
    └── NES.sol # Precompile solidity interface
```

Check out [Concrete](https://github.com/therealbytes/concrete-geth).
