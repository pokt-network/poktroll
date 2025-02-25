package query

import (
	"context"

	"cosmossdk.io/depinject"
	grpc "github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/polylog"
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

	// applicationsCache caches applicationQueryClient.Application requests
	applicationsCache KeyValueCache[apptypes.Application]
	// paramsCache caches applicationQueryClient.Params requests
	paramsCache ParamsCache[apptypes.Params]
}

// NewApplicationQuerier returns a new instance of a client.ApplicationQueryClient
// by injecting the dependecies provided by the depinject.Config
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
		logger.Debug().Msgf("cache hit for key: %s", appAddress)
		return app, nil
	}

	logger.Debug().Msgf("cache miss for key: %s", appAddress)

	req := apptypes.QueryGetApplicationRequest{Address: appAddress}
	res, err := aq.applicationQuerier.Application(ctx, &req)
	if err != nil {
		return apptypes.Application{}, apptypes.ErrAppNotFound.Wrapf("app address: %s [%v]", appAddress, err)
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
	res, err := aq.applicationQuerier.AllApplications(ctx, &req)
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
		logger.Debug().Msg("cache hit")
		return &params, nil
	}

	logger.Debug().Msg("cache miss")

	req := apptypes.QueryParamsRequest{}
	res, err := aq.applicationQuerier.Params(ctx, &req)
	if err != nil {
		return nil, err
	}

	// Update the cache with the newly retrieved application module parameters.
	aq.paramsCache.Set(res.Params)
	return &res.Params, nil
}
