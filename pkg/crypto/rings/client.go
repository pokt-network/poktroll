package rings

import (
	"context"
	"fmt"

	"cosmossdk.io/depinject"
	ringtypes "github.com/athanorlabs/go-dleq/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/noot/ring-go"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/crypto"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

var _ crypto.RingClient = (*ringClient)(nil)

type ringClient struct {
	// logger is the logger for the ring cache.
	logger polylog.Logger

	// applicationQuerier is the querier for the application module, and is
	// used to get the addresses of the gateways an application is delegated to.
	applicationQuerier client.ApplicationQueryClient

	// accountQuerier is the querier for the account module, and is used to get
	// the public keys of the application and its delegated gateways.
	accountQuerier client.AccountQueryClient
}

// NewRingClient returns a new ring client constructed from the given dependencies.
// It returns an error if the required dependencies are not supplied.
//
// Required dependencies:
// - polylog.Logger
// - client.ApplicationQueryClient
// - client.AccountQueryClient
func NewRingClient(deps depinject.Config) (crypto.RingClient, error) {
	rc := new(ringClient)

	if err := depinject.Inject(
		deps,
		&rc.logger,
		&rc.applicationQuerier,
		&rc.accountQuerier,
	); err != nil {
		return nil, err
	}

	return rc, nil
}

// GetRingForAddress returns the ring for the address provided. The ring is created by
// querying for the application address and delegated gateways' account public keys and
// converting them to their secp256k1 curve points.
func (rc *ringClient) GetRingForAddress(
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
		Msg("updating ring ringsByAddr for app")
	return newRingFromPoints(points)
}

// getDelegatedPubKeysForAddress returns the ring used to sign a message for
// the given application address, by querying the application module for it's
// delegated pubkeys and converting them to points on the secp256k1 curve in
// order to create the ring.
func (rc *ringClient) getDelegatedPubKeysForAddress(
	ctx context.Context,
	appAddress string,
) ([]ringtypes.Point, error) {
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
func (rc *ringClient) addressesToPoints(
	ctx context.Context,
	addresses []string,
) ([]ringtypes.Point, error) {
	publicKeys := make([]cryptotypes.PubKey, len(addresses))
	rc.logger.Debug().
		// TODO_TECHDEBT: implement and use `polylog.Event#Strs([]string)` instead of formatting here.
		Str("addresses", fmt.Sprintf("%v", addresses)).
		Msg("converting addresses to points")
	for i, addr := range addresses {
		acc, err := rc.accountQuerier.GetAccount(ctx, addr)
		if err != nil {
			return nil, err
		}
		publicKeys[i] = acc.GetPubKey()
	}

	return pointsFromPublicKeys(publicKeys...)
}
