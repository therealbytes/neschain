package pcs

import (
	_ "embed"
	"encoding/json"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/concrete/api"
	"github.com/ethereum/go-ethereum/concrete/crypto"
	"github.com/ethereum/go-ethereum/concrete/lib"
	"github.com/fogleman/nes/nes"
)

//go:embed abi/NES.json
var nesABIJson []byte

var (
	ABI           abi.ABI
	NESPrecompile api.Precompile
)

var methodPrecompiles = map[string]MethodPrecompile{
	"run":             &runPrecompile{},
	"addPreimage":     &addPreimagePrecompile{},
	"getPreimageSize": &getPreimageSizePrecompile{},
	"getPreimage":     &getPreimagePrecompile{},
}

func init() {
	var jsonAbi struct {
		ABI abi.ABI `json:"abi"`
	}
	err := json.Unmarshal(nesABIJson, &jsonAbi)
	if err != nil {
		panic(err)
	}
	ABI = jsonAbi.ABI
	NESPrecompile = NewPrecompileWithABI(ABI, methodPrecompiles)
}

type runPrecompile struct {
	BlankMethodPrecompile
}

func (p *runPrecompile) typeAssertArgs(args []interface{}) (common.Hash, common.Hash, []struct {
	Button   uint8  "json:\"button\""
	Press    bool   "json:\"press\""
	Duration uint32 "json:\"duration\""
}, error) {
	staticHash := common.Hash(args[0].([32]byte))
	dynHash := common.Hash(args[1].([32]byte))
	activity := args[2].([]struct {
		Button   uint8  "json:\"button\""
		Press    bool   "json:\"press\""
		Duration uint32 "json:\"duration\""
	})
	return staticHash, dynHash, activity, nil
}

func (p *runPrecompile) RequiredGas(input []byte) uint64 {
	return p.CallRequiredGasWithArgs(func(args []interface{}) uint64 {
		_, _, activity, err := p.typeAssertArgs(args)
		if err != nil {
			return 0
		}
		nActions := len(activity)
		totalDuration := uint32(0)
		for _, action := range activity {
			totalDuration += action.Duration
		}
		return 500_000 + uint64(nActions)*100 + uint64(totalDuration)*3
	}, input)
}

func (p *runPrecompile) Run(concrete api.API, input []byte) ([]byte, error) {
	return p.CallRunWithArgs(func(concrete api.API, args []interface{}) ([]interface{}, error) {
		per := concrete.Persistent()

		staticHash, dynHash, activity, err := p.typeAssertArgs(args)
		if err != nil {
			return nil, err
		}

		static := per.GetPreimage(staticHash)
		if len(static) == 0 {
			return nil, errors.New("invalid static hash")
		}

		dyn := per.GetPreimage(dynHash)
		if len(dyn) == 0 {
			return nil, errors.New("invalid dynamic hash")
		}

		console, err := nes.NewHeadlessConsole(static, dyn)
		if err != nil {
			return nil, err
		}

		buttons := [8]bool{}

		for _, action := range activity {
			if action.Button < 8 {
				buttons[action.Button] = action.Press
			}
			console.Controller1.SetButtons(buttons)
			for ii := uint32(0); ii < action.Duration; ii++ {
				console.Step()
			}
		}

		dyn, err = console.SerializeDynamic()
		if err != nil {
			return nil, err
		}
		dynHash = per.AddPreimage(dyn)

		return []interface{}{dynHash}, nil
	}, concrete, input)
}

type addPreimagePrecompile struct {
	BlankMethodPrecompile
}

func (p *addPreimagePrecompile) RequiredGas(input []byte) uint64 {
	sizeBytes := lib.GetData(input, 32, 32)
	size := new(big.Int).SetBytes(sizeBytes).Uint64()
	return size * 100
}

func (p *addPreimagePrecompile) Run(concrete api.API, input []byte) ([]byte, error) {
	return p.CallRunWithArgs(func(concrete api.API, args []interface{}) ([]interface{}, error) {
		per := concrete.Persistent()
		preimage := args[0].([]byte)
		hash := crypto.Keccak256Hash(preimage)
		if per.HasPreimage(hash) {
			return []interface{}{hash}, nil
		}
		per.AddPreimage(preimage)
		return []interface{}{hash}, nil
	}, concrete, input)
}

type getPreimageSizePrecompile struct {
	BlankMethodPrecompile
}

func (p *getPreimageSizePrecompile) RequiredGas(input []byte) uint64 {
	return 100
}

func (p *getPreimageSizePrecompile) Run(concrete api.API, input []byte) ([]byte, error) {
	return p.CallRunWithArgs(func(concrete api.API, args []interface{}) ([]interface{}, error) {
		per := concrete.Persistent()
		hash := common.Hash(args[0].([32]byte))
		if !per.HasPreimage(hash) {
			return nil, errors.New("invalid hash")
		}
		size := per.GetPreimageSize(hash)
		sizeBn := new(big.Int).SetUint64(uint64(size))
		return []interface{}{sizeBn}, nil
	}, concrete, input)
}

type getPreimagePrecompile struct {
	BlankMethodPrecompile
}

func (p *getPreimagePrecompile) RequiredGas(input []byte) uint64 {
	sizeBytes := lib.GetData(input, 0, 32)
	size := new(big.Int).SetBytes(sizeBytes).Uint64()
	return size * 25
}

func (p *getPreimagePrecompile) Run(concrete api.API, input []byte) ([]byte, error) {
	return p.CallRunWithArgs(func(concrete api.API, args []interface{}) ([]interface{}, error) {
		per := concrete.Persistent()
		size := args[0].(*big.Int).Uint64()
		hash := common.Hash(args[1].([32]byte))
		if !per.HasPreimage(hash) {
			return nil, errors.New("invalid hash")
		}
		actualSize := per.GetPreimageSize(hash)
		if uint64(actualSize) != size {
			return nil, errors.New("invalid size")
		}
		preimage := per.GetPreimage(hash)
		return []interface{}{preimage}, nil
	}, concrete, input)
}
