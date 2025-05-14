package types

// ┌────────────────────────────────────────────────────────────────────────────────────┐
// │ 🔑  KV-Store Key Formats (all segments end with "/")                               │
// ├────────────────────────────────────────────────────────────────────────────────────┤
// │  Function / Constant                       Key Layout                              │
// │────────────────────────────────────────────────────────────────────────────────────│
// │ ApplicationKey()                          Application/address/                     │
// │                                           └── <AppAddr>/                           │
// │                                                                                    │
// │ ApplicationUnstakingKeyPrefix   +         Application/unstaking/                   │
// │                                           └── <AppAddr>/                           │
// │                                                                                    │
// │ ApplicationTransferKeyPrefix    +         Application/transfer/                    │
// │                                           └── <AppAddr>/                           │
// │                                                                                    │
// │ UndelegationKey()                         Application/undelegation/                │
// │                                           └── <AppAddr>/                           │
// │                                               <GatewayAddr>/                       │
// │                                                                                    │
// │ DelegationKey()                           Application/delegation/                  │
// │                                           └── <GatewayAddr>/                       │
// │                                               <AppAddr>/                           │
// └────────────────────────────────────────────────────────────────────────────────────┘
//
// Legend
// • <AppAddr>: UTF-8 bytes of the bech-32 or hex-encoded application address
// • <GatewayAddr>: UTF-8 bytes of the bech-32 or hex-encoded gateway address
// • Every segment (including addresses) is terminated with "/" for easy prefix scans

import "encoding/binary"

const (
	// ModuleName defines the module name
	// - Used as the base for store keys
	ModuleName = "application"

	// StoreKey defines the primary module store key
	// - Used for persistent storage
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	// - Used for volatile/in-memory storage
	MemStoreKey = "mem_application"

	// ParamsUpdateKeyPrefix defines the prefix for params updates.
	// This is used to store params updates at a specific height.
	ParamsUpdateKeyPrefix = "application_params_update/effective_height/"
)

// ParamsKey is the key for module parameters
var ParamsKey = []byte("p_application")

// KeyPrefix returns the prefix as a byte slice
func KeyPrefix(p string) []byte { return []byte(p) }

var _ binary.ByteOrder

const (
	// ApplicationKeyPrefix retrieves all Applications
	// - Prefix: Application/address/
	ApplicationKeyPrefix = "Application/address/"

	// ApplicationUnstakingKeyPrefix indexes unstaking applications
	// - Prefix: Application/unstaking/
	ApplicationUnstakingKeyPrefix = "Application/unstaking/"

	// ApplicationTransferKeyPrefix indexes applications being transferred
	// - Prefix: Application/transfer/
	ApplicationTransferKeyPrefix = "Application/transfer/"

	// UndelegationKeyPrefix indexes pending undelegations
	// - Prefix: Application/undelegation/
	UndelegationKeyPrefix = "Application/undelegation/"

	// DelegationKeyPrefix indexes applications delegating to gateways
	// - Prefix: Application/delegation/
	DelegationKeyPrefix = "Application/delegation/"
)

// ApplicationKey returns the store key to retrieve an Application from the index fields.
// - Key format: Application/address/<AppAddr>/
// - <AppAddr>: bech-32 or hex-encoded application address (UTF-8 bytes)
func ApplicationKey(appAddr string) []byte {
	return StringKey(appAddr)
}

// UndelegationKey returns the store key for an undelegation.
// - Key format: Application/undelegation/<AppAddr>/<GatewayAddr>/
// - <AppAddr>: bech-32 or hex-encoded application address (UTF-8 bytes)
// - <GatewayAddr>: bech-32 or hex-encoded gateway address (UTF-8 bytes)
// - Ordering: Application address first for efficient prefix scans by app
func UndelegationKey(appAddr, gatewayAddr string) []byte {
	var key []byte

	appKey := StringKey(appAddr)
	key = append(key, appKey...)

	gatewayKey := StringKey(gatewayAddr)
	key = append(key, gatewayKey...)

	return key
}

// DelegationKey returns the store key for a delegation.
//
// • Key format: Application/delegation/<GatewayAddr>/<AppAddr>/
// • <GatewayAddr>: bech-32 or hex-encoded gateway address (UTF-8 bytes)
// • <AppAddr>: bech-32 or hex-encoded application address (UTF-8 bytes)
// • Ordering: Gateway address first for efficient prefix scans by gateway
func DelegationKey(gatewayAddr, appAddr string) []byte {
	var key []byte

	gatewayKey := StringKey(gatewayAddr)
	key = append(key, gatewayKey...)

	appAddrKey := StringKey(appAddr)
	key = append(key, appAddrKey...)

	return key
}

// StringKey converts a string value to a byte slice for store keys.
//
// • Appends a "/" separator to the end for consistent prefix scanning.
// • Used for all address segments in key construction.
// - Used for all address segments in key construction.
func StringKey(strIndex string) []byte {
	var key []byte

	strIndexBz := []byte(strIndex)
	key = append(key, strIndexBz...)
	key = append(key, []byte("/")...)

	return key
}

// IntKey converts an integer value to a byte slice for use in store keys
// Appends a '/' separator to the end of the key for consistent prefix scanning
func IntKey(intIndex int64) []byte {
	var key []byte

	heightBz := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBz, uint64(intIndex))
	key = append(key, heightBz...)
	key = append(key, []byte("/")...)

	return key
}
