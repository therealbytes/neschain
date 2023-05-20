package pcs

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"errors"
	"io/ioutil"
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
		string(ABI.Methods["run"].ID):         &RunPrecompile{},
		string(ABI.Methods["addPreimage"].ID): &AddPreimagePrecompile{},
	}
}

type RunPrecompile struct {
	lib.BlankPrecompile
}

func (p *RunPrecompile) MutatesStorage(input []byte) bool {
	return true
}

func (p *RunPrecompile) RequiredGas(input []byte) uint64 {
	actionOffset := uint64(64 + 32)
	actionBytesRaw := lib.GetData(input, actionOffset, uint64(len(input))-actionOffset)
	nActions := uint64(len(actionBytesRaw) / 96)
	totalDuration := uint64(0)
	for i := uint64(0); i < nActions; i++ {
		durationBytes := lib.GetData(actionBytesRaw, i*96+64, 32)
		duration := new(big.Int).SetBytes(durationBytes).Uint64()
		totalDuration += duration
	}
	return 1_000_000 + nActions*100 + totalDuration*2 + 200
}

type Action struct {
	Button   uint8
	Press    bool
	Duration uint32
}

func decodeActivity(input []byte) []Action {
	var activity []Action
	_, input = lib.SplitData(input, 32)
	sizeBytes, input := lib.SplitData(input, 32)
	size := new(big.Int).SetBytes(sizeBytes).Uint64()
	for i := uint64(0); i < size; i++ {
		actionBytes := lib.GetData(input, i*3*32, 3*32)
		buttonBytes, actionBytes := lib.SplitData(actionBytes, 32)
		pressBytes, actionBytes := lib.SplitData(actionBytes, 32)
		durationBytes, _ := lib.SplitData(actionBytes, 32)
		action := Action{
			Button:   uint8(new(big.Int).SetBytes(buttonBytes).Uint64()),
			Press:    new(big.Int).SetBytes(pressBytes).Uint64() == 1,
			Duration: uint32(new(big.Int).SetBytes(durationBytes).Uint64()),
		}
		activity = append(activity, action)
	}
	return activity
}

func compress(data []byte) ([]byte, error) {
	var buffer bytes.Buffer
	gz := gzip.NewWriter(&buffer)
	if _, err := gz.Write(data); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func decompress(data []byte) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	return ioutil.ReadAll(gz)
}

func (p *RunPrecompile) Run(concrete api.API, input []byte) ([]byte, error) {
	per := concrete.Persistent()

	staticHashBytes, input := lib.SplitData(input, 32)
	dynHashBytes, input := lib.SplitData(input, 32)
	activityBytes := input

	staticHash := common.BytesToHash(staticHashBytes)
	dynHash := common.BytesToHash(dynHashBytes)

	if !per.HasPreimage(staticHash) {
		return nil, errors.New("invalid static hash")
	}
	if !per.HasPreimage(dynHash) {
		return nil, errors.New("invalid dynamic hash")
	}

	staticZip := per.GetPreimage(staticHash)
	dynZip := per.GetPreimage(dynHash)

	static, err := decompress(staticZip)
	if err != nil {
		return nil, err
	}
	dyn, err := decompress(dynZip)
	if err != nil {
		return nil, err
	}

	activity := decodeActivity(activityBytes)

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
	dynZip, err = compress(dyn)
	if err != nil {
		return nil, err
	}
	dynHash = per.AddPreimage(dynZip)

	return dynHash.Bytes(), nil
}

type AddPreimagePrecompile struct {
	lib.BlankPrecompile
}

func (p *AddPreimagePrecompile) MutatesStorage(input []byte) bool {
	return true
}

func (p *AddPreimagePrecompile) RequiredGas(input []byte) uint64 {
	return uint64(200 * len(input))
}

func (p *AddPreimagePrecompile) Run(concrete api.API, input []byte) ([]byte, error) {
	per := concrete.Persistent()
	_, input = lib.SplitData(input, 32)
	sizeBytes, dataRaw := lib.SplitData(input, 32)
	size := new(big.Int).SetBytes(sizeBytes).Uint64()
	input = dataRaw[:size]
	hash := crypto.Keccak256Hash(input)
	if per.HasPreimage(hash) {
		return hash.Bytes(), nil
	}
	per.AddPreimage(input)
	return hash.Bytes(), nil
}
