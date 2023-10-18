package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/gateway module sentinel errors
var (
	ErrGatewayInvalidAddress = sdkerrors.Register(ModuleName, 1, "invalid gateway address")
	ErrGatewayInvalidStake   = sdkerrors.Register(ModuleName, 2, "invalid gateway stake")
	ErrGatewayUnauthorized   = sdkerrors.Register(ModuleName, 3, "unauthorized signer")
)
