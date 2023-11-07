package keyring

import (
	cosmoskeyring "github.com/cosmos/cosmos-sdk/crypto/keyring"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
)

// KeyNameToAddr attempts to retrieve the key with the given name from the
// given keyring and compute its address.
func KeyNameToAddr(
	keyName string,
	keyring cosmoskeyring.Keyring,
) (cosmostypes.AccAddress, error) {
	if keyName == "" {
		return nil, ErrEmptySigningKeyName
	}

	keyRecord, err := keyring.Key(keyName)
	if err != nil {
		return nil, ErrNoSuchSigningKey.Wrapf("name %q: %s", keyName, err)
	}

	signingAddr, err := keyRecord.GetAddress()
	if err != nil {
		return nil, ErrSigningKeyAddr.Wrapf("name %q: %s", keyName, err)
	}

	return signingAddr, nil
}
