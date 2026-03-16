package fastrlp

import "errors"

var (
	// ErrInvalidRLP is returned when the RLP payload is malformed.
	ErrInvalidRLP = errors.New("invalid RLP payload")

	// ErrUintOverflow is returned when an RLP string is too large to fit into a uint64.
	ErrUintOverflow = errors.New("RLP uint64 overflow")

	// ErrInvalidTxType is returned for unsupported transaction types.
	ErrInvalidTxType = errors.New("invalid transaction type")

	// ErrNotImplemented is a temporary error for scaffolded methods.
	ErrNotImplemented = errors.New("not implemented")
)
