package query

import (
	"context"
	"sync"

	"cosmossdk.io/depinject"
	"github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/cache"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/retry"
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
	logger         polylog.Logger

	// servicesCache caches serviceQueryClient.Service query requests
	servicesCache cache.KeyValueCache[sharedtypes.Service]
	// relayMiningDifficultyCache caches serviceQueryClient.RelayMiningDifficulty query requests
	relayMiningDifficultyCache cache.KeyValueCache[servicetypes.RelayMiningDifficulty]
	// servicesMutex to protect cache access patterns for services and relay mining difficulties
	servicesMutex sync.Mutex

	// paramsCache caches serviceQueryClient.Params query requests
	paramsCache client.ParamsCache[servicetypes.Params]
	// paramsMutex to protect cache access patterns for params
	paramsMutex sync.Mutex
}

// NewServiceQuerier returns a new instance of a client.ServiceQueryClient by
// injecting the dependencies provided by the depinject.Config.
//
// Required dependencies:
// - clientCtx (grpc.ClientConn)
// - polylog.Logger
// - cache.KeyValueCache[sharedtypes.Service]
// - cache.KeyValueCache[servicetypes.RelayMiningDifficulty]
func NewServiceQuerier(deps depinject.Config) (client.ServiceQueryClient, error) {
	servq := &serviceQuerier{}

	if err := depinject.Inject(
		deps,
		&servq.clientConn,
		&servq.logger,
		&servq.servicesCache,
		&servq.relayMiningDifficultyCache,
		&servq.paramsCache,
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
	logger := servq.logger.With("query_client", "service", "method", "GetService")

	// Check if the service is present in the cache.
	if service, found := servq.servicesCache.Get(serviceId); found {
		logger.Debug().Msgf("service cache hit for service id key: %s", serviceId)
		return service, nil
	}

	// Use mutex to prevent multiple concurrent cache updates
	servq.servicesMutex.Lock()
	defer servq.servicesMutex.Unlock()

	// Double-check cache after acquiring lock (follows standard double-checked locking pattern)
	if service, found := servq.servicesCache.Get(serviceId); found {
		logger.Debug().Msgf("service cache hit for service id key after lock: %s", serviceId)
		return service, nil
	}

	logger.Debug().Msgf("service cache miss for service id key: %s", serviceId)

	req := &servicetypes.QueryGetServiceRequest{
		Id: serviceId,
	}
	res, err := retry.Call(ctx, func() (*servicetypes.QueryGetServiceResponse, error) {
		return servq.serviceQuerier.Service(ctx, req)
	}, retry.GetStrategy(ctx))
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
	logger := servq.logger.With("query_client", "service", "method", "GetServiceRelayDifficulty")

	// Check if the relay mining difficulty is present in the cache.
	if relayMiningDifficulty, found := servq.relayMiningDifficultyCache.Get(serviceId); found {
		logger.Debug().Msgf("relay mining difficulty cache hit for service id key: %s", serviceId)
		return relayMiningDifficulty, nil
	}

	// Use mutex to prevent multiple concurrent cache updates
	servq.servicesMutex.Lock()
	defer servq.servicesMutex.Unlock()

	// Double-check cache after acquiring lock (follows standard double-checked locking pattern)
	if relayMiningDifficulty, found := servq.relayMiningDifficultyCache.Get(serviceId); found {
		logger.Debug().Msgf("relay mining difficulty cache hit for service id key after lock: %s", serviceId)
		return relayMiningDifficulty, nil
	}

	logger.Debug().Msgf("relay mining difficulty cache miss for service id key: %s", serviceId)

	req := &servicetypes.QueryGetRelayMiningDifficultyRequest{
		ServiceId: serviceId,
	}
	res, err := retry.Call(ctx, func() (*servicetypes.QueryGetRelayMiningDifficultyResponse, error) {
		return servq.serviceQuerier.RelayMiningDifficulty(ctx, req)
	}, retry.GetStrategy(ctx))
	if err != nil {
		return servicetypes.RelayMiningDifficulty{}, err
	}

	// Cache the relay mining difficulty for future use.
	servq.relayMiningDifficultyCache.Set(serviceId, res.RelayMiningDifficulty)
	return res.RelayMiningDifficulty, nil
}

// GetParams returns the service module parameters.
func (servq *serviceQuerier) GetParams(ctx context.Context) (*servicetypes.Params, error) {
	logger := servq.logger.With("query_client", "service", "method", "GetParams")

	// Check if the service module parameters are present in the cache.
	if params, found := servq.paramsCache.Get(); found {
		logger.Debug().Msg("cache HIT for service params")
		return &params, nil
	}

	// Use mutex to prevent multiple concurrent cache updates
	servq.paramsMutex.Lock()
	defer servq.paramsMutex.Unlock()

	// Double-check cache after acquiring lock (follows standard double-checked locking pattern)
	if params, found := servq.paramsCache.Get(); found {
		logger.Debug().Msg("cache HIT for service params after lock")
		return &params, nil
	}

	logger.Debug().Msg("cache MISS for service params")

	req := servicetypes.QueryParamsRequest{}
	res, err := retry.Call(ctx, func() (*servicetypes.QueryParamsResponse, error) {
		return servq.serviceQuerier.Params(ctx, &req)
	}, retry.GetStrategy(ctx))
	if err != nil {
		return nil, err
	}

	// Cache the parameters for future queries.
	servq.paramsCache.Set(res.Params)
	return &res.Params, nil
}
