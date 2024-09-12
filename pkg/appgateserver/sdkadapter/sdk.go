package sdkadapter

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"

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
// for the given appAddress and serviceId.
func (shannonSDK *ShannonSDK) GetSessionSupplierEndpoints(
	ctx context.Context,
	appAddress, serviceId string,
) (*shannonsdk.SessionFilter, error) {
	currentHeight := shannonSDK.blockClient.LastBlock(ctx).Height()

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

// cacheKey generates a unique key for the session cache.
func cacheKey(appAddress, serviceId string, height int64) string {
	return fmt.Sprintf("%s-%s-%d", appAddress, serviceId, height)
}

// sessionCacheInstance is a global instance of the session cache.
var sessionCacheInstance = sessionCache{
	cache: make(map[string]*cachedSession),
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
