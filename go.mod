module github.com/therealbytes/neschain

go 1.18

replace github.com/fogleman/nes => github.com/therealbytes/nes v0.0.0-20230520140439-926367a99db8

replace github.com/ethereum/go-ethereum => github.com/therealbytes/concrete-geth v0.0.0-20230518143737-14e116b335b8

require github.com/ethereum/go-ethereum v0.0.0-00010101000000-000000000000

require (
	github.com/btcsuite/btcd/btcec/v2 v2.3.2 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.1.0 // indirect
	golang.org/x/crypto v0.7.0 // indirect
	golang.org/x/sys v0.6.0 // indirect
)
