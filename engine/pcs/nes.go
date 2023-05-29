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
	NESPrecompile MethodDemux
)

func init() {
	// Get ABI
	var jsonAbi struct {
		ABI abi.ABI `json:"abi"`
	}
	err := json.Unmarshal(nesABIJson, &jsonAbi)
	if err != nil {
		panic(err)
	}
	ABI = jsonAbi.ABI
	// Set methods
	NESPrecompile = MethodDemux{
		string(ABI.Methods["run"].ID):             &runPrecompile{},
		string(ABI.Methods["addPreimage"].ID):     &addPreimagePrecompile{},
		string(ABI.Methods["getPreimageSize"].ID): &getPreimageSizePrecompile{},
		string(ABI.Methods["getPreimage"].ID):     &getPreimagePrecompile{},
	}
}

type runPrecompile struct {
	lib.BlankPrecompile
}

func (p *runPrecompile) unpackValues(input []byte) (common.Hash, common.Hash, []struct {
	Button   uint8  "json:\"button\""
	Press    bool   "json:\"press\""
	Duration uint32 "json:\"duration\""
}, error) {
	args, err := ABI.Methods["run"].Inputs.UnpackValues(input)
	if err != nil {
		return common.Hash{}, common.Hash{}, nil, err
	}
	staticHash := common.Hash(args[0].([32]byte))
	dynHash := common.Hash(args[1].([32]byte))
	activity := args[2].([]struct {
		Button   uint8  "json:\"button\""
		Press    bool   "json:\"press\""
		Duration uint32 "json:\"duration\""
	})
	return staticHash, dynHash, activity, nil
}

func (p *runPrecompile) MutatesStorage(input []byte) bool {
	return true
}

func (p *runPrecompile) RequiredGas(input []byte) uint64 {
	_, _, activity, err := p.unpackValues(input)
	if err != nil {
		return 0
	}
	nActions := len(activity)
	totalDuration := uint32(0)
	for _, action := range activity {
		totalDuration += action.Duration
	}
	return 500_000 + uint64(nActions)*100 + uint64(totalDuration)*3
}

func (p *runPrecompile) Run(concrete api.API, input []byte) ([]byte, error) {
	per := concrete.Persistent()

	staticHash, dynHash, activity, err := p.unpackValues(input)
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

	return dynHash.Bytes(), nil
}

type addPreimagePrecompile struct {
	lib.BlankPrecompile
}

func (p *addPreimagePrecompile) MutatesStorage(input []byte) bool {
	return true
}

func (p *addPreimagePrecompile) RequiredGas(input []byte) uint64 {
	sizeBytes := lib.GetData(input, 32, 32)
	size := new(big.Int).SetBytes(sizeBytes).Uint64()
	return size * 100
}

func (p *addPreimagePrecompile) Run(concrete api.API, input []byte) ([]byte, error) {
	per := concrete.Persistent()
	args, err := ABI.Methods["addPreimage"].Inputs.UnpackValues(input)
	if err != nil {
		return nil, err
	}
	preimage := args[0].([]byte)
	hash := crypto.Keccak256Hash(preimage)
	if per.HasPreimage(hash) {
		return hash.Bytes(), nil
	}
	per.AddPreimage(preimage)
	return hash.Bytes(), nil
}

type getPreimageSizePrecompile struct {
	lib.BlankPrecompile
}

func (p *getPreimageSizePrecompile) MutatesStorage(input []byte) bool {
	return false
}

func (p *getPreimageSizePrecompile) RequiredGas(input []byte) uint64 {
	return 100
}

func (p *getPreimageSizePrecompile) Run(concrete api.API, input []byte) ([]byte, error) {
	per := concrete.Persistent()
	args, err := ABI.Methods["getPreimageSize"].Inputs.UnpackValues(input)
	if err != nil {
		return nil, err
	}
	hash := common.Hash(args[0].([32]byte))
	if !per.HasPreimage(hash) {
		return nil, errors.New("invalid hash")
	}
	size := per.GetPreimageSize(hash)
	sizeBn := new(big.Int).SetUint64(uint64(size))
	return common.BigToHash(sizeBn).Bytes(), nil
}

type getPreimagePrecompile struct {
	lib.BlankPrecompile
}

func (p *getPreimagePrecompile) MutatesStorage(input []byte) bool {
	return false
}

func (p *getPreimagePrecompile) RequiredGas(input []byte) uint64 {
	sizeBytes := lib.GetData(input, 0, 32)
	size := new(big.Int).SetBytes(sizeBytes).Uint64()
	return size * 25
}

func (p *getPreimagePrecompile) Run(concrete api.API, input []byte) ([]byte, error) {
	per := concrete.Persistent()
	args, err := ABI.Methods["getPreimage"].Inputs.UnpackValues(input)
	if err != nil {
		return nil, err
	}
	size := args[0].(*big.Int).Uint64()
	hash := common.Hash(args[1].([32]byte))
	if !per.HasPreimage(hash) {
		return nil, errors.New("invalid hash")
	}
	actualSize := per.GetPreimageSize(hash)
	if uint64(actualSize) != size {
		return nil, errors.New("invalid size")
	}
	return per.GetPreimage(hash), nil
}
