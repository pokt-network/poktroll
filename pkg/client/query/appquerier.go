package query

import (
	"context"

	"cosmossdk.io/depinject"
	gogogrpc "github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/client"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ client.ApplicationQueryClient = (*appQuerier)(nil)

// appQuerier is a wrapper around the apptypes.QueryClient that enables the
// querying of on-chain application information through a single exposed method
// which returns an apptypes.Application interface
type appQuerier struct {
	client.ParamsQuerier[*apptypes.Params]

	clientConn         gogogrpc.ClientConn
	applicationQuerier apptypes.QueryClient
}

// NewApplicationQuerier returns a new instance of a client.ApplicationQueryClient
// by injecting the dependecies provided by the depinject.Config
//
// Required dependencies:
// - clientCtx (gogogrpc.ClientConn)
func NewApplicationQuerier(
	deps depinject.Config,
	opts ...ParamsQuerierOptionFn,
) (client.ApplicationQueryClient, error) {
	cfg := DefaultParamsQuerierConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	paramsQuerier, err := NewCachedParamsQuerier[*apptypes.Params, apptypes.ApplicationQueryClient](
		deps, apptypes.NewAppQueryClient,
		WithModuleInfo[*sharedtypes.Params](sharedtypes.ModuleName, sharedtypes.ErrSharedParamInvalid),
		WithParamsCacheOptions(cfg.CacheOpts...),
	)
	if err != nil {
		return nil, err
	}

	aq := &appQuerier{
		ParamsQuerier: paramsQuerier,
	}

	if err = depinject.Inject(
		deps,
		&aq.clientConn,
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
	req := apptypes.QueryGetApplicationRequest{Address: appAddress}
	res, err := aq.applicationQuerier.Application(ctx, &req)
	if err != nil {
		return apptypes.Application{}, apptypes.ErrAppNotFound.Wrapf("app address: %s [%v]", appAddress, err)
	}
	return res.Application, nil
}

// GetAllApplications returns all staked applications
func (aq *appQuerier) GetAllApplications(ctx context.Context) ([]apptypes.Application, error) {
	req := apptypes.QueryAllApplicationsRequest{}
	res, err := aq.applicationQuerier.AllApplications(ctx, &req)
	if err != nil {
		return []apptypes.Application{}, err
	}
	return res.Applications, nil
}
