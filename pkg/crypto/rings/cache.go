package rings

import (
	"context"
	"sync"

	"cosmossdk.io/depinject"
	"github.com/noot/ring-go"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/crypto"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

var _ crypto.RingCache = (*ringCache)(nil)

type ringCache struct {
	// logger is the logger for the ring cache.
	logger polylog.Logger

	// ringsByAddr maintains a map of application addresses to the ring composed of
	// the public keys of the application and the gateways the application is delegated to.
	ringsByAddr   map[string]*ring.Ring
	ringsByAddrMu *sync.RWMutex

	// delegationClient is used to listen for on-chain delegation events and
	// invalidate ringsByAddr entries for rings that have been updated on chain.
	delegationClient client.DelegationClient

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
	if err := depinject.Inject(
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

// Start starts the ring cache by subscribing to on-chain redelegation events.
func (rc *ringCache) Start(ctx context.Context) {
	rc.logger.Info().Msg("starting ring ringsByAddr")
	// Listen for redelegation events and invalidate the cache if contains a ring
	// corresponding to the redelegation event's address .
	go func() {
		select {
		case <-ctx.Done():
			// Stop the ring cache if the context is cancelled.
			rc.Stop()
		}
	}()
	go rc.goInvalidateCache(ctx)
}

// goInvalidateCache listens for redelegation events and invalidates the
// cache if ring corresponding to the app address in the redelegation event
// exists in the cache.
// This function is intended to be run in a goroutine.
func (rc *ringCache) goInvalidateCache(ctx context.Context) {
	// Get the latest redelegation replay observable.
	redelegationObs := rc.delegationClient.RedelegationsSequence(ctx)
	// For each redelegation event, check if the redelegation events'
	// app address is in the cache. If it is, invalidate the cache entry.
	channel.ForEach[client.Redelegation](
		ctx, redelegationObs,
		func(ctx context.Context, redelegation client.Redelegation) {
			// Lock ringsByAddr for writing.
			rc.ringsByAddrMu.Lock()
			defer rc.ringsByAddrMu.Unlock()
			// Check if the redelegation event's app address is in the cache.
			if _, ok := rc.ringsByAddr[redelegation.GetAppAddress()]; ok {
				rc.logger.Debug().
					Str("app_address", redelegation.GetAppAddress()).
					Msg("redelegation event received; invalidating ringsByAddr entry")
				// Invalidate the ringsByAddr entry.
				delete(rc.ringsByAddr, redelegation.GetAppAddress())
			}
		})
}

// Stop stops the ring cache by unsubscribing from on-chain redelegation events.
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
	keys := make([]string, 0, len(rc.ringsByAddr))
	for k := range rc.ringsByAddr {
		keys = append(keys, k)
	}
	return keys
}

// GetRingForAddress returns the ring for the address provided. If it does not
// exist in the cache, it will be created by querying the application module.
// and converting the addresses into their corresponding public key points on
// the secp256k1 curve. It will then be cached for future use.
func (rc *ringCache) GetRingForAddress(
	ctx context.Context,
	appAddress string,
) (ring *ring.Ring, err error) {
	// Lock the ringsByAddr map.
	rc.ringsByAddrMu.Lock()
	defer rc.ringsByAddrMu.Unlock()

	// Check if the ring is in the cache.
	ring, ok := rc.ringsByAddr[appAddress]

	if !ok {
		// If the ring is not in the cache, get it from the ring client.
		rc.logger.Debug().
			Str("app_address", appAddress).
			Msg("ring ringsByAddr miss; fetching from application module")
		ring, err = rc.ringClient.GetRingForAddress(ctx, appAddress)

		// Add the address points to the cache.
		rc.ringsByAddr[appAddress] = ring
	} else {
		// If the ring is in the cache, create it from the points.
		rc.logger.Debug().
			Str("app_address", appAddress).
			Msg("ring ringsByAddr hit; creating from points")
	}
	if err != nil {
		return nil, err
	}

	// Return the ring.
	return ring, nil
}
