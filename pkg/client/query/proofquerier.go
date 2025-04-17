package query

import (
	"context"
	"fmt"
	"sync"

	"cosmossdk.io/depinject"
	"github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/cache"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/retry"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
)

// proofQuerier is a wrapper around the prooftypes.QueryClient that enables the
// querying and caching the onchain proof module.
type proofQuerier struct {
	clientConn   grpc.ClientConn
	proofQuerier prooftypes.QueryClient
	logger       polylog.Logger

	// paramsCache caches proofQuerier.Params requests
	paramsCache client.ParamsCache[prooftypes.Params]

	// claimsCache caches proofQuerier.Claim requests
	// It keys the Claims by sessionId and supplierOperatorAddress
	claimsCache cache.KeyValueCache[prooftypes.Claim]

	// Mutexes to protect cache access patterns
	paramsMutex sync.Mutex
	claimsMutex sync.Mutex
}

// NewProofQuerier returns a new instance of a client.ProofQueryClient by
// injecting the dependencies provided by the depinject.Config.
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

	// Use mutex to prevent multiple concurrent cache updates
	pq.paramsMutex.Lock()
	defer pq.paramsMutex.Unlock()

	// Double-check cache after acquiring lock
	if params, found := pq.paramsCache.Get(); found {
		logger.Debug().Msg("cache hit for proof params after lock")
		return &params, nil
	}

	logger.Debug().Msg("cache miss for proof params")

	req := &prooftypes.QueryParamsRequest{}
	res, err := retry.Call(ctx, func() (*prooftypes.QueryParamsResponse, error) {
		return pq.proofQuerier.Params(ctx, req)
	}, retry.GetStrategy(ctx))
	if err != nil {
		return nil, err
	}

	// Update the cache with the newly retrieved params.
	pq.paramsCache.Set(res.Params)
	return &res.Params, nil
}

// GetClaim queries the chain for the claim associated with the given session id and supplier operator address.
// If a claim is available in the cache, it is returned instead.
func (pq *proofQuerier) GetClaim(
	ctx context.Context,
	supplierOperatorAddress string,
	sessionId string,
) (client.Claim, error) {
	logger := pq.logger.With("query_client", "proof", "method", "GetClaim")

	// Get the claim from the cache if it exists.
	claimCacheKey := getClaimCacheKey(supplierOperatorAddress, sessionId)
	if claim, found := pq.claimsCache.Get(claimCacheKey); found {
		logger.Debug().Msgf("claim cache HIT for claim with sessionId %q", sessionId)
		return &claim, nil
	}

	// Use mutex to prevent multiple concurrent cache updates
	pq.claimsMutex.Lock()
	defer pq.claimsMutex.Unlock()

	// Double-check cache after acquiring lock
	if claim, found := pq.claimsCache.Get(claimCacheKey); found {
		logger.Debug().Msgf("claim cache HIT for claim with sessionId %q after lock", sessionId)
		return &claim, nil
	}

	logger.Debug().Msgf("claim cache MISS for claim with sessionId %q", sessionId)

	req := &prooftypes.QueryGetClaimRequest{
		SupplierOperatorAddress: supplierOperatorAddress,
		SessionId:               sessionId,
	}
	res, err := pq.proofQuerier.Claim(ctx, req)
	if err != nil {
		return nil, err
	}

	// Update the cache with the newly retrieved claim.
	pq.claimsCache.Set(claimCacheKey, res.Claim)

	// Return the query claim
	return &res.Claim, nil
}

// getClaimCacheKey constructs the cache key for a claim in the form of: supplierOperatorAddress/sessionId.
func getClaimCacheKey(supplierOperatorAddress, sessionId string) string {
	return fmt.Sprintf("%s/%s", supplierOperatorAddress, sessionId)
}
