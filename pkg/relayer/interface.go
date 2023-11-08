package relayer

import (
	"context"

	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// RelayerProxy is the interface for the proxy that serves relays to the application.
// It is responsible for starting and stopping all supported RelayServers.
// While handling requests and responding in a closed loop, it also notifies
// the miner about the relays that have been served.
type RelayerProxy interface {

	// Start starts all advertised relay servers and returns an error if any of them fail to start.
	Start(ctx context.Context) error

	// Stop stops all advertised relay servers and returns an error if any of them fail.
	Stop(ctx context.Context) error

	// ServedRelays returns an observable that notifies the miner about the relays that have been served.
	// A served relay is one whose RelayRequest's signature and session have been verified,
	// and its RelayResponse has been signed and successfully sent to the client.
	ServedRelays() observable.Observable[*types.Relay]

	// VerifyRelayRequest is a shared method used by RelayServers to check the
	// relay request signature and session validity.
	// This method may mutate the relay request in the future, so the returned
	// relay request should be used instead of the passed in one.
	// TODO_TECHDEBT(@red-0ne): This method should be moved out of the RelayerProxy interface
	// that should not be responsible for verifying relay requests.
	VerifyRelayRequest(
		ctx context.Context,
		relayRequest *types.RelayRequest,
		serviceId *sharedtypes.ServiceId,
	) (*types.RelayRequest, error)

	// SignRelayResponse is a shared method used by RelayServers to sign
	// and return a signed RelayResponse the relay response.
	// TODO_TECHDEBT(@red-0ne): This method should be moved out of the RelayerProxy interface
	// that should not be responsible for signing relay responses.
	SignRelayResponse(relayResponse *types.RelayResponse) (*types.RelayResponse, error)
}

// RelayServer is the interface of the advertised relay servers provided by the RelayerProxy.
type RelayServer interface {
	// Start starts the service server and returns an error if it fails.
	Start(ctx context.Context) error

	// Stop terminates the service server and returns an error if it fails.
	Stop(ctx context.Context) error

	// ServiceId returns the serviceId of the service.
	ServiceId() *sharedtypes.ServiceId
}
