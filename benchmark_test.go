package fastrlp

import (
	"bytes"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

func makeLegacyPayload() []byte {
	tx := types.NewTx(&types.LegacyTx{
		Nonce:    42,
		GasPrice: big.NewInt(50_000_000_000),
		Gas:      21000,
		To:       &common.Address{0xab, 0xcd, 0xef},
		Value:    big.NewInt(1_000_000_000_000_000_000),
		Data:     []byte("test data"),
		V:        big.NewInt(1),
		R:        big.NewInt(2),
		S:        big.NewInt(3),
	})

	payload, err := rlp.EncodeToBytes(tx)
	if err != nil {
		panic(err)
	}
	return payload
}

func makeContractCreatePayload() []byte {
	tx := types.NewTx(&types.LegacyTx{
		Nonce:    1,
		GasPrice: big.NewInt(20_000_000_000),
		Gas:      100000,
		To:       nil, // Contract Creation
		Value:    big.NewInt(0),
		Data:     []byte{0x60, 0x60, 0x01}, // dummy bytecode
		V:        big.NewInt(1),
		R:        big.NewInt(2),
		S:        big.NewInt(3),
	})
	payload, err := rlp.EncodeToBytes(tx)
	if err != nil {
		panic(err)
	}
	return payload
}

func makeEIP2930Payload() []byte {
	tx := types.NewTx(&types.AccessListTx{
		ChainID:  big.NewInt(1),
		Nonce:    42,
		GasPrice: big.NewInt(50_000_000_000),
		Gas:      21000,
		To:       &common.Address{0xab, 0xcd, 0xef},
		Value:    big.NewInt(1_000_000_000_000_000_000),
		Data:     []byte("test data"),
		AccessList: types.AccessList{
			{
				Address:     common.Address{0xaa},
				StorageKeys: []common.Hash{{0x01}},
			},
		},
		V: big.NewInt(1),
		R: big.NewInt(2),
		S: big.NewInt(3),
	})
	payload, err := rlp.EncodeToBytes(tx)
	if err != nil {
		panic(err)
	}
	return payload
}

// makeGethEIP1559Payload creates a RLP-encoded EIP-1559 transaction.
func makeEIP1559Payload() []byte {
	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:    big.NewInt(1),
		Nonce:      42,
		GasTipCap:  big.NewInt(2_000_000_000),  // 2 Gwei
		GasFeeCap:  big.NewInt(50_000_000_000), // 50 Gwei
		Gas:        21000,
		To:         &common.Address{0xab, 0xcd, 0xef},
		Value:      big.NewInt(1_000_000_000_000_000_000), // 1 ETH
		Data:       []byte("test data"),
		AccessList: types.AccessList{},
		V:          big.NewInt(0), // y-parity byte for EIP-1559
		R:          big.NewInt(1),
		S:          big.NewInt(2),
	})

	// We use rlp.EncodeToBytes so the resulting byte-slice
	// can be correctly decoded by rlp.DecodeBytes()
	payload, err := rlp.EncodeToBytes(tx)
	if err != nil {
		panic(err)
	}
	return payload
}

func makeEIP1559AccessListPayload() []byte {
	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   big.NewInt(1),
		Nonce:     42,
		GasTipCap: big.NewInt(2_000_000_000),
		GasFeeCap: big.NewInt(50_000_000_000),
		Gas:       100000,
		To:        &common.Address{0xab, 0xcd, 0xef},
		Value:     big.NewInt(1_000_000_000_000_000_000),
		Data:      []byte("test data"),
		AccessList: types.AccessList{
			{
				Address:     common.Address{0xaa},
				StorageKeys: []common.Hash{{0x01}, {0x02}},
			},
		},
		V: big.NewInt(0),
		R: big.NewInt(1),
		S: big.NewInt(2),
	})

	payload, err := rlp.EncodeToBytes(tx)
	if err != nil {
		panic(err)
	}
	return payload
}

// compareTransactions is a helper that provides detailed, field-by-field comparisons
// of two transactions, failing the test with a specific error if any field mismatches.
func compareTransactions(t *testing.T, want, got *types.Transaction) {
	t.Helper()

	if want.Type() != got.Type() {
		t.Errorf("Type mismatch: want %d, got %d", want.Type(), got.Type())
		return // Stop comparison if types are different
	}
	if want.ChainId().Cmp(got.ChainId()) != 0 {
		t.Errorf("ChainID mismatch: want %s, got %s", want.ChainId(), got.ChainId())
	}
	if want.Nonce() != got.Nonce() {
		t.Errorf("Nonce mismatch: want %d, got %d", want.Nonce(), got.Nonce())
	}
	if want.GasPrice().Cmp(got.GasPrice()) != 0 {
		t.Errorf("GasPrice mismatch: want %s, got %s", want.GasPrice(), got.GasPrice())
	}
	if want.Type() == types.DynamicFeeTxType {
		if want.GasTipCap().Cmp(got.GasTipCap()) != 0 {
			t.Errorf("GasTipCap mismatch: want %s, got %s", want.GasTipCap(), got.GasTipCap())
		}
		if want.GasFeeCap().Cmp(got.GasFeeCap()) != 0 {
			t.Errorf("GasFeeCap mismatch: want %s, got %s", want.GasFeeCap(), got.GasFeeCap())
		}
	}
	if want.Gas() != got.Gas() {
		t.Errorf("Gas mismatch: want %d, got %d", want.Gas(), got.Gas())
	}
	if (want.To() == nil) != (got.To() == nil) {
		t.Errorf("To address nil mismatch: want %v, got %v", want.To(), got.To())
	} else if want.To() != nil && *want.To() != *got.To() {
		t.Errorf("To address mismatch: want %s, got %s", want.To().Hex(), got.To().Hex())
	}
	if want.Value().Cmp(got.Value()) != 0 {
		t.Errorf("Value mismatch: want %s, got %s", want.Value(), got.Value())
	}
	if !bytes.Equal(want.Data(), got.Data()) {
		t.Errorf("Data mismatch: want %x, got %x", want.Data(), got.Data())
	}
	if !reflect.DeepEqual(want.AccessList(), got.AccessList()) {
		t.Errorf("AccessList mismatch: want %#v, got %#v", want.AccessList(), got.AccessList())
	}

	vWant, rWant, sWant := want.RawSignatureValues()
	vGot, rGot, sGot := got.RawSignatureValues()
	if vWant.Cmp(vGot) != 0 || rWant.Cmp(rGot) != 0 || sWant.Cmp(sGot) != 0 {
		t.Errorf("Signature mismatch:\n- V want: %s, got: %s\n- R want: %s, got: %s\n- S want: %s, got: %s", vWant, vGot, rWant, rGot, sWant, sGot)
	}
}

var (
	legacyPayload            = makeLegacyPayload()
	eip2930Payload           = makeEIP2930Payload()
	eip1559Payload           = makeEIP1559Payload()
	eip1559AccessListPayload = makeEIP1559AccessListPayload()
	contractCreatePayload    = makeContractCreatePayload()
)

func TestTransactionDecodingConsistency(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
	}{
		{"Legacy", legacyPayload},
		{"EIP-2930", eip2930Payload},
		{"EIP-1559", eip1559Payload},
		{"EIP-1559 with AccessList", eip1559AccessListPayload},
		{"Contract Creation", contractCreatePayload},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gethTx types.Transaction
			if err := rlp.DecodeBytes(tt.payload, &gethTx); err != nil {
				t.Fatalf("geth rlp.DecodeBytes failed: %v", err)
			}

			fastTx, err := DecodeTransaction(tt.payload)
			if err != nil {
				t.Fatalf("fastrlp.DecodeTransaction failed: %v", err)
			}

			compareTransactions(t, &gethTx, fastTx)
		})
	}
}

// BenchmarkGethDecode/Legacy-8             1683134               598.6 ns/op           360 B/op         14 allocs/op
// BenchmarkGethDecode/EIP1559-8            1405242               843.0 ns/op           512 B/op         19 allocs/op
// BenchmarkGethDecode/AccessList-8         1000000              1097.0 ns/op           856 B/op         22 allocs/op
func BenchmarkGethDecode(b *testing.B) {
	benchmarks := []struct {
		name    string
		payload []byte
	}{
		{"Legacy", legacyPayload},
		{"EIP1559", eip1559Payload},
		{"AccessList", eip1559AccessListPayload},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			var decodedTx types.Transaction
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := rlp.DecodeBytes(bm.payload, &decodedTx); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkFastRLPParse/Legacy-8          32542416                34.72 ns/op            0 B/op          0 allocs/op
// BenchmarkFastRLPParse/EIP1559-8         22326325                53.51 ns/op            0 B/op          0 allocs/op
// BenchmarkFastRLPParse/AccessList-8      21619786                55.30 ns/op            0 B/op          0 allocs/op
func BenchmarkFastRLPParse(b *testing.B) {
	benchmarks := []struct {
		name    string
		payload []byte
	}{
		{"Legacy", legacyPayload},
		{"EIP1559", eip1559Payload},
		{"AccessList", eip1559AccessListPayload},
	}

	var tx Transaction
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := ParseTransaction(bm.payload, &tx); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkFastRLPToTx/Legacy-8            3243415               363.2 ns/op           728 B/op         20 allocs/op
// BenchmarkFastRLPToTx/EIP1559-8           2973186               395.6 ns/op           936 B/op         24 allocs/op
// BenchmarkFastRLPToTx/AccessList-8        2371609               497.2 ns/op          1128 B/op         28 allocs/op
func BenchmarkFastRLPToTx(b *testing.B) {
	benchmarks := []struct {
		name    string
		payload []byte
	}{
		{"Legacy", legacyPayload},
		{"EIP1559", eip1559Payload},
		{"AccessList", eip1559AccessListPayload},
	}
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			var tx Transaction
			_ = ParseTransaction(bm.payload, &tx)

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = tx.ToTx()
			}
		})
	}
}
