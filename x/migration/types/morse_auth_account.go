package types

import (
	"fmt"

	cmtjson "github.com/cometbft/cometbft/libs/json"
	crypto "github.com/cosmos/cosmos-sdk/crypto/types"
)

// SetAddress is a convenience method for setting the address of a MorseAuthAccount.
// It is intended for use in tests.
func (morseAuthAcct *MorseAuthAccount) SetAddress(address crypto.Address) error {
	morseAccount, err := morseAuthAcct.AsMorseAccount()
	if err != nil {
		return err
	}

	morseAccount.Address = address
	morseAccountJSONBz, err := cmtjson.Marshal(morseAccount)
	if err != nil {
		return err
	}

	morseAuthAcct.Value = morseAccountJSONBz
	return nil
}

// AsMorseAccount converts a MorseAuthAccount to a MorseAccount.
// If the account is not an externally owned account, an error is returned.
func (morseAuthAcct *MorseAuthAccount) AsMorseAccount() (*MorseAccount, error) {
	switch morseAuthAcct.Type {
	case MorseExternallyOwnedAccountType:
		morseAccount := new(MorseAccount)
		if err := cmtjson.Unmarshal(morseAuthAcct.Value, morseAccount); err != nil {
			return nil, err
		}
		return morseAccount, nil
	case MorseModuleAccountType:
		return nil, fmt.Errorf("refusing to unmarshal a module account as a morse account")
	default:
		return nil, fmt.Errorf("unknown account type %s", morseAuthAcct.Type)
	}
}

// AsMorseModuleAccount converts a MorseAuthAccount to a MorseModuleAccount.
// If the account is not a module account, an error is returned.
func (morseAuthAcct *MorseAuthAccount) AsMorseModuleAccount() (*MorseModuleAccount, error) {
	switch morseAuthAcct.Type {
	case MorseExternallyOwnedAccountType:
		return nil, fmt.Errorf("refusing to unmarshal an externally owned account as a module account")
	case MorseModuleAccountType:
		morseModuleAccount := new(MorseModuleAccount)
		if err := cmtjson.Unmarshal(morseAuthAcct.Value, morseModuleAccount); err != nil {
			return nil, err
		}
		return morseModuleAccount, nil
	default:
		return nil, fmt.Errorf("unknown account type %s", morseAuthAcct.Type)
	}
}
