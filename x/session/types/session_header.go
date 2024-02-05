package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TODO_TECHDEBT: Make sure this is used everywhere we validate components
// of the session header.
func (sh *SessionHeader) ValidateBasic() error {
	// Validate the application address
	if _, err := sdk.AccAddressFromBech32(sh.ApplicationAddress); err != nil {
		return sdkerrors.Wrapf(ErrSessionInvalidAppAddress, "invalid application address: %s; (%v)", sh.ApplicationAddress, err)
	}

	// Validate the session ID
	// TODO_TECHDEBT: Introduce a `SessionId#ValidateBasic` method.
	if sh.SessionId == "" {
		return sdkerrors.Wrapf(ErrSessionInvalidSessionId, "invalid session ID: %s", sh.SessionId)
	}

	// Validate the service
	// TODO_TECHDEBT: Introduce a `Service#ValidateBasic` method.
	if sh.Service == nil {
		return sdkerrors.Wrapf(ErrSessionInvalidService, "invalid service: %s", sh.Service)
	}

	// Check if session end height is greater than session start height
	if sh.SessionEndBlockHeight <= sh.SessionStartBlockHeight {
		return sdkerrors.Wrapf(ErrSessionInvalidBlockHeight, "session end block height cannot be less than or equal to session start block height")
	}

	return nil
}
