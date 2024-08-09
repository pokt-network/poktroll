package types

// DONTCOVER

import sdkerrors "cosmossdk.io/errors"

// x/supplier module sentinel errors
var (
	ErrSupplierInvalidSigner             = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrSupplierInvalidStake              = sdkerrors.Register(ModuleName, 1101, "invalid supplier stake")
	ErrSupplierInvalidAddress            = sdkerrors.Register(ModuleName, 1102, "invalid address")
	ErrSupplierNotFound                  = sdkerrors.Register(ModuleName, 1103, "supplier not found")
	ErrSupplierInvalidServiceConfig      = sdkerrors.Register(ModuleName, 1104, "invalid service config")
	ErrSupplierInvalidSessionStartHeight = sdkerrors.Register(ModuleName, 1105, "invalid session start height")
	ErrSupplierInvalidSessionId          = sdkerrors.Register(ModuleName, 1106, "invalid session ID")
	ErrSupplierInvalidService            = sdkerrors.Register(ModuleName, 1107, "invalid service in supplier")
	ErrSupplierInvalidSessionEndHeight   = sdkerrors.Register(ModuleName, 1108, "invalid session ending height")
	ErrSupplierIsUnstaking               = sdkerrors.Register(ModuleName, 1109, "supplier is in unbonding period")
	ErrSupplierParamsInvalid             = sdkerrors.Register(ModuleName, 1110, "invalid supplier params")
	ErrSupplierServiceNotFound           = sdkerrors.Register(ModuleName, 1111, "service not found")
)
