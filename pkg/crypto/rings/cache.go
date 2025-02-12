package rings

import (
	"context"
	"sync"

	"cosmossdk.io/depinject"
	"github.com/pokt-network/ring-go"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/crypto"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/polylog"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	"github.com/pokt-network/poktroll/x/service/types"
)

var _ crypto.RingCache = (*ringCache)(nil)

type ringCache struct {
	// logger is the logger for the ring cache.
	logger polylog.Logger

	// ringsByAddr maintains a map from app addresses to the ring composed of
	// the public keys of both the application and its delegated gateways.
	ringsByAddr   map[string]*ring.Ring
	ringsByAddrMu *sync.RWMutex

	// delegationClient is used to listen for onchain delegation events and
	// invalidate entries in ringsByAddr if an associated updated has been made.
	delegationClient client.DelegationClient

	// ringClient is used to retrieve cached rings and verify relay requests.
	ringClient crypto.RingClient
}

// NewRingCache returns a new RingCache instance. It requires a depinject.Config
// to be passed in, which is used to inject the dependencies of the RingCache.
//
// Required dependencies:
// - polylog.Logger
// - client.DelegationClient
// - client.ApplicationQueryClient
// - client.AccountQueryClient
func NewRingCache(deps depinject.Config) (_ crypto.RingCache, err error) {
	rc := &ringCache{
		ringsByAddr:   make(map[string]*ring.Ring),
		ringsByAddrMu: &sync.RWMutex{},
	}

	// Supply the account and application queriers to the RingCache.
	if err = depinject.Inject(
		deps,
		&rc.logger,
		&rc.delegationClient,
	); err != nil {
		return nil, err
	}

	// Construct and assign underlying ring client.
	rc.ringClient, err = NewRingClient(deps)
	if err != nil {
		return nil, err
	}

	return rc, nil
}

// Start starts the ring cache by subscribing to onchain redelegation events.
func (rc *ringCache) Start(ctx context.Context) {
	rc.logger.Info().Msg("starting ring cache")
	// Stop the ringCache when the context is cancelled.
	go func() {
		<-ctx.Done()
		rc.Stop()
	}()
	// Listen for redelegation events and invalidate the cache if it contains an
	// address corresponding to the redelegation event's.
	go rc.goInvalidateCache(ctx)
}

// goInvalidateCache listens for redelegation events and invalidates the
// cache if the ring corresponding to the app address in the redelegation event
// exists in the cache.
// This function is intended to be run in a goroutine.
func (rc *ringCache) goInvalidateCache(ctx context.Context) {
	// Get the latest redelegation replay observable.
	redelegationObs := rc.delegationClient.RedelegationsSequence(ctx)
	// For each redelegation event, check if the redelegation events'
	// app address is in the cache. If it is, invalidate the cache entry.
	channel.ForEach[*apptypes.EventRedelegation](
		ctx, redelegationObs,
		func(ctx context.Context, redelegation *apptypes.EventRedelegation) {
			// Lock ringsByAddr for writing.
			rc.ringsByAddrMu.Lock()
			defer rc.ringsByAddrMu.Unlock()
			// Check if the redelegation event's app address is in the cache.
			appAddr := redelegation.GetApplication().GetAddress()
			if _, ok := rc.ringsByAddr[appAddr]; ok {
				rc.logger.Debug().
					Str("app_address", appAddr).
					Msg("redelegation event received; invalidating ringsByAddr entry")
				// Invalidate the ringsByAddr entry.
				delete(rc.ringsByAddr, appAddr)
			}
		})
}

// Stop stops the ring cache by unsubscribing from onchain redelegation events
// and clears any existing entries.
func (rc *ringCache) Stop() {
	// Clear the cache.
	rc.ringsByAddrMu.Lock()
	rc.ringsByAddr = make(map[string]*ring.Ring)
	rc.ringsByAddrMu.Unlock()
	// Close the delegation client.
	rc.delegationClient.Close()
}

// GetCachedAddresses returns the addresses of the applications that are
// currently cached in the ring cache.
func (rc *ringCache) GetCachedAddresses() []string {
	rc.ringsByAddrMu.RLock()
	defer rc.ringsByAddrMu.RUnlock()

	appAddresses := make([]string, 0, len(rc.ringsByAddr))
	for appAddr := range rc.ringsByAddr {
		appAddresses = append(appAddresses, appAddr)
	}
	return appAddresses
}

// GetRingForAddressAtHeight returns the ring for the address and block height provided.
// If it does not exist in the cache, it will be created by querying the application
// module and converting the addresses into their corresponding public key points on
// the secp256k1 curve. It will then be cached for future use.
func (rc *ringCache) GetRingForAddressAtHeight(
	ctx context.Context,
	appAddress string,
	blockHeight int64,
) (ring *ring.Ring, err error) {
	rc.ringsByAddrMu.Lock()
	defer rc.ringsByAddrMu.Unlock()

	// Check if the ring is in the cache.
	ring, ok := rc.ringsByAddr[appAddress]

	// Use the existing ring if it's cached.
	if ok {
		rc.logger.Debug().
			Str("app_address", appAddress).
			Msg("ring cache hit; using cached ring")

		return ring, nil
	}

	// If the ring is not in the cache, get it from the ring client.
	rc.logger.Debug().
		Str("app_address", appAddress).
		Msg("ring cache miss; fetching from application module")

	ring, err = rc.ringClient.GetRingForAddressAtHeight(ctx, appAddress, blockHeight)
	if err != nil {
		return nil, err
	}
	rc.ringsByAddr[appAddress] = ring

	return ring, nil
}

// VerifyRelayRequestSignature verifies the relay request signature against the
// ring for the application address in the relay request.
func (rc *ringCache) VerifyRelayRequestSignature(
	ctx context.Context,
	relayRequest *types.RelayRequest,
) error {
	return rc.ringClient.VerifyRelayRequestSignature(ctx, relayRequest)
}
