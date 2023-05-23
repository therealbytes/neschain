package pcs

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/concrete/api"
	"github.com/ethereum/go-ethereum/concrete/crypto"
	"github.com/ethereum/go-ethereum/concrete/lib"
	"github.com/fogleman/nes/nes"
)

// #### DEBUGGING ####

//go:embed testdata/mario.static
var marioStatic []byte

//go:embed testdata/mario.dyn
var marioDyn []byte

var (
	marioStaticHash = crypto.Keccak256Hash(marioStatic)
	marioDynHash    = crypto.Keccak256Hash(marioDyn)
)

// ##################

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

func (p *runPrecompile) MutatesStorage(input []byte) bool {
	return true
}

func (p *runPrecompile) RequiredGas(input []byte) uint64 {
	return 100
	// actionOffset := uint64(64 + 32)
	// actionBytesRaw := lib.GetData(input, actionOffset, uint64(len(input))-actionOffset)
	// nActions := uint64(len(actionBytesRaw) / 96)
	// totalDuration := uint64(0)
	// for i := uint64(0); i < nActions; i++ {
	// 	durationBytes := lib.GetData(actionBytesRaw, i*96+64, 32)
	// 	duration := new(big.Int).SetBytes(durationBytes).Uint64()
	// 	totalDuration += duration
	// }
	// return 1_000_000 + nActions*100 + totalDuration*2
}

type Action struct {
	Button   uint8
	Press    bool
	Duration uint32
}

func decodeActivity(input []byte) []Action {
	var activity []Action
	_, input = lib.SplitData(input, 32)
	sizeBytes, activityBytes := lib.SplitData(input, 32)
	size := new(big.Int).SetBytes(sizeBytes).Uint64()
	for i := uint64(0); i < size; i++ {
		actionBytes := lib.GetData(activityBytes, i*3*32, 3*32)
		buttonBytes := lib.GetData(actionBytes, 0, 32)
		pressBytes := lib.GetData(actionBytes, 32, 32)
		durationBytes := lib.GetData(actionBytes, 64, 32)
		action := Action{
			Button:   uint8(new(big.Int).SetBytes(buttonBytes).Uint64()),
			Press:    new(big.Int).SetBytes(pressBytes).Uint64() == 1,
			Duration: uint32(new(big.Int).SetBytes(durationBytes).Uint64()),
		}
		activity = append(activity, action)
	}
	return activity
}

func (p *runPrecompile) Run(concrete api.API, input []byte) ([]byte, error) {

	fmt.Println("runPrecompile.Run", "input", input)

	per := concrete.Persistent()

	staticHashBytes := lib.GetData(input, 0, 32)
	dynHashBytes := lib.GetData(input, 32, 32)
	activityBytes := lib.GetData(input, 64, uint64(len(input))-64)

	staticHash := common.BytesToHash(staticHashBytes)
	dynHash := common.BytesToHash(dynHashBytes)

	if !per.HasPreimage(staticHash) {
		if staticHash == marioStaticHash {
			per.AddPreimage(marioStatic)
		} else {
			fmt.Println("runPrecompile.Run: invalid static hash", staticHash.Hex())
			return nil, errors.New("invalid static hash")
		}
	}
	if !per.HasPreimage(dynHash) {
		if dynHash == marioDynHash {
			per.AddPreimage(marioDyn)
		} else {
			fmt.Println("runPrecompile.Run: invalid dyn hash", dynHash.Hex())
			return nil, errors.New("invalid dynamic hash")
		}
	}

	static := per.GetPreimage(staticHash)
	dyn := per.GetPreimage(dynHash)

	fmt.Println("runPrecompile.Run: static pi size", len(static))
	fmt.Println("runPrecompile.Run: dyn pi size", len(dyn))

	activity := decodeActivity(activityBytes)

	console, err := nes.NewHeadlessConsole(static, dyn)
	if err != nil {
		return nil, err
	}

	buttons := [8]bool{}

	fmt.Println("runPrecompile.Run: running activity")

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

	fmt.Println("runPrecompile.Run: ok")
	fmt.Println("runPrecompile.Run: dyn pi size", len(dyn))
	fmt.Println("runPrecompile.Run: dyn pi hash", dynHash.Hex())

	return dynHash.Bytes(), nil
}

type addPreimagePrecompile struct {
	lib.BlankPrecompile
}

func (p *addPreimagePrecompile) MutatesStorage(input []byte) bool {
	return true
}

func (p *addPreimagePrecompile) RequiredGas(input []byte) uint64 {
	return 1000
	// return uint64(len(input) * 100)
}

func (p *addPreimagePrecompile) Run(concrete api.API, input []byte) ([]byte, error) {
	fmt.Println("addPreimagePrecompile.Run")
	per := concrete.Persistent()
	_, input = lib.SplitData(input, 32)
	sizeBytes, dataRaw := lib.SplitData(input, 32)
	size := new(big.Int).SetBytes(sizeBytes).Uint64()
	preimage := lib.GetData(dataRaw, 0, size)
	hash := crypto.Keccak256Hash(preimage)
	fmt.Println("addPreimagePrecompile.Run", "hash", hash.Hex())
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
	hashBytes := lib.GetData(input, 0, 32)
	hash := common.BytesToHash(hashBytes)
	fmt.Println("getPreimageSizePrecompile.Run", "hash", hash.Hex())
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
	return 100
	// sizeBytes := lib.GetData(input, 0, 32)
	// size := new(big.Int).SetBytes(sizeBytes).Uint64()
	// return size * 100
}

func (p *getPreimagePrecompile) Run(concrete api.API, input []byte) ([]byte, error) {
	per := concrete.Persistent()
	// sizeBytes := lib.GetData(input, 0, 32)
	hashBytes := lib.GetData(input, 32, 32)
	// size := new(big.Int).SetBytes(sizeBytes).Uint64()
	hash := common.BytesToHash(hashBytes)
	fmt.Println("getPreimagePrecompile.Run", "hash", hash.Hex())
	if !per.HasPreimage(hash) {
		fmt.Println("getPreimagePrecompile.Run: invalid hash", hash.Hex())
		return nil, errors.New("invalid hash")
	}
	// actualSize := uint64(per.GetPreimageSize(hash))
	// if actualSize != size {
	// 	return nil, errors.New("invalid size")
	// }
	return per.GetPreimage(hash), nil
}
