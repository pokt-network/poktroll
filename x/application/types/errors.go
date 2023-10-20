package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/application module sentinel errors
var (
	ErrAppInvalidStake   = sdkerrors.Register(ModuleName, 1, "invalid application stake")
	ErrAppInvalidAddress = sdkerrors.Register(ModuleName, 2, "invalid application address")
	ErrAppUnauthorized   = sdkerrors.Register(ModuleName, 3, "unauthorized application signer")
	ErrAppNotFound       = sdkerrors.Register(ModuleName, 4, "application not found")
)
