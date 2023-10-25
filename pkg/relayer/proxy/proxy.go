package proxy

import (
	"context"

	sdkclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	accounttypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"golang.org/x/sync/errgroup"

	// TODO_INCOMPLETE(@red-0ne): Import the appropriate block client interface once available.
	// blocktypes "pocket/pkg/client"
	"pocket/pkg/observable"
	"pocket/pkg/observable/channel"
	"pocket/x/service/types"
	sessiontypes "pocket/x/session/types"
	suppliertypes "pocket/x/supplier/types"
)

var _ RelayerProxy = (*relayerProxy)(nil)

type (
	serviceId       = string
	RelayServersMap = map[serviceId][]RelayServer
)

type relayerProxy struct {
	// keyName is the supplier's key name in the Cosmos's keybase. It is used along with the keyring to
	// get the supplier address and sign the relay responses.
	keyName string
	keyring keyring.Keyring

	// blocksClient is the client used to get the block at the latest height from the blockchain
	// and be notified of new incoming blocks. It is used to update the current session data.
	// TODO_INCOMPLETE(@red-0ne): Uncomment once the BlockClient interface is available.
	// blockClient blocktypes.BlockClient

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
	// the client that relays the request to the supported native service.
	advertisedRelayServers RelayServersMap


	// servedRelays is an observable that notifies the miner about the relays that have been served.
	servedRelays observable.Observable[*types.Relay]

	// servedRelaysProducer is a channel that emits the relays that have been served so that the
	// servedRelays observable can fan out the notifications to its subscribers.
	servedRelaysProducer chan<- *types.Relay
}

func NewRelayerProxy(
	ctx context.Context,
	clientCtx sdkclient.Context,
	keyName string,
	keyring keyring.Keyring,

	// TODO_INCOMPLETE(@red-0ne): Uncomment once the BlockClient interface is available.
	// blockClient blocktypes.BlockClient,
) RelayerProxy {
	accountQuerier := accounttypes.NewQueryClient(clientCtx)
	supplierQuerier := suppliertypes.NewQueryClient(clientCtx)
	sessionQuerier := sessiontypes.NewQueryClient(clientCtx)
	servedRelays, servedRelaysProducer := channel.NewObservable[*types.Relay]()

	return &relayerProxy{
		// TODO_INCOMPLETE(@red-0ne): Uncomment once the BlockClient interface is available.
		// blockClient:       blockClient,
		keyName:              keyName,
		keyring:              keyring,
		accountsQuerier:      accountQuerier,
		supplierQuerier:      supplierQuerier,
		sessionQuerier:       sessionQuerier,
		servedRelays:         servedRelays,
		servedRelaysProducer: servedRelaysProducer,
	}
}

// Start concurrently starts all advertised relay servers and returns an error if any of them fails to start.
func (rp *relayerProxy) Start(ctx context.Context) error {
	// The provided services map is built from the supplier's on-chain advertised information,
	// which is a runtime parameter that can be changed by the supplier.
	// Build the provided services map at Start instead of NewRelayerProxy to avoid having to
	// return an error from the constructor.
	err := rp.BuildProvidedServices(ctx)
	if err != nil {
		return err
	}

	startGroup, gctx := errgroup.WithContext(ctx)

	for _, relayServer := range rp.advertisedRelayServers {
		for _, svr := range relayServer {
			server := svr // create a new variable scoped to the anonymous function
			startGroup.Go(func() error { return server.Start(gctx) })
		}
	}

	return startGroup.Wait()
}

// Stop concurrently stops all advertised relay servers and returns an error if any of them fails.
func (rp *relayerProxy) Stop(ctx context.Context) error {
	stopGroup, gctx := errgroup.WithContext(ctx)

	for _, providedService := range rp.advertisedRelayServers {
		for _, svr := range providedService {
			server := svr // create a new variable scoped to the anonymous function
			stopGroup.Go(func() error { return server.Stop(gctx) })
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
