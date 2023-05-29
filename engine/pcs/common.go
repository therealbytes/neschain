package pcs

import (
	"errors"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/concrete/api"
	"github.com/ethereum/go-ethereum/concrete/lib"
)

type MethodPrecompile interface {
	api.Precompile
	Init(method abi.Method)
}

type BlankMethodPrecompile struct {
	lib.BlankPrecompile
	Method abi.Method
}

func (p *BlankMethodPrecompile) Init(method abi.Method) {
	p.Method = method
}

func (p *BlankMethodPrecompile) MutatesStorage(input []byte) bool {
	return !p.Method.IsConstant()
}

func (p *BlankMethodPrecompile) CallRequiredGasWithArgs(requiredGas func(args []interface{}) uint64, input []byte) uint64 {
	args, err := p.Method.Inputs.UnpackValues(input)
	if err != nil {
		return 0
	}
	return requiredGas(args)
}

func (p *BlankMethodPrecompile) CallRunWithArgs(run func(concrete api.API, args []interface{}) ([]interface{}, error), concrete api.API, input []byte) ([]byte, error) {
	args, err := p.Method.Inputs.UnpackValues(input)
	if err != nil {
		return nil, errors.New("error unpacking arguments: " + err.Error())
	}
	returns, err := run(concrete, args)
	if err != nil {
		return nil, err
	}
	output, err := p.Method.Outputs.PackValues(returns)
	if err != nil {
		return nil, errors.New("error packing return values: " + err.Error())
	}
	return output, nil
}

var _ MethodPrecompile = &BlankMethodPrecompile{}

type PrecompileWithABI struct {
	Implementations map[string]api.Precompile
}

func NewPrecompileWithABI(contractABI abi.ABI, implementations map[string]MethodPrecompile) *PrecompileWithABI {
	p := &PrecompileWithABI{
		Implementations: make(map[string]api.Precompile),
	}
	for name, method := range contractABI.Methods {
		impl, ok := implementations[name]
		if !ok {
			panic("missing implementation for " + name)
		}
		impl.Init(method)
		p.Implementations[string(method.ID)] = impl
	}
	return p
}

func (p *PrecompileWithABI) getImplementation(input []byte) (api.Precompile, []byte, error) {
	id := input[:4]
	input = input[4:]
	impl, ok := p.Implementations[string(id)]
	if !ok {
		return nil, nil, errors.New("invalid method ID")
	}
	return impl, input, nil
}

func (p *PrecompileWithABI) MutatesStorage(input []byte) bool {
	pc, input, err := p.getImplementation(input)
	if err != nil {
		return false
	}
	return pc.MutatesStorage(input)
}

func (p *PrecompileWithABI) RequiredGas(input []byte) uint64 {
	pc, input, err := p.getImplementation(input)
	if err != nil {
		return 0
	}
	return pc.RequiredGas(input)
}

func (p *PrecompileWithABI) Finalise(api api.API) error {
	for _, pc := range p.Implementations {
		if err := pc.Finalise(api); err != nil {
			return err
		}
	}
	return nil
}

func (p *PrecompileWithABI) Commit(api api.API) error {
	for _, pc := range p.Implementations {
		if err := pc.Commit(api); err != nil {
			return err
		}
	}
	return nil
}

func (p *PrecompileWithABI) Run(api api.API, input []byte) ([]byte, error) {
	pc, input, err := p.getImplementation(input)
	if err != nil {
		return nil, err
	}
	return pc.Run(api, input)
}
