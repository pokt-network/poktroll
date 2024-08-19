package proxy

import (
	"context"

	"cosmossdk.io/depinject"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"golang.org/x/sync/errgroup"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/crypto"
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

	// signingKeyNames are the supplier operator key names in the Cosmos's keybase.
	// They are used along with the keyring to get the supplier operator addresses
	// and sign relay responses.
	// A unique list of operator key names from all suppliers configured on RelayMiner
	// is passed to relayerProxy, and the address for each signing key is looked up
	// in `BuildProvidedServices`.
	signingKeyNames []string
	keyring         keyring.Keyring

	// blockClient is the client used to get the block at the latest height from the blockchain
	// and be notified of new incoming blocks. It is used to update the current session data.
	blockClient client.BlockClient

	// supplierQuerier is the querier used to get the supplier's advertised information from the blockchain,
	// which contains the supported services, RPC types, and endpoints, etc...
	supplierQuerier client.SupplierQueryClient

	// sessionQuerier is the query client used to get the current session & session params
	// from the blockchain, which are needed to check if the relay proxy should be serving an
	// incoming relay request.
	sessionQuerier client.SessionQueryClient

	// sharedQuerier is the query client used to get the current shared & shared params
	// from the blockchain, which are needed to check if the relay proxy should be serving an
	// incoming relay request.
	sharedQuerier client.SharedQueryClient

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

	// ringCache is used to obtain and store the ring for the application.
	ringCache crypto.RingCache

	// OperatorAddressToSigningKeyNameMap is a map with a CosmoSDK address as a key,
	// and the keyring signing key name as a value.
	// We use this map in:
	// 1. Relay verification to check if the incoming relay matches the supplier hosted by the relay miner;
	// 2. Relay signing to resolve which keyring key name to use for signing;
	OperatorAddressToSigningKeyNameMap map[string]string
}

// NewRelayerProxy creates a new relayer proxy with the given dependencies or returns
// an error if the dependencies fail to resolve or the options are invalid.
//
// Required dependencies:
//   - cosmosclient.Context
//   - client.BlockClient
//   - client.SessionQueryClient
//   - client.SharedQueryClient
//   - client.SupplierQueryClient
//
// Available options:
//   - WithSigningKeyNames
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
		&rp.ringCache,
		&rp.supplierQuerier,
		&rp.sessionQuerier,
		&rp.sharedQuerier,
		&rp.keyring,
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
	// The provided services map is built from the supplier's on-chain advertised information,
	// which is a runtime parameter that can be changed by the supplier.
	// NOTE: We build the provided services map at Start instead of NewRelayerProxy to avoid having to
	// return an error from the constructor.
	if err := rp.BuildProvidedServices(ctx); err != nil {
		return err
	}
	// Start the ring cache.
	rp.ringCache.Start(ctx)

	startGroup, ctx := errgroup.WithContext(ctx)

	for _, relayServer := range rp.servers {
		server := relayServer // create a new variable scoped to the anonymous function

		if err := server.Ping(ctx); err != nil {
			return err
		}

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
	if rp.signingKeyNames == nil || len(rp.signingKeyNames) == 0 || rp.signingKeyNames[0] == "" {
		return ErrRelayerProxyUndefinedSigningKeyNames
	}

	if rp.serverConfigs == nil || len(rp.serverConfigs) == 0 {
		return ErrRelayerServicesConfigsUndefined
	}

	return nil
}

// Ping tests the connectivity between all the managed relay servers and their respective backend URLs.
func (rp *relayerProxy) Ping(ctx context.Context) []error {
	var errs []error

	var i int
	for _, srv := range rp.servers {
		if err := srv.Ping(ctx); err != nil {
			rp.logger.Error().Err(err).
				Msg("an unexpected error occured while pinging backend URL")
			errs = append(errs, err)
		}

		i++
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}
