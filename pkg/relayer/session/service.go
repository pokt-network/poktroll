package session

import (
	"context"

	"github.com/pokt-network/poktroll/x/service/types"
)

// getServiceComputeUnitsPerRelay returns the compute units per relay (cupr) to weight
// a relay by, pinned to the relay's SESSION-START height.
//
// cupr is read at the session-start height — NOT live — so every relay in a session is
// weighted uniformly by the value the chain validates the claim against. Reading the
// live cupr (which can change mid-session) produced mixed-weight SMSTs that the chain
// rejected with ErrProofComputeUnitsMismatch, forfeiting the whole session. Pinning to
// session-start matches the chain's claim check (which also reads cupr at session-start)
// and mirrors how relay mining difficulty is already pinned.
func (rs *relayerSessionsManager) getServiceComputeUnitsPerRelay(
	ctx context.Context,
	relayRequestMetadata *types.RelayRequestMetadata,
) (uint64, error) {
	sessionHeader := relayRequestMetadata.GetSessionHeader()
	computeUnitsPerRelay, err := rs.serviceQueryClient.GetServiceComputeUnitsPerRelayAtHeight(
		ctx,
		sessionHeader.ServiceId,
		sessionHeader.GetSessionStartBlockHeight(),
	)
	if err != nil {
		return 0, ErrSessionRelayMetaHasInvalidServiceID.Wrapf(
			"getServiceComputeUnitsPerRelay: could not get onchain compute units per relay for service %s at session start height %d: %v",
			sessionHeader.ServiceId,
			sessionHeader.GetSessionStartBlockHeight(),
			err,
		)
	}

	return computeUnitsPerRelay, nil
}
