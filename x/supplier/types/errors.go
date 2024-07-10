package types

// DONTCOVER

import sdkerrors "cosmossdk.io/errors"

// x/supplier module sentinel errors
var (
	ErrSupplierInvalidSigner             = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrSupplierInvalidStake              = sdkerrors.Register(ModuleName, 1101, "invalid supplier stake")
	ErrSupplierInvalidAddress            = sdkerrors.Register(ModuleName, 1102, "invalid address")
	ErrSupplierUnauthorized              = sdkerrors.Register(ModuleName, 1103, "unauthorized supplier signer")
	ErrSupplierNotFound                  = sdkerrors.Register(ModuleName, 1104, "supplier not found")
	ErrSupplierInvalidServiceConfig      = sdkerrors.Register(ModuleName, 1105, "invalid service config")
	ErrSupplierInvalidSessionStartHeight = sdkerrors.Register(ModuleName, 1106, "invalid session start height")
	ErrSupplierInvalidSessionId          = sdkerrors.Register(ModuleName, 1107, "invalid session ID")
	ErrSupplierInvalidService            = sdkerrors.Register(ModuleName, 1108, "invalid service in supplier")
	ErrSupplierInvalidSessionEndHeight   = sdkerrors.Register(ModuleName, 1109, "invalid session ending height")
	ErrSupplierUnbonding                 = sdkerrors.Register(ModuleName, 1110, "supplier is in unbonding period")
	ErrSupplierParamsInvalid             = sdkerrors.Register(ModuleName, 1111, "invalid supplier params")
)
