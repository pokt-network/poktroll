package proxy

import (
	"context"
	"net/url"
	"sync"

	"cosmossdk.io/depinject"
	ringtypes "github.com/athanorlabs/go-dleq/types"
	sdkclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	accounttypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"golang.org/x/sync/errgroup"

	blocktypes "github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/relayer"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	"github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

var _ relayer.RelayerProxy = (*relayerProxy)(nil)

type (
	serviceId            = string
	relayServersMap      = map[serviceId][]relayer.RelayServer
	servicesEndpointsMap = map[serviceId]url.URL
)

// relayerProxy is the main relayer proxy that takes relay requests of supported services from the client
// and proxies them to the supported proxied services.
// It is responsible for notifying the miner about the relays that have been served so they can be counted
// when the miner enters the claim/proof phase.
// TODO_TEST: Have tests for the relayer proxy.
type relayerProxy struct {
	// signingKeyName is the supplier's key name in the Cosmos's keybase. It is used along with the keyring to
	// get the supplier address and sign the relay responses.
	signingKeyName string
	keyring        keyring.Keyring

	// blocksClient is the client used to get the block at the latest height from the blockchain
	// and be notified of new incoming blocks. It is used to update the current session data.
	blockClient blocktypes.BlockClient

	// accountsQuerier is the querier used to get account data (e.g. app publicKey) from the blockchain,
	// which, in the context of the RelayerProxy, is used to verify the relay request signatures.
	accountsQuerier accounttypes.QueryClient

	// supplierQuerier is the querier used to get the supplier's advertised information from the blockchain,
	// which contains the supported services, RPC types, and endpoints, etc...
	supplierQuerier suppliertypes.QueryClient

	// sessionQuerier is the querier used to get the current session from the blockchain,
	// which is needed to check if the relay proxy should be serving an incoming relay request.
	sessionQuerier sessiontypes.QueryClient

	// applicationQuerier is the querier for the application module.
	// It is used to get the ring for a given application address.
	applicationQuerier apptypes.QueryClient

	// advertisedRelayServers is a map of the services provided by the relayer proxy. Each provided service
	// has the necessary information to start the server that listens for incoming relay requests and
	// the client that relays the request to the supported proxied service.
	advertisedRelayServers relayServersMap

	// proxiedServicesEndpoints is a map of the proxied services endpoints that the relayer proxy supports.
	proxiedServicesEndpoints servicesEndpointsMap

	// servedRelays is an observable that notifies the miner about the relays that have been served.
	servedRelays observable.Observable[*types.Relay]

	// servedRelaysProducer is a channel that emits the relays that have been served so that the
	// servedRelays observable can fan out the notifications to its subscribers.
	servedRelaysProducer chan<- *types.Relay

	// ringCache is a cache of the public keys used to create the ring for a given application
	// they are stored in a map of application address to a slice of points on the secp256k1 curve
	// TODO(@h5law): subscribe to on-chain events to update this cache as the ring changes over time
	ringCache      map[string][]ringtypes.Point
	ringCacheMutex *sync.RWMutex

	// clientCtx is the Cosmos' client context used to build the needed query clients and unmarshal their replies.
	clientCtx sdkclient.Context

	// supplierAddress is the address of the supplier that the relayer proxy is running for.
	supplierAddress string
}

// NewRelayerProxy creates a new relayer proxy with the given dependencies or returns
// an error if the dependencies fail to resolve or the options are invalid.
func NewRelayerProxy(
	deps depinject.Config,
	opts ...relayer.RelayerProxyOption,
) (relayer.RelayerProxy, error) {
	rp := &relayerProxy{}

	if err := depinject.Inject(
		deps,
		&rp.clientCtx,
		&rp.blockClient,
	); err != nil {
		return nil, err
	}

	servedRelays, servedRelaysProducer := channel.NewObservable[*types.Relay]()

	rp.servedRelays = servedRelays
	rp.servedRelaysProducer = servedRelaysProducer
	rp.accountsQuerier = accounttypes.NewQueryClient(rp.clientCtx)
	rp.supplierQuerier = suppliertypes.NewQueryClient(rp.clientCtx)
	rp.sessionQuerier = sessiontypes.NewQueryClient(rp.clientCtx)
	rp.keyring = rp.clientCtx.Keyring

	for _, opt := range opts {
		opt(rp)
	}

	if err := rp.validateConfig(); err != nil {
		return nil, err
	}

	return rp, nil
}

// Start concurrently starts all advertised relay servers and returns an error if any of them fails to start.
// This method is blocking as long as all RelayServers are running.
func (rp *relayerProxy) Start(ctx context.Context) error {
	// The provided services map is built from the supplier's on-chain advertised information,
	// which is a runtime parameter that can be changed by the supplier.
	// NOTE: We build the provided services map at Start instead of NewRelayerProxy to avoid having to
	// return an error from the constructor.
	if err := rp.BuildProvidedServices(ctx); err != nil {
		return err
	}

	startGroup, ctx := errgroup.WithContext(ctx)

	for _, relayServer := range rp.advertisedRelayServers {
		for _, svr := range relayServer {
			server := svr // create a new variable scoped to the anonymous function
			startGroup.Go(func() error { return server.Start(ctx) })
		}
	}

	return startGroup.Wait()
}

// Stop concurrently stops all advertised relay servers and returns an error if any of them fails.
// This method is blocking until all RelayServers are stopped.
func (rp *relayerProxy) Stop(ctx context.Context) error {
	stopGroup, ctx := errgroup.WithContext(ctx)

	for _, providedService := range rp.advertisedRelayServers {
		for _, svr := range providedService {
			server := svr // create a new variable scoped to the anonymous function
			stopGroup.Go(func() error { return server.Stop(ctx) })
		}
	}

	return stopGroup.Wait()
}

// ServedRelays returns an observable that notifies the miner about the relays that have been served.
// A served relay is one whose RelayRequest's signature and session have been verified,
// and its RelayResponse has been signed and successfully sent to the client.
func (rp *relayerProxy) ServedRelays() observable.Observable[*types.Relay] {
	return rp.servedRelays
}

// validateConfig validates the relayer proxy's configuration options and returns an error if it is invalid.
// TODO_TEST: Add tests for validating these configurations.
func (rp *relayerProxy) validateConfig() error {
	if rp.signingKeyName == "" {
		return ErrRelayerProxyUndefinedSigningKeyName
	}

	if rp.proxiedServicesEndpoints == nil {
		return ErrRelayerProxyUndefinedProxiedServicesEndpoints
	}

	return nil
}
