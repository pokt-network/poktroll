package application

// DONTCOVER

import sdkerrors "cosmossdk.io/errors"

// Not using types.ModuleName as codespace due to cyclic dependency
const codespace = "application"

// x/application module sentinel errors
var (
	ErrAppInvalidSigner               = sdkerrors.Register(codespace, 1100, "expected gov account as only signer for proposal message")
	ErrAppInvalidStake                = sdkerrors.Register(codespace, 1101, "invalid application stake")
	ErrAppInvalidAddress              = sdkerrors.Register(codespace, 1102, "invalid application address")
	ErrAppUnauthorized                = sdkerrors.Register(codespace, 1103, "unauthorized application signer")
	ErrAppNotFound                    = sdkerrors.Register(codespace, 1104, "application not found")
	ErrAppInvalidServiceConfigs       = sdkerrors.Register(codespace, 1106, "invalid service configs")
	ErrAppGatewayNotFound             = sdkerrors.Register(codespace, 1107, "gateway not found")
	ErrAppInvalidGatewayAddress       = sdkerrors.Register(codespace, 1108, "invalid gateway address")
	ErrAppAlreadyDelegated            = sdkerrors.Register(codespace, 1109, "application already delegated to gateway")
	ErrAppMaxDelegatedGateways        = sdkerrors.Register(codespace, 1110, "maximum number of delegated gateways reached")
	ErrAppInvalidMaxDelegatedGateways = sdkerrors.Register(codespace, 1111, "invalid MaxDelegatedGateways parameter")
	ErrAppNotDelegated                = sdkerrors.Register(codespace, 1112, "application not delegated to gateway")
)
