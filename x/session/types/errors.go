package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/session module sentinel errors
var (
	ErrHydratingSession          = sdkerrors.Register(ModuleName, 1, "error during session hydration")
	ErrAppNotFound               = sdkerrors.Register(ModuleName, 2, "application not found")
	ErrSuppliersNotFound         = sdkerrors.Register(ModuleName, 3, "suppliers not found")
	ErrSessionInvalidAppAddress  = sdkerrors.Register(ModuleName, 4, "invalid application address")
	ErrSessionInvalidServiceId   = sdkerrors.Register(ModuleName, 5, "invalid serviceID")
	ErrSessionInvalidBlockHeight = sdkerrors.Register(ModuleName, 6, "invalid block height")
)
