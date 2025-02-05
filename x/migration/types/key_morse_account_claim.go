package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// MorseAccountClaimKeyPrefix is the prefix to retrieve all MorseAccountClaim
	MorseAccountClaimKeyPrefix = "MorseAccountClaim/value/"
)

// MorseAccountClaimKey returns the store key to retrieve a MorseAccountClaim from the index fields
func MorseAccountClaimKey(
	morseSrcAddress string,
) []byte {
	var key []byte

	morseSrcAddressBytes := []byte(morseSrcAddress)
	key = append(key, morseSrcAddressBytes...)
	key = append(key, []byte("/")...)

	return key
}
