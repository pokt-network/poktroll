package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/supplier module sentinel errors
var (
	ErrSupplierInvalidStake   = sdkerrors.Register(ModuleName, 1, "invalid supplier stake")
	ErrSupplierInvalidAddress = sdkerrors.Register(ModuleName, 2, "invalid supplier address")
	ErrSupplierUnauthorized   = sdkerrors.Register(ModuleName, 3, "unauthorized supplier signer")
)
