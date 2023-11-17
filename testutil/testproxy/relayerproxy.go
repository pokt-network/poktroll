package testproxy

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"cosmossdk.io/depinject"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	keyringtypes "github.com/cosmos/cosmos-sdk/crypto/keyring"
	secp256k1 "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/types"
	accounttypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/proxy"
	mockclient "github.com/pokt-network/poktroll/testutil/mockclient"
	mockaccount "github.com/pokt-network/poktroll/testutil/mockrelayer/account"
	mockapp "github.com/pokt-network/poktroll/testutil/mockrelayer/application"
	mockkeyring "github.com/pokt-network/poktroll/testutil/mockrelayer/keyring"
	mocksession "github.com/pokt-network/poktroll/testutil/mockrelayer/session"
	mocksupplier "github.com/pokt-network/poktroll/testutil/mockrelayer/supplier"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

type ProvidedServiceConfig struct {
	Url     string
	RpcType sharedtypes.RPCType
}

type relayerProxyMocks struct {
	blockClientMock     *mockclient.MockBlockClient
	accountQuerierMock  *mockaccount.MockQueryClient
	appQuerierMock      *mockapp.MockQueryClient
	sessionQuerierMock  *mocksession.MockQueryClient
	supplierQuerierMock *mocksupplier.MockQueryClient
	keyringMock         *mockkeyring.MockKeyring
}

type relayerProxyDeps struct {
	clientCtx       relayer.QueryClientContext
	blockClient     client.BlockClient
	accountQuerier  accounttypes.QueryClient
	appQuerier      apptypes.QueryClient
	sessionQuerier  sessiontypes.QueryClient
	supplierQuerier suppliertypes.QueryClient
	keyring         keyringtypes.Keyring
}

type TestBehavior struct {
	ctx   context.Context
	t     *testing.T
	ctrl  *gomock.Controller
	Deps  depinject.Config
	mocks *relayerProxyMocks

	proxiedServicesConfig    map[string]string
	ProxiedServicesEndpoints map[string]*url.URL
	proxiedServices          map[string]*http.Server

	ProvidedServices map[string]ProvidedServiceConfig

	SupplierKeyName string
	supplierAddress types.AccAddress
}

type RelayerProxyConfig struct {
	SupplierKeyName       string
	ProxiedServicesConfig map[string]string
	ProvidedServices      map[string]ProvidedServiceConfig
}

func NewRelayerProxyTestBehavior(
	ctx context.Context,
	t *testing.T,
	ctrl *gomock.Controller,
	config RelayerProxyConfig,
	behaviors ...func(*TestBehavior),
) *TestBehavior {
	test := &TestBehavior{
		ctx:                   ctx,
		t:                     t,
		ctrl:                  ctrl,
		SupplierKeyName:       config.SupplierKeyName,
		proxiedServicesConfig: config.ProxiedServicesConfig,
		ProvidedServices:      config.ProvidedServices,
	}

	for _, behavior := range behaviors {
		behavior(test)
	}

	return test
}

func WithRelayerProxyMocks(test *TestBehavior) {
	test.mocks = &relayerProxyMocks{
		blockClientMock:     mockclient.NewMockBlockClient(test.ctrl),
		accountQuerierMock:  mockaccount.NewMockQueryClient(test.ctrl),
		appQuerierMock:      mockapp.NewMockQueryClient(test.ctrl),
		sessionQuerierMock:  mocksession.NewMockQueryClient(test.ctrl),
		supplierQuerierMock: mocksupplier.NewMockQueryClient(test.ctrl),
		keyringMock:         mockkeyring.NewMockKeyring(test.ctrl),
	}
}

func WithRelayerProxyDependencies(test *TestBehavior) {
	proxyDeps := &relayerProxyDeps{
		clientCtx:       relayer.QueryClientContext{},
		blockClient:     test.mocks.blockClientMock,
		accountQuerier:  test.mocks.accountQuerierMock,
		appQuerier:      test.mocks.appQuerierMock,
		sessionQuerier:  test.mocks.sessionQuerierMock,
		supplierQuerier: test.mocks.supplierQuerierMock,
		keyring:         test.mocks.keyringMock,
	}

	deps := depinject.Supply(
		proxyDeps.clientCtx,
		proxyDeps.blockClient,
		proxyDeps.accountQuerier,
		proxyDeps.appQuerier,
		proxyDeps.sessionQuerier,
		proxyDeps.supplierQuerier,
		proxyDeps.keyring,
	)

	test.Deps = deps
}

func WithRelayerProxiedServices(test *TestBehavior) {
	proxiedServicesEndpoints := proxy.ServicesEndpointsMap{}
	for serviceId, endpoint := range test.proxiedServicesConfig {
		endpointUrl, err := url.Parse(endpoint)
		require.NoError(test.t, err)

		proxiedServicesEndpoints[serviceId] = endpointUrl
	}

	test.ProxiedServicesEndpoints = proxiedServicesEndpoints
}

func WithProxiedServiceDefaultBehavior(test *TestBehavior) {
	servers := make(map[string]*http.Server)
	for serviceId, endpoint := range test.ProxiedServicesEndpoints {
		host := endpoint.Host
		srv := &http.Server{Addr: host}
		srv.Handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Write([]byte(serviceId))
		})
		go func() { srv.ListenAndServe() }()
		go func() {
			<-test.ctx.Done()
			srv.Shutdown(test.ctx)
		}()

		servers[serviceId] = srv
	}

	test.proxiedServices = servers
}

func WithUnavailableProxiedService(test *TestBehavior) {
	test.proxiedServices = map[string]*http.Server{}
}

func WithSupplierDefaultBehavior(test *TestBehavior) {
	services := []*sharedtypes.SupplierServiceConfig{}

	for serviceId, providedService := range test.ProvidedServices {
		endpoint := &sharedtypes.SupplierServiceConfig{
			Service: &sharedtypes.Service{
				Id: serviceId,
			},
			Endpoints: []*sharedtypes.SupplierEndpoint{
				{
					Url:     providedService.Url,
					RpcType: providedService.RpcType,
				},
			},
		}

		services = append(services, endpoint)
	}
	supplierReq := &suppliertypes.QueryGetSupplierRequest{Address: test.supplierAddress.String()}
	supplier := sharedtypes.Supplier{Address: test.supplierAddress.String(), Services: services}
	test.mocks.supplierQuerierMock.EXPECT().
		Supplier(gomock.Any(), supplierReq).
		AnyTimes().
		Return(&suppliertypes.QueryGetSupplierResponse{Supplier: supplier}, nil)
}

func WithKeyringDefaultBehavior(test *TestBehavior) {
	sk := secp256k1.GenPrivKey()
	pk, err := codectypes.NewAnyWithValue(sk.PubKey())
	require.NoError(test.t, err)

	record := &keyringtypes.Record{Name: test.SupplierKeyName, PubKey: pk}

	test.mocks.keyringMock.EXPECT().
		Key(gomock.Eq(test.SupplierKeyName)).
		AnyTimes().
		Return(record, nil)

	test.mocks.keyringMock.EXPECT().
		Key(gomock.Not(gomock.Eq(test.SupplierKeyName))).
		AnyTimes().
		Return(nil, fmt.Errorf("key not found"))

	address, err := record.GetAddress()
	require.NoError(test.t, err)

	test.supplierAddress = address
}
