package types

// DONTCOVER

import sdkerrors "cosmossdk.io/errors"

// x/gateway module sentinel errors
var (
	ErrGatewayInvalidSigner  = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrGatewayInvalidAddress = sdkerrors.Register(ModuleName, 1101, "invalid gateway address")
	ErrGatewayInvalidStake   = sdkerrors.Register(ModuleName, 1102, "invalid gateway stake")
	ErrGatewayUnauthorized   = sdkerrors.Register(ModuleName, 1103, "unauthorized signer")
	ErrGatewayNotFound       = sdkerrors.Register(ModuleName, 1104, "gateway not found")
	ErrGatewayParamInvalid   = sdkerrors.Register(ModuleName, 1105, "the provided param is invalid")
	ErrGatewayEmitEvent      = sdkerrors.Register(ModuleName, 1106, "unable to emit onchain event")
	ErrGatewayIsUnstaking    = sdkerrors.Register(ModuleName, 1107, "gateway is in unbonding period")
	ErrGatewayIsInactive     = sdkerrors.Register(ModuleName, 1108, "gateway is no longer active")
)
