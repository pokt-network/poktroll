package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/gateway module sentinel errors
var (
	ErrInvalidSigner         = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrSample                = sdkerrors.Register(ModuleName, 1101, "sample error")
	ErrGatewayInvalidAddress = sdkerrors.Register(ModuleName, 1102, "invalid gateway address")
	ErrGatewayInvalidStake   = sdkerrors.Register(ModuleName, 1103, "invalid gateway stake")
	ErrGatewayUnauthorized   = sdkerrors.Register(ModuleName, 1104, "unauthorized signer")
	ErrGatewayNotFound       = sdkerrors.Register(ModuleName, 1105, "gateway not found")
)
