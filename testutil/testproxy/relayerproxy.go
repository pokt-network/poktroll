package testproxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"

	"cosmossdk.io/depinject"
	ring_secp256k1 "github.com/athanorlabs/go-dleq/secp256k1"
	ringtypes "github.com/athanorlabs/go-dleq/types"
	"github.com/cometbft/cometbft/crypto"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	keyringtypes "github.com/cosmos/cosmos-sdk/crypto/keyring"
	secp256k1 "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/types"
	accounttypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/golang/mock/gomock"
	"github.com/noot/ring-go"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/proxy"
	"github.com/pokt-network/poktroll/pkg/signer"
	mockclient "github.com/pokt-network/poktroll/testutil/mockclient"
	mockaccount "github.com/pokt-network/poktroll/testutil/mockrelayer/account"
	mockapp "github.com/pokt-network/poktroll/testutil/mockrelayer/application"
	mockkeyring "github.com/pokt-network/poktroll/testutil/mockrelayer/keyring"
	mocksession "github.com/pokt-network/poktroll/testutil/mockrelayer/session"
	mocksupplier "github.com/pokt-network/poktroll/testutil/mockrelayer/supplier"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
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

	ApplicationPrivateKey *secp256k1.PrivKey
}

var ValidPayload = &servicetypes.JSONRPCRequestPayload{
	Method:  "someMethod",
	Id:      1,
	Jsonrpc: "2.0",
	Params:  []string{"someParam"},
}

func GetAddressFromPrivateKey(test *TestBehavior, privKey *secp256k1.PrivKey) string {
	applicationPublicKey, err := codectypes.NewAnyWithValue(privKey.PubKey())

	require.NoError(test.t, err)
	record := &keyringtypes.Record{Name: "app1", PubKey: applicationPublicKey}

	applicationAddress, err := record.GetAddress()
	require.NoError(test.t, err)
	return applicationAddress.String()
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
		ApplicationPrivateKey: secp256k1.GenPrivKey(),
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
			payload := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"result":"%s"}`, serviceId)
			w.Write([]byte(payload))
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
		Supplier(gomock.Eq(test.ctx), supplierReq).
		AnyTimes().
		Return(&suppliertypes.QueryGetSupplierResponse{Supplier: supplier}, nil)
}

func WithApplicationDefaultBehavior(test *TestBehavior) {
	applicationReq := &apptypes.QueryGetApplicationRequest{
		Address: GetAddressFromPrivateKey(test, test.ApplicationPrivateKey),
	}
	application := apptypes.Application{
		Address: test.supplierAddress.String(),
	}
	test.mocks.appQuerierMock.EXPECT().
		Application(gomock.Any(), applicationReq).
		AnyTimes().
		Return(&apptypes.QueryGetApplicationResponse{Application: application}, nil)

	test.mocks.appQuerierMock.EXPECT().
		Application(gomock.Any(), gomock.Any()).
		AnyTimes().
		Return(nil, fmt.Errorf("key not found"))
}

func WithAccountsDefaultBehavior(test *TestBehavior) {
	accountReq := &accounttypes.QueryAccountRequest{
		Address: GetAddressFromPrivateKey(test, test.ApplicationPrivateKey),
	}
	pubKeyAny, err := codectypes.NewAnyWithValue(test.ApplicationPrivateKey.PubKey())
	require.NoError(test.t, err)
	accountAny, err := codectypes.NewAnyWithValue(&accounttypes.BaseAccount{
		Address: test.supplierAddress.String(),
		PubKey:  pubKeyAny,
	})
	require.NoError(test.t, err)
	test.mocks.accountQuerierMock.EXPECT().
		Account(gomock.Any(), accountReq).
		AnyTimes().
		Return(&accounttypes.QueryAccountResponse{Account: accountAny}, nil)
}

func WithSessionSupplierMismatchBehavior(test *TestBehavior) {
	sessionReq := &sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: GetAddressFromPrivateKey(test, test.ApplicationPrivateKey),
		Service:            &sharedtypes.Service{Id: "service1"},
		BlockHeight:        1,
	}
	session := sessiontypes.Session{
		SessionId: "",
		Suppliers: []*sharedtypes.Supplier{},
	}
	test.mocks.sessionQuerierMock.EXPECT().
		GetSession(gomock.Any(), sessionReq).
		AnyTimes().
		Return(&sessiontypes.QueryGetSessionResponse{Session: &session}, nil)
}

func WithSessionDefaultBehavior(test *TestBehavior) {
	sessionReq := &sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: GetAddressFromPrivateKey(test, test.ApplicationPrivateKey),
		Service:            &sharedtypes.Service{Id: "service1"},
		BlockHeight:        1,
	}
	session := sessiontypes.Session{
		Header: &sessiontypes.SessionHeader{
			Service:                 &sharedtypes.Service{Id: "service1"},
			ApplicationAddress:      GetAddressFromPrivateKey(test, test.ApplicationPrivateKey),
			SessionStartBlockHeight: 1,
		},
		SessionId: "",
		Suppliers: []*sharedtypes.Supplier{
			{
				Address: test.supplierAddress.String(),
			},
		},
	}
	test.mocks.sessionQuerierMock.EXPECT().
		GetSession(gomock.Any(), sessionReq).
		AnyTimes().
		Return(&sessiontypes.QueryGetSessionResponse{Session: &session}, nil)
}

func WithKeyringDefaultBehavior(test *TestBehavior) {
	supplierPrivateKey := secp256k1.GenPrivKey()
	supplierPublicKey, err := codectypes.NewAnyWithValue(supplierPrivateKey.PubKey())

	require.NoError(test.t, err)

	record := &keyringtypes.Record{Name: test.SupplierKeyName, PubKey: supplierPublicKey}

	test.mocks.keyringMock.EXPECT().
		Key(gomock.Eq(test.SupplierKeyName)).
		AnyTimes().
		Return(record, nil)

	test.mocks.keyringMock.EXPECT().
		Key(gomock.Not(gomock.Eq(test.SupplierKeyName))).
		AnyTimes().
		Return(nil, fmt.Errorf("key not found"))

	test.mocks.keyringMock.EXPECT().
		Sign(gomock.Eq(test.SupplierKeyName), gomock.AssignableToTypeOf([]byte{})).
		AnyTimes().
		Return([]byte("signature"), nil, nil)

	address, err := record.GetAddress()
	require.NoError(test.t, err)

	test.supplierAddress = address
}

func WithBlockClientDefaultBehavior(test *TestBehavior) {
	test.mocks.blockClientMock.EXPECT().
		LatestBlock(gomock.Any()).
		AnyTimes().
		Return(newBlock(1))
}

func MarshalAndSend(
	test *TestBehavior,
	request *servicetypes.RelayRequest,
) (errCode int32, errorMessage string) {
	reqBz, err := request.Marshal()
	require.NoError(test.t, err)

	reader := io.NopCloser(bytes.NewReader(reqBz))
	res, err := http.DefaultClient.Post(test.ProvidedServices["service1"].Url, "application/json", reader)
	require.NoError(test.t, err)
	require.NotNil(test.t, res)

	return GetRelayResponseError(test.t, res)
}

func GetRelayResponseError(t *testing.T, res *http.Response) (errCode int32, errMsg string) {
	responseBody, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	relayResponse := &servicetypes.RelayResponse{}
	err = relayResponse.Unmarshal(responseBody)
	require.NoError(t, err)

	payload := relayResponse.Payload.(*servicetypes.RelayResponse_JsonRpcPayload).JsonRpcPayload

	if payload.Error == nil {
		return 0, ""
	}

	return payload.Error.Code, payload.Error.Message
}

func GetApplicationRingSignature(
	t *testing.T,
	req *servicetypes.RelayRequest,
	appPrivateKey *secp256k1.PrivKey,
) []byte {
	publicKey := appPrivateKey.PubKey()
	curve := ring_secp256k1.NewCurve()

	point, err := curve.DecodeToPoint(publicKey.Bytes())
	require.NoError(t, err)

	points := []ringtypes.Point{point, point}
	pointsRing, err := ring.NewFixedKeyRingFromPublicKeys(curve, points)
	require.NoError(t, err)

	scalar, err := curve.DecodeToScalar(appPrivateKey.Bytes())
	require.NoError(t, err)

	signer := signer.NewRingSigner(pointsRing, scalar)

	signableBz, err := req.GetSignableBytes()
	require.NoError(t, err)

	hash := crypto.Sha256(signableBz)
	signature, err := signer.Sign(hash)
	require.NoError(t, err)

	return signature
}

type block struct {
	height int64
}

func (b *block) Height() int64 {
	return b.height
}

func (b *block) Hash() []byte {
	return []byte{}
}

func newBlock(height int64) *block {
	return &block{height: height}
}
