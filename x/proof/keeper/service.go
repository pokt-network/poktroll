package keeper

import (
	"context"

	"github.com/pokt-network/pocket/x/proof/types"
)

// getServiceComputeUnitsPerRelay is used to ensure that a service with the ServiceID exists.
// exists.
// It returns the compute units per relay for the service with the given id.
func (k Keeper) getServiceComputeUnitsPerRelay(
	ctx context.Context,
	serviceId string,
) (uint64, error) {
	logger := k.Logger().With("method", "getServiceComputeUnitsPerRelay")

	service, found := k.serviceKeeper.GetService(ctx, serviceId)
	if !found {
		return 0, types.ErrProofServiceNotFound.Wrapf("service %s not found", serviceId)
	}

	logger.
		With("service_id", serviceId).
		Debug("got service for proof")

	return service.ComputeUnitsPerRelay, nil
}
