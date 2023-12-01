package testproxy

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
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
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/noot/ring-go"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/crypto/rings"
	"github.com/pokt-network/poktroll/pkg/signer"
	"github.com/pokt-network/poktroll/testutil/testclient/testblock"
	testkeyring "github.com/pokt-network/poktroll/testutil/testclient/testkeyring"
	"github.com/pokt-network/poktroll/testutil/testclient/testqueryclients"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

type ProvidedServiceConfig struct {
	Url     string
	RpcType sharedtypes.RPCType
}

// TestBehavior is a struct that holds the test context and mocks
// for the relayer proxy tests
type TestBehavior struct {
	ctx  context.Context
	t    *testing.T
	Deps depinject.Config

	proxiedServices map[string]*http.Server
}

// JSONRpcError is the error struct for the JSON RPC response
type JSONRpcError struct {
	Code    int32  `json:"code"`
	Message string `json:"message"`
}

// JSONRpcErrorReply is the error reply struct for the JSON RPC response
type JSONRpcErrorReply struct {
	Id      int32  `json:"id"`
	Jsonrpc string `json:"jsonrpc"`
	Error   *JSONRpcError
}

// NewRelayerProxyTestBehavior creates a TestBehavior with the provided config
func NewRelayerProxyTestBehavior(
	ctx context.Context,
	t *testing.T,
	behaviors ...func(*TestBehavior),
) *TestBehavior {
	test := &TestBehavior{
		ctx:             ctx,
		t:               t,
		proxiedServices: make(map[string]*http.Server),
	}

	for _, behavior := range behaviors {
		behavior(test)
	}

	return test
}

// WithRelayerProxyDependencies creates the dependencies for the relayer proxy
// from the TestBehavior.mocks so they have the right interface and can be
// used by the dependency injection framework.
func WithRelayerProxyDependencies(keyName string) func(*TestBehavior) {
	return func(test *TestBehavior) {
		accountQueryClient := testqueryclients.NewTestAccountQueryClient(test.t)
		applicationQueryClient := testqueryclients.NewTestApplicationQueryClient(test.t)
		sessionQueryClient := testqueryclients.NewTestSessionQueryClient(test.t)
		supplierQueryClient := testqueryclients.NewTestSupplierQueryClient(test.t)

		blockClient := testblock.NewAnyTimeLatestBlockBlockClient(test.t, []byte{}, 1)
		keyring, _ := testkeyring.NewTestKeyringWithKey(test.t, keyName)

		ringDeps := depinject.Supply(accountQueryClient, applicationQueryClient)
		ringCache, err := rings.NewRingCache(ringDeps)
		require.NoError(test.t, err)

		deps := depinject.Configs(ringDeps, depinject.Supply(
			ringCache,
			blockClient,
			sessionQueryClient,
			supplierQueryClient,
			keyring,
		))

		test.Deps = deps
	}
}

// WithRelayerProxiedServices creates the services that the relayer proxy will
// proxy requests to.
func WithRelayerProxiedServices(proxiedServices map[string]*url.URL) func(*TestBehavior) {
	return func(test *TestBehavior) {
		for serviceId, endpoint := range proxiedServices {
			server := &http.Server{Addr: endpoint.Host}
			server.Handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				payload := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"result":"%s"}`, serviceId)
				w.Write([]byte(payload))
			})
			go func() { server.ListenAndServe() }()
			go func() {
				<-test.ctx.Done()
				server.Shutdown(test.ctx)
			}()

			test.proxiedServices[serviceId] = server
		}
	}
}

// WithDefaultApplications creates the default actors (application and supplier)
// for the test. And mock they are staked on-chain.
// If the supplierKeyName is empty, the supplier will not be staked so we can
// test the case where the supplier is not in the application's session's supplier list.
func WithDefaultActors(
	supplierKeyName string,
	appPrivateKey *secp256k1.PrivKey,
) func(*TestBehavior) {
	return func(test *TestBehavior) {
		var keyring keyringtypes.Keyring

		err := depinject.Inject(test.Deps, &keyring)
		require.NoError(test.t, err)

		supplierAccount, err := keyring.Key(supplierKeyName)
		require.NoError(test.t, err)

		supplierAccAddress, err := supplierAccount.GetAddress()
		require.NoError(test.t, err)

		supplierAddress := supplierAccAddress.String()
		supplierEndpoints := []*sharedtypes.SupplierEndpoint{
			{
				Url:     "http://",
				RpcType: sharedtypes.RPCType_JSON_RPC,
			},
		}

		appPubKey := appPrivateKey.PubKey()
		appAddress := GetAddressFromPrivateKey(test, appPrivateKey)
		delegateeAccounts := map[string]cryptotypes.PubKey{}

		testqueryclients.AddAddressToApplicationMap(
			test.t,
			appAddress,
			appPubKey,
			delegateeAccounts,
		)

		testqueryclients.AddSuppliersWithServiceEndpoints(
			test.t,
			supplierAddress,
			"service1",
			supplierEndpoints,
		)
	}
}

func WithDefaultSessionSupplier(
	supplierKeyName string,
	appPrivateKey *secp256k1.PrivKey,
) func(*TestBehavior) {
	return func(test *TestBehavior) {
		appAddress := GetAddressFromPrivateKey(test, appPrivateKey)

		sessionSuppliers := []string{}

		if supplierKeyName != "" {
			var keyring keyringtypes.Keyring
			err := depinject.Inject(test.Deps, &keyring)
			require.NoError(test.t, err)

			supplierAccount, err := keyring.Key(supplierKeyName)
			require.NoError(test.t, err)

			supplierAccAddress, err := supplierAccount.GetAddress()
			require.NoError(test.t, err)

			supplierAddress := supplierAccAddress.String()
			sessionSuppliers = append(sessionSuppliers, supplierAddress)
		}

		testqueryclients.AddToExistingSessions(
			test.t,
			appAddress,
			"service1",
			1,
			sessionSuppliers,
		)
	}
}

// MarshalAndSend marshals the request and sends it to the provided service
func MarshalAndSend(
	test *TestBehavior,
	url string,
	request *servicetypes.RelayRequest,
) (errCode int32, errorMessage string) {
	reqBz, err := request.Marshal()
	require.NoError(test.t, err)

	reader := io.NopCloser(bytes.NewReader(reqBz))
	res, err := http.DefaultClient.Post(url, "application/json", reader)
	require.NoError(test.t, err)
	require.NotNil(test.t, res)

	return GetRelayResponseError(test.t, res)
}

// GetRelayResponseError returns the error code and message from the relay response
// if the response is not an error, it returns 0, ""
func GetRelayResponseError(t *testing.T, res *http.Response) (errCode int32, errMsg string) {
	responseBody, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	relayResponse := &servicetypes.RelayResponse{}
	err = relayResponse.Unmarshal(responseBody)
	if err != nil {
		return 0, "cannot unmarshal response body"
	}

	var payload JSONRpcErrorReply
	err = json.Unmarshal(relayResponse.Payload, &payload)
	if err != nil {
		return 0, "cannot unmarshal response payload"
	}

	if payload.Error == nil {
		return 0, ""
	}

	return payload.Error.Code, payload.Error.Message
}

// GetRelayResponseResult crafts a ring signer for test purposes and uses it to
// sign the relay request
func GetApplicationRingSignature(
	t *testing.T,
	req *servicetypes.RelayRequest,
	appPrivateKey *secp256k1.PrivKey,
) []byte {
	publicKey := appPrivateKey.PubKey()
	curve := ring_secp256k1.NewCurve()

	point, err := curve.DecodeToPoint(publicKey.Bytes())
	require.NoError(t, err)

	// At least two points are required to create a ring signer
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

// GetAddressFromPrivateKey returns the address of the provided private key
func GetAddressFromPrivateKey(test *TestBehavior, privKey *secp256k1.PrivKey) string {
	applicationPublicKey, err := codectypes.NewAnyWithValue(privKey.PubKey())

	require.NoError(test.t, err)
	record := &keyringtypes.Record{Name: "app1", PubKey: applicationPublicKey}

	applicationAddress, err := record.GetAddress()
	require.NoError(test.t, err)
	return applicationAddress.String()
}

// GenerateRelayRequest generates a relay request with the provided parameters
func GenerateRelayRequest(
	test *TestBehavior,
	privKey *secp256k1.PrivKey,
	serviceId string,
	blockHeight int64,
	payload []byte,
) *servicetypes.RelayRequest {
	appAddress := GetAddressFromPrivateKey(test, privKey)
	sessionId := sha256.Sum256([]byte(fmt.Sprintf("%s-%s-%d", appAddress, serviceId, blockHeight)))

	return &servicetypes.RelayRequest{
		Meta: &servicetypes.RelayRequestMetadata{
			SessionHeader: &sessiontypes.SessionHeader{
				ApplicationAddress: appAddress,
				SessionId:          string(sessionId[:]),
				Service:            &sharedtypes.Service{Id: serviceId},
			},
			Signature: []byte(""),
		},
		Payload: payload,
	}
}
