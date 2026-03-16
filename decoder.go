package fastrlp

import (
	"github.com/ethereum/go-ethereum/core/types"
)

// DecodeTransaction decodes raw RLP bytes into a go-ethereum types.Transaction
// without triggering reflection-based heap allocations.
func DecodeTransaction(payload []byte) (*types.Transaction, error) {
	if len(payload) == 0 {
		return nil, ErrInvalidRLP
	}

	// Use a temporary Fast1559Tx to perform the zero-allocation parsing.
	var tx FastDynamicFeeTx
	if err := ParseTransaction(payload, &tx); err != nil {
		return nil, err
	}

	// Bridge the zero-alloc struct to the go-ethereum type. This is where
	// allocations for big.Int, etc., will occur.
	return tx.ToGethTransaction()
}

// ParseTransaction performs a zero-allocation decoding of a raw RLP transaction
// payload into a Fast1559Tx struct. The fields of the resulting struct are
// slices that point directly into the input payload.
func ParseTransaction(payload []byte, tx *FastDynamicFeeTx) error {
	if len(payload) == 0 {
		return ErrInvalidRLP
	}

	prefix := payload[0]
	var strOffset, strLen uint64

	if prefix <= 0x7f {
		return ErrInvalidRLP // Single byte, not a valid tx encoding
	} else if prefix <= 0xb7 {
		strLen = uint64(prefix - 0x80)
		strOffset = 1
	} else if prefix <= 0xbf {
		lenOfLen := int(prefix - 0xb7)
		if len(payload) < 1+lenOfLen {
			return ErrInvalidRLP
		}
		for _, b := range payload[1 : 1+lenOfLen] {
			strLen = (strLen << 8) | uint64(b)
		}
		strOffset = uint64(1 + lenOfLen)
	} else {
		// TODO: Handle legacy transaction (starts with a list prefix)
		return ErrNotImplemented
	}

	if uint64(len(payload)) != strOffset+strLen {
		return ErrInvalidRLP
	}

	inner := payload[strOffset:]
	if len(inner) == 0 || inner[0] != types.DynamicFeeTxType {
		return ErrInvalidTxType
	}

	listPayload := inner[1:]
	if len(listPayload) == 0 {
		return ErrInvalidRLP
	}

	listPrefix := listPayload[0]
	var listOffset, listLen uint64

	if listPrefix <= 0xbf {
		return ErrInvalidRLP
	} else if listPrefix <= 0xf7 { // Short list
		listLen = uint64(listPrefix - 0xc0)
		listOffset = 1
	} else { // Long list
		lenOfLen := int(listPrefix - 0xf7)
		if len(listPayload) < 1+lenOfLen {
			return ErrInvalidRLP
		}
		for _, b := range listPayload[1 : 1+lenOfLen] {
			listLen = (listLen << 8) | uint64(b)
		}
		listOffset = uint64(1 + lenOfLen)
	}

	if uint64(len(listPayload)) != listOffset+listLen {
		return ErrInvalidRLP
	}

	fields := listPayload[listOffset:]
	return parseTransactionBody(fields, tx)
}

// parseTransactionBody is the zero-allocation core of the decoder. It takes the
// RLP list payload of a transaction and populates a Fast1559Tx with slices
// pointing to the original payload.
func parseTransactionBody(fields []byte, tx *FastDynamicFeeTx) (err error) {
	var b []byte

	if b, fields, err = decodeBytes(fields); err != nil {
		return err
	}
	tx.ChainID.SetBytes(b)

	if b, fields, err = decodeBytes(fields); err != nil {
		return err
	}
	if tx.Nonce, err = bytesToUint64(b); err != nil {
		return err
	}

	if b, fields, err = decodeBytes(fields); err != nil {
		return err
	}
	tx.GasTipCap.SetBytes(b)

	if b, fields, err = decodeBytes(fields); err != nil {
		return err
	}
	tx.GasFeeCap.SetBytes(b)

	if b, fields, err = decodeBytes(fields); err != nil {
		return err
	}
	if tx.Gas, err = bytesToUint64(b); err != nil {
		return err
	}

	if b, fields, err = decodeBytes(fields); err != nil {
		return err
	}
	if len(b) == 20 {
		copy(tx.To[:], b)
		tx.Create = false
	} else if len(b) == 0 {
		tx.Create = true
	} else {
		return ErrInvalidRLP
	}

	if b, fields, err = decodeBytes(fields); err != nil {
		return err
	}
	tx.Value.SetBytes(b)

	if tx.Data, fields, err = decodeBytes(fields); err != nil {
		return err
	}

	if tx.AccessList, fields, err = decodeList(fields); err != nil {
		return err
	}

	if b, fields, err = decodeBytes(fields); err != nil {
		return err
	}
	tx.V.SetBytes(b)

	if b, fields, err = decodeBytes(fields); err != nil {
		return err
	}
	tx.R.SetBytes(b)

	if b, fields, err = decodeBytes(fields); err != nil {
		return err
	}
	tx.S.SetBytes(b)
	if len(fields) > 0 {
		return ErrInvalidRLP // Extraneous data
	}
	return nil
}
