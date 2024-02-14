package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/application module sentinel errors
var (
	ErrAppInvalidSigner               = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrAppSample                      = sdkerrors.Register(ModuleName, 1101, "sample error")
	ErrAppInvalidStake                = sdkerrors.Register(ModuleName, 1102, "invalid application stake")
	ErrAppInvalidAddress              = sdkerrors.Register(ModuleName, 1103, "invalid application address")
	ErrAppUnauthorized                = sdkerrors.Register(ModuleName, 1104, "unauthorized application signer")
	ErrAppNotFound                    = sdkerrors.Register(ModuleName, 1105, "application not found")
	ErrAppInvalidServiceConfigs       = sdkerrors.Register(ModuleName, 1107, "invalid service configs")
	ErrAppGatewayNotFound             = sdkerrors.Register(ModuleName, 1108, "gateway not found")
	ErrAppInvalidGatewayAddress       = sdkerrors.Register(ModuleName, 1109, "invalid gateway address")
	ErrAppAlreadyDelegated            = sdkerrors.Register(ModuleName, 1110, "application already delegated to gateway")
	ErrAppMaxDelegatedGateways        = sdkerrors.Register(ModuleName, 1111, "maximum number of delegated gateways reached")
	ErrAppInvalidMaxDelegatedGateways = sdkerrors.Register(ModuleName, 1112, "invalid MaxDelegatedGateways parameter")
	ErrAppNotDelegated                = sdkerrors.Register(ModuleName, 1113, "application not delegated to gateway")
)
