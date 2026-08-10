package types

import "encoding/binary"

const (
	// ServiceComputeUnitsPerRelayHistoryKeyPrefix is the prefix for storing the
	// historical compute_units_per_relay (cupr) of a service.
	// Key format: ServiceComputeUnitsPerRelayHistoryKeyPrefix | serviceId | "/" | BigEndian(effectiveHeight)
	// This enables efficient range queries to find the cupr effective at a given height.
	ServiceComputeUnitsPerRelayHistoryKeyPrefix = "ServiceComputeUnitsPerRelay/history/"
)

// ServiceComputeUnitsPerRelayHistoryKey returns the store key for a service's cupr
// at a given effective height. Uses big-endian encoding so lexicographic ordering
// matches numeric ordering (required for the reverse-iteration at-height lookup).
func ServiceComputeUnitsPerRelayHistoryKey(serviceId string, effectiveHeight int64) []byte {
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, uint64(effectiveHeight))

	key := append([]byte(serviceId), []byte("/")...)
	key = append(key, heightBytes...)
	return append([]byte(ServiceComputeUnitsPerRelayHistoryKeyPrefix), key...)
}

// ServiceComputeUnitsPerRelayHistoryKeyPrefixForService returns the prefix for all
// cupr history entries of a service.
func ServiceComputeUnitsPerRelayHistoryKeyPrefixForService(serviceId string) []byte {
	key := append([]byte(serviceId), []byte("/")...)
	return append([]byte(ServiceComputeUnitsPerRelayHistoryKeyPrefix), key...)
}
