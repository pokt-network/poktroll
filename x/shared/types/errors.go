package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/shared module sentinel errors
var (
	ErrInvalidSigner           = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrSessionParamNameInvalid = sdkerrors.Register(ModuleName, 1101, "the provided param name is invalid")
	ErrSessionParamInvalid     = sdkerrors.Register(ModuleName, 1102, "the provided param is invalid")
)
