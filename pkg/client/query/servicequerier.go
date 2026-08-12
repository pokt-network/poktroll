package query

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"cosmossdk.io/depinject"
	"github.com/cosmos/gogoproto/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/pkg/cache"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/retry"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ client.ServiceQueryClient = (*serviceQuerier)(nil)

// serviceQuerier is a wrapper around the servicetypes.QueryClient that enables the
// querying of onchain service information through a single exposed method
// which returns a sharedtypes.Service struct
type serviceQuerier struct {
	clientConn     grpc.ClientConn
	serviceQuerier servicetypes.QueryClient
	logger         polylog.Logger

	// servicesCache caches serviceQueryClient.Service query requests
	servicesCache cache.KeyValueCache[sharedtypes.Service]
	// relayMiningDifficultyCache caches serviceQueryClient.RelayMiningDifficulty query requests
	relayMiningDifficultyCache cache.KeyValueCache[servicetypes.RelayMiningDifficulty]
	// computeUnitsPerRelayCache caches serviceQueryClient.ComputeUnitsPerRelayAtHeight
	// query requests. The cupr effective at a past height is immutable, so entries are
	// keyed by "serviceId:height" and never invalidated.
	computeUnitsPerRelayCache cache.KeyValueCache[uint64]
	// servicesMutex to protect cache access patterns for services and relay mining difficulties
	servicesMutex sync.Mutex

	// cuprAtHeightUnsupported latches once the node has answered codes.Unimplemented for
	// ComputeUnitsPerRelayAtHeight (i.e. it predates v0.1.35). Without the latch every
	// relay pays a full failed gRPC round trip -- under servicesMutex -- plus a warning
	// log, in exactly the configuration the fallback exists to keep usable.
	cuprAtHeightUnsupported atomic.Bool
	// cuprUnsupportedWarnOnce keeps the degrade notice to one line, not one per relay.
	cuprUnsupportedWarnOnce sync.Once

	// paramsCache caches serviceQueryClient.Params query requests
	paramsCache client.ParamsCache[servicetypes.Params]
	// paramsMutex to protect cache access patterns for params
	paramsMutex sync.Mutex
}

// NewServiceQuerier returns a new instance of a client.ServiceQueryClient by
// injecting the dependencies provided by the depinject.Config.
//
// Required dependencies:
// - clientCtx (grpc.ClientConn)
// - polylog.Logger
// - cache.KeyValueCache[sharedtypes.Service]
// - cache.KeyValueCache[servicetypes.RelayMiningDifficulty]
// - cache.KeyValueCache[uint64] (compute units per relay at height)
func NewServiceQuerier(deps depinject.Config) (client.ServiceQueryClient, error) {
	servq := &serviceQuerier{}

	if err := depinject.Inject(
		deps,
		&servq.clientConn,
		&servq.logger,
		&servq.servicesCache,
		&servq.relayMiningDifficultyCache,
		&servq.computeUnitsPerRelayCache,
		&servq.paramsCache,
	); err != nil {
		return nil, err
	}

	servq.serviceQuerier = servicetypes.NewQueryClient(servq.clientConn)

	return servq, nil
}

// GetService returns a sharedtypes.Service struct for a given serviceId.
// It implements the ServiceQueryClient#GetService function.
func (servq *serviceQuerier) GetService(
	ctx context.Context,
	serviceId string,
) (sharedtypes.Service, error) {
	logger := servq.logger.With("query_client", "service", "method", "GetService")

	// Check if the service is present in the cache.
	if service, found := servq.servicesCache.Get(serviceId); found {
		logger.Debug().Msgf("service cache hit for service id key: %s", serviceId)
		return service, nil
	}

	// Use mutex to prevent multiple concurrent cache updates
	servq.servicesMutex.Lock()
	defer servq.servicesMutex.Unlock()

	// Double-check cache after acquiring lock (follows standard double-checked locking pattern)
	if service, found := servq.servicesCache.Get(serviceId); found {
		logger.Debug().Msgf("service cache hit for service id key after lock: %s", serviceId)
		return service, nil
	}

	logger.Debug().Msgf("service cache miss for service id key: %s", serviceId)

	req := &servicetypes.QueryGetServiceRequest{
		Id: serviceId,
	}
	res, err := retry.Call(ctx, func() (*servicetypes.QueryGetServiceResponse, error) {
		queryCtx, cancelQueryCtx := context.WithTimeout(ctx, defaultQueryTimeout)
		defer cancelQueryCtx()
		return servq.serviceQuerier.Service(queryCtx, req)
	}, retry.GetStrategy(ctx), logger)
	if err != nil {
		return sharedtypes.Service{}, ErrQueryRetrieveService.Wrapf(
			"serviceId: %s; error: [%v]",
			serviceId, err,
		)
	}

	// Cache the service for future use.
	servq.servicesCache.Set(serviceId, res.Service)
	return res.Service, nil
}

// GetServiceRelayDifficulty queries the onchain data for
// the relay mining difficulty associated with the given service.
func (servq *serviceQuerier) GetServiceRelayDifficulty(
	ctx context.Context,
	serviceId string,
) (servicetypes.RelayMiningDifficulty, error) {
	logger := servq.logger.With("query_client", "service", "method", "GetServiceRelayDifficulty")

	// Check if the relay mining difficulty is present in the cache.
	if relayMiningDifficulty, found := servq.relayMiningDifficultyCache.Get(serviceId); found {
		logger.Debug().Msgf("relay mining difficulty cache hit for service id key: %s", serviceId)
		return relayMiningDifficulty, nil
	}

	// Use mutex to prevent multiple concurrent cache updates
	servq.servicesMutex.Lock()
	defer servq.servicesMutex.Unlock()

	// Double-check cache after acquiring lock (follows standard double-checked locking pattern)
	if relayMiningDifficulty, found := servq.relayMiningDifficultyCache.Get(serviceId); found {
		logger.Debug().Msgf("relay mining difficulty cache hit for service id key after lock: %s", serviceId)
		return relayMiningDifficulty, nil
	}

	logger.Debug().Msgf("relay mining difficulty cache miss for service id key: %s", serviceId)

	req := &servicetypes.QueryGetRelayMiningDifficultyRequest{
		ServiceId: serviceId,
	}
	res, err := retry.Call(ctx, func() (*servicetypes.QueryGetRelayMiningDifficultyResponse, error) {
		queryCtx, cancelQueryCtx := context.WithTimeout(ctx, defaultQueryTimeout)
		defer cancelQueryCtx()
		return servq.serviceQuerier.RelayMiningDifficulty(queryCtx, req)
	}, retry.GetStrategy(ctx), logger)
	if err != nil {
		return servicetypes.RelayMiningDifficulty{}, err
	}

	// Cache the relay mining difficulty for future use.
	servq.relayMiningDifficultyCache.Set(serviceId, res.RelayMiningDifficulty)
	return res.RelayMiningDifficulty, nil
}

// GetServiceComputeUnitsPerRelayAtHeight queries the onchain compute units per relay
// (cupr) that was effective for a service at the given block height.
//
// The RelayMiner calls this with a session's start height so every relay in the
// session is weighted by the same cupr the chain will validate the claim against —
// eliminating the mid-session cupr flip that forfeited claims with
// ErrProofComputeUnitsMismatch. Because the cupr at a past height never changes, the
// result is cached permanently under a "serviceId:height" key.
func (servq *serviceQuerier) GetServiceComputeUnitsPerRelayAtHeight(
	ctx context.Context,
	serviceId string,
	blockHeight int64,
) (uint64, error) {
	logger := servq.logger.With("query_client", "service", "method", "GetServiceComputeUnitsPerRelayAtHeight")

	cacheKey := fmt.Sprintf("%s:%d", serviceId, blockHeight)

	// Check if the cupr is present in the cache.
	if computeUnitsPerRelay, found := servq.computeUnitsPerRelayCache.Get(cacheKey); found {
		logger.Debug().Msgf("compute units per relay cache hit for key: %s", cacheKey)
		return computeUnitsPerRelay, nil
	}

	// ComputeUnitsPerRelayAtHeight is NEW in v0.1.35, and a node still running v0.1.34
	// answers codes.Unimplemented -- not a transient code, so retry.Call gives up on the
	// first attempt. Once that has been observed, skip the doomed round trip entirely.
	//
	// This check MUST come before servicesMutex is taken: the live path calls GetService,
	// which takes that SAME non-reentrant mutex.
	if servq.cuprAtHeightUnsupported.Load() {
		return servq.liveComputeUnitsPerRelay(ctx, serviceId)
	}

	cupr, unsupported, err := servq.queryComputeUnitsPerRelayAtHeight(ctx, serviceId, blockHeight, cacheKey, logger)
	switch {
	case unsupported:
		// Degrade to the live cupr, which is exactly what this call site read before
		// v0.1.35, so behaviour matches the old binary until the node catches up.
		//
		// Reached only AFTER queryComputeUnitsPerRelayAtHeight has returned and released
		// servicesMutex. Calling GetService while still holding it deadlocks the goroutine
		// against itself, and because the lock is held every other service, difficulty and
		// cupr lookup blocks behind it forever -- a total hang of serving AND mining,
		// strictly worse than the outage this fallback exists to prevent.
		servq.cuprAtHeightUnsupported.Store(true)
		servq.cuprUnsupportedWarnOnce.Do(func() {
			logger.Warn().Msg(
				"node does not implement ComputeUnitsPerRelayAtHeight (pre-v0.1.35); " +
					"falling back to the LIVE compute units per relay. Session-start cupr " +
					"pricing is INACTIVE until the full node is upgraded to v0.1.35.",
			)
		})
		return servq.liveComputeUnitsPerRelay(ctx, serviceId)
	case err != nil:
		return 0, err
	}

	return cupr, nil
}

// queryComputeUnitsPerRelayAtHeight performs the cache-guarded at-height query.
//
// It reports unsupported=true rather than handling the pre-v0.1.35 fallback itself,
// because that fallback needs GetService, which contends on the mutex held here.
func (servq *serviceQuerier) queryComputeUnitsPerRelayAtHeight(
	ctx context.Context,
	serviceId string,
	blockHeight int64,
	cacheKey string,
	logger polylog.Logger,
) (cupr uint64, unsupported bool, err error) {
	// Use mutex to prevent multiple concurrent cache updates.
	servq.servicesMutex.Lock()
	defer servq.servicesMutex.Unlock()

	// Double-check cache after acquiring lock (standard double-checked locking pattern).
	if computeUnitsPerRelay, found := servq.computeUnitsPerRelayCache.Get(cacheKey); found {
		logger.Debug().Msgf("compute units per relay cache hit for key after lock: %s", cacheKey)
		return computeUnitsPerRelay, false, nil
	}

	logger.Debug().Msgf("compute units per relay cache miss for key: %s", cacheKey)

	req := &servicetypes.QueryComputeUnitsPerRelayAtHeightRequest{
		ServiceId:   serviceId,
		BlockHeight: blockHeight,
	}
	res, err := retry.Call(ctx, func() (*servicetypes.QueryComputeUnitsPerRelayAtHeightResponse, error) {
		queryCtx, cancelQueryCtx := context.WithTimeout(ctx, defaultQueryTimeout)
		defer cancelQueryCtx()
		return servq.serviceQuerier.ComputeUnitsPerRelayAtHeight(queryCtx, req)
	}, retry.GetStrategy(ctx), logger)
	if err != nil {
		if status.Code(err) == codes.Unimplemented {
			return 0, true, nil
		}

		return 0, false, ErrQueryRetrieveService.Wrapf(
			"serviceId: %s; height: %d; error: [%v]",
			serviceId, blockHeight, err,
		)
	}

	// Cache the cupr for future use (immutable for a past height).
	servq.computeUnitsPerRelayCache.Set(cacheKey, res.ComputeUnitsPerRelay)
	return res.ComputeUnitsPerRelay, false, nil
}

// liveComputeUnitsPerRelay reads the service's CURRENT compute units per relay.
//
// MUST NOT be called while holding servicesMutex -- GetService takes it.
func (servq *serviceQuerier) liveComputeUnitsPerRelay(ctx context.Context, serviceId string) (uint64, error) {
	service, err := servq.GetService(ctx, serviceId)
	if err != nil {
		return 0, err
	}
	return service.ComputeUnitsPerRelay, nil
}

// GetServiceRelayDifficultyAtHeight queries the onchain relay mining difficulty that was
// effective for a service at the given block height.
//
// The RelayMiner calls this with a session's START height because every onchain consumer
// resolves difficulty the same way — claim creation, proof submission,
// ProofRequirementForClaim, and settlement all use
// GetRelayMiningDifficultyAtHeight(sessionStartHeight). Since
//
//	claimeduPOKT = numEstimatedComputeUnits(difficulty) * CUTTM / granularity
//
// pinning only CUTTM would leave the other factor of the same product unpinned: the miner
// would compare a differently-derived claimeduPOKT against proof_requirement_threshold
// than the chain does, and could skip a proof the chain requires — PROOF_MISSING, and the
// supplier is slashed.
//
// Difficulty at a past height never changes, so the result is cached permanently under a
// "serviceId:height" key. That key shape cannot collide with the plain serviceId keys used
// by GetServiceRelayDifficulty.
func (servq *serviceQuerier) GetServiceRelayDifficultyAtHeight(
	ctx context.Context,
	serviceId string,
	blockHeight int64,
) (servicetypes.RelayMiningDifficulty, error) {
	logger := servq.logger.With("query_client", "service", "method", "GetServiceRelayDifficultyAtHeight")

	cacheKey := fmt.Sprintf("%s:%d", serviceId, blockHeight)

	// Check if the relay mining difficulty is present in the cache.
	if relayMiningDifficulty, found := servq.relayMiningDifficultyCache.Get(cacheKey); found {
		logger.Debug().Msgf("relay mining difficulty cache hit for key: %s", cacheKey)
		return relayMiningDifficulty, nil
	}

	// Use mutex to prevent multiple concurrent cache updates.
	servq.servicesMutex.Lock()
	defer servq.servicesMutex.Unlock()

	// Double-check cache after acquiring lock (standard double-checked locking pattern).
	if relayMiningDifficulty, found := servq.relayMiningDifficultyCache.Get(cacheKey); found {
		logger.Debug().Msgf("relay mining difficulty cache hit for key after lock: %s", cacheKey)
		return relayMiningDifficulty, nil
	}

	logger.Debug().Msgf("relay mining difficulty cache miss for key: %s", cacheKey)

	req := &servicetypes.QueryGetRelayMiningDifficultyAtHeightRequest{
		ServiceId:   serviceId,
		BlockHeight: blockHeight,
	}
	res, err := retry.Call(ctx, func() (*servicetypes.QueryGetRelayMiningDifficultyAtHeightResponse, error) {
		queryCtx, cancelQueryCtx := context.WithTimeout(ctx, defaultQueryTimeout)
		defer cancelQueryCtx()
		return servq.serviceQuerier.RelayMiningDifficultyAtHeight(queryCtx, req)
	}, retry.GetStrategy(ctx), logger)
	if err != nil {
		return servicetypes.RelayMiningDifficulty{}, ErrQueryRetrieveService.Wrapf(
			"serviceId: %s; height: %d; error: [%v]",
			serviceId, blockHeight, err,
		)
	}

	// Cache the difficulty for future use (immutable for a past height).
	servq.relayMiningDifficultyCache.Set(cacheKey, res.RelayMiningDifficulty)
	return res.RelayMiningDifficulty, nil
}

// GetParams returns the service module parameters.
func (servq *serviceQuerier) GetParams(ctx context.Context) (*servicetypes.Params, error) {
	logger := servq.logger.With("query_client", "service", "method", "GetParams")

	// Check if the service module parameters are present in the cache.
	if params, found := servq.paramsCache.Get(); found {
		logger.Debug().Msg("cache HIT for service params")
		return &params, nil
	}

	// Use mutex to prevent multiple concurrent cache updates
	servq.paramsMutex.Lock()
	defer servq.paramsMutex.Unlock()

	// Double-check cache after acquiring lock (follows standard double-checked locking pattern)
	if params, found := servq.paramsCache.Get(); found {
		logger.Debug().Msg("cache HIT for service params after lock")
		return &params, nil
	}

	logger.Debug().Msg("cache MISS for service params")

	req := servicetypes.QueryParamsRequest{}
	res, err := retry.Call(ctx, func() (*servicetypes.QueryParamsResponse, error) {
		queryCtx, cancelQueryCtx := context.WithTimeout(ctx, defaultQueryTimeout)
		defer cancelQueryCtx()
		return servq.serviceQuerier.Params(queryCtx, &req)
	}, retry.GetStrategy(ctx), logger)
	if err != nil {
		return nil, err
	}

	// Cache the parameters for future queries.
	servq.paramsCache.Set(res.Params)
	return &res.Params, nil
}
