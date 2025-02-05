package query

import (
	"context"

	"cosmossdk.io/depinject"
	"github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/client"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ client.ServiceQueryClient = (*serviceQuerier)(nil)

// serviceQuerier is a wrapper around the servicetypes.QueryClient that enables the
// querying of onchain service information through a single exposed method
// which returns a sharedtypes.Service struct
type serviceQuerier struct {
	clientConn     grpc.ClientConn
	serviceQuerier servicetypes.QueryClient

	// servicesCache caches serviceQueryClient.Service requests
	servicesCache KeyValueCache[sharedtypes.Service]
	// relayMiningDifficultyCache caches serviceQueryClient.RelayMiningDifficulty requests
	relayMiningDifficultyCache KeyValueCache[servicetypes.RelayMiningDifficulty]
}

// NewServiceQuerier returns a new instance of a client.ServiceQueryClient by
// injecting the dependecies provided by the depinject.Config.
//
// Required dependencies:
// - clientCtx (grpc.ClientConn)
func NewServiceQuerier(deps depinject.Config) (client.ServiceQueryClient, error) {
	servq := &serviceQuerier{}

	if err := depinject.Inject(
		deps,
		&servq.clientConn,
		&servq.servicesCache,
		&servq.relayMiningDifficultyCache,
	); err != nil {
		return nil, err
	}

	servq.serviceQuerier = servicetypes.NewQueryClient(servq.clientConn)

	return servq, nil
}

// GetService returns a sharedtypes.Service struct for a given serviceId.
// It implements the ServiceQueryClient#GetService function.
func (servq *serviceQuerier) GetService(
	ctx context.Context,
	serviceId string,
) (sharedtypes.Service, error) {
	// Check if the service is present in the cache.
	if service, found := servq.servicesCache.Get(serviceId); found {
		return service, nil
	}

	req := &servicetypes.QueryGetServiceRequest{
		Id: serviceId,
	}

	res, err := servq.serviceQuerier.Service(ctx, req)
	if err != nil {
		return sharedtypes.Service{}, ErrQueryRetrieveService.Wrapf(
			"serviceId: %s; error: [%v]",
			serviceId, err,
		)
	}

	// Cache the service for future use.
	servq.servicesCache.Set(serviceId, res.Service)
	return res.Service, nil
}

// GetServiceRelayDifficulty queries the onchain data for
// the relay mining difficulty associated with the given service.
func (servq *serviceQuerier) GetServiceRelayDifficulty(
	ctx context.Context,
	serviceId string,
) (servicetypes.RelayMiningDifficulty, error) {
	// Check if the relay mining difficulty is present in the cache.
	if relayMiningDifficulty, found := servq.relayMiningDifficultyCache.Get(serviceId); found {
		return relayMiningDifficulty, nil
	}

	req := &servicetypes.QueryGetRelayMiningDifficultyRequest{
		ServiceId: serviceId,
	}

	res, err := servq.serviceQuerier.RelayMiningDifficulty(ctx, req)
	if err != nil {
		return servicetypes.RelayMiningDifficulty{}, err
	}

	// Cache the relay mining difficulty for future use.
	servq.relayMiningDifficultyCache.Set(serviceId, res.RelayMiningDifficulty)
	return res.RelayMiningDifficulty, nil
}
