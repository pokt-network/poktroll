package supplier

// DONTCOVER

import sdkerrors "cosmossdk.io/errors"

const codespace = "supplier"

// x/supplier module sentinel errors
var (
	ErrSupplierInvalidSigner             = sdkerrors.Register(codespace, 1100, "expected gov account as only signer for proposal message")
	ErrSupplierInvalidStake              = sdkerrors.Register(codespace, 1101, "invalid supplier stake")
	ErrSupplierInvalidAddress            = sdkerrors.Register(codespace, 1102, "invalid address")
	ErrSupplierUnauthorized              = sdkerrors.Register(codespace, 1103, "unauthorized supplier signer")
	ErrSupplierNotFound                  = sdkerrors.Register(codespace, 1104, "supplier not found")
	ErrSupplierInvalidServiceConfig      = sdkerrors.Register(codespace, 1105, "invalid service config")
	ErrSupplierInvalidSessionStartHeight = sdkerrors.Register(codespace, 1106, "invalid session start height")
	ErrSupplierInvalidSessionId          = sdkerrors.Register(codespace, 1107, "invalid session ID")
	ErrSupplierInvalidService            = sdkerrors.Register(codespace, 1108, "invalid service in supplier")
	ErrSupplierInvalidSessionEndHeight   = sdkerrors.Register(codespace, 1109, "invalid session ending height")
	ErrSupplierServiceNotFound           = sdkerrors.Register(codespace, 1110, "service not found")
)
