package rings

import (
	"context"
	"fmt"
	"slices"

	"cosmossdk.io/depinject"
	ring_secp256k1 "github.com/athanorlabs/go-dleq/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	ring "github.com/noot/ring-go"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/crypto"
	"github.com/pokt-network/poktroll/pkg/polylog"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	"github.com/pokt-network/poktroll/x/service/types"
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
)

var _ crypto.RingClient = (*ringClient)(nil)

// ringClient is an implementation of the RingClient interface that uses the
// client.ApplicationQueryClient to get application's delegation information
// needed to construct the ring for signing relay requests.
type ringClient struct {
	logger polylog.Logger

	// applicationQuerier is the querier for the application module, and is
	// used to get the gateways an application is delegated to.
	applicationQuerier client.ApplicationQueryClient

	// accountQuerier is used to fetch accounts for a given an account address.
	accountQuerier client.AccountQueryClient
}

// NewRingClient returns a new ring client constructed from the given dependencies.
// It returns an error if the required dependencies are not supplied.
//
// Required dependencies:
// - polylog.Logger
// - client.ApplicationQueryClient
// - client.AccountQueryClient
func NewRingClient(deps depinject.Config) (_ crypto.RingClient, err error) {
	rc := new(ringClient)

	if err := depinject.Inject(
		deps,
		&rc.logger,
		&rc.accountQuerier,
		&rc.applicationQuerier,
	); err != nil {
		return nil, err
	}

	return rc, nil
}

// GetRingForAddressAtHeight returns the ring for the address and block height provided.
// The height provided is used to determine the appropriate delegated gateways
// to use at that height since signature verification may be performed for
// delegations that are no longer active.
// The height provided will be rounded up to the session end height to ensure
// the ring is constructed from the correct past delegations since they become
// effective at the next session's start height.
// TODO(@red-0ne): Link to the docs once they are available.
// The ring is created by querying for the application's and its delegated
// gateways public keys. These keys are converted to secp256k1 curve points
// before forming the ring.
func (rc *ringClient) GetRingForAddressAtHeight(
	ctx context.Context,
	appAddress string,
	blockHeight int64,
) (*ring.Ring, error) {
	ringPubKeys, err := rc.getRingPubKeysForAddress(ctx, appAddress, blockHeight)
	if err != nil {
		return nil, err
	}

	// Get the points on the secp256k1 curve for the public keys in the ring.
	points, err := pointsFromPublicKeys(ringPubKeys...)
	if err != nil {
		return nil, err
	}

	// Return the ring the constructed from the points retrieved above.
	return newRingFromPoints(points)
}

// VerifyRelayRequestSignature verifies the signature of the relay request
// provided against the corresponding ring for the application address in
// the same request.
func (rc *ringClient) VerifyRelayRequestSignature(
	ctx context.Context,
	relayRequest *types.RelayRequest,
) error {
	relayRequestMeta := relayRequest.GetMeta()

	sessionHeader := relayRequestMeta.GetSessionHeader()
	if err := sessionHeader.ValidateBasic(); err != nil {
		return ErrRingClientInvalidRelayRequest.Wrapf("invalid session header: %v", err)
	}

	rc.logger.Debug().
		Fields(map[string]any{
			"session_id":          sessionHeader.GetSessionId(),
			"application_address": sessionHeader.GetApplicationAddress(),
			"service_id":          sessionHeader.GetService().GetId(),
		}).
		Msg("verifying relay request signature")

	// Extract the relay request's ring signature.
	signature := relayRequestMeta.GetSignature()
	if signature == nil {
		return ErrRingClientInvalidRelayRequest.Wrap("missing signature from relay request")
	}

	// Deserialize the request signature bytes back into a ring signature.
	relayRequestRingSig := new(ring.RingSig)
	if err := relayRequestRingSig.Deserialize(ring_secp256k1.NewCurve(), signature); err != nil {
		return ErrRingClientInvalidRelayRequestSignature.Wrapf(
			"error deserializing ring signature: %s", err,
		)
	}

	// Get the ring for the application address of the relay request.
	sessionEndHeight := sessionHeader.GetSessionEndBlockHeight()
	appAddress := sessionHeader.GetApplicationAddress()
	expectedAppRing, err := rc.GetRingForAddressAtHeight(ctx, appAddress, sessionEndHeight)
	if err != nil {
		return ErrRingClientInvalidRelayRequest.Wrapf(
			"error getting ring for application address %s: %v", appAddress, err,
		)
	}

	// Compare the expected ring signature against the one provided in the relay request.
	if !relayRequestRingSig.Ring().Equals(expectedAppRing) {
		return ErrRingClientInvalidRelayRequestSignature.Wrapf(
			"ring signature in the relay request does not match the expected one for the app %s", appAddress,
		)
	}

	// Get and hash the signable bytes of the relay request.
	requestSignableBz, err := relayRequest.GetSignableBytesHash()
	if err != nil {
		return ErrRingClientInvalidRelayRequest.Wrapf("error getting relay request signable bytes: %v", err)
	}

	// Verify the relay request's signature.
	if valid := relayRequestRingSig.Verify(requestSignableBz); !valid {
		return ErrRingClientInvalidRelayRequestSignature.Wrapf("invalid relay request signature or bytes")
	}

	return nil
}

// getRingPubKeysForAddress returns the public keys corresponding to a ring
// It is a slice consisting of the application's public key and the public keys
// of the gateways an application delegated the ability to sign relay requests
// on its behalf at the given block height.
func (rc *ringClient) getRingPubKeysForAddress(
	ctx context.Context,
	appAddress string,
	blockHeight int64,
) ([]cryptotypes.PubKey, error) {
	// Get the application's on chain state.
	app, err := rc.applicationQuerier.GetApplication(ctx, appAddress)
	if err != nil {
		return nil, err
	}

	// Create a slice of addresses for the ring.
	ringAddresses := make([]string, 0)
	ringAddresses = append(ringAddresses, appAddress) // app address is index 0

	// Reconstruct the delegatee gateway addresses at the given block height and
	// add them to the ring addresses.
	delegateeGatewayAddresses := GetRingAddressesAtBlock(&app, blockHeight)
	ringAddresses = append(ringAddresses, delegateeGatewayAddresses...)

	// Sort the ring addresses to ensure the ring is consistent between signing and
	// verification by satisfying relayRequestRingSig.Ring().Equals(expectedAppRing)
	slices.Sort(ringAddresses)

	rc.logger.Debug().
		// TODO_TECHDEBT: implement and use `polylog.Event#Strs([]string)`
		Str("addresses", fmt.Sprintf("%v", ringAddresses)).
		Msg("converting addresses to points")

	return rc.addressesToPubKeys(ctx, ringAddresses)
}

// addressesToPubKeys queries for and returns the public keys for the addresses
// provided.
func (rc *ringClient) addressesToPubKeys(
	ctx context.Context,
	addresses []string,
) ([]cryptotypes.PubKey, error) {
	pubKeys := make([]cryptotypes.PubKey, len(addresses))
	for i, addr := range addresses {
		acc, err := rc.accountQuerier.GetPubKeyFromAddress(ctx, addr)
		if err != nil {
			return nil, err
		}
		pubKeys[i] = acc
	}
	return pubKeys, nil
}

// GetRingAddressesAtBlock returns the active gateway delegations for the given
// application and target block height while accounting for the pending undelegations.
// The ring addresses slice is reconstructed by adding back the past delegated gateways
// that have been undelegated after the target session end height.
func GetRingAddressesAtBlock(app *apptypes.Application, blockHeight int64) []string {
	// Get the target session end height at which we want to get the active delegations.
	targetSessionEndHeight := uint64(sessionkeeper.GetSessionEndBlockHeight(blockHeight))
	// Get the current active delegations for the application and use them as a base.
	activeDelegationsAtHeight := app.DelegateeGatewayAddresses

	// Use a map to keep track of the gateways addresses that have been added to
	// the active delegations slice to avoid duplicates.
	addedDelegations := make(map[string]bool)

	// Iterate over the pending undelegations recorded at their respective block
	// height and check whether to add them back as active delegations.
	for pendingUndelegationHeight, undelegatedGateways := range app.PendingUndelegations {
		// If the pending undelegation happened BEFORE the target session end height,
		// skip it, as it became effective before the target session end height.
		if targetSessionEndHeight > pendingUndelegationHeight {
			continue
		}
		// Add back any gateway address  that was undelegated after the target session
		// end height, as we consider it not happening yet relative to the target height.
		for _, gatewayAddress := range undelegatedGateways.GatewayAddresses {
			if _, ok := addedDelegations[gatewayAddress]; ok {
				continue
			}

			activeDelegationsAtHeight = append(activeDelegationsAtHeight, gatewayAddress)
			// Mark the gateway address as added to avoid duplicates.
			addedDelegations[gatewayAddress] = true
		}

	}

	// add app address twice to make the ring size of minimum 2
	// TODO_IMPROVE: The appAddress is added twice because a ring signature
	// requires AT LEAST two pubKeys. If the Application has not delegated
	// to any gateways, the app's own address needs to be used twice to
	// create a ring. This is not a huge issue but an improvement should
	// be investigated in the future.
	if len(activeDelegationsAtHeight) == 0 {
		activeDelegationsAtHeight = append(activeDelegationsAtHeight, app.Address)
	}

	return activeDelegationsAtHeight
}
