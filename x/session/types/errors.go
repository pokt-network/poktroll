package types

// DONTCOVER

import sdkerrors "cosmossdk.io/errors"

// x/session module sentinel errors
var (
	ErrSessionInvalidSigner          = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrSessionHydration              = sdkerrors.Register(ModuleName, 1101, "error during session hydration")
	ErrSessionAppNotFound            = sdkerrors.Register(ModuleName, 1102, "application for session not found not found ")
	ErrSessionAppNotStakedForService = sdkerrors.Register(ModuleName, 1103, "application in session not staked for requested service")
	ErrSessionSuppliersNotFound      = sdkerrors.Register(ModuleName, 1104, "no suppliers not found for session")
	ErrSessionInvalidAppAddress      = sdkerrors.Register(ModuleName, 1105, "invalid application address for session")
	ErrSessionInvalidService         = sdkerrors.Register(ModuleName, 1106, "invalid service in session")
	ErrSessionInvalidBlockHeight     = sdkerrors.Register(ModuleName, 1107, "invalid block height for session")
	ErrSessionInvalidSessionId       = sdkerrors.Register(ModuleName, 1108, "invalid sessionId")
	ErrSessionAppNotActive           = sdkerrors.Register(ModuleName, 1109, "application is not active")
)
