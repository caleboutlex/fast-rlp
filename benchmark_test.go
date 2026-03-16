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

var eip1559Payload = makeEIP1559Payload()

func TestTransactionDecodingConsistency(t *testing.T) {
	payload := makeEIP1559Payload()

	// 1. Decode with go-ethereum's rlp as the source of truth.
	var gethTx types.Transaction
	if err := rlp.DecodeBytes(payload, &gethTx); err != nil {
		t.Fatalf("geth rlp.DecodeBytes failed: %v", err)
	}

	// 2. Decode with our custom fast-rlp implementation.
	fastTx, err := DecodeTransaction(payload)
	if err != nil {
		t.Fatalf("fastrlp.DecodeTransaction failed: %v", err)
	}

	// 3. Compare the results. They must be identical.
	compareTransactions(t, &gethTx, fastTx)
}

// BenchmarkGethDecodeTransaction-8     1365064               856.8 ns/op           512 B/op         19 allocs/op
func BenchmarkGethDecodeTransaction(b *testing.B) {
	var decodedTx types.Transaction

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Decode payload to types.Transaction struct
		if err := rlp.DecodeBytes(eip1559Payload, &decodedTx); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFastRLPDecodeTransaction-8      1854018               647.2 ns/op          1336 B/op         25 allocs/op
func BenchmarkFastRLPDecodeTransaction(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := DecodeTransaction(eip1559Payload)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFastRLPParseTransaction-8      35819221                33.59 ns/op            0 B/op          0 allocs/op
func BenchmarkParseTransaction(b *testing.B) {
	var tx FastDynamicFeeTx
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := ParseTransaction(eip1559Payload, &tx); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkToGethTransaction-8     2256909               506.8 ns/op          1160 B/op         31 allocs/op
// BenchmarkToGethTransaction-8     1958667               579.8 ns/op          1336 B/op         25 allocs/op
func BenchmarkToGethTransaction(b *testing.B) {
	var tx FastDynamicFeeTx
	// Parse the payload to the RawDynamicFeeTransaction type
	if err := ParseTransaction(eip1559Payload, &tx); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := tx.ToGethTransaction(); err != nil {
			b.Fatal(err)
		}
	}
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
	if want.GasTipCap().Cmp(got.GasTipCap()) != 0 {
		t.Errorf("GasTipCap mismatch: want %s, got %s", want.GasTipCap(), got.GasTipCap())
	}
	if want.GasFeeCap().Cmp(got.GasFeeCap()) != 0 {
		t.Errorf("GasFeeCap mismatch: want %s, got %s", want.GasFeeCap(), got.GasFeeCap())
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
