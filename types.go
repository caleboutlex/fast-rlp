package fastrlp

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
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
	AccessList AccessList

	// Signature values
	V *big.Int
	R *big.Int
	S *big.Int
}

// LegacyTx is the transaction data of the original Ethereum transactions.
type LegacyTx struct {
	Nonce    uint64          // nonce of sender account
	GasPrice *big.Int        // wei per gas
	Gas      uint64          // gas limit
	To       *common.Address `rlp:"nil"` // nil means contract creation
	Value    *big.Int        // wei amount
	Data     []byte          // contract invocation input data
	V, R, S  *big.Int        // signature values
}

// AccessListTx is the data of EIP-2930 access list transactions.
type AccessListTx struct {
	ChainID    *big.Int        // destination chain ID
	Nonce      uint64          // nonce of sender account
	GasPrice   *big.Int        // wei per gas
	Gas        uint64          // gas limit
	To         *common.Address `rlp:"nil"` // nil means contract creation
	Value      *big.Int        // wei amount
	Data       []byte          // contract invocation input data
	AccessList AccessList      // EIP-2930 access list
	V, R, S    *big.Int        // signature values
}

// AccessList is an EIP-2930 access list.
type AccessList []AccessTuple

// AccessTuple is the element type of an access list.
type AccessTuple struct {
	Address     common.Address `json:"address"     gencodec:"required"`
	StorageKeys []common.Hash  `json:"storageKeys" gencodec:"required"`
}

// BlobTx represents an EIP-4844 transaction.
type BlobTx struct {
	ChainID    *uint256.Int
	Nonce      uint64
	GasTipCap  *uint256.Int // a.k.a. maxPriorityFeePerGas
	GasFeeCap  *uint256.Int // a.k.a. maxFeePerGas
	Gas        uint64
	To         common.Address
	Value      *uint256.Int
	Data       []byte
	AccessList AccessList
	BlobFeeCap *uint256.Int // a.k.a. maxFeePerBlobGas
	BlobHashes []common.Hash

	// A blob transaction can optionally contain blobs. This field must be set when BlobTx
	// is used to create a transaction for signing.
	Sidecar *BlobTxSidecar `rlp:"-"`

	// Signature values
	V *uint256.Int
	R *uint256.Int
	S *uint256.Int
}

// BlobTxSidecar contains the blobs of a blob transaction.
type BlobTxSidecar struct {
	Version     byte                 // Version
	Blobs       []kzg4844.Blob       // Blobs needed by the blob pool
	Commitments []kzg4844.Commitment // Commitments needed by the blob pool
	Proofs      []kzg4844.Proof      // Proofs needed by the blob pool
}

// SetCodeTx implements the EIP-7702 transaction type which temporarily installs
// the code at the signer's address.
type SetCodeTx struct {
	ChainID    *uint256.Int
	Nonce      uint64
	GasTipCap  *uint256.Int // a.k.a. maxPriorityFeePerGas
	GasFeeCap  *uint256.Int // a.k.a. maxFeePerGas
	Gas        uint64
	To         common.Address
	Value      *uint256.Int
	Data       []byte
	AccessList AccessList
	AuthList   []SetCodeAuthorization

	// Signature values
	V *uint256.Int
	R *uint256.Int
	S *uint256.Int
}

// SetCodeAuthorization is an authorization from an account to deploy code at its address.
type SetCodeAuthorization struct {
	ChainID uint256.Int    `json:"chainId" gencodec:"required"`
	Address common.Address `json:"address" gencodec:"required"`
	Nonce   uint64         `json:"nonce" gencodec:"required"`
	V       uint8          `json:"yParity" gencodec:"required"`
	R       uint256.Int    `json:"r" gencodec:"required"`
	S       uint256.Int    `json:"s" gencodec:"required"`
}

// FastDynamicFeeTx holds the raw RLP data for each field of an EIP-1559 transaction.
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
// a single memory block.
type eip1559Data struct {
	tx        types.DynamicFeeTx
	chainID   big.Int
	gasTipCap big.Int
	gasFeeCap big.Int
	value     big.Int
	v, r, s   big.Int
	to        common.Address
}

// ToTx() converts the zero-allocation FastDynamicFeeTx into a standard
// go-ethereum types.Transaction. This function will allocate memory for the
// geth transaction struct and its fields.
func (f *FastDynamicFeeTx) ToTx() (*types.Transaction, error) {
	d := new(eip1559Data)
	innerTx := &d.tx
	var buf [32]byte

	setBigInt(&f.ChainID, &d.chainID, buf[:])
	innerTx.ChainID = &d.chainID
	innerTx.Nonce = f.Nonce
	setBigInt(&f.GasTipCap, &d.gasTipCap, buf[:])
	innerTx.GasTipCap = &d.gasTipCap
	setBigInt(&f.GasFeeCap, &d.gasFeeCap, buf[:])
	innerTx.GasFeeCap = &d.gasFeeCap
	innerTx.Gas = f.Gas

	if !f.Create {
		copy(d.to[:], f.To[:])
		innerTx.To = &d.to
	}

	setBigInt(&f.Value, &d.value, buf[:])
	innerTx.Value = &d.value
	innerTx.Data = f.Data

	// Correctly decode the AccessList
	var err error
	if err = decodeAccessList(f.AccessList, &innerTx.AccessList); err != nil {
		return nil, err
	}

	setBigInt(&f.V, &d.v, buf[:])
	innerTx.V = &d.v
	setBigInt(&f.R, &d.r, buf[:])
	innerTx.R = &d.r
	setBigInt(&f.S, &d.s, buf[:])
	innerTx.S = &d.s

	return types.NewTx(innerTx), nil
}
