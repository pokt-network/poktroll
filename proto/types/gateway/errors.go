package gateway

// DONTCOVER

import sdkerrors "cosmossdk.io/errors"

const codespace = "gateway"

// x/gateway module sentinel errors
var (
	ErrGatewayInvalidSigner  = sdkerrors.Register(codespace, 1100, "expected gov account as only signer for proposal message")
	ErrGatewayInvalidAddress = sdkerrors.Register(codespace, 1101, "invalid gateway address")
	ErrGatewayInvalidStake   = sdkerrors.Register(codespace, 1102, "invalid gateway stake")
	ErrGatewayUnauthorized   = sdkerrors.Register(codespace, 1103, "unauthorized signer")
	ErrGatewayNotFound       = sdkerrors.Register(codespace, 1104, "gateway not found")
)
