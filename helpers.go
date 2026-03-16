package fastrlp

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// decodeBytes returns the payload of an RLP-encoded string and the remaining buffer.
func decodeBytes(b []byte) (payload, remainder []byte, err error) {
	if len(b) == 0 {
		return nil, nil, ErrInvalidRLP
	}
	prefix := b[0]
	switch {
	case prefix <= 0x7f:
		// A single byte, itself is the payload.
		return b[:1], b[1:], nil
	case prefix <= 0xb7:
		// Short string.
		strLen := uint64(prefix - 0x80)
		if uint64(len(b)) < 1+strLen {
			return nil, nil, ErrInvalidRLP
		}
		end := 1 + strLen
		return b[1:end], b[end:], nil
	case prefix <= 0xbf:
		// Long string.
		lenOfLen := int(prefix - 0xb7)
		if len(b) < 1+lenOfLen {
			return nil, nil, ErrInvalidRLP
		}
		var strLen uint64
		for _, bt := range b[1 : 1+lenOfLen] {
			strLen = (strLen << 8) | uint64(bt)
		}
		if uint64(len(b)) < 1+uint64(lenOfLen)+strLen {
			return nil, nil, ErrInvalidRLP
		}
		start := 1 + lenOfLen
		end := start + int(strLen)
		return b[start:end], b[end:], nil
	default:
		// It's a list, which is not what we expect for a byte string.
		return nil, nil, ErrInvalidRLP
	}
}

// decodeList returns the payload of an RLP-encoded list and the remaining buffer.
func decodeList(b []byte) (payload, remainder []byte, err error) {
	if len(b) == 0 {
		return nil, nil, ErrInvalidRLP
	}
	prefix := b[0]
	switch {
	case prefix <= 0xbf:
		// Not a list.
		return nil, nil, ErrInvalidRLP
	case prefix <= 0xf7:
		// Short list.
		listLen := uint64(prefix - 0xc0)
		if uint64(len(b)) < 1+listLen {
			return nil, nil, ErrInvalidRLP
		}
		end := 1 + listLen
		return b[1:end], b[end:], nil
	default: // prefix >= 0xf8
		// Long list.
		lenOfLen := int(prefix - 0xf7)
		if len(b) < 1+lenOfLen {
			return nil, nil, ErrInvalidRLP
		}
		var listLen uint64
		for _, bt := range b[1 : 1+lenOfLen] {
			listLen = (listLen << 8) | uint64(bt)
		}
		if uint64(len(b)) < 1+uint64(lenOfLen)+listLen {
			return nil, nil, ErrInvalidRLP
		}
		start := 1 + lenOfLen
		end := start + int(listLen)
		return b[start:end], b[end:], nil
	}
}

// Helper to populate a pre-allocated big.Int.

func setBigInt(b []byte, bi *big.Int) {
	if len(b) > 0 {
		bi.SetBytes(b)
	}
}

func bytesToUint64(b []byte) (uint64, error) {
	if len(b) > 8 {
		return 0, ErrUintOverflow
	}
	var v uint64
	for _, bt := range b {
		v = (v << 8) | uint64(bt)
	}
	return v, nil
}

// decodeAccessList manually parses the RLP list of access tuples without reflection.
func decodeAccessList(b []byte) (types.AccessList, error) {
	if len(b) == 0 {
		return nil, nil
	}

	var al types.AccessList
	for len(b) > 0 {
		var tupleRLP, storageRLP, addrBytes, hashBytes []byte
		var err error

		// Each element is a list: [address, [storageKeys...]]
		if tupleRLP, b, err = decodeList(b); err != nil {
			return nil, err
		}
		if addrBytes, tupleRLP, err = decodeBytes(tupleRLP); err != nil {
			return nil, err
		}
		if storageRLP, tupleRLP, err = decodeList(tupleRLP); err != nil {
			return nil, err
		}

		tuple := types.AccessTuple{
			Address: common.BytesToAddress(addrBytes),
		}

		// Parse the storage keys list
		for len(storageRLP) > 0 {
			if hashBytes, storageRLP, err = decodeBytes(storageRLP); err != nil {
				return nil, err
			}
			tuple.StorageKeys = append(tuple.StorageKeys, common.BytesToHash(hashBytes))
		}
		al = append(al, tuple)
	}
	return al, nil
}
