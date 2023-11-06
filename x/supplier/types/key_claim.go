package types

import "encoding/binary"

var _ binary.ByteOrder

const (
    // ClaimKeyPrefix is the prefix to retrieve all Claim
	ClaimKeyPrefix = "Claim/value/"
)

// ClaimKey returns the store key to retrieve a Claim from the index fields
func ClaimKey(
index string,
) []byte {
	var key []byte
    
    indexBytes := []byte(index)
    key = append(key, indexBytes...)
    key = append(key, []byte("/")...)
    
	return key
}