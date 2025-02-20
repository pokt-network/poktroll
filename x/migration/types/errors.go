package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/migration module sentinel errors
var (
	ErrInvalidSigner     = sdkerrors.Register(ModuleName, 1100, "expected x/gov module account as the only signer for migration state import messages")
	ErrMorseAccountState = sdkerrors.Register(ModuleName, 1101, "morse account state is invalid")
	ErrUnauthorized      = sdkerrors.Register(ModuleName, 1102, "unauthorized")
)
