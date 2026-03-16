package fastrlp

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

// DecodeTransaction decodes raw RLP bytes into a standard go-ethereum types.Transaction.
func DecodeTransaction(payload []byte) (*types.Transaction, error) {
	if len(payload) == 0 {
		return nil, ErrInvalidRLP
	}

	var tx Transaction
	if err := ParseTransaction(payload, &tx); err != nil {
		return nil, err
	}
	return tx.ToTx()
}

// ParseTransaction performs a zero-allocation decoding of any EVM transaction RLP payload.
func ParseTransaction(payload []byte, tx *Transaction) error {
	if len(payload) == 0 {
		return ErrInvalidRLP
	}
	*tx = Transaction{}

	if payload[0] >= 0xc0 {
		tx.Type = types.LegacyTxType
		listPayload, remainder, err := decodeList(payload)
		if err != nil || len(remainder) > 0 {
			return ErrInvalidRLP
		}
		return parseLegacyBody(listPayload, tx)
	}

	txType, inner, err := decodeTypedTxEnvelope(payload)
	if err != nil {
		return err
	}
	tx.Type = txType

	fields, remainder, err := decodeList(inner)
	if err != nil || len(remainder) > 0 {
		return ErrInvalidRLP
	}

	switch txType {
	case types.AccessListTxType:
		return parseAccessListBody(fields, tx)
	case types.DynamicFeeTxType:
		return parseDynamicFeeBody(fields, tx)
	default:
		return ErrInvalidTxType
	}
}

func parseAccessListBody(fields []byte, tx *Transaction) (err error) {
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
	tx.GasPrice.SetBytes(b)
	return decodeTypedTxTail(fields, &tx.Gas, &tx.To, &tx.Create, &tx.Value, &tx.Data, &tx.AccessList, &tx.V, &tx.R, &tx.S)
}

// decodeTypedTxEnvelope is a helper that unwraps the EIP-2718 RLP string envelope.
// It returns the transaction type byte and the remaining inner payload (the RLP list).
func decodeTypedTxEnvelope(payload []byte) (txType byte, inner []byte, err error) {
	content, remainder, err := decodeBytes(payload)
	if err != nil {
		return 0, nil, err
	}
	// The envelope should encompass the entire payload and contain at least a type byte.
	if len(remainder) > 0 || len(content) == 0 {
		return 0, nil, ErrInvalidRLP
	}
	// First byte of the content is the EIP-2718 TransactionType.
	return content[0], content[1:], nil
}

// parseLegacyBody populates a FastTransaction from the inner RLP list bytes.
func parseLegacyBody(fields []byte, tx *Transaction) (err error) {
	var b []byte
	if b, fields, err = decodeBytes(fields); err != nil {
		return err
	}
	if tx.Nonce, err = bytesToUint64(b); err != nil {
		return err
	}

	if b, fields, err = decodeBytes(fields); err != nil {
		return err
	}
	tx.GasPrice.SetBytes(b)

	if b, fields, err = decodeBytes(fields); err != nil {
		return err
	}
	if tx.Gas, err = bytesToUint64(b); err != nil {
		return err
	}

	if fields, err = decodeTo(fields, &tx.To, &tx.Create); err != nil {
		return err
	}

	if b, fields, err = decodeBytes(fields); err != nil {
		return err
	}
	tx.Value.SetBytes(b)

	if tx.Data, fields, err = decodeBytes(fields); err != nil {
		return err
	}

	if fields, err = decodeSignature(fields, &tx.V, &tx.R, &tx.S); err != nil {
		return err
	}

	if len(fields) > 0 {
		return ErrInvalidRLP
	}
	return nil
}

func parseDynamicFeeBody(fields []byte, tx *Transaction) (err error) {
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

	return decodeTypedTxTail(fields, &tx.Gas, &tx.To, &tx.Create, &tx.Value, &tx.Data, &tx.AccessList, &tx.V, &tx.R, &tx.S)
}

// decodeTo parses the 'To' address field, handling both EOA and Contract Creation (empty) targets.
func decodeTo(fields []byte, to *common.Address, create *bool) (remainder []byte, err error) {
	b, remainder, err := decodeBytes(fields)
	if err != nil {
		return nil, err
	}
	if len(b) == 20 {
		copy(to[:], b)
		*create = false
	} else if len(b) == 0 {
		*create = true
	} else {
		return nil, ErrInvalidRLP
	}
	return remainder, nil
}

// decodeSignature parses the V, R, and S signature values into uint256.Ints.
func decodeSignature(fields []byte, v, r, s *uint256.Int) (remainder []byte, err error) {
	var b []byte
	if b, fields, err = decodeBytes(fields); err != nil {
		return nil, err
	}
	v.SetBytes(b)
	if b, fields, err = decodeBytes(fields); err != nil {
		return nil, err
	}
	r.SetBytes(b)
	if b, fields, err = decodeBytes(fields); err != nil {
		return nil, err
	}
	s.SetBytes(b)
	return fields, nil
}

// decodeTypedTxTail parses the common fields shared by EIP-2930 and EIP-1559 transactions.
func decodeTypedTxTail(fields []byte, gas *uint64, to *common.Address, create *bool, value *uint256.Int, data, accessList *[]byte, v, r, s *uint256.Int) error {
	var b []byte
	var err error

	if b, fields, err = decodeBytes(fields); err != nil {
		return err
	}
	if *gas, err = bytesToUint64(b); err != nil {
		return err
	}

	if fields, err = decodeTo(fields, to, create); err != nil {
		return err
	}

	if b, fields, err = decodeBytes(fields); err != nil {
		return err
	}
	value.SetBytes(b)

	if *data, fields, err = decodeBytes(fields); err != nil {
		return err
	}
	if *accessList, fields, err = decodeList(fields); err != nil {
		return err
	}

	if fields, err = decodeSignature(fields, v, r, s); err != nil {
		return err
	}

	if len(fields) > 0 {
		return ErrInvalidRLP
	}
	return nil
}
