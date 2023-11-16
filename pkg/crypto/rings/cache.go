package rings

import (
	"context"
	"log"
	"sync"

	"cosmossdk.io/depinject"
	ring_secp256k1 "github.com/athanorlabs/go-dleq/secp256k1"
	ringtypes "github.com/athanorlabs/go-dleq/types"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	accounttypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/noot/ring-go"

	"github.com/pokt-network/poktroll/pkg/relayer"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
)

var _ RingCache = (*ringCache)(nil)

type ringCache struct {
	// ringPointsCache maintains a map of application addresses to the points
	// on the secp256k1 curve that correspond to the public keys of the gateways
	// the application is delegated to. These are used to build the app's ring.
	ringPointsCache map[string][]ringtypes.Point
	ringPointsMu    *sync.RWMutex

	// clientCtx is the client context for the application, and is used to query
	// the application and account modules.
	clientCtx relayer.QueryClientContext

	// applicationQuerier is the querier for the application module, and is
	// used to get the addresses of the gateways an application is delegated to.
	applicationQuerier apptypes.QueryClient

	// accountQuerier is the querier for the account module, and is used to get
	// the public keys of the application and its delegated gateways.
	accountQuerier accounttypes.QueryClient
}

// NewRingCache returns a new RingCache instance. It requires a depinject.Config
// to be passed in, which is used to inject the dependencies of the RingCache.
func NewRingCache(deps depinject.Config) (RingCache, error) {
	rc := &ringCache{
		ringPointsCache: make(map[string][]ringtypes.Point),
		ringPointsMu:    &sync.RWMutex{},
	}

	if err := depinject.Inject(
		deps,
		&rc.clientCtx,
	); err != nil {
		return nil, err
	}

	clientCtx := cosmosclient.Context(rc.clientCtx)

	rc.accountQuerier = accounttypes.NewQueryClient(clientCtx)
	rc.applicationQuerier = apptypes.NewQueryClient(clientCtx)

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
	var ring *ring.Ring
	var err error

	// lock the cache for reading
	rc.ringPointsMu.RLock()
	// check if the ring is in the cache
	points, ok := rc.ringPointsCache[appAddress]
	// unlock the cache incase it was not cached
	rc.ringPointsMu.RUnlock()

	if !ok {
		// if the ring is not in the cache, get it from the application module
		log.Printf("DEBUG: Ring not in cache, fetching from application module [%s]", appAddress)
		ring, err = rc.getRingForAppAddress(ctx, appAddress)
	} else {
		// if the ring is in the cache, create it from the points
		log.Printf("DEBUG: Ring in cache, creating from points [%s]", appAddress)
		ring, err = newRingFromPoints(points)
	}
	if err != nil {
		return nil, err
	}

	// return the ring
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
	rc.ringPointsMu.Lock()
	defer rc.ringPointsMu.Unlock()

	// get the application's on chain state
	req := &apptypes.QueryGetApplicationRequest{Address: appAddress}
	res, err := rc.applicationQuerier.Application(ctx, req)
	if err != nil {
		return nil, ErrRingsAccountNotFound.Wrapf("app address: %s [%v]", appAddress, err)
	}

	// create a slice of addresses for the ring
	ringAddresses := make([]string, 0)
	ringAddresses = append(ringAddresses, appAddress) // app address is index 0
	if len(res.Application.DelegateeGatewayAddresses) == 0 {
		// add app address twice to make the ring size of mininmum 2
		// TODO_HACK: We are adding the appAddress twice because a ring
		// signature requires AT LEAST two pubKeys. When the Application has
		// not delegated to any gateways, we add the application's own address
		// twice. This is a HACK and should be investigated as to what is the
		// best approach to take in this situation.
		ringAddresses = append(ringAddresses, appAddress)
	} else {
		// add the delegatee gateway addresses
		ringAddresses = append(ringAddresses, res.Application.DelegateeGatewayAddresses...)
	}

	// get the points on the secp256k1 curve for the addresses
	points, err := rc.addressesToPoints(ctx, ringAddresses)
	if err != nil {
		return nil, err
	}

	// update the cache overwriting the previous value
	log.Printf("DEBUG: Updating ring cache for [%s]", appAddress)
	rc.ringPointsCache[appAddress] = points

	// return the public key points on the secp256k1 curve
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
	for i, addr := range addresses {
		pubKeyReq := &accounttypes.QueryAccountRequest{Address: addr}
		pubKeyRes, err := rc.accountQuerier.Account(ctx, pubKeyReq)
		if err != nil {
			return nil, ErrRingsAccountNotFound.Wrapf("address: %s [%v]", addr, err)
		}
		var acc accounttypes.AccountI
		if err = ringCodec.UnpackAny(pubKeyRes.Account, &acc); err != nil {
			return nil, ErrRingsUnableToDeserialiseAccount.Wrapf("address: %s [%v]", addr, err)
		}
		key := acc.GetPubKey()
		if _, ok := key.(*secp256k1.PubKey); !ok {
			return nil, ErrRingsWrongCurve.Wrapf("got %T", key)
		}
		point, err := curve.DecodeToPoint(key.Bytes())
		if err != nil {
			return nil, err
		}
		points[i] = point
	}
	return points, nil
}
