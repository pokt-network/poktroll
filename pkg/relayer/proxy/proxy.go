package proxy

import (
	"context"
	"net/url"
	"sync"

	sdkerrors "cosmossdk.io/errors"
	ringtypes "github.com/athanorlabs/go-dleq/types"
	"github.com/cometbft/cometbft/crypto"
	sdkclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	accounttypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/noot/ring-go"
	"golang.org/x/sync/errgroup"

	// TODO_INCOMPLETE(@red-0ne): Import the appropriate block client interface once available.
	// blocktypes "github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/signer"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	"github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
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
}

func NewRelayerProxy(
	ctx context.Context,
	clientCtx sdkclient.Context,
	keyName string,
	keyring keyring.Keyring,
	proxiedServicesEndpoints servicesEndpointsMap,
	// TODO_INCOMPLETE(@red-0ne): Uncomment once the BlockClient interface is available.
	// blockClient blocktypes.BlockClient,
) RelayerProxy {
	accountQuerier := accounttypes.NewQueryClient(clientCtx)
	supplierQuerier := suppliertypes.NewQueryClient(clientCtx)
	applicationQuerier := apptypes.NewQueryClient(clientCtx)
	sessionQuerier := sessiontypes.NewQueryClient(clientCtx)
	servedRelays, servedRelaysProducer := channel.NewObservable[*types.Relay]()

	return &relayerProxy{
		// TODO_INCOMPLETE(@red-0ne): Uncomment once the BlockClient interface is available.
		// blockClient:       blockClient,
		keyName:                  keyName,
		keyring:                  keyring,
		accountsQuerier:          accountQuerier,
		supplierQuerier:          supplierQuerier,
		applicationQuerier:       applicationQuerier,
		sessionQuerier:           sessionQuerier,
		proxiedServicesEndpoints: proxiedServicesEndpoints,
		servedRelays:             servedRelays,
		servedRelaysProducer:     servedRelaysProducer,
		ringCacheMutex:           &sync.RWMutex{},
		ringCache:                make(map[string][]ringtypes.Point),
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

// VerifyRelayRequest is a shared method used by RelayServers to check the relay request signature and session validity.
func (rp *relayerProxy) VerifyRelayRequest(ctx context.Context, relayRequest *types.RelayRequest) (isValid bool, err error) {
	// extract the relay request's ring signature
	signature := relayRequest.Meta.Signature
	if signature == nil {
		return false, sdkerrors.Wrapf(ErrInvalidRelayRequest, "missing signature from relay request: %v", relayRequest)
	}
	ringSig := new(ring.RingSig)
	if err := ringSig.Deserialize(signature); err != nil {
		return false, sdkerrors.Wrapf(ErrInvalidRequestSignature, "error deserializing signature: %v", err)
	}

	// get the ring for the application address of the relay request
	appAddress := relayRequest.Meta.SessionHeader.ApplicationAddress
	appRing, err := rp.getRingForAppAddress(ctx, appAddress)
	if err != nil {
		return false, sdkerrors.Wrapf(
			ErrInvalidRelayRequest,
			"error getting ring for application address %s: %v", appAddress, err,
		)
	}

	// verify the ring signature against the ring
	if !ringSig.Ring().Equals(appRing) {
		return false, sdkerrors.Wrapf(
			ErrInvalidRequestSignature,
			"ring signature does not match ring for application address %s", appAddress,
		)
	}

	// get and hash the signable bytes of the relay request
	signableBz, err := relayRequest.GetSignableBytes()
	if err != nil {
		return false, sdkerrors.Wrapf(ErrInvalidRelayRequest, "error getting signable bytes: %v", err)
	}
	hash := crypto.Sha256(signableBz)
	var hash32 [32]byte
	copy(hash32[:], hash)

	// verify the relay request's signature
	return ringSig.Verify(hash32), nil
}

// SignRelayResponse is a shared method used by RelayServers to sign the relay response.
func (rp *relayerProxy) SignRelayResponse(relayResponse *types.RelayResponse) ([]byte, error) {
	// create a simple signer for the request
	signer := signer.NewSimpleSigner(rp.keyring, rp.keyName)

	// extract and hash the relay response's signable bytes
	signableBz, err := relayResponse.GetSignableBytes()
	if err != nil {
		return nil, sdkerrors.Wrapf(ErrInvalidRelayResponse, "error getting signable bytes: %v", err)
	}
	hash := crypto.Sha256(signableBz)
	var hash32 [32]byte
	copy(hash32[:], hash)

	// sign the relay response
	return signer.Sign(hash32)
}
