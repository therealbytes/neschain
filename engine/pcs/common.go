package pcs

import (
	"errors"

	cc_api "github.com/ethereum/go-ethereum/concrete/api"
)

type MethodDemux map[string]cc_api.Precompile

func (d MethodDemux) splitInput(input []byte) (string, []byte) {
	return string(input[:4]), input[4:]
}

func (d MethodDemux) MutatesStorage(input []byte) bool {
	sel, input := d.splitInput(input)
	pc, ok := d[sel]
	if !ok {
		// TODO: Should this error?
		return false
	}
	return pc.MutatesStorage(input)
}

func (d MethodDemux) RequiredGas(input []byte) uint64 {
	sel, input := d.splitInput(input)
	pc, ok := d[sel]
	if !ok {
		return 0
	}
	return pc.RequiredGas(input)
}

func (d MethodDemux) Finalise(api cc_api.API) error {
	for _, pc := range d {
		if err := pc.Finalise(api); err != nil {
			return err
		}
	}
	return nil
}

func (d MethodDemux) Commit(api cc_api.API) error {
	for _, pc := range d {
		if err := pc.Commit(api); err != nil {
			return err
		}
	}
	return nil
}

func (d MethodDemux) Run(api cc_api.API, input []byte) ([]byte, error) {
	sel, input := d.splitInput(input)
	pc, ok := d[sel]
	if !ok {
		return nil, errors.New("invalid select value")
	}
	return pc.Run(api, input)
}

var _ cc_api.Precompile = &MethodDemux{}
