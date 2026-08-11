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

// IsUnbonding mirrors Gateway.IsUnbonding for the decode-only projection.
func (l *GatewayLifecycle) IsUnbonding() bool {
	return l.UnstakeSessionEndHeight != GatewayNotUnstaking
}

// IsActive mirrors Gateway.IsActive for the decode-only projection.
func (l *GatewayLifecycle) IsActive(queryHeight int64) bool {
	return !l.IsUnbonding() || uint64(queryHeight) <= l.GetUnstakeSessionEndHeight()
}

// ToGateway inflates a decode-only GatewayLifecycle into a Gateway carrying a nil card.
//
// Callers that only need lifecycle state (unbonding checks, unbonding height, address)
// use this to reuse the Gateway helpers above without ever materializing a card. The
// result MUST NOT be written back to state: doing so would erase the gateway's stored
// card. Every current caller either only reads it, or passes it to UnbondGateway, which
// removes the record rather than re-storing it.
func (l *GatewayLifecycle) ToGateway() Gateway {
	return Gateway{
		Address:                 l.Address,
		Stake:                   l.Stake,
		UnstakeSessionEndHeight: l.UnstakeSessionEndHeight,
	}
}
