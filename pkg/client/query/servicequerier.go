package query

import (
	"context"

	"cosmossdk.io/depinject"
	gogogrpc "github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/query/cache"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ client.ServiceQueryClient = (*serviceQuerier)(nil)

// serviceQuerier is a wrapper around the servicetypes.QueryClient that enables the
// querying of on-chain service information through a single exposed method
// which returns a sharedtypes.Service struct
type serviceQuerier struct {
	client.ParamsQuerier[*servicetypes.Params]

	clientConn     gogogrpc.ClientConn
	serviceQuerier servicetypes.QueryClient
	paramsCache    client.QueryCache[*servicetypes.Params]
}

// NewServiceQuerier returns a new instance of a client.ServiceQueryClient by
// injecting the dependecies provided by the depinject.Config.
//
// Required dependencies:
// - clientCtx (gogogrpc.ClientConn)
func NewServiceQuerier(
	deps depinject.Config,
	paramsQuerierOpts ...ParamsQuerierOptionFn,
) (client.ServiceQueryClient, error) {
	paramsQuerierCfg := DefaultParamsQuerierConfig()
	for _, opt := range paramsQuerierOpts {
		opt(paramsQuerierCfg)
	}

	paramsQuerier, err := NewBaseParamsQuerier[*servicetypes.Params, servicetypes.ServiceQueryClient](
		deps, servicetypes.NewServiceQueryClient,
		WithModuleInfo[*servicetypes.Params](servicetypes.ModuleName, servicetypes.ErrServiceParamInvalid),
		WithParamsCacheOptions(paramsQuerierCfg.CacheOpts...),
	)
	if err != nil {
		return nil, err
	}

	querier := &serviceQuerier{
		// TODO_IN_THIS_COMMIT: extract this to an option.
		// TODO_IMPROVE: add an option for persistent cache.
		paramsCache:   cache.NewInMemoryCache[*servicetypes.Params](paramsQuerierCfg.CacheOpts...),
		ParamsQuerier: paramsQuerier,
	}

	if err := depinject.Inject(
		deps,
		&querier.clientConn,
	); err != nil {
		return nil, err
	}

	querier.serviceQuerier = servicetypes.NewQueryClient(querier.clientConn)

	return querier, nil
}

// GetService returns a sharedtypes.Service struct for a given serviceId.
// It implements the ServiceQueryClient#GetService function.
func (servq *serviceQuerier) GetService(
	ctx context.Context,
	serviceId string,
) (sharedtypes.Service, error) {
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
	return res.Service, nil
}

// GetServiceRelayDifficulty queries the onchain data for
// the relay mining difficulty associated with the given service.
func (servq *serviceQuerier) GetServiceRelayDifficulty(
	ctx context.Context,
	serviceId string,
) (servicetypes.RelayMiningDifficulty, error) {
	req := &servicetypes.QueryGetRelayMiningDifficultyRequest{
		ServiceId: serviceId,
	}

	res, err := servq.serviceQuerier.RelayMiningDifficulty(ctx, req)
	if err != nil {
		return servicetypes.RelayMiningDifficulty{}, err
	}

	return res.RelayMiningDifficulty, nil
}
