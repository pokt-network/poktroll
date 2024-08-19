package session

import (
	"context"
	"fmt"

	"github.com/pokt-network/poktroll/x/service/types"
)

// getServiceComputeUnitsPerRelay returns the compute units per relay for the service specified in
// the relay request's metadata. The session manager's service query client is used to fetch the onchain
// service.
func (rs *relayerSessionsManager) getServiceComputeUnitsPerRelay(
	ctx context.Context,
	relayRequestMetadata *types.RelayRequestMetadata,
) (uint64, error) {
	sessionHeader := relayRequestMetadata.SessionHeader
	if sessionHeader.Service == nil {
		return 0, fmt.Errorf("getServiceComputeUnitsPerRelay: received nil service")
	}

	service, err := rs.serviceQueryClient.GetService(ctx, sessionHeader.Service.Id)
	if err != nil {
		return 0, fmt.Errorf("getServiceComputeUnitsPerRelay: could not get on-chain service %s: %w",
			sessionHeader.Service.Id,
			err,
		)
	}

	return service.ComputeUnitsPerRelay, nil
}
