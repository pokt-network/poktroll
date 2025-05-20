package query

import (
	"context"
	"fmt"
	"sync"

	"cosmossdk.io/depinject"
	"github.com/cosmos/gogoproto/grpc"
	proto "github.com/cosmos/gogoproto/proto"

	"github.com/pokt-network/poktroll/pkg/cache"
	"github.com/pokt-network/poktroll/pkg/client"
	querycache "github.com/pokt-network/poktroll/pkg/client/query/cache"
	"github.com/pokt-network/poktroll/pkg/polylog"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
)

// proofQuerier is a wrapper around the prooftypes.QueryClient that enables the
// querying and caching the onchain proof module.
type proofQuerier struct {
	clientConn   grpc.ClientConn
	proofQuerier prooftypes.QueryClient
	logger       polylog.Logger

	// eventsParamsActivationClient is used to subscribe to proof module parameters updates
	eventsParamsActivationClient client.EventsParamsActivationClient
	// paramsCache caches proofQuerier.Params requests
	paramsCache client.ParamsCache[prooftypes.Params]

	// claimsCache caches proofQuerier.Claim requests
	// It keys the Claims by sessionId and supplierOperatorAddress
	claimsCache cache.KeyValueCache[prooftypes.Claim]
	// claimsMutex to protect cache access patterns for claims
	claimsMutex sync.Mutex
}

// NewProofQuerier returns a new instance of a client.ProofQueryClient by
// injecting the dependencies provided by the depinject.Config.
//
// Required dependencies:
// - grpc.ClientConn
// - polylog.Logger
// - client.EventsParamsActivationClient
// - client.ParamsCache[prooftypes.Params]
// - cache.KeyValueCache[prooftypes.Claim]
func NewProofQuerier(
	ctx context.Context,
	deps depinject.Config,
) (client.ProofQueryClient, error) {
	querier := &proofQuerier{}

	if err := depinject.Inject(
		deps,
		&querier.clientConn,
		&querier.logger,
		&querier.eventsParamsActivationClient,
		&querier.paramsCache,
		&querier.claimsCache,
	); err != nil {
		return nil, err
	}

	querier.proofQuerier = prooftypes.NewQueryClient(querier.clientConn)

	// Initialize the proof module cache with all existing parameters updates:
	// - Parameters are cached as historic data, eliminating the need to invalidate the cache.
	// - The UpdateParamsCache method ensures the querier starts with the current parameters history cached.
	// - Future updates are automatically cached by subscribing to the eventsParamsActivationClient observable.
	err := querycache.UpdateParamsCache(
		ctx,
		&prooftypes.QueryParamsUpdatesRequest{},
		toProofParamsUpdate,
		querier.proofQuerier,
		querier.eventsParamsActivationClient,
		querier.paramsCache,
	)
	if err != nil {
		return nil, err
	}

	return querier, nil
}

// GetParams queries the chain for the current proof module parameters.
func (pq *proofQuerier) GetParams(
	ctx context.Context,
) (client.ProofParams, error) {
	logger := pq.logger.With("query_client", "proof", "method", "GetParams")

	// Attempt to retrieve the latest parameters from the cache.
	params, found := pq.paramsCache.GetLatest()
	if !found {
		logger.Debug().Msg("cache MISS for proof params")
		return nil, fmt.Errorf("expecting proof params to be found in cache")
	}

	logger.Debug().Msg("cache HIT for proof params")

	return &params, nil
}

// GetParamsAtHeight queries & returns the proof module onchain parameters
// that were in effect at the given block height.
func (pq *proofQuerier) GetParamsAtHeight(ctx context.Context, height int64) (client.ProofParams, error) {
	logger := pq.logger.With("query_client", "proof", "method", "GetParamsAtHeight")

	// Get the params from the cache if they exist.
	params, found := pq.paramsCache.GetAtHeight(height)
	if !found {
		logger.Debug().Msgf("cache MISS for proof params at height: %d", height)
		return nil, fmt.Errorf("expecting proof params to be found in cache at height %d", height)
	}

	logger.Debug().Msgf("cache HIT for proof params at height: %d", height)
	return &params, nil
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

	// Double-check cache after acquiring lock (follows standard double-checked locking pattern)
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

func toProofParamsUpdate(protoMessage proto.Message) (*prooftypes.ParamsUpdate, bool) {
	if event, ok := protoMessage.(*prooftypes.EventParamsActivated); ok {
		return &event.ParamsUpdate, true
	}

	return nil, false
}
