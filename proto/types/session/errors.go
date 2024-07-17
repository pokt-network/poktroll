package session

// DONTCOVER

import sdkerrors "cosmossdk.io/errors"

const codespace = "session"

// x/session module sentinel errors
var (
	ErrSessionInvalidSigner          = sdkerrors.Register(codespace, 1100, "expected gov account as only signer for proposal message")
	ErrSessionHydration              = sdkerrors.Register(codespace, 1101, "error during session hydration")
	ErrSessionAppNotFound            = sdkerrors.Register(codespace, 1102, "application for session not found not found ")
	ErrSessionAppNotStakedForService = sdkerrors.Register(codespace, 1103, "application in session not staked for requested service")
	ErrSessionSuppliersNotFound      = sdkerrors.Register(codespace, 1104, "no suppliers not found for session")
	ErrSessionInvalidAppAddress      = sdkerrors.Register(codespace, 1105, "invalid application address for session")
	ErrSessionInvalidService         = sdkerrors.Register(codespace, 1106, "invalid service in session")
	ErrSessionInvalidBlockHeight     = sdkerrors.Register(codespace, 1107, "invalid block height for session")
	ErrSessionInvalidSessionId       = sdkerrors.Register(codespace, 1108, "invalid sessionId")
)
