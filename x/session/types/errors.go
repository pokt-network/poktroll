package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/session module sentinel errors
var (
	ErrInvalidSigner                 = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrSample                        = sdkerrors.Register(ModuleName, 1101, "sample error")
	ErrSessionHydration              = sdkerrors.Register(ModuleName, 1102, "error during session hydration")
	ErrSessionAppNotFound            = sdkerrors.Register(ModuleName, 1103, "application for session not found not found ")
	ErrSessionAppNotStakedForService = sdkerrors.Register(ModuleName, 1104, "application in session not staked for requested service")
	ErrSessionSuppliersNotFound      = sdkerrors.Register(ModuleName, 1105, "no suppliers not found for session")
	ErrSessionInvalidAppAddress      = sdkerrors.Register(ModuleName, 1106, "invalid application address for session")
	ErrSessionInvalidService         = sdkerrors.Register(ModuleName, 1107, "invalid service in session")
	ErrSessionInvalidBlockHeight     = sdkerrors.Register(ModuleName, 1108, "invalid block height for session")
	ErrSessionInvalidSessionId       = sdkerrors.Register(ModuleName, 1109, "invalid sessionId")
)
