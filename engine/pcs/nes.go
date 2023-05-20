package pcs

import (
	"github.com/ethereum/go-ethereum/concrete/lib"
)

type NesPrecompile struct {
	lib.BlankPrecompile
}

func (p *NesPrecompile) MutatesStorage(input []byte) bool {
	return true
}

func (p *NesPrecompile) RequiredGas(input []byte) uint64 {
	return 0
}

func (p *NesPrecompile) Run(input []byte) ([]byte, error) {
	return []byte{}, nil
}
