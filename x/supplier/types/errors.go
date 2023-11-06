package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/supplier module sentinel errors
var (
	ErrSupplierInvalidStake              = sdkerrors.Register(ModuleName, 1, "invalid supplier stake")
	ErrSupplierInvalidAddress            = sdkerrors.Register(ModuleName, 2, "invalid supplier address")
	ErrSupplierUnauthorized              = sdkerrors.Register(ModuleName, 3, "unauthorized supplier signer")
	ErrSupplierNotFound                  = sdkerrors.Register(ModuleName, 4, "supplier not found")
	ErrSupplierInvalidServiceConfig      = sdkerrors.Register(ModuleName, 5, "invalid service config")
	ErrSupplierInvalidSessionStartHeight = sdkerrors.Register(ModuleName, 6, "invalid session start height")
	ErrSupplierInvalidSessionId          = sdkerrors.Register(ModuleName, 7, "invalid session ID")
	ErrSupplierInvalidService            = sdkerrors.Register(ModuleName, 8, "invalid service")
	ErrSupplierInvalidRootHash           = sdkerrors.Register(ModuleName, 9, "invalid root hash")
)
