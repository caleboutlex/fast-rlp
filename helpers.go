package fastrlp

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
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

// decodeAccessList manually parses the RLP list of access tuples without reflection.
func decodeAccessList(b []byte, dest *types.AccessList) error {
	if len(b) == 0 {
		*dest = nil
		return nil
	}

	// Pre-scan to count tuples to avoid multiple append allocations
	count := 0
	temp := b
	for len(temp) > 0 {
		_, remainder, err := rlpSplit(temp)
		if err != nil {
			return err
		}
		count++
		temp = remainder
	}

	al := make(types.AccessList, 0, count)
	for len(b) > 0 {
		var tupleRLP, storageRLP, addrBytes, hashBytes []byte
		var err error

		// Each element is a list: [address, [storageKeys...]]
		if tupleRLP, b, err = decodeList(b); err != nil {
			return err
		}
		if addrBytes, tupleRLP, err = decodeBytes(tupleRLP); err != nil {
			return err
		}
		if storageRLP, tupleRLP, err = decodeList(tupleRLP); err != nil {
			return err
		}

		tuple := types.AccessTuple{
			Address: common.BytesToAddress(addrBytes),
		}

		// Parse the storage keys list
		for len(storageRLP) > 0 {
			if hashBytes, storageRLP, err = decodeBytes(storageRLP); err != nil {
				return err
			}
			tuple.StorageKeys = append(tuple.StorageKeys, common.BytesToHash(hashBytes))
		}
		al = append(al, tuple)
	}
	*dest = al
	return nil
}

// rlpSplit is a zero-allocation helper to get the head and tail of an RLP stream.
func rlpSplit(b []byte) (item, remainder []byte, err error) {
	if len(b) == 0 {
		return nil, nil, ErrInvalidRLP
	}
	prefix := b[0]
	var offset, contentLen uint64
	if prefix <= 0x7f {
		return b[:1], b[1:], nil
	} else if prefix <= 0xb7 {
		offset, contentLen = 1, uint64(prefix-0x80)
	} else if prefix <= 0xbf {
		l := uint64(prefix - 0xb7)
		offset = 1 + l
		for i := uint64(1); i < offset; i++ {
			contentLen = (contentLen << 8) | uint64(b[i])
		}
	} else if prefix <= 0xf7 {
		offset, contentLen = 1, uint64(prefix-0xc0)
	} else {
		l := uint64(prefix - 0xf7)
		offset = 1 + l
		for i := uint64(1); i < offset; i++ {
			contentLen = (contentLen << 8) | uint64(b[i])
		}
	}
	total := offset + contentLen
	if uint64(len(b)) < total {
		return nil, nil, ErrInvalidRLP
	}
	return b[:total], b[total:], nil
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

// setBigInt logic: reuse big.Int headers and optimize for small values.
func setBigInt(src *uint256.Int, dest *big.Int, buf []byte) {
	if src.IsUint64() {
		dest.SetUint64(src.Uint64())
	} else {
		src.WriteToSlice(buf)
		dest.SetBytes(buf)
	}
}
