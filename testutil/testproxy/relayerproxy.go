package testproxy

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

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
	"go.uber.org/mock/gomock"

	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/config"
	"github.com/pokt-network/poktroll/pkg/relayer/relay_authenticator"
	"github.com/pokt-network/poktroll/pkg/signer"
	"github.com/pokt-network/poktroll/testutil/mockrelayer"
	testsession "github.com/pokt-network/poktroll/testutil/session"
	"github.com/pokt-network/poktroll/testutil/testclient/testblock"
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

	// signingKeyNames is the list of key names that are used to sign the relay responses
	signingKeyNames []string

	// Deps is exported so it can be used by the dependency injection framework
	// from the pkg/relayer/proxy/proxy_test.go
	Deps depinject.Config

	// proxyServersMap is a map from ServiceId to the actual Server that handles
	// processing of incoming RPC requests.
	proxyServersMap map[string]*http.Server

	// RelayMeterCallCount is used to track the number of times the relay meter
	// methods are called during the test execution.
	RelayMeterCallCount *relayMeterCallCount
}

// blockHeight is the default block height used in the tests.
const blockHeight = 1

// blockHashBz is the []byte representation of the block hash used in the tests.
var blockHashBz []byte

// testDelays stores artificial delays for specific service IDs for timeout testing
var testDelays = make(map[string]time.Duration)

func init() {
	var err error
	if blockHashBz, err = hex.DecodeString("1B1051B7BF236FEA13EFA65B6BE678514FA5B6EA0AE9A7A4B68D45F95E4F18E0"); err != nil {
		panic(fmt.Errorf("error while trying to decode block hash: %w", err))
	}
}

// SetTestDelay sets an artificial delay for a specific service ID (for timeout testing)
func SetTestDelay(serviceId string, delay time.Duration) {
	testDelays[serviceId] = delay
}

// ClearTestDelays clears all artificial delays
func ClearTestDelays() {
	testDelays = make(map[string]time.Duration)
}

// getTestDelay returns the delay for a service ID, if any
func getTestDelay(serviceId string) (time.Duration, bool) {
	delay, exists := testDelays[serviceId]
	return delay, exists
}

// NewRelayerProxyTestBehavior creates a TestBehavior with the provided set of
// behavior function that are used to instrument the tested subject's dependencies
// and isolate specific execution pathways.
func NewRelayerProxyTestBehavior(
	ctx context.Context,
	t *testing.T,
	signingKeyNames []string,
	behaviors ...func(*TestBehavior),
) *TestBehavior {
	test := &TestBehavior{
		ctx:             ctx,
		t:               t,
		proxyServersMap: make(map[string]*http.Server),
		signingKeyNames: signingKeyNames,
	}

	for _, behavior := range behaviors {
		behavior(test)
	}

	return test
}

// ShutdownServiceID gracefully shuts down the http server for a given service id.
func (t *TestBehavior) ShutdownServiceID(serviceID string) error {
	srv, ok := t.proxyServersMap[serviceID]
	if !ok {
		return fmt.Errorf("shutdown service id: not found")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	return srv.Shutdown(ctx)
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

		ringClientDeps := depinject.Supply(accountQueryClient, applicationQueryClient, sharedQueryClient)
		ringClient := testrings.NewRingClientWithMockDependencies(test.ctx, test.t, ringClientDeps)

		relayAuthenticatorDeps := depinject.Supply(
			logger,
			keyring,
			sessionQueryClient,
			sharedQueryClient,
			blockClient,
			ringClient,
		)

		opts := relay_authenticator.WithSigningKeyNames(test.signingKeyNames)
		relayAuthenticator, err := relay_authenticator.NewRelayAuthenticator(relayAuthenticatorDeps, opts)
		require.NoError(test.t, err)

		test.RelayMeterCallCount = &relayMeterCallCount{}
		relayMeter := newMockRelayMeterWithCallCount(test.t, test.RelayMeterCallCount)

		testDeps := depinject.Configs(
			ringClientDeps,
			depinject.Supply(
				logger,
				ringClient,
				blockClient,
				sessionQueryClient,
				supplierQueryClient,
				keyring,
				relayAuthenticator,
				relayMeter,
			),
		)

		test.Deps = testDeps
	}
}

// WithRelayMeter creates the dependencies mocks for the relayproxy to use a relay meter.
func WithRelayMeter() func(*TestBehavior) {
	return func(test *TestBehavior) {
		relayMeter := newMockRelayMeter(test.t)
		test.Deps = depinject.Configs(test.Deps, depinject.Supply(relayMeter))
	}
}

// WithServicesConfigMap creates the services that the relayer proxy will
// proxy requests to. It creates an HTTP server for each service and starts
// listening on the provided host.
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
				// It is recommended to listen on the main Go routine to ensure
				// that the HTTP servers created for each service are fully initialized
				// and ready to receive requests before executing the test cases.
				listener, err := net.Listen("tcp", supplierConfig.ServiceConfig.BackendUrl.Host)
				require.NoError(test.t, err)

				server := &http.Server{
					Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						// Apply configured test delay for service ID if present.
						if delay, hasDelay := getTestDelay(serviceId); hasDelay {
							time.Sleep(delay)
						}
						sendJSONRPCResponse(test.t, w)
					}),
				}

				go func() {
					err := server.Serve(listener)
					if err != nil && !errors.Is(err, http.ErrServerClosed) {
						require.NoError(test.t, err)
					}
				}()

				go func() {
					<-test.ctx.Done()
					err := server.Shutdown(test.ctx)
					if err != nil {
						require.ErrorIs(test.t, err, context.Canceled)
					} else {
						require.NoError(test.t, err)
					}
				}()

				test.proxyServersMap[serviceId] = server
			}
		}
	}
}

// WithDefaultSupplier creates the default staked supplier for the test
func WithDefaultSupplier(
	supplierOperatorKeyName string,
	supplierEndpoints map[string][]*sharedtypes.SupplierEndpoint,
) func(*TestBehavior) {
	return func(test *TestBehavior) {
		supplierOperatorAddress := getAddressFromKeyName(test, supplierOperatorKeyName)

		for serviceId, endpoints := range supplierEndpoints {
			testqueryclients.AddSuppliersWithServiceEndpoints(
				test.t,
				supplierOperatorAddress,
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
	supplierOperatorKeyName string,
	serviceId string,
	appPrivateKey *secp256k1.PrivKey,
) func(*TestBehavior) {
	return func(test *TestBehavior) {
		if supplierOperatorKeyName == "" {
			return
		}

		appAddress := getAddressFromPrivateKey(test, appPrivateKey)

		sessionSuppliers := []string{}
		supplierOperatorAddress := getAddressFromKeyName(test, supplierOperatorKeyName)
		sessionSuppliers = append(sessionSuppliers, supplierOperatorAddress)

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
	supplierOperatorKeyName string,
	serviceId string,
	appPrivateKey *secp256k1.PrivKey,
	sessionsCount int,
) func(*TestBehavior) {
	return func(test *TestBehavior) {
		appAddress := getAddressFromPrivateKey(test, appPrivateKey)

		sessionSuppliers := []string{}
		supplierOperatorAddress := getAddressFromKeyName(test, supplierOperatorKeyName)
		sessionSuppliers = append(sessionSuppliers, supplierOperatorAddress)

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

func newMockRelayMeter(t *testing.T) relayer.RelayMeter {
	ctrl := gomock.NewController(t)

	relayMeter := mockrelayer.NewMockRelayMeter(ctrl)
	relayMeter.EXPECT().Start(gomock.Any()).Return(nil).AnyTimes()

	return relayMeter
}

// relayMeterCallCount tracks the number of calls made to each method of the RelayMeter interface.
type relayMeterCallCount struct {
	// AccumulateRelayReward counts the number of times AccumulateRelayReward method was called.
	// This tracks how many relays were initially processed and optimistically accounted.
	AccumulateRelayReward int

	// SetNonApplicableRelayReward counts the number of times SetNonApplicableRelayReward method was called.
	// This tracks how many relays were later determined to be non-applicable for rewards.
	SetNonApplicableRelayReward int
}

// newMockRelayMeterWithCallCount creates a mock RelayMeter implementation that
// tracks call counts for its methods.
// It returns a RelayMeter that increments the corresponding counter in the provided
// callCount struct whenever one of its methods is called.
func newMockRelayMeterWithCallCount(
	t *testing.T,
	callCount *relayMeterCallCount,
) relayer.RelayMeter {
	ctrl := gomock.NewController(t)

	relayMeter := mockrelayer.NewMockRelayMeter(ctrl)

	relayMeter.EXPECT().Start(gomock.Any()).Return(nil).AnyTimes()

	relayMeter.EXPECT().IsOverServicing(gomock.Any(), gomock.Any()).
		Do(func(ctx context.Context, meta servicetypes.RelayRequestMetadata) {
			callCount.AccumulateRelayReward++
		}).AnyTimes()

	relayMeter.EXPECT().SetNonApplicableRelayReward(gomock.Any(), gomock.Any()).
		Do(func(ctx context.Context, meta servicetypes.RelayRequestMetadata) {
			callCount.SetNonApplicableRelayReward++
		}).AnyTimes()

	return relayMeter
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

	reqBz, err := request.Marshal()
	require.NoError(test.t, err)
	reader := io.NopCloser(bytes.NewReader(reqBz))
	req := &http.Request{
		Method: http.MethodPost,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		URL:  &url.URL{Scheme: scheme, Host: servicesConfigMap[serviceEndpoint].ListenAddress},
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
	appPrivateKey cryptotypes.PrivKey,
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
	supplierOperatorKeyName string,
	payload []byte,
) *servicetypes.RelayRequest {
	appAddress := getAddressFromPrivateKey(test, privKey)
	sessionId, _ := testsession.GetSessionIdWithDefaultParams(appAddress, serviceId, blockHashBz, blockHeight)
	supplierOperatorAddress := getAddressFromKeyName(test, supplierOperatorKeyName)

	return &servicetypes.RelayRequest{
		Meta: servicetypes.RelayRequestMetadata{
			SessionHeader: &sessiontypes.SessionHeader{
				ApplicationAddress:      appAddress,
				SessionId:               string(sessionId[:]),
				ServiceId:               serviceId,
				SessionStartBlockHeight: testsession.GetSessionStartHeightWithDefaultParams(blockHeight),
				SessionEndBlockHeight:   testsession.GetSessionEndHeightWithDefaultParams(blockHeight),
			},
			SupplierOperatorAddress: supplierOperatorAddress,
			// The returned relay is unsigned and must be signed elsewhere for functionality
			Signature: []byte(""),
		},
		Payload: payload,
	}
}
