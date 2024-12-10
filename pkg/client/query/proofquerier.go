package query

import (
	"cosmossdk.io/depinject"
	"github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/client"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
)

// TODO_IN_THIS_COMMIT: comment explaining why we can't use client.ProofQueryClient;
// tl;dr, it defines ian interface for ProofParams to avoid a dependency cycle
// (i.e. instead of importing prooftypes).
var _ client.ProofQueryClient = (*proofQuerier)(nil)

// proofQuerier is a wrapper around the prooftypes.QueryClient that enables the
// querying of on-chain proof module params.
type proofQuerier struct {
	//client.ParamsQuerier[*prooftypes.Params]
	client.ParamsQuerier[client.ProofParams]

	clientConn   grpc.ClientConn
	proofQuerier prooftypes.QueryClient
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
	// ) (paramsQuerierIface[*prooftypes.Params], error) {
) (paramsQuerierIface[client.ProofParams], error) {
	paramsQuerierCfg := DefaultParamsQuerierConfig()
	for _, opt := range paramsQuerierOpts {
		opt(paramsQuerierCfg)
	}

	paramsQuerier, err := NewCachedParamsQuerier[client.ProofParams, prooftypes.ProofQueryClient](
		deps, prooftypes.NewProofQueryClient,
		WithModuleInfo[*prooftypes.Params](prooftypes.ModuleName, prooftypes.ErrProofParamInvalid),
		WithParamsCacheOptions(paramsQuerierCfg.CacheOpts...),
	)
	if err != nil {
		return nil, err
	}

	querier := &proofQuerier{
		ParamsQuerier: paramsQuerier,
	}

	if err = depinject.Inject(
		deps,
		&querier.clientConn,
	); err != nil {
		return nil, err
	}

	querier.proofQuerier = prooftypes.NewQueryClient(querier.clientConn)

	return querier, nil
}
