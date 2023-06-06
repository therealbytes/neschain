package pcs

import (
	"bytes"
	_ "embed"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/concrete/api"
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

func TestRun(t *testing.T) {
	var (
		addr       = common.HexToAddress("0xc0ffee")
		db         = state.NewDatabase(rawdb.NewMemoryDatabase())
		statedb, _ = state.New(common.Hash{}, db, nil)
		evm        = vm.NewEVM(vm.BlockContext{}, vm.TxContext{}, statedb, params.TestChainConfig, vm.Config{})
		concrete   = api.New(evm.NewConcreteEVM(), addr)
		activity   = Activity{
			{Button: 0, Press: false, Duration: 100_000},
			{Button: 3, Press: true, Duration: 100_000},
			{Button: 3, Press: false, Duration: 1_000_000},
		}
	)

	preimageStore := api.NewPersistentBigPreimageStore(concrete, Radix, LeafSize)

	staticHash := preimageStore.AddPreimage(testStatic)
	dynHash := preimageStore.AddPreimage(testDyn)

	for ii := 0; ii < 3; ii++ {
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
		if !preimageStore.HasPreimage(outDynHash) {
			t.Fatal("preimage not added")
		}
		outDyn := preimageStore.GetPreimage(outDynHash)

		refStatic := preimageStore.GetPreimage(staticHash)
		refInDyn := preimageStore.GetPreimage(dynHash)
		refNes, err := nes.NewHeadlessConsole(refStatic, refInDyn, false, false)
		if err != nil {
			t.Fatal(err)
		}
		runActivity(refNes, activity)
		refOutDyn, err := refNes.SerializeDynamic()
		if err != nil {
			t.Fatal(err)
		}

		if len(outDyn) != len(refOutDyn) {
			t.Fatal("dyn length mismatch")
		}
		if !bytes.Equal(outDyn, refOutDyn) {
			t.Fatal("dyn mismatch")
		}

		dynHash = outDynHash
	}
}
