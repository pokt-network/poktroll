package types

// DONTCOVER

import sdkerrors "cosmossdk.io/errors"

// x/service module sentinel errors
var (
	ErrServiceInvalidSigner               = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrServiceDuplicateIndex              = sdkerrors.Register(ModuleName, 1101, "duplicate index when adding a new service")
	ErrServiceInvalidAddress              = sdkerrors.Register(ModuleName, 1102, "invalid address when adding a new service")
	ErrServiceMissingID                   = sdkerrors.Register(ModuleName, 1103, "missing service ID")
	ErrServiceMissingName                 = sdkerrors.Register(ModuleName, 1104, "missing service name")
	ErrServiceAlreadyExists               = sdkerrors.Register(ModuleName, 1105, "service already exists")
	ErrServiceInvalidServiceFee           = sdkerrors.Register(ModuleName, 1106, "invalid ServiceFee")
	ErrServiceAccountNotFound             = sdkerrors.Register(ModuleName, 1107, "account not found")
	ErrServiceNotEnoughFunds              = sdkerrors.Register(ModuleName, 1108, "not enough funds to add service")
	ErrServiceFailedToDeductFee           = sdkerrors.Register(ModuleName, 1109, "failed to deduct fee")
	ErrServiceInvalidRelayResponse        = sdkerrors.Register(ModuleName, 1110, "invalid relay response")
	ErrServiceInvalidRelayRequest         = sdkerrors.Register(ModuleName, 1111, "invalid relay request")
	ErrServiceInvalidComputeUnitsPerRelay = sdkerrors.Register(ModuleName, 1112, "invalid compute units per relay")
)
