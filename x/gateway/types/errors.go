package types

import "cosmossdk.io/errors"

// DONTCOVER

// x/gateway module sentinel errors
var (
	ErrGatewayInvalidAddress = errors.Register(ModuleName, 1, "invalid gateway address")
	ErrGatewayInvalidStake   = errors.Register(ModuleName, 2, "invalid gateway stake")
	ErrGatewayUnauthorized   = errors.Register(ModuleName, 3, "unauthorized signer")
	ErrGatewayNotFound       = errors.Register(ModuleName, 4, "gateway not found")
)
