# fast-rlp: A High-Performance EVM Transaction Decoder

`fast-rlp` is a specialized, high-performance RLP (Recursive Length Prefix) decoder designed for high-frequency EVM infrastructure and MEV searchers.

It aims to provide a faster, lower-latency alternative to the standard `go-ethereum/rlp` package by replacing its reflection-based decoding with a direct, zero-allocation parsing strategy for critical hot-paths like mempool ingestion.

## The Bottleneck: Why this exists

In high-TPS environments, the standard `go-ethereum/rlp` package is a hidden latency trap. Because it relies on Go's `reflect` package to dynamically map byte streams to arbitrary `interface{}` structs, it triggers unavoidable heap allocations. 
More allocations = more Garbage Collection (GC) pressure = latency spikes during critical mempool ingest phases.

`fast-rlp` solves this by abandoning generic interfaces. It uses static, hardcoded decoding paths and $O(1)$ byte-slicing to map raw RLP streams directly into strongly typed EVM structs.

## The Proof (Benchmarks)

*Note: Benchmarks comparing standard `go-ethereum/rlp` decoding vs `fast-rlp` zero-allocation `ParseTransaction` path.*

| Scenario | Decoder | ns/op | allocs/op | B/op |
| :--- | :--- | :--- | :--- | :--- |
| **EIP-1559** | `go-ethereum/rlp` | `843.0 ns` | `19` | `512 B` |
| **EIP-1559** | `fast-rlp (Parse)` | **`53.5 ns`** | **`0`** | **`0 B`** |
| **Legacy** | `go-ethereum/rlp` | `598.6 ns` | `14` | `360 B` |
| **Legacy** | `fast-rlp (Parse)` | **`34.7 ns`** | **`0`** | **`0 B`** |
| **AccessList** | `go-ethereum/rlp` | `1097.0 ns` | `22` | `856 B` |
| **AccessList** | `fast-rlp (Parse)` | **`55.3 ns`** | **`0`** | **`0 B`** |

**Performance Gain:** ~92-95% reduction in latency across all transaction types.
**Zero-Allocation:** The `ParseTransaction` path provides a unified, zero-overhead entry point for all EVM transaction types.

## Core Architecture

1. **No Reflection:** Strict adherence to explicit pointer passing (`*types.Transaction`). No `interface{}`.
2. **Zero-Copy Slicing:** Instead of allocating new byte slices or relying on `io.Reader` streams, `fast-rlp` calculates payload offsets directly from the prefix and maps sub-slices of the original memory block.
3. **Two-Tier Decoding:** 
    - **`ParseTransaction`**: The "Hot Path". Zero-allocation parsing into a unified `Transaction` struct for immediate filtering.
    - **`DecodeTransaction`**: A convenience bridge to standard `go-ethereum` types.

## How Transaction is Constructed

The unified `Transaction` struct is specifically designed to eliminate heap allocations and pointer chasing:

*   **Envelope Detection:** Identifies the EIP-2718 transaction type (e.g., `0x02` for EIP-1559).
*   **Field Slicing:** The decoder uses O(1) slicing to extract field boundaries.
*   **Data Mapping:** 
    *   Large fields like `Data` and `AccessList` are returned as **slices** of the original input buffer.
    *   **Value Types:** Numeric fields use `uint256.Int`. Unlike `big.Int`, these are stack-allocated value types (fixed-size arrays), preventing heap allocations.
    *   **Flat Address:** The `To` address is a flat `common.Address` array. We avoid `*common.Address` (which is a heap-allocated pointer in Geth) by using a `Create` boolean flag to signify contract creation.

This layout ensures that the memory footprint of the parsing operation is exactly zero bytes on the heap, keeping the CPU cache warm and the GC idle.

## MEV Searcher Context: Filter then Decode

In MEV, 99.9% of mempool transactions are noise. Decoding every transaction into a `types.Transaction` triggers thousands of wasted allocations per second.

With `fast-rlp`, you parse transactions with **zero memory overhead**, filter out the noise, and only pay the allocation cost for transactions you actually intend to simulate or bundle.

```go
package main

import (
    "bytes"
    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/core/types"
    "github.com/caleboutlex/fast-rlp"
)

func main() {
    rawEIP1559Tx := []byte{ /* ... */ } 
    
    // 1. Zero-allocation parse
    var fastTx fastrlp.Transaction
    if err := fastrlp.ParseTransaction(rawEIP1559Tx, &fastTx); err != nil {
        return
    }

    // 2. High-speed filtering (No GC pressure)
    // Example: Only simulate if the 'To' address is a specific DEX contract
    dexAddress := common.HexToAddress("0x...")
    if fastTx.To != dexAddress {
        return 
    }

    // 3. Convert to Geth type only when necessary
    tx, err := fastTx.ToTx()
    // ... simulate or bundle ...
}
```

## Usage (Standard Replacement)

For general use cases where you need a standard Geth object immediately:

```go
func handleTx(data []byte) {
    // This acts as a faster version of rlp.DecodeBytes
    tx, err := fastrlp.DecodeTransaction(data)
    if err != nil {
        return
    }
    
    _ = tx.Hash()
}