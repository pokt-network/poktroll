package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/service module sentinel errors
var (
	ErrServiceDuplicateIndex    = sdkerrors.Register(ModuleName, 1, "duplicate index when adding a new service")
	ErrServiceInvalidAddress    = sdkerrors.Register(ModuleName, 2, "invalid address when adding a new service")
	ErrServiceMissingID         = sdkerrors.Register(ModuleName, 3, "missing service ID")
	ErrServiceMissingName       = sdkerrors.Register(ModuleName, 4, "missing service name")
	ErrServiceAlreadyExists     = sdkerrors.Register(ModuleName, 5, "service already exists")
	ErrServiceInvalidServiceFee = sdkerrors.Register(ModuleName, 6, "invalid service fee")
	ErrServiceAccountNotFound   = sdkerrors.Register(ModuleName, 7, "account not found")
	ErrServiceNotEnoughFunds    = sdkerrors.Register(ModuleName, 8, "not enough funds to add service")
	ErrServiceFailedToDeductFee = sdkerrors.Register(ModuleName, 9, "failed to deduct fee")
	ErrInvalidSigner            = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
)
