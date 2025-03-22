package session

import (
	"context"

	"github.com/pokt-network/poktroll/pkg/retry"
	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// getServiceComputeUnitsPerRelay returns the compute units per relay for the service specified in
// the relay request's metadata. The session manager's service query client is used to fetch the onchain
// service.
func (rs *relayerSessionsManager) getServiceComputeUnitsPerRelay(
	ctx context.Context,
	relayRequestMetadata *types.RelayRequestMetadata,
) (uint64, error) {
	sessionHeader := relayRequestMetadata.GetSessionHeader()
	service, err := retry.Call(
		func() (sharedtypes.Service, error) {
			return rs.serviceQueryClient.GetService(ctx, sessionHeader.ServiceId)
		},
		retry.UntilNextBlock(ctx, rs.blockClient.CommittedBlocksSequence(ctx)),
	)
	if err != nil {
		return 0, ErrSessionRelayMetaHasInvalidServiceID.Wrapf(
			"getServiceComputeUnitsPerRelay: could not get onchain service %s: %v",
			sessionHeader.ServiceId,
			err,
		)
	}

	return service.GetComputeUnitsPerRelay(), nil
}
