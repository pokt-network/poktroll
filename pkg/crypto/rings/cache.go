package rings

import (
	"context"
	"fmt"
	"sync"

	"cosmossdk.io/depinject"
	ring_secp256k1 "github.com/athanorlabs/go-dleq/secp256k1"
	ringtypes "github.com/athanorlabs/go-dleq/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
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

	// ringPointsCache maintains a map of application addresses to the points
	// on the secp256k1 curve that correspond to the public keys of the gateways
	// the application is delegated to. These are used to build the app's ring.
	ringPointsCache map[string][]ringtypes.Point
	ringPointsMu    *sync.RWMutex

	// delegationClient is used to listen for on-chain delegation events and
	// invalidate cache entries for rings that have been updated on chain.
	delegationClient client.DelegationClient

	// applicationQuerier is the querier for the application module, and is
	// used to get the addresses of the gateways an application is delegated to.
	applicationQuerier client.ApplicationQueryClient

	// accountQuerier is the querier for the account module, and is used to get
	// the public keys of the application and its delegated gateways.
	accountQuerier client.AccountQueryClient
}

// NewRingCache returns a new RingCache instance. It requires a depinject.Config
// to be passed in, which is used to inject the dependencies of the RingCache.
//
// Required dependencies:
// - polylog.Logger
// - client.DelegationClient
// - client.ApplicationQueryClient
// - client.AccountQueryClient
func NewRingCache(deps depinject.Config) (crypto.RingCache, error) {
	rc := &ringCache{
		ringPointsCache: make(map[string][]ringtypes.Point),
		ringPointsMu:    &sync.RWMutex{},
	}

	// Supply the account and application queriers to the RingCache.
	if err := depinject.Inject(
		deps,
		&rc.logger,
		&rc.delegationClient,
		&rc.applicationQuerier,
		&rc.accountQuerier,
	); err != nil {
		return nil, err
	}

	return rc, nil
}

// Start starts the ring cache by subscribing to on-chain redelegation events.
func (rc *ringCache) Start(ctx context.Context) {
	rc.logger.Info().Msg("starting ring cache")
	// Listen for redelegation events and invalidate the cache if the
	// redelegation event's address is stored in the cache.
	go rc.goInvalidateCache(ctx)
}

// goInvalidateCache listens for delegatee change events and invalidates the
// cache if the delegatee change's address is stored in the cache.
// It is intended to be run in a goroutine.
func (rc *ringCache) goInvalidateCache(ctx context.Context) {
	// Get the latest redelegation replay observable.
	redelegationObs := rc.delegationClient.RedelegationsSequence(ctx)
	// For each redelegation event, check if the redelegation events's
	// app address is in the cache. If it is, invalidate the cache entry.
	channel.ForEach[client.Redelegation](
		ctx, redelegationObs,
		func(ctx context.Context, redelegation client.Redelegation) {
			// Lock the cache for writing.
			rc.ringPointsMu.Lock()
			// Check if the redelegation event's app address is in the cache.
			if _, ok := rc.ringPointsCache[redelegation.GetAppAddress()]; ok {
				rc.logger.Debug().
					Str("app_address", redelegation.GetAppAddress()).
					Msg("redelegation event received; invalidating cache entry")
				// Invalidate the cache entry.
				delete(rc.ringPointsCache, redelegation.GetAppAddress())
			}
			// Unlock the cache.
			rc.ringPointsMu.Unlock()
		})
}

// Stop stops the ring cache by unsubscribing from on-chain redelegation events.
func (rc *ringCache) Stop() {
	// Clear the cache.
	rc.ringPointsMu.Lock()
	rc.ringPointsCache = make(map[string][]ringtypes.Point)
	rc.ringPointsMu.Unlock()
	// Close the delegation client.
	rc.delegationClient.Close()
}

// GetCachedAddresses returns the addresses of the applications that are
// currently cached in the ring cache.
func (rc *ringCache) GetCachedAddresses() []string {
	rc.ringPointsMu.RLock()
	defer rc.ringPointsMu.RUnlock()
	keys := make([]string, 0, len(rc.ringPointsCache))
	for k := range rc.ringPointsCache {
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
) (*ring.Ring, error) {
	var (
		ring *ring.Ring
		err  error
	)

	// Lock the cache for reading.
	rc.ringPointsMu.RLock()
	// Check if the ring is in the cache.
	points, ok := rc.ringPointsCache[appAddress]
	// Unlock the cache in case it was not cached.
	rc.ringPointsMu.RUnlock()

	if !ok {
		// If the ring is not in the cache, get it from the application module.
		rc.logger.Debug().
			Str("app_address", appAddress).
			Msg("ring cache miss; fetching from application module")
		ring, err = rc.getRingForAppAddress(ctx, appAddress)
	} else {
		// If the ring is in the cache, create it from the points.
		rc.logger.Debug().
			Str("app_address", appAddress).
			Msg("ring cache hit; creating from points")
		ring, err = newRingFromPoints(points)
	}
	if err != nil {
		return nil, err
	}

	// Return the ring.
	return ring, nil
}

// getRingForAppAddress returns the RingSinger used to sign relays. It does so by fetching
// the latest information from the application module and creating the correct ring.
// This method also caches the ring's public keys for future use.
func (rc *ringCache) getRingForAppAddress(
	ctx context.Context,
	appAddress string,
) (*ring.Ring, error) {
	points, err := rc.getDelegatedPubKeysForAddress(ctx, appAddress)
	if err != nil {
		return nil, err
	}
	// Cache the ring's points for future use
	rc.logger.Debug().
		Str("app_address", appAddress).
		Msg("updating ring cache for app")
	rc.ringPointsMu.Lock()
	defer rc.ringPointsMu.Unlock()
	rc.ringPointsCache[appAddress] = points
	return newRingFromPoints(points)
}

// newRingFromPoints creates a new ring from points on the secp256k1 curve
func newRingFromPoints(points []ringtypes.Point) (*ring.Ring, error) {
	return ring.NewFixedKeyRingFromPublicKeys(ring_secp256k1.NewCurve(), points)
}

// getDelegatedPubKeysForAddress returns the ring used to sign a message for
// the given application address, by querying the application module for it's
// delegated pubkeys and converting them to points on the secp256k1 curve in
// order to create the ring.
func (rc *ringCache) getDelegatedPubKeysForAddress(
	ctx context.Context,
	appAddress string,
) ([]ringtypes.Point, error) {
	rc.ringPointsMu.Lock()
	defer rc.ringPointsMu.Unlock()

	// Get the application's on chain state.
	app, err := rc.applicationQuerier.GetApplication(ctx, appAddress)
	if err != nil {
		return nil, err
	}

	// Create a slice of addresses for the ring.
	ringAddresses := make([]string, 0)
	ringAddresses = append(ringAddresses, appAddress) // app address is index 0
	if len(app.DelegateeGatewayAddresses) == 0 {
		// add app address twice to make the ring size of mininmum 2
		// TODO_HACK: We are adding the appAddress twice because a ring
		// signature requires AT LEAST two pubKeys. When the Application has
		// not delegated to any gateways, we add the application's own address
		// twice. This is a HACK and should be investigated as to what is the
		// best approach to take in this situation.
		ringAddresses = append(ringAddresses, appAddress)
	} else {
		// add the delegatee gateway addresses
		ringAddresses = append(ringAddresses, app.DelegateeGatewayAddresses...)
	}

	// Get the points on the secp256k1 curve for the addresses.
	points, err := rc.addressesToPoints(ctx, ringAddresses)
	if err != nil {
		return nil, err
	}

	// Return the public key points on the secp256k1 curve.
	return points, nil
}

// addressesToPoints converts a slice of addresses to a slice of points on the
// secp256k1 curve, by querying the account module for the public key for each
// address and converting them to the corresponding points on the secp256k1 curve
func (rc *ringCache) addressesToPoints(
	ctx context.Context,
	addresses []string,
) ([]ringtypes.Point, error) {
	curve := ring_secp256k1.NewCurve()
	points := make([]ringtypes.Point, len(addresses))
	rc.logger.Debug().
		// TODO_TECHDEBT: implement and use `polylog.Event#Strs([]string)` instead of formatting here.
		Str("addresses", fmt.Sprintf("%v", addresses)).
		Msg("converting addresses to points")
	for i, addr := range addresses {
		// Retrieve the account from the auth module
		acc, err := rc.accountQuerier.GetAccount(ctx, addr)
		if err != nil {
			return nil, err
		}
		key := acc.GetPubKey()
		// Check if the key is a secp256k1 public key
		if _, ok := key.(*secp256k1.PubKey); !ok {
			return nil, ErrRingsNotSecp256k1Curve.Wrapf("got %T", key)
		}
		// Convert the public key to the point on the secp256k1 curve
		point, err := curve.DecodeToPoint(key.Bytes())
		if err != nil {
			return nil, err
		}
		// Insert the point into the slice of points
		points[i] = point
	}
	return points, nil
}
