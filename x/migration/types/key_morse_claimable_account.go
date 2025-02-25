package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// MorseClaimableAccountKeyPrefix is the prefix to retrieve all MorseClaimableAccount
	MorseClaimableAccountKeyPrefix = "MorseClaimableAccount/value/"
)

// MorseClaimableAccountKey returns the store key to retrieve a MorseClaimableAccount from the index fields
func MorseClaimableAccountKey(
	address string,
) []byte {
	var key []byte

	addressBytes := []byte(address)
	key = append(key, addressBytes...)
	key = append(key, []byte("/")...)

	return key
}
