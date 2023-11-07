// TODO(@h5law): Move all this logic out into a shared package
package proxy

import (
	"context"
	"fmt"

	ring_secp256k1 "github.com/athanorlabs/go-dleq/secp256k1"
	ringtypes "github.com/athanorlabs/go-dleq/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	accounttypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	ring "github.com/noot/ring-go"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
)

// getRingForAppAddress returns the RingSinger used to sign relays. It does so by fetching
// the latest information from the application module and creating the correct ring.
// This method also caches the ring's public keys for future use.
func (rp *relayerProxy) getRingForAppAddress(ctx context.Context, appAddress string) (*ring.Ring, error) {
	// lock the cache for reading
	rp.ringCacheMutex.RLock()
	defer rp.ringCacheMutex.RUnlock()

	// check if the ring is in the cache
	points, ok := rp.ringCache[appAddress]
	if !ok {
		// if the ring is not in the cache, get it from the application module
		return rp.getRingForAppAddress(ctx, appAddress)
	}
	// if the ring is in the cache, create it from the points
	return newRingFromPoints(points)
}

// newRingFromPoints creates a new ring from a slice of points on the secp256k1 curve
func newRingFromPoints(points []ringtypes.Point) (*ring.Ring, error) {
	return ring.NewFixedKeyRingFromPublicKeys(ring_secp256k1.NewCurve(), points)
}

// getDelegatedPubKeysForAddress returns the ring used to sign a message for the given application
// address, by querying the portal module for it's delegated pubkeys
func (rp *relayerProxy) getDelegatedPubKeysForAddress(
	ctx context.Context,
	appAddress string,
) ([]ringtypes.Point, error) {
	// get the application's on chain state
	req := &apptypes.QueryGetApplicationRequest{Address: appAddress}
	res, err := rp.applicationQuerier.Application(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve application for address: %s [%w]", appAddress, err)
	}

	// create a slice of addresses for the ring
	ringAddresses := make([]string, len(res.Application.DelegateeGatewayAddresses)+1) // +1 for app address
	ringAddresses[0] = appAddress                                                     // app address is index 0
	copy(ringAddresses[1:], res.Application.DelegateeGatewayAddresses)                // copy the gateway addresses

	// get the points on the secp256k1 curve for the addresses
	points, err := rp.addressesToPoints(ctx, ringAddresses)
	if err != nil {
		return nil, err
	}

	// update the cache overwriting the previous value
	rp.ringCacheMutex.Lock()
	defer rp.ringCacheMutex.Unlock()
	rp.ringCache[appAddress] = points

	// return the public key points on the secp256k1 curve
	return points, nil
}

// addressesToPoints converts a slice of addresses to a slice of points on the secp256k1 curve
// it does so by querying the account module for the public key for each address and converting
// them to the corresponding points on the secp256k1 curve
func (rp *relayerProxy) addressesToPoints(ctx context.Context, addresses []string) ([]ringtypes.Point, error) {
	curve := ring_secp256k1.NewCurve()
	points := make([]ringtypes.Point, len(addresses))
	for i, addr := range addresses {
		pubKeyReq := &accounttypes.QueryAccountRequest{Address: addr}
		pubKeyRes, err := rp.accountsQuerier.Account(ctx, pubKeyReq)
		if err != nil {
			return nil, fmt.Errorf("unable to get account for address: %s [%w]", addr, err)
		}
		acc := new(accounttypes.BaseAccount)
		if err := acc.Unmarshal(pubKeyRes.Account.Value); err != nil {
			return nil, fmt.Errorf("unable to deserialise account for address: %s [%w]", addr, err)
		}
		key := acc.GetPubKey()
		if _, ok := key.(*secp256k1.PubKey); !ok {
			return nil, fmt.Errorf("public key is not a secp256k1 key: got %T", key)
		}
		point, err := curve.DecodeToPoint(key.Bytes())
		if err != nil {
			return nil, err
		}
		points[i] = point
	}
	return points, nil
}
