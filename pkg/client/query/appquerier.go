package query

import (
	"context"

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

	blockClient client.BlockClient
	// applicationsCache caches application.Application returned from applicationQueryClient.Application requests.
	applicationsCache cache.KeyValueCache[apptypes.Application]
	// paramsCache caches application.Params returned from applicationQueryClient.Params requests.
	paramsCache client.ParamsCache[apptypes.Params]
}

// NewApplicationQuerier returns a new instance of a client.ApplicationQueryClient
// by injecting the dependecies provided by the depinject.Config
//
// Required dependencies:
// - clientCtx
// - polylog.Logger
// - client.BlockQueryClient
// - cache.KeyValueCache[apptypes.Application]
// - client.ParamsCache[apptypes.Params]
func NewApplicationQuerier(deps depinject.Config) (client.ApplicationQueryClient, error) {
	aq := &appQuerier{}

	if err := depinject.Inject(
		deps,
		&aq.clientConn,
		&aq.logger,
		&aq.blockClient,
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
		logger.Debug().Msgf("cache hit for application address key: %s", appAddress)
		return app, nil
	}

	logger.Debug().Msgf("cache miss for application address key: %s", appAddress)

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
	if params, found := aq.paramsCache.GetLatest(); found {
		logger.Debug().Msg("cache hit for application params")
		return &params, nil
	}

	logger.Debug().Msg("cache miss for application params")

	res, err := retry.Call(ctx, func() (*apptypes.QueryParamsAtHeightResponse, error) {
		lastBlock := aq.blockClient.LastBlock(ctx)
		req := apptypes.QueryParamsAtHeightRequest{
			Height: uint64(lastBlock.Height()),
		}
		return aq.applicationQuerier.ParamsAtHeight(ctx, &req)
	}, retry.GetStrategy(ctx))
	if err != nil {
		return nil, err
	}

	// Update the cache with the newly retrieved application module parameters.
	aq.paramsCache.SetAtHeight(res.Params, int64(res.EffectiveBlockHeight))
	return &res.Params, nil
}
