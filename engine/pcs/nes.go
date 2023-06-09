package pcs

import (
	_ "embed"
	"encoding/json"
	"errors"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/concrete/api"
	"github.com/ethereum/go-ethereum/concrete/lib"
	"github.com/ethereum/go-ethereum/concrete/precompiles"
	"github.com/fogleman/nes/nes"
)

//go:embed abi/NES.json
var nesABIJson []byte

var (
	Radix    = precompiles.BigPreimageStoreRadix
	LeafSize = precompiles.BigPreimageStoreLeafSize
)

var (
	ABI           abi.ABI
	NESPrecompile api.Precompile
)

type Activity = []struct {
	Button   uint8  "json:\"button\""
	Press    bool   "json:\"press\""
	Duration uint32 "json:\"duration\""
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
	NESPrecompile = lib.NewPrecompileWithABI(ABI, map[string]lib.MethodPrecompile{"run": &runPrecompile{}})
}

func runActivity(console *nes.Console, activity Activity) {
	buttons := [8]bool{}
	for _, action := range activity {
		if action.Button < 8 {
			buttons[action.Button] = action.Press
		}
		console.Controller1.SetButtons(buttons)
		cycles := int(action.Duration)
		for cycles > 0 {
			cycles -= console.Step()
		}
	}
}

type runPrecompile struct {
	lib.BlankMethodPrecompile
}

func (p *runPrecompile) RequiredGas(input []byte) uint64 {
	return p.CallRequiredGasWithArgs(func(args []interface{}) uint64 {
		activity := args[2].(Activity)
		nActions := len(activity)
		totalDuration := uint32(0)
		for _, action := range activity {
			totalDuration += action.Duration
		}
		return 500_000 + uint64(nActions)*100 + uint64(totalDuration)/2
	}, input)
}

func (p *runPrecompile) Run(concrete api.API, input []byte) ([]byte, error) {
	return p.CallRunWithArgs(func(concrete api.API, args []interface{}) ([]interface{}, error) {
		staticRoot := common.Hash(args[0].([32]byte))
		dynRoot := common.Hash(args[1].([32]byte))
		activity := args[2].(Activity)

		preimageStore := api.NewPersistentBigPreimageStore(concrete, Radix, LeafSize)

		static := preimageStore.GetPreimage(staticRoot)
		if len(static) == 0 {
			return nil, errors.New("invalid static root")
		}
		dyn := preimageStore.GetPreimage(dynRoot)
		if len(dyn) == 0 {
			return nil, errors.New("invalid dynamic root")
		}

		console, err := nes.NewHeadlessConsole(static, dyn, false)
		if err != nil {
			return nil, err
		}
		runActivity(console, activity)

		dyn, err = console.SerializeDynamic()
		if err != nil {
			return nil, err
		}
		dynRoot = preimageStore.AddPreimage(dyn)

		return []interface{}{dynRoot}, nil
	}, concrete, input)
}
