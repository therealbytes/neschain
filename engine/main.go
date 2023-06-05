package main

import (
	"github.com/therealbytes/neschain/engine/pcs"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/concrete"
	"github.com/ethereum/go-ethereum/concrete/precompiles"
)

func setup(engine concrete.ConcreteApp) {
	engine.SetConfig(concrete.ConcreteConfig{
		PreimageRegistryConfig: precompiles.PreimageRegistryConfig{
			Enabled:  true,
			Writable: true,
		},
		BigPreimageRegistryConfig: precompiles.PreimageRegistryConfig{
			Enabled:  true,
			Writable: true,
		},
	})
	engine.AddPrecompile(common.HexToAddress("0x80"), pcs.NESPrecompile, precompiles.PrecompileMetadata{
		Name:        "NES",
		Version:     precompiles.Version{common.Big0, common.Big1, common.Big0},
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
