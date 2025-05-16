package types

import (
	"encoding/binary"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// TODO_IN_THIS_COMMIT: comments / table...

var _ binary.ByteOrder

const (
	// MorseClaimableAccountKeyPrefix is the prefix to retrieve all MorseClaimableAccount
	MorseClaimableAccountKeyPrefix = "MorseClaimableAccount/morse_src_address/"

	// TODO_IN_THIS_COMMIT: godoc...
	MorseClaimableAccountMorseOutputAddressKeyPrefix = "MorseClaimableAccount/morse_output_address/"

	// TODO_IN_THIS_COMMIT: godoc...
	MorseClaimableAccountShannonDestAddressKeyPrefix = "MorseClaimableAccount/shannon_dest_address/"
)

// MorseClaimableAccountKey returns the store key to retrieve a MorseClaimableAccount from the index fields
func MorseClaimableAccountKey(
	morseSrcAddress string,
) []byte {
	return sharedtypes.StringKey(morseSrcAddress)
}

// TODO_IN_THIS_COMMIT: godoc...
func MorseClaimableAccountMorseOutputAddressKey(morseClaimableAccount MorseClaimableAccount) []byte {
	var key []byte

	morseOutputAddressKey := sharedtypes.StringKey(morseClaimableAccount.GetMorseOutputAddress())
	key = append(key, morseOutputAddressKey...)

	morseSrcAddressKey := sharedtypes.StringKey(morseClaimableAccount.GetMorseSrcAddress())
	key = append(key, morseSrcAddressKey...)

	return key
}

// TODO_IN_THIS_COMMIT: godoc...
func MorseClaimableAccountShannonDestAddressKey(morseClaimableAccount MorseClaimableAccount) []byte {
	var key []byte

	shannonDestAddressKey := sharedtypes.StringKey(morseClaimableAccount.GetShannonDestAddress())
	key = append(key, shannonDestAddressKey...)

	morseSrcAddressKey := sharedtypes.StringKey(morseClaimableAccount.GetMorseSrcAddress())
	key = append(key, morseSrcAddressKey...)

	return key
}
