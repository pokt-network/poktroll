package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// NOTE: Please note that `QueryGetClaimRequest` is not a `sdk.Msg`, and is therefore not a message/request
// that will be signable or invoke a state transition. However, following a similar `ValidateBasic` pattern
// allows us to localize & reuse validation logic.
func (query *QueryGetClaimRequest) ValidateBasic() error {
	// Validate the supplier address
	if _, err := sdk.AccAddressFromBech32(query.SupplierAddress); err != nil {
		return sdkerrors.Wrapf(ErrSupplierInvalidAddress, "invalid supplier address for claim being retrieved %s; (%v)", query.SupplierAddress, err)
	}

	// TODO_TECHDEBT: Validate the session ID once we have a deterministic way to generate it
	return nil
}
