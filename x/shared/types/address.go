package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ValidateCanonicalAddress checks that address is valid bech32 AND in its canonical
// (lowercase) encoding. State is keyed and compared by the address string, so every
// encoding of one account must map to the same string.
func ValidateCanonicalAddress(address string) error {
	accAddress, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return err
	}
	if canonical := accAddress.String(); address != canonical {
		return fmt.Errorf("address %q is not in canonical form %q", address, canonical)
	}
	return nil
}
