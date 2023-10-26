package proxy

import (
	"context"
	"net/url"

	sdkclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	accounttypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"golang.org/x/sync/errgroup"

	blocktypes "pocket/pkg/client"
	"pocket/pkg/observable"
	"pocket/pkg/observable/channel"
	"pocket/x/service/types"
	sessiontypes "pocket/x/session/types"
	suppliertypes "pocket/x/supplier/types"
)

var _ RelayerProxy = (*relayerProxy)(nil)

type (
	serviceId            = string
	relayServersMap      = map[serviceId][]RelayServer
	servicesEndpointsMap = map[serviceId]url.URL
)

type relayerProxy struct {
	// keyName is the supplier's key name in the Cosmos's keybase. It is used along with the keyring to
	// get the supplier address and sign the relay responses.
	keyName string
	keyring keyring.Keyring

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

	// clientCtx is the Cosmos' client context used to build the needed query clients and unmarshal their replies.
	clientCtx sdkclient.Context

	// supplierAddress is the address of the supplier that the relayer proxy is running for.
	supplierAddress string
}

func NewRelayerProxy(
	ctx context.Context,
	clientCtx sdkclient.Context,
	keyName string,
	keyring keyring.Keyring,
	proxiedServicesEndpoints servicesEndpointsMap,
	blockClient blocktypes.BlockClient,
) RelayerProxy {
	accountQuerier := accounttypes.NewQueryClient(clientCtx)
	supplierQuerier := suppliertypes.NewQueryClient(clientCtx)
	sessionQuerier := sessiontypes.NewQueryClient(clientCtx)
	servedRelays, servedRelaysProducer := channel.NewObservable[*types.Relay]()

	return &relayerProxy{
		blockClient:              blockClient,
		keyName:                  keyName,
		keyring:                  keyring,
		accountsQuerier:          accountQuerier,
		supplierQuerier:          supplierQuerier,
		sessionQuerier:           sessionQuerier,
		proxiedServicesEndpoints: proxiedServicesEndpoints,
		servedRelays:             servedRelays,
		servedRelaysProducer:     servedRelaysProducer,
		clientCtx:                clientCtx,
	}
}

// Start concurrently starts all advertised relay servers and returns an error if any of them fails to start.
// This method is blocking until all RelayServers are started.
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
