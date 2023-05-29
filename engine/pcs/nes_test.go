package pcs

import (
	"bytes"
	_ "embed"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/concrete/api"
	"github.com/ethereum/go-ethereum/concrete/crypto"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/fogleman/nes/nes"
)

//go:embed testdata/mario.static
var testStatic []byte

//go:embed testdata/mario.dyn
var testDyn []byte

func NewTestAPI() api.API {
	db := state.NewDatabase(rawdb.NewMemoryDatabase())
	statedb, _ := state.New(common.Hash{}, db, nil)
	addr := common.HexToAddress("0xc0ffee")
	return NewTestAPIWithStateDB(statedb, addr)
}

func NewTestAPIWithStateDB(statedb vm.StateDB, addr common.Address) api.API {
	evm := vm.NewEVM(vm.BlockContext{}, vm.TxContext{}, statedb, params.TestChainConfig, vm.Config{})
	return api.New(evm.NewConcreteEVM(), addr)
}

func TestRun(t *testing.T) {
	concrete := NewTestAPI()
	staticHash := concrete.Persistent().AddPreimage(testStatic)
	dynHash := concrete.Persistent().AddPreimage(testDyn)
	activity := []struct {
		Button   uint8
		Press    bool
		Duration uint32
	}{
		{Button: 0, Press: false, Duration: 100_000},
	}

	input, err := ABI.Pack("run", staticHash, dynHash, activity)
	if err != nil {
		t.Fatal(err)
	}

	output, err := NESPrecompile.Run(concrete, input)
	if err != nil {
		t.Fatal(err)
	}

	_outDynHash, err := ABI.Unpack("run", output)
	if err != nil {
		t.Fatal(err)
	}
	outDynHash := common.Hash(_outDynHash[0].([32]byte))

	refNes, err := nes.NewHeadlessConsole(testStatic, testDyn)
	if err != nil {
		t.Fatal(err)
	}

	for ii := 0; ii < int(activity[0].Duration); ii++ {
		refNes.Step()
	}

	refOutDyn, err := refNes.SerializeDynamic()
	if err != nil {
		t.Fatal(err)
	}

	refOutDynHash := crypto.Keccak256Hash(refOutDyn)

	if outDynHash != refOutDynHash {
		t.Fatal("output mismatch: got", outDynHash, "expected", refOutDynHash)
	}

	if !concrete.Persistent().HasPreimage(outDynHash) {
		t.Fatal("preimage not added")
	}
}

func TestAddPreimage(t *testing.T) {
	concrete := NewTestAPI()
	preimage := []byte("hello world")

	input, err := ABI.Pack("addPreimage", preimage)
	if err != nil {
		t.Fatal(err)
	}

	output, err := NESPrecompile.Run(concrete, input)
	if err != nil {
		t.Fatal(err)
	}

	_hash, err := ABI.Unpack("addPreimage", output)
	if err != nil {
		t.Fatal(err)
	}
	hash := common.Hash(_hash[0].([32]byte))

	if hash != crypto.Keccak256Hash(preimage) {
		t.Fatal("output mismatch: got", hash, "expected", crypto.Keccak256Hash(preimage))
	}
	if !concrete.Persistent().HasPreimage(hash) {
		t.Fatal("preimage not added")
	}
}

func TestGetPreimageSize(t *testing.T) {
	concrete := NewTestAPI()
	preimage := []byte("hello world")
	hash := concrete.Persistent().AddPreimage(preimage)

	input, err := ABI.Pack("getPreimageSize", hash)
	if err != nil {
		t.Fatal(err)
	}

	output, err := NESPrecompile.Run(concrete, input)
	if err != nil {
		t.Fatal(err)
	}

	_outSize, err := ABI.Unpack("getPreimageSize", output)
	if err != nil {
		t.Fatal(err)
	}
	outSize := int(_outSize[0].(*big.Int).Uint64())

	if outSize != len(preimage) {
		t.Fatal("output mismatch: got", outSize, "expected", len(preimage))
	}
}

func TestGetPreimage(t *testing.T) {
	concrete := NewTestAPI()
	preimage := []byte("hello world")
	hash := concrete.Persistent().AddPreimage(preimage)

	input, err := ABI.Pack("getPreimage", big.NewInt(int64(len(preimage))), hash)
	if err != nil {
		t.Fatal(err)
	}

	output, err := NESPrecompile.Run(concrete, input)
	if err != nil {
		t.Fatal(err)
	}

	_outPreimage, err := ABI.Unpack("getPreimage", output)
	if err != nil {
		t.Fatal(err)
	}
	outPreimage := _outPreimage[0].([]byte)

	if !bytes.Equal(outPreimage, preimage) {
		t.Fatal("output mismatch: got", output, "expected", preimage)
	}
}
