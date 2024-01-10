package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

var (
	ErrServiceDuplicateIndex = sdkerrors.Register(ModuleName, 1, "duplicate index")
	ErrServiceInvalidAddress = sdkerrors.Register(ModuleName, 2, "invalid address")
	ErrServiceMissingID      = sdkerrors.Register(ModuleName, 3, "missing service ID")
	ErrServiceMissingName    = sdkerrors.Register(ModuleName, 4, "missing service name")
)
