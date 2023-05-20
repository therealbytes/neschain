package pcs

import (
	"bytes"
	_ "embed"
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

	staticZip, err := compress(testStatic)
	if err != nil {
		t.Fatal(err)
	}
	dynZip, err := compress(testDyn)
	if err != nil {
		t.Fatal(err)
	}

	staticHash := concrete.Persistent().AddPreimage(staticZip)
	dynHash := concrete.Persistent().AddPreimage(dynZip)

	activity := []Action{
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

	refOutDynZip, err := compress(refOutDyn)
	if err != nil {
		t.Fatal(err)
	}
	refOutDynHash := crypto.Keccak256Hash(refOutDynZip)

	if outDynHash != refOutDynHash {
		t.Fatal("wrong output")
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
	hash := common.BytesToHash(output)

	if hash != crypto.Keccak256Hash(preimage) {
		t.Fatal("wrong output")
	}
	if !concrete.Persistent().HasPreimage(hash) {
		t.Fatal("preimage not added")
	}
}
func TestCompression(t *testing.T) {
	data := []byte("hello world")
	compressed, err := compress(data)
	if err != nil {
		t.Fatal(err)
	}
	decompressed, err := decompress(compressed)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, decompressed) {
		t.Fatal("wrong output")
	}
}
