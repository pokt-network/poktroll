package proxy

import (
	"context"

	"cosmossdk.io/depinject"
	"golang.org/x/sync/errgroup"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/config"
	"github.com/pokt-network/poktroll/x/service/types"
)

var _ relayer.RelayerProxy = (*relayerProxy)(nil)

// relayerProxy is the main relayer proxy that takes relay requests of supported
// services from the client and proxies them to the supported backend services.
// It is responsible for notifying the miner about the relays that have been
// served so they can be counted when the miner enters the claim/proof phase.
type relayerProxy struct {
	logger polylog.Logger

	// blockClient is the client used to get the block at the latest height from the blockchain
	// and be notified of new incoming blocks. It is used to update the current session data.
	blockClient client.BlockClient

	// supplierQuerier is the querier used to get the supplier's advertised information from the blockchain,
	// which contains the supported services, RPC types, and endpoints, etc...
	supplierQuerier client.SupplierQueryClient

	// sharedQuerier is the query client used to get the current shared & shared params
	// from the blockchain, which are needed to check if the relay proxy should be serving an
	// incoming relay request.
	sharedQuerier client.SharedQueryClient

	// relayMeter keeps track of the total amount of stake an onchhain Application
	// will owe an onchain Supplier (backed by this RelayMiner) once the session settles.
	// It also configures application over-servicing allowance.
	relayMeter relayer.RelayMeter

	// relayAuthenticator is responsible for authenticating relay requests and responses.
	// It verifies the relay request signature and session validity, and signs relay responses.
	relayAuthenticator relayer.RelayAuthenticator

	// servers is a map of listenAddress -> RelayServer provided by the relayer proxy,
	// where listenAddress is the address of the server defined in the config file and
	// RelayServer is the server that listens for incoming relay requests.
	servers map[string]relayer.RelayServer

	// serverConfigs is a map of listenAddress -> RelayMinerServerConfig where listenAddress
	// is the address of the server defined in the config file and RelayMinerServerConfig
	// is its configuration.
	serverConfigs map[string]*config.RelayMinerServerConfig

	// servedRelays is an observable that notifies the miner about the relays that have been served.
	servedRelays relayer.RelaysObservable

	// servedRelaysPublishCh is a channel that emits the relays that have been served so that the
	// servedRelays observable can fan out the notifications to its subscribers.
	servedRelaysPublishCh chan<- *types.Relay
}

// NewRelayerProxy creates a new relayer proxy with the given dependencies or returns
// an error if the dependencies fail to resolve or the options are invalid.
//
// Required dependencies:
//   - polylog.Logger
//   - client.BlockClient
//   - client.SupplierQueryClient
//   - client.SharedQueryClient
//   - relayer.RelayMeter
//   - relayer.RelayAuthenticator
//
// Available options:
//   - WithServicesConfigMap
func NewRelayerProxy(
	deps depinject.Config,
	opts ...relayer.RelayerProxyOption,
) (relayer.RelayerProxy, error) {
	rp := &relayerProxy{}

	if err := depinject.Inject(
		deps,
		&rp.logger,
		&rp.blockClient,
		&rp.supplierQuerier,
		&rp.sharedQuerier,
		&rp.relayMeter,
		&rp.relayAuthenticator,
	); err != nil {
		return nil, err
	}

	servedRelays, servedRelaysProducer := channel.NewObservable[*types.Relay]()

	rp.servedRelays = servedRelays
	rp.servedRelaysPublishCh = servedRelaysProducer

	for _, opt := range opts {
		opt(rp)
	}

	if err := rp.validateConfig(); err != nil {
		return nil, err
	}

	return rp, nil
}

// Start concurrently starts all advertised relay services and returns an error
// if any of them errors.
// NB: This method IS BLOCKING until all RelayServers are stopped.
func (rp *relayerProxy) Start(ctx context.Context) error {
	// The provided services map is built from the supplier's onchain advertised information,
	// which is a runtime parameter that can be changed by the supplier.
	// NOTE: We build the provided services map at Start instead of NewRelayerProxy to avoid having to
	// return an error from the constructor.
	if err := rp.BuildProvidedServices(ctx); err != nil {
		return err
	}

	// Start the relay authenticator.
	rp.relayAuthenticator.Start(ctx)

	// Start the relay meter by subscribing to the onchain events.
	// This function is non-blocking and the subscription cancellation is handled
	// by the context passed to the Start method.
	if err := rp.relayMeter.Start(ctx); err != nil {
		return err
	}

	startGroup, ctx := errgroup.WithContext(ctx)

	for _, relayServer := range rp.servers {
		server := relayServer // create a new variable scoped to the anonymous function
		startGroup.Go(func() error { return server.Start(ctx) })
	}

	return startGroup.Wait()
}

// Stop concurrently stops all advertised relay servers and returns an error if any of them fails.
// This method is blocking until all RelayServers are stopped.
func (rp *relayerProxy) Stop(ctx context.Context) error {
	stopGroup, ctx := errgroup.WithContext(ctx)

	for _, relayServer := range rp.servers {
		// Create a new object (i.e. deep copy) variable scoped to the anonymous function below
		server := relayServer
		stopGroup.Go(func() error { return server.Stop(ctx) })
	}

	return stopGroup.Wait()
}

// ServedRelays returns an observable that notifies the miner about the relays that have been served.
// A served relay is one whose RelayRequest's signature and session have been verified,
// and its RelayResponse has been signed and successfully sent to the client.
func (rp *relayerProxy) ServedRelays() relayer.RelaysObservable {
	return rp.servedRelays
}

// validateConfig validates the relayer proxy's configuration options and returns an error if it is invalid.
// TODO_TEST: Add tests for validating these configurations.
func (rp *relayerProxy) validateConfig() error {
	if len(rp.serverConfigs) == 0 {
		return ErrRelayerServicesConfigsUndefined
	}

	return nil
}
