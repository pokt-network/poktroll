package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/shared module sentinel errors
var (
	ErrSharedInvalidSigner    = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrSharedInvalidAddress   = sdkerrors.Register(ModuleName, 1101, "invalid address")
	ErrSharedParamNameInvalid = sdkerrors.Register(ModuleName, 1102, "the provided param name is invalid")
	ErrSharedParamInvalid     = sdkerrors.Register(ModuleName, 1103, "the provided param is invalid")
)
