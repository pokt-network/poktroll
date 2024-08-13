package keeper

import (
	"context"
	"fmt"
)

// getServiceCupr ensures that a service with the ServiceID of the given service
// exists.
// It returns the compute units per relay for the service with the given id.
func (k Keeper) getServiceCupr(
	ctx context.Context,
	serviceId string,
) (uint64, error) {
	logger := k.Logger().With("method", "getServiceCupr")

	service, found := k.serviceKeeper.GetService(ctx, serviceId)
	if !found {
		return 0, fmt.Errorf("service %s not found", serviceId)
	}

	logger.
		With("service_id", serviceId).
		Debug("got service for proof")

	return service.ComputeUnitsPerRelay, nil
}
