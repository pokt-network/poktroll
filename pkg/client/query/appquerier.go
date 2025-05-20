package query

import (
	"context"
	"fmt"
	"sync"

	"cosmossdk.io/depinject"
	"github.com/cosmos/gogoproto/grpc"
	proto "github.com/cosmos/gogoproto/proto"

	"github.com/pokt-network/poktroll/pkg/cache"
	"github.com/pokt-network/poktroll/pkg/client"
	querycache "github.com/pokt-network/poktroll/pkg/client/query/cache"
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

	// eventsParamsActivationClient is used to subscribe to application parameters updates
	eventsParamsActivationClient client.EventsParamsActivationClient
	// paramsCache caches application.Params returned from applicationQueryClient.Params requests.
	paramsCache client.ParamsCache[apptypes.Params]
}

// NewApplicationQuerier returns a new instance of a client.ApplicationQueryClient
// by injecting the dependencies provided by the depinject.Config
//
// Required dependencies:
// - clientCtx
// - polylog.Logger
// - client.EventsParamsActivationClient
// - client.BlockQueryClient
// - cache.KeyValueCache[apptypes.Application]
// - client.ParamsCache[apptypes.Params]
func NewApplicationQuerier(
	ctx context.Context,
	deps depinject.Config,
) (client.ApplicationQueryClient, error) {
	aq := &appQuerier{}

	if err := depinject.Inject(
		deps,
		&aq.clientConn,
		&aq.logger,
		&aq.eventsParamsActivationClient,
		&aq.applicationsCache,
		&aq.paramsCache,
	); err != nil {
		return nil, err
	}

	aq.applicationQuerier = apptypes.NewQueryClient(aq.clientConn)

	// Initialize the application cache with all existing application parameters updates:
	// - Parameters are cached as historic data, eliminating the need to invalidate the cache.
	// - The UpdateParamsCache method ensures the querier starts with the current parameters history cached.
	// - Future updates are automatically cached by subscribing to the eventsParamsActivationClient observable.
	err := querycache.UpdateParamsCache(
		ctx,
		&apptypes.QueryParamsUpdatesRequest{},
		toAppParamsUpdate,
		aq.applicationQuerier,
		aq.eventsParamsActivationClient,
		aq.paramsCache,
	)
	if err != nil {
		return nil, err
	}

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

// GetParams returns the latest application module parameters
func (aq *appQuerier) GetParams(ctx context.Context) (*apptypes.Params, error) {
	logger := aq.logger.With("query_client", "application", "method", "GetParams")

	// Attempt to retrieve the latest parameters from the cache.
	params, found := aq.paramsCache.GetLatest()
	if !found {
		logger.Debug().Msg("cache MISS for application params")
		return nil, fmt.Errorf("expecting application params to be found in cache")
	}
	logger.Debug().Msg("cache HIT for application params")
	return &params, nil
}

func toAppParamsUpdate(protoMessage proto.Message) (*apptypes.ParamsUpdate, bool) {
	if event, ok := protoMessage.(*apptypes.EventParamsActivated); ok {
		return &event.ParamsUpdate, true
	}

	return nil, false
}
