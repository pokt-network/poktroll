package rings

import (
	"context"
	"fmt"

	"cosmossdk.io/depinject"
	ring_secp256k1 "github.com/athanorlabs/go-dleq/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	ring "github.com/noot/ring-go"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/crypto"
	"github.com/pokt-network/poktroll/pkg/polylog"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	"github.com/pokt-network/poktroll/x/service/types"
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

// GetRingForAddress returns the ring for the address and the block height provided.
// The block height provided is used to determine the appropriate delegated gateways
// to use at that height since signature verification may be performed for
// delegations that are no longer active.
// The ring is created by querying for the application's and its delegated
// gateways public keys. These keys are converted to secp256k1 curve points
// before forming the ring.
func (rc *ringClient) GetRingForAddress(
	ctx context.Context,
	appAddress string,
	blockHeight int64,
) (*ring.Ring, error) {
	pubKeys, err := rc.getDelegatedPubKeysForAddress(ctx, appAddress, blockHeight)
	if err != nil {
		return nil, err
	}

	// Get the points on the secp256k1 curve for the public keys in the ring.
	points, err := pointsFromPublicKeys(pubKeys...)
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

	// Get the session end block height from the relay request's session header to
	// determine the ring to use for signature verification.
	sessionEndHeight := sessionHeader.GetSessionEndBlockHeight()
	appAddress := sessionHeader.GetApplicationAddress()

	// Get the applicable ring for the application address at sessionEndHeight.
	// If the session end height is not reached yet, the active delegations will
	// be used to form the ring.
	expectedAppRing, err := rc.GetRingForAddress(ctx, appAddress, sessionEndHeight)
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

// getDelegatedPubKeysForAddress returns the gateway public keys an application
// delegated the ability to sign relay requests on its behalf.
func (rc *ringClient) getDelegatedPubKeysForAddress(
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
	if len(app.DelegateeGatewayAddresses) == 0 {
		// add app address twice to make the ring size of minimum 2
		// TODO_IMPROVE: The appAddress is added twice because a ring signature
		// requires AT LEAST two pubKeys. If the Application has not delegated
		// to any gateways, the app's own address needs to be used twice to
		// create a ring. This is not a huge issue but an improvement should
		// be investigated in the future.
		ringAddresses = append(ringAddresses, appAddress)
	} else {
		// Chose the appropriate delegated gateway addresses based on the provided
		// block height and add them to the ring addresses.
		delegateeGatewayAddresses := getDelegationsAtBlock(&app, blockHeight)
		ringAddresses = append(ringAddresses, delegateeGatewayAddresses...)
	}

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

// getDelegationsAtBlock returns the appropriate delegations for the given
// application and target block height.
// The delegations selection is determined by the following:
// - The delegations must be archived AFTER the provided block height.
// - Among the delegations that meet the above criteria, the oldest one is chosen.
// - If no suitable archived delegation is found, the active one is used.
//
// The illustration below shows the selection process:
//
// Legend:
// x : Delegations archival
// > : Active delegations
// - : Activity time for a delegation set prior to its archival "x"
// | : Target block height for which the delegations are requested
// ^ : Selected delegations, archived "x" or active ">"
//
// Case: Picking an archived delegation
// -----x---x-----x---x---->
// -----------|---^
//
// Case: Picking the currently active delegation
// -----x---x-----x---x---->
// ---------------------|--^
func getDelegationsAtBlock(
	app *apptypes.Application,
	blockHeight int64,
) []string {
	// neoProximateArchivedDelegations will hold the appropriate archived delegations
	// for the provided block height or nil if no suitable one is found.
	// It will be the oldest archived delegations among the ones that are past the
	// provided block height.
	var neoProximateArchivedDelegations []string
	// neoProximateArchivalHeight will hold the block height of the appropriate
	// archived delegation or 0 if no suitable one is found.
	neoProximateArchivalHeight := int64(0)

	// The chronological order of the archived delegations may not be guaranteed
	// and iterating through all of them is necessary to find the appropriate one.
	for _, archivedDelegateeSet := range app.ArchivedDelegations {
		// A valid archived delegation is one that is archived AFTER the provided height.
		isArchivedAfterHeight := archivedDelegateeSet.LastActiveBlockHeight >= blockHeight
		// And is the oldest archived set found so far.
		isNeoProximate := archivedDelegateeSet.LastActiveBlockHeight < neoProximateArchivalHeight

		if isArchivedAfterHeight && isNeoProximate {
			neoProximateArchivalHeight = archivedDelegateeSet.LastActiveBlockHeight
			neoProximateArchivedDelegations = archivedDelegateeSet.GatewayAddresses
		}
	}

	// Return the archived delegations if one is found.
	if neoProximateArchivedDelegations != nil {
		return neoProximateArchivedDelegations
	}

	// Use the active one otherwise.
	return app.DelegateeGatewayAddresses
}
