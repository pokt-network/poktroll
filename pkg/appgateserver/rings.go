// TODO(@h5law): Move all this logic out into a shared package
package appgateserver

import (
	"context"
	"fmt"
	"log"

	ring_secp256k1 "github.com/athanorlabs/go-dleq/secp256k1"
	ringtypes "github.com/athanorlabs/go-dleq/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	accounttypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	ring "github.com/noot/ring-go"

	"github.com/pokt-network/poktroll/pkg/signer"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
)

// getRingSingerForAppAddress returns the RingSinger used to sign relays.
// This method first attempts to get the points of the ring from the cache, if it
// fails it queries the application module for the points and creates the ring.
func (app *appGateServer) getRingSingerForAppAddress(ctx context.Context, appAddress string) (*signer.RingSigner, error) {
	var ring *ring.Ring
	var err error

	// lock the cache for reading
	app.ringCacheMutex.RLock()
	defer app.ringCacheMutex.RUnlock()

	// check if the ring is in the cache
	points, ok := app.ringCache[appAddress]
	if !ok {
		// if the ring is not in the cache, get it from the application module
		log.Printf("DEBUG: No ring cached for address: %s", appAddress)
		ring, err = app.getRingForAppAddress(ctx, appAddress)
	} else {
		// if the ring is in the cache, create it from the points
		log.Printf("DEBUG: Ring cached for address: %s", appAddress)
		ring, err = newRingFromPoints(points)
	}
	if err != nil {
		log.Printf("ERROR: Unable to get ring for address: %s [%v]", appAddress, err)
		return nil, err
	}

	// return the ring signer
	return signer.NewRingSigner(ring, app.signingInformation.SigningKey), nil
}

// getRingForAppAddress returns the RingSinger used to sign relays. It does so by fetching
// the latest information from the application module and creating the correct ring.
// This method also caches the ring's public keys for future use.
func (app *appGateServer) getRingForAppAddress(ctx context.Context, appAddress string) (*ring.Ring, error) {
	points, err := app.getDelegatedPubKeysForAddress(ctx, appAddress)
	if err != nil {
		return nil, err
	}
	return newRingFromPoints(points)
}

// newRingFromPoints creates a new ring from a slice of points on the secp256k1 curve
func newRingFromPoints(points []ringtypes.Point) (*ring.Ring, error) {
	return ring.NewFixedKeyRingFromPublicKeys(ring_secp256k1.NewCurve(), points)
}

// getDelegatedPubKeysForAddress returns the ring used to sign a message for the given
// application address, by querying the application module for it's delegated pubkeys
func (app *appGateServer) getDelegatedPubKeysForAddress(
	ctx context.Context,
	appAddress string,
) ([]ringtypes.Point, error) {
	app.ringCacheMutex.RLock()
	defer app.ringCacheMutex.RUnlock()

	// get the application's on chain state
	req := &apptypes.QueryGetApplicationRequest{Address: appAddress}
	res, err := app.applicationQuerier.Application(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve application for address: %s [%w]", appAddress, err)
	}

	// create a slice of addresses for the ring
	ringAddresses := make([]string, 0)
	ringAddresses = append(ringAddresses, appAddress) // app address is index 0
	if len(res.Application.DelegateeGatewayAddresses) < 1 {
		// add app address twice to make the ring size of mininmum 2
		// TODO_TECHDEBT: We are adding the appAddress twice because a ring
		// signature requires AT LEAST two pubKeys. When the Application has
		// not delegated to any gateways, we add the application's own address
		// twice. This is a HACK and should be investigated as to what is the
		// best approach to take in this situation.
		ringAddresses = append(ringAddresses, appAddress)
	} else if len(res.Application.DelegateeGatewayAddresses) > 0 {
		// add the delegatee gateway addresses
		ringAddresses = append(ringAddresses, res.Application.DelegateeGatewayAddresses...)
	}

	// get the points on the secp256k1 curve for the addresses
	points, err := app.addressesToPoints(ctx, ringAddresses)
	if err != nil {
		return nil, err
	}

	// update the cache overwriting the previous value
	app.ringCache[appAddress] = points

	// return the public key points on the secp256k1 curve
	return points, nil
}

// addressesToPoints converts a slice of addresses to a slice of points on the
// secp256k1 curve, by querying the account module for the public key for each
// address and converting them to the corresponding points on the secp256k1 curve
func (app *appGateServer) addressesToPoints(ctx context.Context, addresses []string) ([]ringtypes.Point, error) {
	curve := ring_secp256k1.NewCurve()
	points := make([]ringtypes.Point, len(addresses))
	for i, addr := range addresses {
		pubKeyReq := &accounttypes.QueryAccountRequest{Address: addr}
		pubKeyRes, err := app.accountQuerier.Account(ctx, pubKeyReq)
		if err != nil {
			return nil, fmt.Errorf("unable to get account for address: %s [%w]", addr, err)
		}
		var acc accounttypes.AccountI
		reg := codectypes.NewInterfaceRegistry()
		accounttypes.RegisterInterfaces(reg)
		cdc := codec.NewProtoCodec(reg)
		if err := cdc.UnpackAny(pubKeyRes.Account, &acc); err != nil {
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
