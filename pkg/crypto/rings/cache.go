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
	"github.com/pokt-network/poktroll/pkg/polylog"
)

var _ crypto.RingCache = (*ringCache)(nil)

type ringCache struct {
	// ringPointsCache maintains a map of application addresses to the points
	// on the secp256k1 curve that correspond to the public keys of the gateways
	// the application is delegated to. These are used to build the app's ring.
	ringPointsCache map[string][]ringtypes.Point
	ringPointsMu    *sync.RWMutex

	// applicationQuerier is the querier for the application module, and is
	// used to get the addresses of the gateways an application is delegated to.
	applicationQuerier client.ApplicationQueryClient

	// accountQuerier is the querier for the account module, and is used to get
	// the public keys of the application and its delegated gateways.
	accountQuerier client.AccountQueryClient
}

// NewRingCache returns a new RingCache instance. It requires a depinject.Config
// to be passed in, which is used to inject the dependencies of the RingCache.
func NewRingCache(deps depinject.Config) (crypto.RingCache, error) {
	rc := &ringCache{
		ringPointsCache: make(map[string][]ringtypes.Point),
		ringPointsMu:    &sync.RWMutex{},
	}

	// Supply the account and application queriers to the RingCache.
	if err := depinject.Inject(
		deps,
		&rc.applicationQuerier,
		&rc.accountQuerier,
	); err != nil {
		return nil, err
	}

	return rc, nil
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
		ring   *ring.Ring
		err    error
		logger = polylog.Ctx(ctx)
	)

	// Lock the cache for reading.
	rc.ringPointsMu.RLock()
	// Check if the ring is in the cache.
	points, ok := rc.ringPointsCache[appAddress]
	// Unlock the cache incase it was not cached.
	rc.ringPointsMu.RUnlock()

	if !ok {
		// If the ring is not in the cache, get it from the application module.
		logger.Debug().
			Str("app_address", appAddress).
			Msg("ring cache miss; fetching from application module")
		ring, err = rc.getRingForAppAddress(ctx, appAddress)
	} else {
		// If the ring is in the cache, create it from the points.
		logger.Debug().
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
	logger := polylog.Ctx(ctx)

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

	// Update the cache overwriting the previous value.
	logger.Debug().
		Str("app_address", appAddress).
		Msg("updating ring cache for app")
	rc.ringPointsCache[appAddress] = points

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
	logger := polylog.Ctx(ctx)

	curve := ring_secp256k1.NewCurve()
	points := make([]ringtypes.Point, len(addresses))
	logger.Debug().
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
