package query

import (
	"context"

	"cosmossdk.io/depinject"
	"github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/cache"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/polylog"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
)

// proofQuerier is a wrapper around the prooftypes.QueryClient that enables the
// querying of onchain proof module params.
type proofQuerier struct {
	clientConn   grpc.ClientConn
	proofQuerier prooftypes.QueryClient
	logger       polylog.Logger

	// paramsCache caches proofQuerier.Params requests
	paramsCache client.ParamsCache[prooftypes.Params]

	// claimsCache caches proofQuerier.Claim requests
	claimsCache cache.KeyValueCache[prooftypes.Claim]
}

// NewProofQuerier returns a new instance of a client.ProofQueryClient by
// injecting the dependecies provided by the depinject.Config.
//
// Required dependencies:
// - grpc.ClientConn
// - polylog.Logger
// - client.ParamsCache[prooftypes.Params]
// - cache.KeyValueCache[prooftypes.Claim]
func NewProofQuerier(deps depinject.Config) (client.ProofQueryClient, error) {
	querier := &proofQuerier{}

	if err := depinject.Inject(
		deps,
		&querier.clientConn,
		&querier.logger,
		&querier.paramsCache,
		&querier.claimsCache,
	); err != nil {
		return nil, err
	}

	querier.proofQuerier = prooftypes.NewQueryClient(querier.clientConn)

	return querier, nil
}

// GetParams queries the chain for the current proof module parameters.
func (pq *proofQuerier) GetParams(
	ctx context.Context,
) (client.ProofParams, error) {
	logger := pq.logger.With("query_client", "proof", "method", "GetParams")

	// Get the params from the cache if they exist.
	if params, found := pq.paramsCache.Get(); found {
		logger.Debug().Msg("cache hit for proof params")
		return &params, nil
	}

	logger.Debug().Msg("cache miss proof params")

	req := &prooftypes.QueryParamsRequest{}
	res, err := pq.proofQuerier.Params(ctx, req)
	if err != nil {
		return nil, err
	}

	// Update the cache with the newly retrieved params.
	pq.paramsCache.Set(res.Params)
	return &res.Params, nil
}

// GetClaim queries the chain for the claim associated with the given session and supplier operator address.
func (pq *proofQuerier) GetClaim(
	ctx context.Context,
	supplierOperatorAddress string,
	sessionId string,
) (client.Claim, error) {
	logger := pq.logger.With("query_client", "proof", "method", "GetClaim")

	// Get the claim from the cache if it exists.
	if claim, found := pq.claimsCache.Get(sessionId); found {
		logger.Debug().Msgf("claim cache hit for claim with sessionId %q", sessionId)
		return &claim, nil
	}

	logger.Debug().Msgf("claim cache miss for claim with sessionId %q", sessionId)

	req := &prooftypes.QueryGetClaimRequest{
		SupplierOperatorAddress: supplierOperatorAddress,
		SessionId:               sessionId,
	}
	res, err := pq.proofQuerier.Claim(ctx, req)
	if err != nil {
		return nil, err
	}

	// Update the cache with the newly retrieved claim.
	pq.claimsCache.Set(sessionId, res.Claim)
	return &res.Claim, nil
}
