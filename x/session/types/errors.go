package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/session module sentinel errors
var (
	ErrHydratingSession = sdkerrors.Register(ModuleName, 1, "error during session hydration")
)
