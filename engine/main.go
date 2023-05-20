package main

import (
	"github.com/therealbytes/neschain/engine/pcs"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/concrete"
)

func setup(engine concrete.ConcreteApp) {
	engine.AddPrecompile(common.HexToAddress("0x80"), pcs.NESPrecompile)
}

func main() {
	engine := concrete.ConcreteGeth
	setup(engine)
	engine.Run()
}
