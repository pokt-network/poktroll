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

// relayerProxy is the main relayer proxy that takes relay requests of supported services from the client
// and proxies them to the supported proxied services.
// It is responsible for notifying the miner about the relays that have been served so they can be counted
// when the miner enters the claim/proof phase.
// TODO_TEST: Have tests for the relayer proxy.
type relayerProxy struct {
	logger polylog.Logger

	// signingKeyName is the supplier's key name in the Cosmos's keybase. It is used along with the keyring to
	// get the supplier address and sign the relay responses.
	signingKeyName string
	keyring        keyring.Keyring

	// blockClient is the client used to get the block at the latest height from the blockchain
	// and be notified of new incoming blocks. It is used to update the current session data.
	blockClient client.BlockClient

	// supplierQuerier is the querier used to get the supplier's advertised information from the blockchain,
	// which contains the supported services, RPC types, and endpoints, etc...
	supplierQuerier client.SupplierQueryClient

	// sessionQuerier is the querier used to get the current session from the blockchain,
	// which is needed to check if the relay proxy should be serving an incoming relay request.
	sessionQuerier client.SessionQueryClient

	// proxyServers is a map of proxyName -> RelayServer provided by the relayer proxy,
	// where proxyName is the name of the proxy defined in the config file and
	// RelayServer is the server that listens for incoming relay requests.
	proxyServers map[string]relayer.RelayServer

	// proxyConfigs is a map of proxyName -> RelayMinerProxyConfig where proxyName
	// is the name of the proxy defined in the config file and RelayMinerProxyConfig
	// is the configuration of the proxy.
	proxyConfigs map[string]*config.RelayMinerProxyConfig

	// servedRelays is an observable that notifies the miner about the relays that have been served.
	servedRelays relayer.RelaysObservable

	// servedRelaysPublishCh is a channel that emits the relays that have been served so that the
	// servedRelays observable can fan out the notifications to its subscribers.
	servedRelaysPublishCh chan<- *types.Relay

	// ringCache is used to obtain and store the ring for the application.
	ringCache crypto.RingCache

	// supplierAddress is the address of the supplier that the relayer proxy is running for.
	supplierAddress string
}

// NewRelayerProxy creates a new relayer proxy with the given dependencies or returns
// an error if the dependencies fail to resolve or the options are invalid.
//
// Required dependencies:
//   - cosmosclient.Context
//   - client.BlockClient
//
// Available options:
//   - WithSigningKeyName
//   - WithProxiedServicesEndpoints
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
// This method IS BLOCKING until all RelayServers are stopped.
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

	for _, relayServer := range rp.proxyServers {
		server := relayServer // create a new variable scoped to the anonymous function
		startGroup.Go(func() error { return server.Start(ctx) })
	}

	return startGroup.Wait()
}

// Stop concurrently stops all advertised relay servers and returns an error if any of them fails.
// This method is blocking until all RelayServers are stopped.
func (rp *relayerProxy) Stop(ctx context.Context) error {
	stopGroup, ctx := errgroup.WithContext(ctx)

	for _, relayServer := range rp.proxyServers {
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
	if rp.signingKeyName == "" {
		return ErrRelayerProxyUndefinedSigningKeyName
	}

	if rp.proxyConfigs == nil || len(rp.proxyConfigs) == 0 {
		return ErrRelayerProxyUndefinedProxiedServicesEndpoints
	}

	return nil
}
