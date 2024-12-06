package query

import (
	"cosmossdk.io/depinject"
	"github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/query/cache"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
)

// TODO_IN_THIS_COMMIT: comment explaining why we can't use client.ProofQueryClient;
// tl;dr, it defines ian interface for ProofParams to avoid a dependency cycle
// (i.e. instead of importing prooftypes).
//
//var _ prooftypes.ProofQueryClient = (*proofQuerier)(nil)

// proofQuerier is a wrapper around the prooftypes.QueryClient that enables the
// querying of on-chain proof module params.
type proofQuerier struct {
	*baseParamsQuerier[*prooftypes.Params, prooftypes.ProofQueryClient]

	clientConn    grpc.ClientConn
	proofQuerier  prooftypes.QueryClient
	paramsQuerier client.ParamsQuerier[*prooftypes.Params]
	paramsCache   client.QueryCache[*prooftypes.Params]
}

// NewProofQuerier returns a new instance of a client.ProofQueryClient by
// injecting the dependecies provided by the depinject.Config.
//
// Required dependencies:
// - grpc.ClientConn
func NewProofQuerier(
	deps depinject.Config,
	paramsQuerierOpts ...ParamsQuerierOptionFn,
	// TODO_IN_THIS_COMMIT: comment explaining why we can't use client.ProofQueryClient;
	// tl;dr, it defines ian interface for ProofParams to avoid a dependency cycle
	// (i.e. instead of importing prooftypes).
) (paramsQuerierIface[*prooftypes.Params], error) {
	paramsQuerierCfg := DefaultParamsQuerierConfig()
	for _, opt := range paramsQuerierOpts {
		opt(paramsQuerierCfg)
	}

	paramsQuerier, err := NewParamsQuerier[*prooftypes.Params, prooftypes.ProofQueryClient](
		deps, prooftypes.NewProofQueryClient,
		WithModuleInfo[*prooftypes.Params](prooftypes.ModuleName, prooftypes.ErrProofParamInvalid),
		WithParamsCacheOptions(paramsQuerierCfg.CacheOpts...),
	)
	if err != nil {
		return nil, err
	}

	querier := &proofQuerier{
		// TODO_IN_THIS_COMMIT: extract this to an option.
		// TODO_IMPROVE: add an option for persistent cache.
		paramsCache:   cache.NewInMemoryCache[*prooftypes.Params](paramsQuerierCfg.CacheOpts...),
		paramsQuerier: paramsQuerier,
	}

	if err := depinject.Inject(
		deps,
		&querier.clientConn,
	); err != nil {
		return nil, err
	}

	querier.proofQuerier = prooftypes.NewQueryClient(querier.clientConn)

	return querier, nil
}
