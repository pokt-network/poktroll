package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// ValidateBasic performs basic stateless validation of a SessionHeader.
func (sh *SessionHeader) ValidateBasic() error {
	// Validate the application address
	if _, err := sdk.AccAddressFromBech32(sh.ApplicationAddress); err != nil {
		return ErrSessionInvalidAppAddress.Wrapf("%q; (%s)", sh.ApplicationAddress, err)
	}

	// Validate the session ID
	if len(sh.SessionId) == 0 {
		return ErrSessionInvalidSessionId.Wrapf("%q", sh.SessionId)
	}

	if !sharedtypes.IsValidServiceId(sh.GetServiceId()) {
		return ErrSessionInvalidService.Wrapf("invalid service ID: %q", sh.GetServiceId())
	}

	// Sessions can only start at height 1
	if sh.SessionStartBlockHeight <= 0 {
		return ErrSessionInvalidBlockHeight.Wrapf("sessions can only start at height 1 or greater")
	}

	// Check if session end height is greater than session start height
	if sh.SessionEndBlockHeight <= sh.SessionStartBlockHeight {
		return ErrSessionInvalidBlockHeight.Wrapf("session end block height cannot be less than or equal to session start block height")
	}

	return nil
}
