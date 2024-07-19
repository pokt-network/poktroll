package query

import (
	"context"

	"cosmossdk.io/depinject"
	"github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/proto/types/proof"
)

// proofQuerier is a wrapper around the prooftypes.QueryClient that enables the
// querying of on-chain proof module params.
type proofQuerier struct {
	clientConn   grpc.ClientConn
	proofQuerier proof.QueryClient
}

// NewProofQuerier returns a new instance of a client.ProofQueryClient by
// injecting the dependecies provided by the depinject.Config.
//
// Required dependencies:
// - grpc.ClientConn
func NewProofQuerier(deps depinject.Config) (client.ProofQueryClient, error) {
	querier := &proofQuerier{}

	if err := depinject.Inject(
		deps,
		&querier.clientConn,
	); err != nil {
		return nil, err
	}

	querier.proofQuerier = proof.NewQueryClient(querier.clientConn)

	return querier, nil
}

// GetParams queries the chain for the current proof module parameters.
func (pq *proofQuerier) GetParams(
	ctx context.Context,
) (client.ProofParams, error) {
	req := &proof.QueryParamsRequest{}
	res, err := pq.proofQuerier.Params(ctx, req)
	if err != nil {
		return nil, err
	}
	return &res.Params, nil
}
