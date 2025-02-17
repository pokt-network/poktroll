package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/migration module sentinel errors
var (
	ErrInvalidSigner         = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrMorseAccountState     = sdkerrors.Register(ModuleName, 1101, "morse account state is invalid")
	ErrUnauthorized          = sdkerrors.Register(ModuleName, 1102, "unauthorized")
	ErrMorseClaimableAccount = sdkerrors.Register(ModuleName, 1103, "morse claimable account is invalid")
)
