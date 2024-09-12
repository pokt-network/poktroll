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

// ShannonSDK is a wrapper around the Shannon SDK that is used by the AppGateServer
// to encapsulate the SDK's functionality and dependencies.
type ShannonSDK struct {
	blockClient   client.BlockClient
	sessionClient client.SessionQueryClient
	appClient     client.ApplicationQueryClient
	accountClient client.AccountQueryClient
	relayClient   *http.Client
	signer        *shannonsdk.Signer
}

// Cache structure to store session results and ensure thread-safety
type sessionCache struct {
	mu    sync.RWMutex
	cache map[string]*cachedSession
}

// Cached session data
type cachedSession struct {
	filter *shannonsdk.SessionFilter
	height int64
}

var sessionCacheInstance = sessionCache{
	cache: make(map[string]*cachedSession),
}

// Cache structure for block height
type heightCache struct {
	mu              sync.RWMutex
	cachedHeight    int64
	lastUpdated     time.Time
	invalidationDur time.Duration
}

// Height cache instance with a configurable invalidation duration
var blockHeightCache = heightCache{
	invalidationDur: time.Second, // Default 1 second
}

// Generate cache key from appAddress, serviceId, and height
func cacheKey(appAddress, serviceId string, height int64) string {
	return fmt.Sprintf("%s-%s-%d", appAddress, serviceId, height)
}

// Remove old cache entries with heights smaller than the current height
func (c *sessionCache) cleanupCache(currentHeight int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, entry := range c.cache {
		if entry.height < currentHeight {
			delete(c.cache, key)
		}
	}
}

// GetCachedHeight retrieves the block height and caches it. It invalidates
// the cached value if the configured duration has passed since the last update.
func (shannonSDK *ShannonSDK) GetCachedHeight(ctx context.Context) (int64, error) {
	blockHeightCache.mu.RLock()
	// Check if cached height is still valid
	if time.Since(blockHeightCache.lastUpdated) < blockHeightCache.invalidationDur {
		cachedHeight := blockHeightCache.cachedHeight
		blockHeightCache.mu.RUnlock()
		return cachedHeight, nil
	}
	blockHeightCache.mu.RUnlock()

	// Fetch new height from the block client
	newHeight := shannonSDK.blockClient.LastBlock(ctx).Height()

	// Update the cached height
	blockHeightCache.mu.Lock()
	blockHeightCache.cachedHeight = newHeight
	blockHeightCache.lastUpdated = time.Now()
	blockHeightCache.mu.Unlock()

	return newHeight, nil
}

// NewShannonSDK creates a new ShannonSDK instance with the given signing key and dependencies.
// It initializes the necessary clients and signer for the SDK.
func NewShannonSDK(
	ctx context.Context,
	signingKey cryptotypes.PrivKey,
	deps depinject.Config,
) (*ShannonSDK, error) {
	sessionClient, sessionClientErr := query.NewSessionQuerier(deps)
	if sessionClientErr != nil {
		return nil, sessionClientErr
	}

	accountClient, accountClientErr := query.NewAccountQuerier(deps)
	if accountClientErr != nil {
		return nil, accountClientErr
	}

	appClient, appClientErr := query.NewApplicationQuerier(deps)
	if appClientErr != nil {
		return nil, appClientErr
	}

	blockClient := client.BlockClient(nil)
	if depsErr := depinject.Inject(deps, &blockClient); depsErr != nil {
		return nil, depsErr
	}

	signer, signerErr := NewSigner(signingKey)
	if signerErr != nil {
		return nil, signerErr
	}

	shannonSDK := &ShannonSDK{
		blockClient:   blockClient,
		sessionClient: sessionClient,
		accountClient: accountClient,
		appClient:     appClient,
		relayClient:   http.DefaultClient,
		signer:        signer,
	}

	return shannonSDK, nil
}

// SendRelay builds a relay request from the given requestBz, signs it with the
// application address, then sends it to the given endpoint.
func (shannonSDK *ShannonSDK) SendRelay(
	ctx context.Context,
	appAddress string,
	endpoint shannonsdk.Endpoint,
	requestBz []byte,
) (*types.RelayResponse, error) {
	relayRequest, err := shannonsdk.BuildRelayRequest(endpoint, requestBz)
	if err != nil {
		return nil, err
	}

	application, err := shannonSDK.appClient.GetApplication(ctx, appAddress)
	if err != nil {
		return nil, err
	}

	appRing := shannonsdk.ApplicationRing{
		PublicKeyFetcher: shannonSDK.accountClient,
		Application:      application,
	}

	if _, err = shannonSDK.signer.Sign(ctx, relayRequest, appRing); err != nil {
		return nil, err
	}

	relayRequestBz, err := relayRequest.Marshal()
	if err != nil {
		return nil, err
	}

	response, err := shannonSDK.relayClient.Post(
		endpoint.Endpoint().Url,
		"application/json",
		bytes.NewReader(relayRequestBz),
	)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	responseBodyBz, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return shannonsdk.ValidateRelayResponse(
		ctx,
		endpoint.Supplier(),
		responseBodyBz,
		shannonSDK.accountClient,
	)
}

// GetSessionSupplierEndpoints returns the current session's supplier endpoints
// for the given appAddress and serviceId, using a cached result if available.
func (shannonSDK *ShannonSDK) GetSessionSupplierEndpoints(
	ctx context.Context,
	appAddress, serviceId string,
) (*shannonsdk.SessionFilter, error) {
	// Use the cached block height
	currentHeight, err := shannonSDK.GetCachedHeight(ctx)
	if err != nil {
		return nil, err
	}

	key := cacheKey(appAddress, serviceId, currentHeight)

	// Check if result is cached
	sessionCacheInstance.mu.RLock()
	if result, found := sessionCacheInstance.cache[key]; found {
		sessionCacheInstance.mu.RUnlock()
		return result.filter, nil
	}
	sessionCacheInstance.mu.RUnlock()

	// Fetch session from the session client if not cached
	session, err := shannonSDK.sessionClient.GetSession(ctx, appAddress, serviceId, currentHeight)
	if err != nil {
		return nil, err
	}

	filteredSession := &shannonsdk.SessionFilter{
		Session: session,
	}

	// Store the result in cache
	sessionCacheInstance.mu.Lock()
	sessionCacheInstance.cache[key] = &cachedSession{
		filter: filteredSession,
		height: currentHeight,
	}
	sessionCacheInstance.mu.Unlock()

	// Cleanup old cache entries
	sessionCacheInstance.cleanupCache(currentHeight)

	return filteredSession, nil
}

// SetHeightCacheInvalidationDuration allows configuration of the block height cache invalidation duration.
func SetHeightCacheInvalidationDuration(duration time.Duration) {
	blockHeightCache.mu.Lock()
	defer blockHeightCache.mu.Unlock()
	blockHeightCache.invalidationDur = duration
}
