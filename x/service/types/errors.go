package types

import (
	sdkerrors "cosmossdk.io/errors"
)

var (
	ErrServiceDuplicateIndex = sdkerrors.Register(ModuleName, 1, "duplicate index when adding a new service")
	ErrServiceInvalidAddress = sdkerrors.Register(ModuleName, 2, "invalid address when adding a new service")
	ErrServiceMissingID      = sdkerrors.Register(ModuleName, 3, "missing service ID")
	ErrServiceMissingName    = sdkerrors.Register(ModuleName, 4, "missing service name")
	ErrServiceAlreadyExists  = sdkerrors.Register(ModuleName, 5, "service already exists")
)
