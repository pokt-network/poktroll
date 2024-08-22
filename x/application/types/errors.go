package types

// DONTCOVER

import sdkerrors "cosmossdk.io/errors"

// x/application module sentinel errors
var (
	ErrAppInvalidSigner               = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrAppInvalidStake                = sdkerrors.Register(ModuleName, 1101, "invalid application stake")
	ErrAppInvalidAddress              = sdkerrors.Register(ModuleName, 1102, "invalid application address")
	ErrAppUnauthorized                = sdkerrors.Register(ModuleName, 1103, "unauthorized application signer")
	ErrAppNotFound                    = sdkerrors.Register(ModuleName, 1104, "application not found")
	ErrAppInvalidServiceConfigs       = sdkerrors.Register(ModuleName, 1106, "invalid service configs")
	ErrAppGatewayNotFound             = sdkerrors.Register(ModuleName, 1107, "gateway not found")
	ErrAppInvalidGatewayAddress       = sdkerrors.Register(ModuleName, 1108, "invalid gateway address")
	ErrAppAlreadyDelegated            = sdkerrors.Register(ModuleName, 1109, "application already delegated to gateway")
	ErrAppMaxDelegatedGateways        = sdkerrors.Register(ModuleName, 1110, "maximum number of delegated gateways reached")
	ErrAppInvalidMaxDelegatedGateways = sdkerrors.Register(ModuleName, 1111, "invalid MaxDelegatedGateways parameter")
	ErrAppNotDelegated                = sdkerrors.Register(ModuleName, 1112, "application not delegated to gateway")
	ErrAppDuplicateAddress            = sdkerrors.Register(ModuleName, 1113, "duplicate application address")
)
