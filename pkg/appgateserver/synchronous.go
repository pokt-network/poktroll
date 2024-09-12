package appgateserver

import (
	"context"
	"net/http"
	"time"

	shannonsdk "github.com/pokt-network/shannon-sdk"
	sdktypes "github.com/pokt-network/shannon-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// requestInfo is a struct that holds the information needed to handle a relay request.
type requestInfo struct {
	appAddress  string
	serviceId   string
	rpcType     sharedtypes.RPCType
	poktRequest *sdktypes.POKTHTTPRequest
	requestBz   []byte
}

// handleSynchronousRelay handles relay requests for synchronous protocols, where
// there is a one-to-one correspondence between the request and response.
// It does everything from preparing, signing and sending the request.
// It then blocks on the response to come back and forward it to the provided writer.
func (app *appGateServer) handleSynchronousRelay(
	ctx context.Context,
	reqInfo *requestInfo,
	writer http.ResponseWriter,
) error {
	serviceId := reqInfo.serviceId
	rpcType := reqInfo.rpcType
	poktRequest := reqInfo.poktRequest
	requestBz := reqInfo.requestBz
	appAddress := reqInfo.appAddress

	relaysTotal.
		With("service_id", serviceId, "rpc_type", rpcType.String()).
		Add(1)

	currentHeight := app.sdk.GetHeight(ctx)

	// Cache matching endpoints
	cacheKey := cacheKey{height: currentHeight, serviceId: serviceId, rpcType: rpcType}
	matchingEndpoints, err := app.getCachedEndpoints(ctx, cacheKey, appAddress)
	if err != nil {
		return ErrAppGateHandleRelay.Wrapf("getting matching endpoints: %s", err)
	}

	// Get a supplier URL and address for the given service and session.
	supplierEndpoint, err := app.getRelayerUrl(poktRequest.Url, matchingEndpoints)
	if err != nil {
		return ErrAppGateHandleRelay.Wrapf("getting supplier URL: %s", err)
	}

	relayResponse, err := app.sdk.SendRelay(ctx, appAddress, supplierEndpoint, requestBz)
	// If the relayResponse is nil, it means that err is not nil and the error
	// should be handled by the appGateServer.
	if relayResponse == nil {
		return err
	}
	// Here, neither the relayResponse nor the error are nil, so the relayResponse's
	// contains the upstream service's error response.
	if err != nil {
		return ErrAppGateUpstreamError.Wrap(string(relayResponse.Payload))
	}

	// Deserialize the RelayResponse payload to get the serviceResponse that will
	// be forwarded to the client.
	serviceResponse, err := sdktypes.DeserializeHTTPResponse(relayResponse.Payload)
	if err != nil {
		return ErrAppGateHandleRelay.Wrapf("deserializing response: %s", err)
	}

	app.logger.Debug().
		Str("relay_response_payload", string(serviceResponse.BodyBz)).
		Msg("writing relay response payload")

	// Reply to the client with the service's response status code and headers.
	// At this point the AppGateServer has not generated any internal errors, so
	// the whole response will be forwarded to the client as is, including the
	// status code and headers, be it an error or not.
	serviceResponse.CopyToHTTPHeader(writer.Header())
	writer.WriteHeader(int(serviceResponse.StatusCode))

	// Transmit the service's response body to the client.
	if _, err := writer.Write(serviceResponse.BodyBz); err != nil {
		return ErrAppGateHandleRelay.Wrapf("writing relay response payload: %s", err)
	}

	relaysSuccessTotal.
		With("service_id", serviceId, "rpc_type", rpcType.String()).
		Add(1)

	return nil
}

// getCachedEndpoints retrieves matching endpoints from cache or fetches and caches them
func (app *appGateServer) getCachedEndpoints(
	ctx context.Context,
	key cacheKey,
	appAddress string,
) ([]shannonsdk.Endpoint, error) {
	app.endpointCacheMu.RLock()
	entry, found := app.endpointCache[key]
	app.endpointCacheMu.RUnlock()

	if found {
		return entry.endpoints, nil
	}

	// If not found in cache, fetch and cache the endpoints
	sessionSuppliers, err := app.sdk.GetSessionSupplierEndpoints(ctx, key.height, appAddress, key.serviceId)
	if err != nil {
		return nil, ErrAppGateHandleRelay.Wrapf("getting current session: %s", err)
	}

	matchingEndpoints, err := app.getMatchingEndpoints(*sessionSuppliers, key.rpcType)
	if err != nil {
		return nil, ErrAppGateHandleRelay.Wrapf("getting matching endpoints: %s", err)
	}

	app.endpointCacheMu.Lock()
	app.endpointCache[key] = cacheEntry{
		endpoints: matchingEndpoints,
		timestamp: time.Now(),
	}
	app.endpointCacheMu.Unlock()

	// Trigger cache cleanup if not already running
	app.triggerCacheCleanup()

	return matchingEndpoints, nil
}

// triggerCacheCleanup starts a timer to clean up old cache entries
func (app *appGateServer) triggerCacheCleanup() {
	if app.cacheCleanupTimer == nil {
		app.cacheCleanupTimer = time.AfterFunc(3*time.Minute, func() {
			app.cleanupCache()
		})
	}
}

// cleanupCache removes old cache entries
func (app *appGateServer) cleanupCache() {
	app.endpointCacheMu.Lock()
	defer app.endpointCacheMu.Unlock()

	currentHeight := app.sdk.GetHeight(context.Background())
	for key, entry := range app.endpointCache {
		if key.height < currentHeight || time.Since(entry.timestamp) > 5*time.Minute {
			delete(app.endpointCache, key)
		}
	}

	// Reset the cleanup timer
	app.cacheCleanupTimer = nil
}
