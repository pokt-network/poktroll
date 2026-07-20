package keeper

import (
	"context"

	"github.com/pokt-network/poktroll/x/proof/types"
)

// getServiceComputeUnitsPerRelay returns the compute_units_per_relay (cupr) that was
// effective for the service at the given session-start height.
//
// cupr is pinned to the session-start height — NOT read live at claim creation — so
// an in-flight session is always validated against the cupr that was live when its
// relays were mined. Reading the live value previously forfeited every claim for a
// service whose cupr changed mid-session with ErrProofComputeUnitsMismatch, because
// the append-only SMST bakes the mine-time cupr while the chain checked the new one.
// This mirrors how relay mining difficulty is already pinned to session-start.
func (k Keeper) getServiceComputeUnitsPerRelay(
	ctx context.Context,
	serviceId string,
	sessionStartHeight int64,
) (uint64, error) {
	logger := k.Logger().With("method", "getServiceComputeUnitsPerRelay")

	computeUnitsPerRelay, found := k.serviceKeeper.GetServiceComputeUnitsPerRelayAtHeight(ctx, serviceId, sessionStartHeight)
	if !found {
		return 0, types.ErrProofServiceNotFound.Wrapf("service %s not found", serviceId)
	}

	logger.
		With("service_id", serviceId, "session_start_height", sessionStartHeight).
		Debug("got service compute units per relay at session start for proof")

	return computeUnitsPerRelay, nil
}
