package types

import sharedtypes "github.com/pokt-network/poktroll/x/shared/types"

// GatewayNotUnstaking is the value of `unstake_session_end_height` if the
// gateway is not actively unbonding.
const GatewayNotUnstaking uint64 = iota

// IsUnbonding returns true if the gateway is actively unbonding.
// It determines if the gateway has submitted an unstake message, in which case
// the gateway has its UnstakeSessionEndHeight set.
func (s *Gateway) IsUnbonding() bool {
	return s.UnstakeSessionEndHeight != GatewayNotUnstaking
}

// GetGatewayUnbondingHeight returns the session end height at which the given
// gateway finishes unbonding.
func GetGatewayUnbondingHeight(
	sharedParams *sharedtypes.Params,
	gateway *Gateway,
) int64 {
	gatewayUnbondingPeriodBlocks := sharedParams.GatewayUnbondingPeriodSessions * sharedParams.NumBlocksPerSession

	return int64(gateway.UnstakeSessionEndHeight + gatewayUnbondingPeriodBlocks)
}

// IsActive returns whether the gateway is allowed to handle services at the given query height.
//
// Gateway activity rules:
// - Gateway without unstake message: Always active
// - Gateway with unstake message: Active until end of session containing unstake height
func (s *Gateway) IsActive(queryHeight int64) bool {
	return !s.IsUnbonding() || uint64(queryHeight) <= s.GetUnstakeSessionEndHeight()
}
