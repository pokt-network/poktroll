package testproxy

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"testing"

	"cosmossdk.io/depinject"
	ring_secp256k1 "github.com/athanorlabs/go-dleq/secp256k1"
	ringtypes "github.com/athanorlabs/go-dleq/types"
	keyringtypes "github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/pokt-network/ring-go"
	sdktypes "github.com/pokt-network/shannon-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/relayer/config"
	"github.com/pokt-network/poktroll/pkg/signer"
	testsession "github.com/pokt-network/poktroll/testutil/session"
	"github.com/pokt-network/poktroll/testutil/testclient/testblock"
	"github.com/pokt-network/poktroll/testutil/testclient/testdelegation"
	"github.com/pokt-network/poktroll/testutil/testclient/testkeyring"
	"github.com/pokt-network/poktroll/testutil/testclient/testqueryclients"
	testrings "github.com/pokt-network/poktroll/testutil/testcrypto/rings"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// JSONRPCInternalErrorCode is the default JSON-RPC error code to be used when
// generating a JSON-RPC error reply.
// JSON-RPC specification uses -32000 to -32099 as implementation-defined server-errors.
// See: https://www.jsonrpc.org/specification#error_object
const JSONRPCInternalErrorCode = -32000

// TestBehavior is a struct that holds the test context and mocks
// for the relayer proxy tests.
// It is used to provide the context needed by the instrumentation functions
// in order to isolate specific execution paths of the subject under test.
type TestBehavior struct {
	ctx context.Context
	t   *testing.T

	// Deps is exported so it can be used by the dependency injection framework
	// from the pkg/relayer/proxy/proxy_test.go
	Deps depinject.Config

	// proxyServersMap is a map from ServiceId to the actual Server that handles
	// processing of incoming RPC requests.
	proxyServersMap map[string]*http.Server
}

// blockHeight is the default block height used in the tests.
const blockHeight = 1

// blockHashBz is the []byte representation of the block hash used in the tests.
var blockHashBz []byte

func init() {
	var err error
	if blockHashBz, err = hex.DecodeString("1B1051B7BF236FEA13EFA65B6BE678514FA5B6EA0AE9A7A4B68D45F95E4F18E0"); err != nil {
		panic(fmt.Errorf("error while trying to decode block hash: %w", err))
	}
}

// NewRelayerProxyTestBehavior creates a TestBehavior with the provided set of
// behavior function that are used to instrument the tested subject's dependencies
// and isolate specific execution pathways.
func NewRelayerProxyTestBehavior(
	ctx context.Context,
	t *testing.T,
	behaviors ...func(*TestBehavior),
) *TestBehavior {
	test := &TestBehavior{
		ctx:             ctx,
		t:               t,
		proxyServersMap: make(map[string]*http.Server),
	}

	for _, behavior := range behaviors {
		behavior(test)
	}

	return test
}

// WithRelayerProxyDependenciesForBlockHeight creates the dependencies for the relayer proxy
// from the TestBehavior.mocks so they have the right interface and can be
// used by the dependency injection framework.
// blockHeight being the block height that will be returned by the block client's
// LastNBlock method
func WithRelayerProxyDependenciesForBlockHeight(
	keyName string,
	blockHeight int64,
) func(*TestBehavior) {
	return func(test *TestBehavior) {
		logger := polylog.Ctx(test.ctx)
		accountQueryClient := testqueryclients.NewTestAccountQueryClient(test.t)
		applicationQueryClient := testqueryclients.NewTestApplicationQueryClient(test.t)
		sessionQueryClient := testqueryclients.NewTestSessionQueryClient(test.t)
		supplierQueryClient := testqueryclients.NewTestSupplierQueryClient(test.t)
		sharedQueryClient := testqueryclients.NewTestSharedQueryClient(test.t)

		blockClient := testblock.NewAnyTimeLastBlockBlockClient(test.t, []byte{}, blockHeight)
		keyring, _ := testkeyring.NewTestKeyringWithKey(test.t, keyName)

		redelegationObs, _ := channel.NewReplayObservable[client.Redelegation](test.ctx, 1)
		delegationClient := testdelegation.NewAnyTimesRedelegationsSequence(test.ctx, test.t, "", redelegationObs)

		ringCacheDeps := depinject.Supply(accountQueryClient, applicationQueryClient, delegationClient, sharedQueryClient)
		ringCache := testrings.NewRingCacheWithMockDependencies(test.ctx, test.t, ringCacheDeps)

		testDeps := depinject.Configs(
			ringCacheDeps,
			depinject.Supply(
				logger,
				ringCache,
				blockClient,
				sessionQueryClient,
				supplierQueryClient,
				keyring,
			),
		)

		test.Deps = testDeps
	}
}

// WithServicesConfigMap creates the services that the relayer proxy will
// proxy requests to.
// It creates an HTTP server for each service and starts listening on the
// provided host.
func WithServicesConfigMap(
	servicesConfigMap map[string]*config.RelayMinerServerConfig,
) func(*TestBehavior) {
	return func(test *TestBehavior) {
		if os.Getenv("INCLUDE_FLAKY_TESTS") != "true" {
			test.t.Skip("Skipping known flaky test: 'TestRelayerProxy'")
		} else {
			test.t.Log(`TODO_FLAKY: Running known flaky test: 'TestRelayerProxy'

Run the following command a few times to verify it passes at least once:

$ go test -v -count=1 -run TestRelayerProxy ./pkg/relayer/...`)
		}
		for _, serviceConfig := range servicesConfigMap {
			for serviceId, supplierConfig := range serviceConfig.SupplierConfigsMap {
				server := &http.Server{Addr: supplierConfig.ServiceConfig.BackendUrl.Host}
				server.Handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					sendJSONRPCResponse(test.t, w)
				})

				go func() {
					err := server.ListenAndServe()
					if err != nil && !errors.Is(err, http.ErrServerClosed) {
						require.NoError(test.t, err)
					}
				}()

				go func() {
					<-test.ctx.Done()
					err := server.Shutdown(test.ctx)
					require.NoError(test.t, err)
				}()

				test.proxyServersMap[serviceId] = server
			}
		}
	}
}

// WithDefaultSupplier creates the default staked supplier for the test
func WithDefaultSupplier(
	supplierKeyName string,
	supplierEndpoints map[string][]*sharedtypes.SupplierEndpoint,
) func(*TestBehavior) {
	return func(test *TestBehavior) {
		supplierAddress := getAddressFromKeyName(test, supplierKeyName)

		for serviceId, endpoints := range supplierEndpoints {
			testqueryclients.AddSuppliersWithServiceEndpoints(
				test.t,
				supplierAddress,
				serviceId,
				endpoints,
			)
		}
	}
}

// WithDefaultApplication creates the default staked application actor for the test
func WithDefaultApplication(appPrivateKey *secp256k1.PrivKey) func(*TestBehavior) {
	return func(test *TestBehavior) {
		appPubKey := appPrivateKey.PubKey()
		appAddress := getAddressFromPrivateKey(test, appPrivateKey)
		delegateeAccounts := map[string]cryptotypes.PubKey{}

		testqueryclients.AddAddressToApplicationMap(
			test.t,
			appAddress,
			appPubKey,
			delegateeAccounts,
		)
	}
}

// WithDefaultSessionSupplier adds the default staked supplier to the
// application's current session
// If the supplierKeyName is empty, the supplier will not be staked so we can
// test the case where the supplier is not in the application's session's supplier list.
func WithDefaultSessionSupplier(
	supplierKeyName string,
	serviceId string,
	appPrivateKey *secp256k1.PrivKey,
) func(*TestBehavior) {
	return func(test *TestBehavior) {
		if supplierKeyName == "" {
			return
		}

		appAddress := getAddressFromPrivateKey(test, appPrivateKey)

		sessionSuppliers := []string{}
		supplierAddress := getAddressFromKeyName(test, supplierKeyName)
		sessionSuppliers = append(sessionSuppliers, supplierAddress)

		testqueryclients.AddToExistingSessions(
			test.t,
			appAddress,
			serviceId,
			blockHeight,
			sessionSuppliers,
		)
	}
}

// WithSuccessiveSessions creates sessions with SessionNumber 0 through SessionCount -1
// and adds all of them to the sessionMap.
// Each session is configured for the same serviceId and application provided.
func WithSuccessiveSessions(
	supplierKeyName string,
	serviceId string,
	appPrivateKey *secp256k1.PrivKey,
	sessionsCount int,
) func(*TestBehavior) {
	return func(test *TestBehavior) {
		appAddress := getAddressFromPrivateKey(test, appPrivateKey)

		sessionSuppliers := []string{}
		supplierAddress := getAddressFromKeyName(test, supplierKeyName)
		sessionSuppliers = append(sessionSuppliers, supplierAddress)

		// Adding `sessionCount` sessions to the sessionsMap to make them available
		// to the MockSessionQueryClient.
		for i := 0; i < sessionsCount; i++ {
			testqueryclients.AddToExistingSessions(
				test.t,
				appAddress,
				serviceId,
				sharedtypes.DefaultNumBlocksPerSession*int64(i),
				sessionSuppliers,
			)
		}
	}
}

// MarshalAndSend marshals the request and sends it to the provided service.
func MarshalAndSend(
	test *TestBehavior,
	servicesConfigMap map[string]*config.RelayMinerServerConfig,
	serviceEndpoint string,
	serviceId string,
	request *servicetypes.RelayRequest,
) (errCode int32, errorMessage string) {
	var scheme string
	switch servicesConfigMap[serviceEndpoint].ServerType {
	case config.RelayMinerServerTypeHTTP:
		scheme = "http"
	default:
		require.FailNow(test.t, "unsupported server type")
	}

	// originHost is the endpoint that the client will retrieve from the on-chain supplier record.
	// The supplier may have multiple endpoints (e.g. for load geo-balancing, host failover, etc.).
	// In the current test setup, we only have one endpoint per supplier, which is why we are accessing `[0]`.
	// In a real-world scenario, the publicly exposed endpoint would reach a load balancer
	// or a reverse proxy that would route the request to the address specified by ListenAddress.
	originHost := servicesConfigMap[serviceEndpoint].SupplierConfigsMap[serviceId].PubliclyExposedEndpoints[0]

	reqBz, err := request.Marshal()
	require.NoError(test.t, err)
	reader := io.NopCloser(bytes.NewReader(reqBz))
	req := &http.Request{
		Method: http.MethodPost,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		URL:  &url.URL{Scheme: scheme, Host: servicesConfigMap[serviceEndpoint].ListenAddress},
		Host: originHost,
		Body: reader,
	}
	res, err := http.DefaultClient.Do(req)
	require.NoError(test.t, err)
	require.NotNil(test.t, res)

	return GetRelayResponseError(test.t, res)
}

// GetRelayResponseError returns the error code and message from the relay response.
// If the response is not an error, it returns `0, ""`.
func GetRelayResponseError(t *testing.T, res *http.Response) (errCode int32, errMsg string) {
	responseBody, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	relayResponse := &servicetypes.RelayResponse{}
	err = relayResponse.Unmarshal(responseBody)
	require.NoError(t, err)

	// If the relayResponse basic validation fails then consider the payload as an error.
	if err = relayResponse.ValidateBasic(); err != nil {
		return JSONRPCInternalErrorCode, string(relayResponse.Payload)
	}

	response, err := sdktypes.DeserializeHTTPResponse(relayResponse.Payload)
	if err != nil {
		return 0, "cannot unmarshal response"
	}

	var payload JSONRPCErrorReply
	err = json.Unmarshal(response.BodyBz, &payload)
	if err != nil {
		return 0, "cannot unmarshal response payload"
	}

	if payload.Error == nil {
		return 0, ""
	}

	return payload.Error.Code, payload.Error.Message
}

// GetApplicationRingSignature crafts a ring signer for test purposes and uses
// it to sign the relay request
func GetApplicationRingSignature(
	t *testing.T,
	req *servicetypes.RelayRequest,
	appPrivateKey *secp256k1.PrivKey,
) []byte {
	publicKey := appPrivateKey.PubKey()
	curve := ring_secp256k1.NewCurve()

	point, err := curve.DecodeToPoint(publicKey.Bytes())
	require.NoError(t, err)

	// At least two points are required to create a ring signer so we are reusing
	// the same key for it
	points := []ringtypes.Point{point, point}
	pointsRing, err := ring.NewFixedKeyRingFromPublicKeys(curve, points)
	require.NoError(t, err)

	scalar, err := curve.DecodeToScalar(appPrivateKey.Bytes())
	require.NoError(t, err)

	signer := signer.NewRingSigner(pointsRing, scalar)

	signableBz, err := req.GetSignableBytesHash()
	require.NoError(t, err)

	signature, err := signer.Sign(signableBz)
	require.NoError(t, err)

	return signature
}

// getAddressFromPrivateKey returns the address of the provided private key
func getAddressFromPrivateKey(test *TestBehavior, privKey *secp256k1.PrivKey) string {
	addressBz := privKey.PubKey().Address()
	address, err := bech32.ConvertAndEncode("pokt", addressBz)
	require.NoError(test.t, err)
	return address
}

// getAddressFromKeyName returns the address of the provided keyring key name
func getAddressFromKeyName(test *TestBehavior, keyName string) string {
	test.t.Helper()

	var keyring keyringtypes.Keyring

	err := depinject.Inject(test.Deps, &keyring)
	require.NoError(test.t, err)

	account, err := keyring.Key(keyName)
	require.NoError(test.t, err)

	accAddress, err := account.GetAddress()
	require.NoError(test.t, err)

	return accAddress.String()
}

// GenerateRelayRequest generates a relay request with the provided parameters
func GenerateRelayRequest(
	test *TestBehavior,
	privKey *secp256k1.PrivKey,
	serviceId string,
	blockHeight int64,
	supplierKeyName string,
	payload []byte,
) *servicetypes.RelayRequest {
	appAddress := getAddressFromPrivateKey(test, privKey)
	sessionId, _ := testsession.GetSessionIdWithDefaultParams(appAddress, serviceId, blockHashBz, blockHeight)
	supplierAddress := getAddressFromKeyName(test, supplierKeyName)

	return &servicetypes.RelayRequest{
		Meta: servicetypes.RelayRequestMetadata{
			SessionHeader: &sessiontypes.SessionHeader{
				ApplicationAddress:      appAddress,
				SessionId:               string(sessionId[:]),
				Service:                 &sharedtypes.Service{Id: serviceId},
				SessionStartBlockHeight: testsession.GetSessionStartHeightWithDefaultParams(blockHeight),
				SessionEndBlockHeight:   testsession.GetSessionEndHeightWithDefaultParams(blockHeight),
			},
			SupplierAddress: supplierAddress,
			// The returned relay is unsigned and must be signed elsewhere for functionality
			Signature: []byte(""),
		},
		Payload: payload,
	}
}
