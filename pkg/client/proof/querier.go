package proof

import (
	"context"

	"cosmossdk.io/depinject"
	"github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/client"
	prooftypes "github.com/pokt-network/poktroll/proto/types/proof"
)

var _ client.ProofQueryClient = (*proofQueryClient)(nil)

// proofQueryClient is a wrapper around the prooftypes.QueryClient that enables the
// querying of on-chain proof module params.
type proofQueryClient struct {
	clientConn   grpc.ClientConn
	proofQuerier prooftypes.QueryClient
}

// NewProofQueryClient returns a new instance of a client.ProofQueryClient by
// injecting the dependecies provided by the depinject.Config.
//
// Required dependencies:
// - grpc.ClientConn
func NewProofQueryClient(deps depinject.Config) (client.ProofQueryClient, error) {
	querier := &proofQueryClient{}

	if err := depinject.Inject(
		deps,
		&querier.clientConn,
	); err != nil {
		return nil, err
	}

	querier.proofQuerier = prooftypes.NewQueryClient(querier.clientConn)

	return querier, nil
}

// GetParams queries the chain for the current proof module parameters.
func (pq *proofQueryClient) GetParams(
	ctx context.Context,
) (client.ProofParams, error) {
	req := &prooftypes.QueryParamsRequest{}
	res, err := pq.proofQuerier.Params(ctx, req)
	if err != nil {
		return nil, err
	}
	return &res.Params, nil
}
