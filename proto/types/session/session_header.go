package session

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	sharedhelpers "github.com/pokt-network/poktroll/x/shared/helpers"
)

// TODO_BETA: Ensure this is used everywhere a SessionHeader is validated.
// ValidateBasic performs basic stateless validation of a SessionHeader.
func (sh *SessionHeader) ValidateBasic() error {
	// Validate the application address
	if _, err := sdk.AccAddressFromBech32(sh.ApplicationAddress); err != nil {
		return ErrSessionInvalidAppAddress.Wrapf("invalid application address: %s; (%v)", sh.ApplicationAddress, err)
	}

	// Validate the session ID
	if len(sh.SessionId) == 0 {
		return ErrSessionInvalidSessionId.Wrapf("invalid session ID: %s", sh.SessionId)
	}

	// Validate the service
	if sh.Service == nil || !sharedhelpers.IsValidService(sh.Service) {
		return ErrSessionInvalidService.Wrapf("invalid service: %s", sh.Service)
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
