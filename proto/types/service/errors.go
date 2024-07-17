package service

// DONTCOVER

import sdkerrors "cosmossdk.io/errors"

const codespace = "service"

// x/service module sentinel errors
var (
	ErrServiceInvalidSigner        = sdkerrors.Register(codespace, 1100, "expected gov account as only signer for proposal message")
	ErrServiceDuplicateIndex       = sdkerrors.Register(codespace, 1101, "duplicate index when adding a new service")
	ErrServiceInvalidAddress       = sdkerrors.Register(codespace, 1102, "invalid address when adding a new service")
	ErrServiceMissingID            = sdkerrors.Register(codespace, 1103, "missing service ID")
	ErrServiceMissingName          = sdkerrors.Register(codespace, 1104, "missing service name")
	ErrServiceAlreadyExists        = sdkerrors.Register(codespace, 1105, "service already exists")
	ErrServiceInvalidServiceFee    = sdkerrors.Register(codespace, 1106, "invalid ServiceFee")
	ErrServiceAccountNotFound      = sdkerrors.Register(codespace, 1107, "account not found")
	ErrServiceNotEnoughFunds       = sdkerrors.Register(codespace, 1108, "not enough funds to add service")
	ErrServiceFailedToDeductFee    = sdkerrors.Register(codespace, 1109, "failed to deduct fee")
	ErrServiceInvalidRelayResponse = sdkerrors.Register(codespace, 1110, "invalid relay response")
	ErrServiceInvalidRelayRequest  = sdkerrors.Register(codespace, 1111, "invalid relay request")
)
