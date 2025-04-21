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
	apptypes "github.com/pokt-network/poktroll/x/application/types"
)

var _ client.ApplicationQueryClient = (*appQuerier)(nil)

// appQuerier is a wrapper around the apptypes.QueryClient that enables the
// querying of onchain application information through a single exposed method
// which returns an apptypes.Application interface
type appQuerier struct {
	clientConn         grpc.ClientConn
	applicationQuerier apptypes.QueryClient
	logger             polylog.Logger

	// applicationsCache caches application.Application returned from applicationQueryClient.Application requests.
	applicationsCache cache.KeyValueCache[apptypes.Application]
	// Mutex to protect applicationsCache access patterns
	applicationsMutex sync.Mutex

	// paramsCache caches application.Params returned from applicationQueryClient.Params requests.
	paramsCache client.ParamsCache[apptypes.Params]
	// Mutex to protect paramsCache access patterns
	paramsMutex sync.Mutex
}

// NewApplicationQuerier returns a new instance of a client.ApplicationQueryClient
// by injecting the dependencies provided by the depinject.Config
//
// Required dependencies:
// - clientCtx
func NewApplicationQuerier(deps depinject.Config) (client.ApplicationQueryClient, error) {
	aq := &appQuerier{}

	if err := depinject.Inject(
		deps,
		&aq.clientConn,
		&aq.logger,
		&aq.applicationsCache,
		&aq.paramsCache,
	); err != nil {
		return nil, err
	}

	aq.applicationQuerier = apptypes.NewQueryClient(aq.clientConn)

	return aq, nil
}

// GetApplication returns an apptypes.Application interface for a given address
func (aq *appQuerier) GetApplication(
	ctx context.Context,
	appAddress string,
) (apptypes.Application, error) {
	logger := aq.logger.With("query_client", "application", "method", "GetApplication")

	// Check if the application is present in the cache.
	if app, found := aq.applicationsCache.Get(appAddress); found {
		logger.Debug().Msgf("cache HIT for application with address: %s", appAddress)
		return app, nil
	}

	// Use mutex to prevent multiple concurrent cache updates
	aq.applicationsMutex.Lock()
	defer aq.applicationsMutex.Unlock()

	// Double-check cache after acquiring lock (follows standard double-checked locking pattern)
	if app, found := aq.applicationsCache.Get(appAddress); found {
		logger.Debug().Msgf("cache HIT for application with address after lock: %s", appAddress)
		return app, nil
	}

	logger.Debug().Msgf("cache MISS for application with address: %s", appAddress)

	req := apptypes.QueryGetApplicationRequest{Address: appAddress}
	res, err := retry.Call(ctx, func() (*apptypes.QueryGetApplicationResponse, error) {
		return aq.applicationQuerier.Application(ctx, &req)
	}, retry.GetStrategy(ctx))
	if err != nil {
		return apptypes.Application{}, err
	}

	// Cache the application.
	aq.applicationsCache.Set(appAddress, res.Application)
	return res.Application, nil
}

// GetAllApplications returns all staked applications
func (aq *appQuerier) GetAllApplications(ctx context.Context) ([]apptypes.Application, error) {
	req := apptypes.QueryAllApplicationsRequest{}
	// TODO_OPTIMIZE: Fill the cache with all applications and mark it as
	// having been filled, such that subsequent calls to this function will
	// return the cached value.
	res, err := retry.Call(ctx, func() (*apptypes.QueryAllApplicationsResponse, error) {
		return aq.applicationQuerier.AllApplications(ctx, &req)
	}, retry.GetStrategy(ctx))
	if err != nil {
		return []apptypes.Application{}, err
	}
	return res.Applications, nil
}

// GetParams returns the application module parameters
func (aq *appQuerier) GetParams(ctx context.Context) (*apptypes.Params, error) {
	logger := aq.logger.With("query_client", "application", "method", "GetParams")

	// Check if the application module parameters are present in the cache.
	if params, found := aq.paramsCache.Get(); found {
		logger.Debug().Msg("cache HIT for application params")
		return &params, nil
	}

	// Use mutex to prevent multiple concurrent cache updates
	aq.paramsMutex.Lock()
	defer aq.paramsMutex.Unlock()

	// Double-check cache after acquiring lock (follows standard double-checked locking pattern)
	if params, found := aq.paramsCache.Get(); found {
		logger.Debug().Msg("cache HIT for application params after lock")
		return &params, nil
	}

	logger.Debug().Msg("cache MISS for application params")

	req := apptypes.QueryParamsRequest{}
	res, err := retry.Call(ctx, func() (*apptypes.QueryParamsResponse, error) {
		return aq.applicationQuerier.Params(ctx, &req)
	}, retry.GetStrategy(ctx))
	if err != nil {
		return nil, err
	}

	// Update the cache with the newly retrieved application module parameters.
	aq.paramsCache.Set(res.Params)
	return &res.Params, nil
}
