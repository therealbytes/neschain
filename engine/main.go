package main

import (
	"github.com/therealbytes/neschain/engine/pcs"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/concrete"
	"github.com/ethereum/go-ethereum/concrete/precompiles"
)

func setup(engine concrete.ConcreteApp) {
	engine.AddPrecompile(common.HexToAddress("0x80"), pcs.NESPrecompile, precompiles.PrecompileMetadata{
		Name:        "NES",
		Version:     "0.1.0",
		Author:      "therealbytes",
		Description: "Run activity on top of a given NES state",
		Source:      "https://github.com/therealbytes/neschain/blob/main/engine/pcs/nes.go",
	})
}

func main() {
	engine := concrete.ConcreteGeth
	setup(engine)
	engine.Run()
}
