package types

// DONTCOVER

import (
	errorsmod "cosmossdk.io/errors"
)

// x/application module sentinel errors
var (
	ErrInvalidAppStake   = errorsmod.Register(ModuleName, 1, "invalid application stake")
	ErrInvalidAppAddress = errorsmod.Register(ModuleName, 2, "invalid application address")
)
