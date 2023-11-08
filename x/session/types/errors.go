package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/session module sentinel errors
var (
	ErrSessionHydration              = sdkerrors.Register(ModuleName, 1, "error during session hydration")
	ErrSessionAppNotFound            = sdkerrors.Register(ModuleName, 2, "application for session not found not found ")
	ErrSessionAppNotStakedForService = sdkerrors.Register(ModuleName, 3, "application in session not staked for requested service")
	ErrSessionSuppliersNotFound      = sdkerrors.Register(ModuleName, 4, "no suppliers not found for session")
	ErrSessionInvalidAppAddress      = sdkerrors.Register(ModuleName, 5, "invalid application address for session")
	ErrSessionInvalidService         = sdkerrors.Register(ModuleName, 6, "invalid service in session")
	ErrSessionInvalidBlockHeight     = sdkerrors.Register(ModuleName, 7, "invalid block height for session")
)
