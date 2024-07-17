package shared

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

const codespace = "shared"

// x/shared module sentinel errors
var (
	ErrSharedInvalidSigner    = sdkerrors.Register(codespace, 1100, "expected gov account as only signer for proposal message")
	ErrSharedInvalidAddress   = sdkerrors.Register(codespace, 1101, "invalid address")
	ErrSharedParamNameInvalid = sdkerrors.Register(codespace, 1102, "the provided param name is invalid")
	ErrSharedParamInvalid     = sdkerrors.Register(codespace, 1103, "the provided param is invalid")
	ErrSharedEmitEvent        = sdkerrors.Register(codespace, 1104, "failed to emit event")
)
