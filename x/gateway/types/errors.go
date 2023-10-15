package types

// DONTCOVER

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/gateway module sentinel errors
var (
	ErrSample                = sdkerrors.Register(ModuleName, 1100, "sample error")
	ErrGatewayInvalidAddress = sdkerrors.Register(ModuleName, 1101, "invalid gateway address")
	ErrGatewayInvalidStake   = sdkerrors.Register(ModuleName, 1102, "invalid gateway stake")
)
