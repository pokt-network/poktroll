package types

import sdk "github.com/cosmos/cosmos-sdk/types"

// TODO_TECHDEBT: Make sure this is used everywhere we validate components
// of the session header.
func (sh *SessionHeader) ValidateBasic() error {
	// Validate the application address
	if _, err := sdk.AccAddressFromBech32(sh.ApplicationAddress); err != nil {
		return ErrSessionInvalidAppAddress.Wrapf("invalid application address: %s; (%v)", sh.ApplicationAddress, err)
	}

	// Validate the session ID
	// TODO_TECHDEBT: Introduce a `SessionId#ValidateBasic` method.
	if sh.SessionId == "" {
		return ErrSessionInvalidSessionId.Wrapf("invalid session ID: %s", sh.SessionId)
	}

	// Validate the service
	// TODO_TECHDEBT: Introduce a `Service#ValidateBasic` method.
	if sh.Service == nil {
		return ErrSessionInvalidService.Wrapf("invalid service: %s", sh.Service)
	}

	// Check if session end height is greater than session start height
	if sh.SessionEndBlockHeight <= sh.SessionStartBlockHeight {
		return ErrSessionInvalidBlockHeight.Wrapf("session end block height cannot be less than or equal to session start block height")
	}

	return nil
}
