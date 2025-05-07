package types

import crypto "github.com/cosmos/cosmos-sdk/crypto/types"

// SetAddress is a convenience method for setting the address of a MorseAuthAccount.
func (morseAuthAcct *MorseAuthAccount) SetAddress(address crypto.Address) {
	morseAccount := morseAuthAcct.GetMorseAccount()
	morseAccount.Address = address
	morseAuthAcct.Value = &MorseAuthAccount_MorseAccount{
		MorseAccount: morseAccount,
	}
}
