package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// ClaimKeyPrefix is the prefix to retrieve all Claim
	ClaimKeyPrefix = "Claim/value/"
)

// ClaimKey returns the store key to retrieve a Claim
func ClaimKey(sessionId, supplierAddr string) []byte {
	var key []byte

	sessionBz := []byte(sessionId)
	key = append(key, sessionBz...)
	key = append(key, []byte("/")...)

	supplierAddrBz := []byte(supplierAddr)
	key = append(key, supplierAddrBz...)
	key = append(key, []byte("/")...)

	return key
}
