package fastrlp

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

// FastTransaction holds the raw RLP data for any EVM transaction type.
type Transaction struct {
	Type byte

	// Hot fields optimized for filtering
	Value     uint256.Int
	GasPrice  uint256.Int // Legacy & EIP-2930
	GasTipCap uint256.Int // EIP-1559
	GasFeeCap uint256.Int // EIP-1559
	ChainID   uint256.Int

	To     common.Address
	Nonce  uint64
	Gas    uint64
	Create bool

	Data       []byte
	AccessList []byte
	V, R, S    uint256.Int
}

// ToTx converts the unified FastTransaction into a go-ethereum types.Transaction.
func (f *Transaction) ToTx() (*types.Transaction, error) {
	var buf [32]byte
	switch f.Type {
	case types.LegacyTxType:
		d := new(legacyData)
		d.tx.Nonce, d.tx.Gas = f.Nonce, f.Gas
		setBigInt(&f.GasPrice, &d.gasPrice, buf[:])
		d.tx.GasPrice = &d.gasPrice
		f.populateCommon(&d.tx.Value, &d.tx.To, &d.tx.Data, &d.tx.V, &d.tx.R, &d.tx.S, &d.value, &d.v, &d.r, &d.s, &d.to, buf[:])
		return types.NewTx(&d.tx), nil

	case types.AccessListTxType:
		d := new(accessListData)
		d.tx.Nonce, d.tx.Gas = f.Nonce, f.Gas
		setBigInt(&f.ChainID, &d.chainID, buf[:])
		d.tx.ChainID = &d.chainID
		setBigInt(&f.GasPrice, &d.gasPrice, buf[:])
		d.tx.GasPrice = &d.gasPrice
		f.populateCommon(&d.tx.Value, &d.tx.To, &d.tx.Data, &d.tx.V, &d.tx.R, &d.tx.S, &d.value, &d.v, &d.r, &d.s, &d.to, buf[:])
		if err := decodeAccessList(f.AccessList, &d.tx.AccessList); err != nil {
			return nil, err
		}
		return types.NewTx(&d.tx), nil

	case types.DynamicFeeTxType:
		d := new(eip1559Data)
		d.tx.Nonce, d.tx.Gas = f.Nonce, f.Gas
		setBigInt(&f.ChainID, &d.chainID, buf[:])
		d.tx.ChainID = &d.chainID
		setBigInt(&f.GasTipCap, &d.gasTipCap, buf[:])
		d.tx.GasTipCap = &d.gasTipCap
		setBigInt(&f.GasFeeCap, &d.gasFeeCap, buf[:])
		d.tx.GasFeeCap = &d.gasFeeCap
		f.populateCommon(&d.tx.Value, &d.tx.To, &d.tx.Data, &d.tx.V, &d.tx.R, &d.tx.S, &d.value, &d.v, &d.r, &d.s, &d.to, buf[:])
		if err := decodeAccessList(f.AccessList, &d.tx.AccessList); err != nil {
			return nil, err
		}
		return types.NewTx(&d.tx), nil

	default:
		return nil, ErrInvalidTxType
	}
}

func (f *Transaction) populateCommon(val **big.Int, to **common.Address, data *[]byte, v, r, s **big.Int, bVal, bV, bR, bS *big.Int, bTo *common.Address, buf []byte) {
	setBigInt(&f.Value, bVal, buf)
	*val = bVal
	if !f.Create {
		copy(bTo[:], f.To[:])
		*to = bTo
	}
	*data = f.Data
	setBigInt(&f.V, bV, buf)
	*v = bV
	setBigInt(&f.R, bR, buf)
	*r = bR
	setBigInt(&f.S, bS, buf)
	*s = bS
}

type legacyData struct {
	tx              types.LegacyTx
	gasPrice, value big.Int
	v, r, s         big.Int
	to              common.Address
}

type accessListData struct {
	tx                       types.AccessListTx
	chainID, gasPrice, value big.Int
	v, r, s                  big.Int
	to                       common.Address
}

type eip1559Data struct {
	tx                                   types.DynamicFeeTx
	chainID, gasTipCap, gasFeeCap, value big.Int
	v, r, s                              big.Int
	to                                   common.Address
}
