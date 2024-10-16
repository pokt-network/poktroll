package types

// DONTCOVER

import sdkerrors "cosmossdk.io/errors"

// x/supplier module sentinel errors
var (
	ErrSupplierInvalidSigner        = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrSupplierInvalidStake         = sdkerrors.Register(ModuleName, 1101, "invalid supplier stake")
	ErrSupplierInvalidAddress       = sdkerrors.Register(ModuleName, 1102, "invalid address")
	ErrSupplierNotFound             = sdkerrors.Register(ModuleName, 1103, "supplier not found")
	ErrSupplierInvalidServiceConfig = sdkerrors.Register(ModuleName, 1104, "invalid service config")
	ErrSupplierIsUnstaking          = sdkerrors.Register(ModuleName, 1105, "supplier is in unbonding period")
	ErrSupplierServiceNotFound      = sdkerrors.Register(ModuleName, 1106, "service not found")
	ErrSupplierParamInvalid         = sdkerrors.Register(ModuleName, 1107, "the provided param is invalid")
	ErrSupplierEmitEvent            = sdkerrors.Register(ModuleName, 1108, "failed to emit event")
)
