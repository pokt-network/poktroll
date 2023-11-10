package types

import (
	fmt "fmt"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// NOTE: Please note that these messages are not of type `sdk.Msg`, and are therefore not a message/request
// that will be signable or invoke a state transition. However, following a similar `ValidateBasic` pattern
// allows us to localize & reuse validation logic.

// ValidateBasic performs basic (non-state-dependant) validation on a QueryGetClaimRequest.
func (query *QueryGetClaimRequest) ValidateBasic() error {
	// Validate the supplier address
	if _, err := sdk.AccAddressFromBech32(query.SupplierAddress); err != nil {
		return sdkerrors.Wrapf(ErrSupplierInvalidAddress, "invalid supplier address for claim being retrieved %s; (%v)", query.SupplierAddress, err)
	}

	// TODO_TECHDEBT: Validate the session ID once we have a deterministic way to generate it
	return nil
}

// ValidateBasic performs basic (non-state-dependant) validation on a QueryAllClaimsRequest.
func (query *QueryAllClaimsRequest) ValidateBasic() error {
	switch filter := query.Filter.(type) {
	case *QueryAllClaimsRequest_SupplierAddress:
		if _, err := sdk.AccAddressFromBech32(filter.SupplierAddress); err != nil {
			return sdkerrors.Wrapf(ErrSupplierInvalidAddress, "invalid supplier address for claims being retrieved %s; (%v)", filter.SupplierAddress, err)
		}

	case *QueryAllClaimsRequest_SessionId:
		// TODO_TECHDEBT: Validate the session ID once we have a deterministic way to generate it
		fmt.Println("TODO: SessionID check is currently a noop: ", filter.SessionId)

	case *QueryAllClaimsRequest_SessionEndHeight:
		if filter.SessionEndHeight < 0 {
			return sdkerrors.Wrapf(ErrSupplierInvalidSessionEndHeight, "invalid session end height for claims being retrieved %d", filter.SessionEndHeight)
		}

	default:
		// No filter is set
		fmt.Println("No specific filter set")
	}
	return nil
}
