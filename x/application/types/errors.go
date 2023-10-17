package types

// DONTCOVER

import (
	errorsmod "cosmossdk.io/errors"
)

// x/application module sentinel errors
var (
	ErrAppInvalidStake   = errorsmod.Register(ModuleName, 1, "invalid application stake")
	ErrAppInvalidAddress = errorsmod.Register(ModuleName, 2, "invalid application address")
	ErrAppUnauthorized   = errorsmod.Register(ModuleName, 3, "unauthorized application signer")
	ErrAppNotFound       = errorsmod.Register(ModuleName, 4, "application not found")
)
