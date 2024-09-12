package sdkadapter

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"cosmossdk.io/depinject"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	shannonsdk "github.com/pokt-network/shannon-sdk"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/query"
	"github.com/pokt-network/poktroll/x/service/types"
)

// ShannonSDK is a wrapper around the Shannon SDK used by the AppGateServer
// to encapsulate the SDK's functionality and dependencies.
type ShannonSDK struct {
	blockClient   client.BlockClient
	sessionClient client.SessionQueryClient
	appClient     client.ApplicationQueryClient
	accountClient client.AccountQueryClient
	relayClient   *http.Client
	signer        *shannonsdk.Signer
}

// sessionCache stores session results to reduce redundant queries.
type sessionCache struct {
	mu    sync.RWMutex
	cache map[string]*cachedSession
}

// cachedSession represents a cached session data.
type cachedSession struct {
	filter *shannonsdk.SessionFilter
	height int64
}

// sessionCacheInstance is a global instance of the session cache.
var sessionCacheInstance = sessionCache{
	cache: make(map[string]*cachedSession),
}

// heightCache caches the latest block height to reduce queries.
type heightCache struct {
	mu              sync.RWMutex
	cachedHeight    int64
	lastUpdated     time.Time
	invalidationDur time.Duration
}

// blockHeightCache is a global instance of the height cache with a default 1-second invalidation duration.
var blockHeightCache = heightCache{
	invalidationDur: time.Second,
}

// cacheKey generates a unique key for the session cache.
func cacheKey(appAddress, serviceId string, height int64) string {
	return fmt.Sprintf("%s-%s-%d", appAddress, serviceId, height)
}

// cleanupCache removes outdated entries from the session cache.
func (c *sessionCache) cleanupCache(currentHeight int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, entry := range c.cache {
		if entry.height < currentHeight {
			delete(c.cache, key)
		}
	}
}

func (shannonSDK *ShannonSDK) GetCachedHeight(ctx context.Context) (int64, error) {
	blockHeightCache.mu.RLock()
	if blockHeightCache.cachedHeight != 0 && time.Since(blockHeightCache.lastUpdated) < blockHeightCache.invalidationDur {
		cachedHeight := blockHeightCache.cachedHeight
		blockHeightCache.mu.RUnlock()
		return cachedHeight, nil
	}
	blockHeightCache.mu.RUnlock()

	blockHeightCache.mu.Lock()
	defer blockHeightCache.mu.Unlock()

	// Double-check the condition after acquiring the write lock
	if blockHeightCache.cachedHeight != 0 && time.Since(blockHeightCache.lastUpdated) < blockHeightCache.invalidationDur {
		return blockHeightCache.cachedHeight, nil
	}

	lastBlock := shannonSDK.blockClient.LastBlock(ctx)
	if lastBlock == nil {
		return 0, fmt.Errorf("failed to get last block")
	}

	newHeight := lastBlock.Height()
	blockHeightCache.cachedHeight = newHeight
	blockHeightCache.lastUpdated = time.Now()

	return newHeight, nil
}

// NewShannonSDK creates a new ShannonSDK instance with the given signing key and dependencies.
func NewShannonSDK(
	ctx context.Context,
	signingKey cryptotypes.PrivKey,
	deps depinject.Config,
) (*ShannonSDK, error) {
	sessionClient, err := query.NewSessionQuerier(deps)
	if err != nil {
		return nil, fmt.Errorf("failed to create session querier: %w", err)
	}

	accountClient, err := query.NewAccountQuerier(deps)
	if err != nil {
		return nil, fmt.Errorf("failed to create account querier: %w", err)
	}

	appClient, err := query.NewApplicationQuerier(deps)
	if err != nil {
		return nil, fmt.Errorf("failed to create application querier: %w", err)
	}

	var blockClient client.BlockClient
	if err := depinject.Inject(deps, &blockClient); err != nil {
		return nil, fmt.Errorf("failed to inject block client: %w", err)
	}

	signer, err := NewSigner(signingKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %w", err)
	}

	return &ShannonSDK{
		blockClient:   blockClient,
		sessionClient: sessionClient,
		accountClient: accountClient,
		appClient:     appClient,
		relayClient:   http.DefaultClient,
		signer:        signer,
	}, nil
}

// SendRelay builds and sends a relay request to the given endpoint.
func (shannonSDK *ShannonSDK) SendRelay(
	ctx context.Context,
	appAddress string,
	endpoint shannonsdk.Endpoint,
	requestBz []byte,
) (*types.RelayResponse, error) {
	relayRequest, err := shannonsdk.BuildRelayRequest(endpoint, requestBz)
	if err != nil {
		return nil, fmt.Errorf("failed to build relay request: %w", err)
	}

	application, err := shannonSDK.appClient.GetApplication(ctx, appAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get application: %w", err)
	}

	appRing := shannonsdk.ApplicationRing{
		PublicKeyFetcher: shannonSDK.accountClient,
		Application:      application,
	}

	if _, err = shannonSDK.signer.Sign(ctx, relayRequest, appRing); err != nil {
		return nil, fmt.Errorf("failed to sign relay request: %w", err)
	}

	relayRequestBz, err := relayRequest.Marshal()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal relay request: %w", err)
	}

	response, err := shannonSDK.relayClient.Post(
		endpoint.Endpoint().Url,
		"application/json",
		bytes.NewReader(relayRequestBz),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send relay request: %w", err)
	}
	defer response.Body.Close()

	responseBodyBz, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return shannonsdk.ValidateRelayResponse(
		ctx,
		endpoint.Supplier(),
		responseBodyBz,
		shannonSDK.accountClient,
	)
}

// GetSessionSupplierEndpoints returns the current session's supplier endpoints,
// using a cached result if available.
func (shannonSDK *ShannonSDK) GetSessionSupplierEndpoints(
	ctx context.Context,
	appAddress, serviceId string,
) (*shannonsdk.SessionFilter, error) {
	currentHeight, err := shannonSDK.GetCachedHeight(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current height: %w", err)
	}

	key := cacheKey(appAddress, serviceId, currentHeight)

	sessionCacheInstance.mu.RLock()
	if result, found := sessionCacheInstance.cache[key]; found {
		sessionCacheInstance.mu.RUnlock()
		return result.filter, nil
	}
	sessionCacheInstance.mu.RUnlock()

	session, err := shannonSDK.sessionClient.GetSession(ctx, appAddress, serviceId, currentHeight)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Create a deep copy of the session to ensure immutability
	sessionCopy := *session

	filteredSession := &shannonsdk.SessionFilter{
		Session: &sessionCopy, // Store a pointer to our copy
	}

	sessionCacheInstance.mu.Lock()
	sessionCacheInstance.cache[key] = &cachedSession{
		filter: filteredSession,
		height: currentHeight,
	}
	sessionCacheInstance.mu.Unlock()

	sessionCacheInstance.cleanupCache(currentHeight)

	return filteredSession, nil
}

// SetHeightCacheInvalidationDuration allows configuration of the block height cache invalidation duration.
func SetHeightCacheInvalidationDuration(duration time.Duration) {
	blockHeightCache.mu.Lock()
	defer blockHeightCache.mu.Unlock()
	blockHeightCache.invalidationDur = duration
}
