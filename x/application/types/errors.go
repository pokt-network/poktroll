package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/application module sentinel errors
var (
	ErrAppInvalidStake          = sdkerrors.Register(ModuleName, 1, "invalid application stake")
	ErrAppInvalidAddress        = sdkerrors.Register(ModuleName, 2, "invalid application address")
	ErrAppUnauthorized          = sdkerrors.Register(ModuleName, 3, "unauthorized application signer")
	ErrAppNotFound              = sdkerrors.Register(ModuleName, 4, "application not found")
	ErrAppInvalidServiceConfigs = sdkerrors.Register(ModuleName, 6, "invalid service configs")
	ErrAppGatewayNotFound       = sdkerrors.Register(ModuleName, 7, "gateway not found")
	ErrAppInvalidGatewayAddress = sdkerrors.Register(ModuleName, 8, "invalid gateway address")
	ErrAppAlreadyDelegated      = sdkerrors.Register(ModuleName, 9, "application already delegated to gateway")
)
