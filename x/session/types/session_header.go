package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (sh *SessionHeader) ValidateBasic() error {
	if sh.ApplicationAddress == "" {
		return sdkerrors.Wrapf(ErrSessionInvalidAppAddress, "invalid application address: %s", sh.ApplicationAddress)
	}

	// Validate the session ID
	if sh.SessionId == "" {
		return sdkerrors.Wrapf(ErrSessionInvalidSessionId, "invalid session ID: %s", sh.SessionId)
	}

	// Validate the service
	if sh.Service == nil {
		return sdkerrors.Wrapf(ErrSessionInvalidService, "invalid service: %s", sh.Service)
	}

	// Validate the application address
	if _, err := sdk.AccAddressFromBech32(sh.ApplicationAddress); err != nil {
		return sdkerrors.Wrapf(ErrSessionInvalidAppAddress, "invalid application address: %s; (%v)", sh.ApplicationAddress, err)
	}

	// Check if session end height is greater than session start height
	if sh.SessionEndBlockHeight <= sh.SessionStartBlockHeight {
		return sdkerrors.Wrapf(ErrSessionInvalidBlockHeight, "session end block height must be greater than session start block height")
	}

	return nil
}
