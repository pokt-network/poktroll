package proxy

import (
	"context"

	sdkclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	accounttypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	blocktypes "pocket/pkg/client"
	"pocket/pkg/observable"
	"pocket/pkg/observable/channel"
	"pocket/x/service/types"
	sessiontypes "pocket/x/session/types"
	suppliertypes "pocket/x/supplier/types"
)

var (
	_ RelayerProxy = &relayerProxy{}
)

type relayerProxy struct {
	// keyName is the supplier's key name in the keyring. It is used along with the keyring to
	// get the supplier address and sign the relay responses.
	keyName string
	keyring keyring.Keyring

	// blocksClient is the client used to get the latest block height from the blockchain
	// and be notified about new blocks to update the current session when needed.
	blocksClient blocktypes.BlockClient

	// accountsQuerier is the querier used to get the application public key from the blockchain,
	// which is used to verify the relay request signatures.
	accountsQuerier accounttypes.QueryClient

	// supplierQuerier is the querier used to get the supplier's advertised information from the blockchain,
	// which contains the supported services, the RPC types, and endpoints needed to query the services.
	supplierQuerier suppliertypes.QueryClient

	// sessionQuerier is the querier used to get the current session from the blockchain,
	// which is needed to check if the relay proxy should be serving an incoming relay request.
	sessionQuerier sessiontypes.QueryClient

	// providedServices is a map of the services provided by the relayer proxy. Each provided service
	// has the necessary information to start the server that listens for incoming relay requests and
	// the client that proxies the request to the supported native service.
	providedServices map[string]ProvidedService

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
	blocksClient blocktypes.BlockClient,
) RelayerProxy {
	accountQuerier := accounttypes.NewQueryClient(clientCtx)
	supplierQuerier := suppliertypes.NewQueryClient(clientCtx)
	sessionQuerier := sessiontypes.NewQueryClient(clientCtx)
	providedServices := buildProvidedServices(ctx, supplierQuerier)
	servedRelays, servedRelaysProducer := channel.NewObservable[*types.Relay]()

	return &relayerProxy{
		keyName:              keyName,
		keyring:              keyring,
		blocksClient:         blocksClient,
		accountsQuerier:      accountQuerier,
		supplierQuerier:      supplierQuerier,
		sessionQuerier:       sessionQuerier,
		providedServices:     providedServices,
		servedRelays:         servedRelays,
		servedRelaysProducer: servedRelaysProducer,
	}
}

func (rp *relayerProxy) Start(ctx context.Context) error {
	panic("not implemented")
}

func (rp *relayerProxy) Stop() error {
	panic("not implemented")
}

func (rp *relayerProxy) ServedRelays() observable.Observable[*types.Relay] {
	panic("not implemented")
}

// buildProvidedServices builds the provided services map from the supplier's advertised information.
// It loops over the retrieved `SupplierServiceConfig` and, for each `SupplierEndpoint`, it creates the necessary
// server and client to populate the corresponding `ProvidedService` struct in the map.
func buildProvidedServices(
	ctx context.Context,
	supplierQuerier suppliertypes.QueryClient,
) map[string]ProvidedService {
	panic("not implemented")
}

// TODO_INCOMPLETE: Add the appropriate server and client interfaces to be implemented by each RPC type.
type ProvidedService struct {
	serviceId string
	server    struct{}
	client    struct{}
}
