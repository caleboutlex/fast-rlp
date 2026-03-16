package fastrlp

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

// DynamicFeeTx represents an EIP-1559 transaction.
type DynamicFeeTx struct {
	ChainID    *big.Int
	Nonce      uint64
	GasTipCap  *big.Int // a.k.a. maxPriorityFeePerGas
	GasFeeCap  *big.Int // a.k.a. maxFeePerGas
	Gas        uint64
	To         *common.Address `rlp:"nil"` // nil means contract creation
	Value      *big.Int
	Data       []byte
	AccessList types.AccessList

	// Signature values
	V *big.Int
	R *big.Int
	S *big.Int
}

// RawDynamicFeeTx holds the raw RLP data for each field of an EIP-1559 transaction.
// This struct is designed for zero-allocation decoding. All fields are slices
// of the original RLP payload.
type FastDynamicFeeTx struct {
	ChainID    uint256.Int
	Nonce      uint64
	GasTipCap  uint256.Int // a.k.a. maxPriorityFeePerGas
	GasFeeCap  uint256.Int // a.k.a. maxFeePerGas
	Gas        uint64
	To         common.Address
	Create     bool // true if 'To' is nil (contract creation)
	Value      uint256.Int
	Data       []byte
	AccessList []byte
	V, R, S    uint256.Int // Signature values
}

// eip1559Data groups all the heap-allocated fields of a DynamicFeeTx into
// a single memory block. This reduces the number of heap allocations from
// ~9 down to 1.
type eip1559Data struct {
	tx        types.DynamicFeeTx
	chainID   big.Int
	gasTipCap big.Int
	gasFeeCap big.Int
	value     big.Int
	v, r, s   big.Int
	to        common.Address
}

// ToGethTransaction converts the zero-allocation Fast1559Tx into a standard
// go-ethereum types.Transaction. This function will allocate memory for the
// geth transaction struct and its fields.
func (f *FastDynamicFeeTx) ToGethTransaction() (*types.Transaction, error) {
	d := new(eip1559Data)
	innerTx := &d.tx

	var buf [32]byte

	// Optimization: Use SetUint64 for small values to reduce processing overhead
	if f.ChainID.IsUint64() {
		d.chainID.SetUint64(f.ChainID.Uint64())
	} else {
		f.ChainID.WriteToSlice(buf[:])
		d.chainID.SetBytes(buf[:])
	}
	innerTx.ChainID = &d.chainID
	innerTx.Nonce = f.Nonce

	if f.GasTipCap.IsUint64() {
		d.gasTipCap.SetUint64(f.GasTipCap.Uint64())
	} else {
		f.GasTipCap.WriteToSlice(buf[:])
		d.gasTipCap.SetBytes(buf[:])
	}
	innerTx.GasTipCap = &d.gasTipCap

	if f.GasFeeCap.IsUint64() {
		d.gasFeeCap.SetUint64(f.GasFeeCap.Uint64())
	} else {
		f.GasFeeCap.WriteToSlice(buf[:])
		d.gasFeeCap.SetBytes(buf[:])
	}
	innerTx.GasFeeCap = &d.gasFeeCap
	innerTx.Gas = f.Gas

	if !f.Create {
		copy(d.to[:], f.To[:])
		innerTx.To = &d.to
	}

	if f.Value.IsUint64() {
		d.value.SetUint64(f.Value.Uint64())
	} else {
		f.Value.WriteToSlice(buf[:])
		d.value.SetBytes(buf[:])
	}
	innerTx.Value = &d.value
	innerTx.Data = f.Data

	// Correctly decode the AccessList
	var err error
	if innerTx.AccessList, err = decodeAccessList(f.AccessList); err != nil {
		return nil, err
	}

	f.V.WriteToSlice(buf[:])
	d.v.SetBytes(buf[:])
	innerTx.V = &d.v

	f.R.WriteToSlice(buf[:])
	d.r.SetBytes(buf[:])
	innerTx.R = &d.r

	f.S.WriteToSlice(buf[:])
	d.s.SetBytes(buf[:])
	innerTx.S = &d.s

	return types.NewTx(innerTx), nil
}
