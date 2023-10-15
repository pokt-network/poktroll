package types

import "cosmossdk.io/errors"

// DONTCOVER

// x/gateway module sentinel errors
var (
	ErrSample                = errors.Register(ModuleName, 1100, "sample error")
	ErrGatewayInvalidAddress = errors.Register(ModuleName, 1101, "invalid gateway address")
	ErrGatewayInvalidStake   = errors.Register(ModuleName, 1102, "invalid gateway stake")
	ErrGatewayUnauthorized   = errors.Register(ModuleName, 1103, "unauthorized signer")
)
