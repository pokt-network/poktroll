package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

var (
	ErrServiceDuplicateIndex = sdkerrors.Register(ModuleName, 1, "duplicate index")
	ErrServiceInvalidAddress = sdkerrors.Register(ModuleName, 2, "invalid address")
)
