package proxy_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/foxcpp/go-mockdns"
	sdktypes "github.com/pokt-network/shannon-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/pkg/relayer/config"
	"github.com/pokt-network/poktroll/pkg/relayer/proxy"
	"github.com/pokt-network/poktroll/testutil/testproxy"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const (
	blockHeight               = 1
	defaultService            = "service1"
	secondaryService          = "service2"
	thirdService              = "service3"
	defaultRelayMinerServer   = "127.0.0.1:8080"
	secondaryRelayMinerServer = "127.0.0.1:8081"
)

var (
	// helpers used for tests that are initialized in init()
	supplierOperatorKeyName string

	// supplierEndpoints is the map of serviceName -> []SupplierEndpoint
	// where serviceName is the name of the service the supplier staked for
	// and SupplierEndpoint is the endpoint of the service advertised onchain
	// by the supplier
	supplierEndpoints map[string][]*sharedtypes.SupplierEndpoint

	// appPrivateKey is the private key of the application that is used to sign
	// relay responses.
	// It is also used in these tests to derive the public key and address of the
	// application.
	appPrivateKey *secp256k1.PrivKey

	// servicesConfigMap is a map from the service endpoint to its respective
	// respective parsed RelayMiner configuration.
	servicesConfigMap map[string]*config.RelayMinerServerConfig

	// defaultRelayerServerBehavior is the list of functions that are used to
	// define the behavior of the RelayerProxy in the tests.
	defaultRelayerProxyBehavior []func(*testproxy.TestBehavior)
)

func init() {
	supplierOperatorKeyName = "supplierKeyName"
	appPrivateKey = secp256k1.GenPrivKey()

	supplierEndpoints = map[string][]*sharedtypes.SupplierEndpoint{
		defaultService: {
			{
				Url: "http://supplier:8545/",
				// TODO_EXTEND: Consider adding support for non JSON RPC services in the future
				RpcType: sharedtypes.RPCType_JSON_RPC,
			},
		},
		secondaryService: {
			{
				Url:     "http://supplier1:8546/",
				RpcType: sharedtypes.RPCType_GRPC,
			},
		},
		thirdService: {
			{
				Url:     "http://supplier2:8547/",
				RpcType: sharedtypes.RPCType_GRPC,
			},
		},
	}

	servicesConfigMap = map[string]*config.RelayMinerServerConfig{
		defaultRelayMinerServer: {
			ServerType:    config.RelayMinerServerTypeHTTP,
			ListenAddress: defaultRelayMinerServer,
			SupplierConfigsMap: map[string]*config.RelayMinerSupplierConfig{
				defaultService: {
					ServiceId:                defaultService,
					ServerType:               config.RelayMinerServerTypeHTTP,
					PubliclyExposedEndpoints: []string{"supplier"},
					ServiceConfig: &config.RelayMinerSupplierServiceConfig{
						BackendUrl: &url.URL{Scheme: "http", Host: "127.0.0.1:8545", Path: "/"},
					},
					SigningKeyNames: []string{supplierOperatorKeyName},
				},
				secondaryService: {
					ServiceId:                secondaryService,
					ServerType:               config.RelayMinerServerTypeHTTP,
					PubliclyExposedEndpoints: []string{"supplier1"},
					ServiceConfig: &config.RelayMinerSupplierServiceConfig{
						BackendUrl: &url.URL{Scheme: "http", Host: "127.0.0.1:8546", Path: "/"},
					},
					SigningKeyNames: []string{supplierOperatorKeyName},
				},
			},
		},
		secondaryRelayMinerServer: {
			ServerType:    config.RelayMinerServerTypeHTTP,
			ListenAddress: secondaryRelayMinerServer,
			SupplierConfigsMap: map[string]*config.RelayMinerSupplierConfig{
				thirdService: {
					ServiceId:                thirdService,
					ServerType:               config.RelayMinerServerTypeHTTP,
					PubliclyExposedEndpoints: []string{"supplier2"},
					ServiceConfig: &config.RelayMinerSupplierServiceConfig{
						BackendUrl: &url.URL{Scheme: "http", Host: "127.0.0.1:8547", Path: "/"},
					},
				},
			},
		},
	}

	defaultRelayerProxyBehavior = []func(*testproxy.TestBehavior){
		testproxy.WithRelayerProxyDependenciesForBlockHeight(supplierOperatorKeyName, blockHeight),
		testproxy.WithServicesConfigMap(servicesConfigMap),
		testproxy.WithDefaultSupplier(supplierOperatorKeyName, supplierEndpoints),
		testproxy.WithDefaultApplication(appPrivateKey),
		testproxy.WithDefaultSessionSupplier(supplierOperatorKeyName, defaultService, appPrivateKey),
	}
}

// RelayerProxy should start and stop without errors
func TestRelayerProxy_StartAndStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	// Setup the RelayerProxy instrumented behavior
	signingKeyNames := []string{supplierOperatorKeyName}
	test := testproxy.NewRelayerProxyTestBehavior(ctx, t, signingKeyNames, defaultRelayerProxyBehavior...)

	// Create a RelayerProxy
	rp, err := proxy.NewRelayerProxy(
		test.Deps,
		proxy.WithServicesConfigMap(servicesConfigMap),
	)
	require.NoError(t, err)

	// Start RelayerProxy
	go rp.Start(ctx)
	// Block so relayerProxy has sufficient time to start
	time.Sleep(100 * time.Millisecond)

	err = rp.PingAll(ctx)
	require.NoError(t, err)

	// Test that RelayerProxy is handling requests (ignoring the actual response content)
	res, err := http.DefaultClient.Get(fmt.Sprintf("http://%s/", servicesConfigMap[defaultRelayMinerServer].ListenAddress))
	require.NoError(t, err)
	require.NotNil(t, res)

	// Test that RelayerProxy is handling requests from the other server
	res, err = http.DefaultClient.Get(fmt.Sprintf("http://%s/", servicesConfigMap[secondaryRelayMinerServer].ListenAddress))
	require.NoError(t, err)
	require.NotNil(t, res)

	// Stop RelayerProxy
	err = rp.Stop(ctx)
	require.NoError(t, err)
}

// RelayerProxy should fail to build if the service configs are not provided
func TestRelayerProxy_EmptyServicesConfigMap(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	signingKeyNames := []string{supplierOperatorKeyName}
	test := testproxy.NewRelayerProxyTestBehavior(ctx, t, signingKeyNames, defaultRelayerProxyBehavior...)

	_, err := proxy.NewRelayerProxy(
		test.Deps,
		proxy.WithServicesConfigMap(make(map[string]*config.RelayMinerServerConfig)),
	)
	require.Error(t, err)
}

// RelayerProxy should fail to start if it cannot spawn a server for the
// services it advertized onchain.
func TestRelayerProxy_UnsupportedRpcType(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	unsupportedSupplierEndpoint := map[string][]*sharedtypes.SupplierEndpoint{
		defaultService: {
			{
				Url: "http://unsupported:8545/jsonrpc",
				// TODO_EXTEND: Consider adding support for non JSON RPC services in the future
				RpcType: sharedtypes.RPCType_JSON_RPC,
			},
		},
	}

	unsupportedRPCTypeBehavior := []func(*testproxy.TestBehavior){
		testproxy.WithRelayerProxyDependenciesForBlockHeight(supplierOperatorKeyName, blockHeight),
		testproxy.WithServicesConfigMap(servicesConfigMap),

		// The supplier is staked onchain but the service it provides is not supported by the proxy
		testproxy.WithDefaultSupplier(supplierOperatorKeyName, unsupportedSupplierEndpoint),
		testproxy.WithDefaultApplication(appPrivateKey),
		testproxy.WithDefaultSessionSupplier(supplierOperatorKeyName, defaultService, appPrivateKey),
	}

	signingKeyNames := []string{supplierOperatorKeyName}
	test := testproxy.NewRelayerProxyTestBehavior(ctx, t, signingKeyNames, unsupportedRPCTypeBehavior...)

	rp, err := proxy.NewRelayerProxy(
		test.Deps,
		proxy.WithServicesConfigMap(servicesConfigMap),
	)
	require.NoError(t, err)

	err = rp.Start(ctx)
	require.Error(t, err)
}

func TestRelayerProxy_UnsupportedTransportType(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	badTransportSupplierEndpoints := map[string][]*sharedtypes.SupplierEndpoint{
		defaultService: {
			{
				Url:     "xttp://supplier:8545/",
				RpcType: sharedtypes.RPCType_JSON_RPC,
			},
		},
	}

	unsupportedTransportProxy := map[string]*config.RelayMinerServerConfig{
		defaultRelayMinerServer: {
			// The proxy is configured with an unsupported transport type
			ServerType:    config.RelayMinerServerType(100),
			ListenAddress: defaultRelayMinerServer,
			SupplierConfigsMap: map[string]*config.RelayMinerSupplierConfig{
				defaultService: {
					ServiceId: defaultService,
					// The proxy is configured with an unsupported transport type
					ServerType:               config.RelayMinerServerType(100),
					PubliclyExposedEndpoints: []string{"supplier"},
					ServiceConfig: &config.RelayMinerSupplierServiceConfig{
						BackendUrl: &url.URL{Scheme: "http", Host: "127.0.0.1:8545", Path: "/"},
					},
				},
			},
		},
	}

	unsupportedTransportTypeBehavior := []func(*testproxy.TestBehavior){
		testproxy.WithRelayerProxyDependenciesForBlockHeight(supplierOperatorKeyName, blockHeight),

		// The proxy is configured with an unsupported transport type for the proxy
		testproxy.WithServicesConfigMap(unsupportedTransportProxy),
		testproxy.WithDefaultSupplier(supplierOperatorKeyName, badTransportSupplierEndpoints),
		testproxy.WithDefaultApplication(appPrivateKey),
		testproxy.WithDefaultSessionSupplier(supplierOperatorKeyName, defaultService, appPrivateKey),
	}

	signingKeyNames := []string{supplierOperatorKeyName}
	test := testproxy.NewRelayerProxyTestBehavior(ctx, t, signingKeyNames, unsupportedTransportTypeBehavior...)

	rp, err := proxy.NewRelayerProxy(
		test.Deps,
		proxy.WithServicesConfigMap(unsupportedTransportProxy),
	)
	require.NoError(t, err)

	err = rp.Start(ctx)
	require.ErrorIs(t, err, proxy.ErrRelayerProxyUnsupportedTransportType)
}

func TestRelayerProxy_NonConfiguredSupplierServices(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	missingServicesProxy := map[string]*config.RelayMinerServerConfig{
		defaultRelayMinerServer: {
			ServerType:    config.RelayMinerServerTypeHTTP,
			ListenAddress: defaultRelayMinerServer,
			SupplierConfigsMap: map[string]*config.RelayMinerSupplierConfig{
				defaultService: {
					ServiceId:                defaultService,
					ServerType:               config.RelayMinerServerTypeHTTP,
					PubliclyExposedEndpoints: []string{"supplier"},
					ServiceConfig: &config.RelayMinerSupplierServiceConfig{
						BackendUrl: &url.URL{Scheme: "http", Host: "127.0.0.1:8545", Path: "/"},
					},
				},
			},
		},
	}

	unsupportedTransportTypeBehavior := []func(*testproxy.TestBehavior){
		testproxy.WithRelayerProxyDependenciesForBlockHeight(supplierOperatorKeyName, blockHeight),

		// The proxy is configured with an unsupported transport type for the proxy
		testproxy.WithServicesConfigMap(missingServicesProxy),
		testproxy.WithDefaultSupplier(supplierOperatorKeyName, supplierEndpoints),
		testproxy.WithDefaultApplication(appPrivateKey),
		testproxy.WithDefaultSessionSupplier(supplierOperatorKeyName, defaultService, appPrivateKey),
	}

	signingKeyNames := []string{supplierOperatorKeyName}
	test := testproxy.NewRelayerProxyTestBehavior(ctx, t, signingKeyNames, unsupportedTransportTypeBehavior...)

	rp, err := proxy.NewRelayerProxy(
		test.Deps,
		proxy.WithServicesConfigMap(missingServicesProxy),
	)
	require.NoError(t, err)

	err = rp.Start(ctx)
	require.ErrorIs(t, err, proxy.ErrRelayerProxyServiceEndpointNotHandled)
}

// Test different RelayRequest scenarios
func TestRelayerProxy_Relays(t *testing.T) {

	sharedParams := sharedtypes.DefaultParams()

	sessionTwoStartHeight := sharedtypes.GetSessionEndHeight(&sharedParams, blockHeight) + 1

	// blockOutsideSessionGracePeriod is the block height that is after the first
	// session's grace period and within the second session's grace period,
	// meaning a relay should be handled as part of the session two AND NOT session one.
	blockOutsideSessionGracePeriod := sharedtypes.GetSessionGracePeriodEndHeight(&sharedParams, sessionTwoStartHeight)

	// blockWithinSessionGracePeriod is the block height that is after the first
	// session but within its session's grace period, meaning a relay should be
	// handled at this block height.
	blockWithinSessionGracePeriod := sharedtypes.GetSessionGracePeriodEndHeight(&sharedParams, blockHeight) - 1

	tests := []struct {
		desc string
		// RelayerProxy instrumented behavior
		relayerProxyBehavior []func(*testproxy.TestBehavior)
		// Input scenario builds a RelayRequest, marshals it and sends it to the RelayerProxy
		inputScenario func(
			t *testing.T,
			test *testproxy.TestBehavior,
		) (errCode int32, errMsg string)

		// The request result should return any error form the http.DefaultClient.Do call.
		// We infer the behavior from the response's code and message prefix
		expectedErrCode int32
		expectedErrMsg  string
	}{
		{
			desc: "Unparsable relay request",

			relayerProxyBehavior: defaultRelayerProxyBehavior,
			inputScenario:        sendRequestWithUnparsableBody,

			// expectedErrCode is because the proxy won't be able to unmarshal the request
			// so it does not know how to format the error response
			expectedErrCode: testproxy.JSONRPCInternalErrorCode,
			expectedErrMsg:  "proto: RelayRequest: wiretype end group for non-group",
		},
		{
			desc: "Missing signature from relay request",

			relayerProxyBehavior: defaultRelayerProxyBehavior,
			inputScenario:        sendRequestWithMissingSignature,

			expectedErrCode: testproxy.JSONRPCInternalErrorCode,
			expectedErrMsg:  "missing application signature",
		},
		{
			desc: "Invalid signature associated with relay request",

			relayerProxyBehavior: defaultRelayerProxyBehavior,
			inputScenario:        sendRequestWithInvalidSignature,

			expectedErrCode: testproxy.JSONRPCInternalErrorCode,
			expectedErrMsg:  "error deserializing ring signature",
		},
		{
			desc: "Missing session header application address associated with relay request",

			relayerProxyBehavior: defaultRelayerProxyBehavior,
			inputScenario:        sendRequestWithMissingSessionHeaderApplicationAddress,

			expectedErrCode: testproxy.JSONRPCInternalErrorCode,
			expectedErrMsg:  "invalid session header: invalid application address",
		},
		{
			desc: "Non staked application address",

			relayerProxyBehavior: defaultRelayerProxyBehavior,
			inputScenario:        sendRequestWithNonStakedApplicationAddress,

			expectedErrCode: testproxy.JSONRPCInternalErrorCode,
			expectedErrMsg:  "error getting ring for application address",
		},
		{
			desc: "Ring signature mismatch",

			relayerProxyBehavior: defaultRelayerProxyBehavior,
			inputScenario:        sendRequestWithRingSignatureMismatch,

			expectedErrCode: testproxy.JSONRPCInternalErrorCode,
			expectedErrMsg:  "ring signature in the relay request does not match the expected one for the app",
		},
		{
			desc: "Session mismatch",

			relayerProxyBehavior: defaultRelayerProxyBehavior,
			inputScenario:        sendRequestWithDifferentSession,

			expectedErrCode: testproxy.JSONRPCInternalErrorCode,
			expectedErrMsg:  "session ID mismatch",
		},
		{
			desc: "Invalid relay supplier",

			relayerProxyBehavior: []func(*testproxy.TestBehavior){
				testproxy.WithRelayerProxyDependenciesForBlockHeight(supplierOperatorKeyName, blockHeight),
				testproxy.WithServicesConfigMap(servicesConfigMap),
				testproxy.WithDefaultSupplier(supplierOperatorKeyName, supplierEndpoints),
				testproxy.WithDefaultApplication(appPrivateKey),
				// Missing session supplier
				testproxy.WithDefaultSessionSupplier("", defaultService, appPrivateKey),
			},
			inputScenario: sendRequestWithInvalidRelaySupplier,

			expectedErrCode: testproxy.JSONRPCInternalErrorCode,
			expectedErrMsg:  "error while trying to retrieve a session",
		},
		{
			desc: "Relay request signature does not match the request payload",

			relayerProxyBehavior: defaultRelayerProxyBehavior,
			inputScenario:        sendRequestWithSignatureForDifferentPayload,

			expectedErrCode: testproxy.JSONRPCInternalErrorCode,
			expectedErrMsg:  "invalid relay request signature or bytes",
		},
		{
			desc:                 "Successful relay",
			relayerProxyBehavior: defaultRelayerProxyBehavior,
			inputScenario:        sendRequestWithSuccessfulReply,

			expectedErrCode: 0,
			expectedErrMsg:  "",
		},
		{
			desc: "Successful late relay with session grace period",
			relayerProxyBehavior: []func(*testproxy.TestBehavior){
				// blockHeight is past the first session but within its session grace period
				testproxy.WithRelayerProxyDependenciesForBlockHeight(
					supplierOperatorKeyName,
					blockWithinSessionGracePeriod,
				),
				testproxy.WithServicesConfigMap(servicesConfigMap),
				testproxy.WithDefaultSupplier(supplierOperatorKeyName, supplierEndpoints),
				testproxy.WithDefaultApplication(appPrivateKey),
				// Add 2 sessions, with the first one being within the withing grace period
				// and the second one being the current session
				testproxy.WithSuccessiveSessions(supplierOperatorKeyName, defaultService, appPrivateKey, 2),
			},
			inputScenario: sendRequestWithCustomSessionHeight(blockHeight),

			expectedErrCode: 0,
			expectedErrMsg:  "", // Relay handled successfully
		},
		{
			desc: "Failed late relay outside session grace period",

			relayerProxyBehavior: []func(*testproxy.TestBehavior){
				// blockHeight is past the first session's grace period
				testproxy.WithRelayerProxyDependenciesForBlockHeight(
					supplierOperatorKeyName,
					// Set the current block height value returned by the block provider
					blockOutsideSessionGracePeriod,
				),
				testproxy.WithServicesConfigMap(servicesConfigMap),
				testproxy.WithDefaultSupplier(supplierOperatorKeyName, supplierEndpoints),
				testproxy.WithDefaultApplication(appPrivateKey),
				// Add 3 sessions, with the first one that is no longer within its
				// session grace period
				testproxy.WithSuccessiveSessions(supplierOperatorKeyName, defaultService, appPrivateKey, 3),
			},
			// Send a request that has a late session past the grace period
			inputScenario: sendRequestWithCustomSessionHeight(blockHeight),

			expectedErrCode: testproxy.JSONRPCInternalErrorCode,
			expectedErrMsg:  "session expired", // Relay rejected by the supplier
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.TODO())
			defer cancel()

			signingKeyNames := []string{supplierOperatorKeyName}
			testBehavior := testproxy.NewRelayerProxyTestBehavior(ctx, t, signingKeyNames, test.relayerProxyBehavior...)

			rp, err := proxy.NewRelayerProxy(
				testBehavior.Deps,
				proxy.WithServicesConfigMap(servicesConfigMap),
			)
			require.NoError(t, err)

			go rp.Start(ctx)
			// Block so relayerProxy has sufficient time to start
			time.Sleep(100 * time.Millisecond)

			errCode, errMsg := test.inputScenario(t, testBehavior)
			require.Equal(t, test.expectedErrCode, errCode)
			require.True(t, strings.HasPrefix(errMsg, test.expectedErrMsg))

			cancel()
		})
	}
}

// RelayProxyPingAllSuite implements the suite to test the relay proxy ping
// application logic.
type RelayProxyPingAllSuite struct {
	suite.Suite
	relayerProxyBehavior []func(*testproxy.TestBehavior)
	servicesConfigMap    map[string]*config.RelayMinerServerConfig
}

// TestRelayProxyPingAllSuite executes the RelayProxyPingAllSuite test suite.
func TestRelayProxyPingAllSuite(t *testing.T) {
	suite.Run(t, new(RelayProxyPingAllSuite))
}

// SetupSuite setups a single default relayminer with one supplier. The
// default relayminer will be reused in every subsequent tests in the
// suite.
func (t *RelayProxyPingAllSuite) SetupSuite() {
	appPrivateKey := secp256k1.GenPrivKey()
	defaultRelayMinerServerAddress := "127.0.0.1:8245"
	supplierEndpoints := map[string][]*sharedtypes.SupplierEndpoint{
		defaultService: {
			{
				Url:     "http://supplier1pingall:8645",
				RpcType: sharedtypes.RPCType_JSON_RPC,
			},
		},
	}

	t.servicesConfigMap = map[string]*config.RelayMinerServerConfig{
		defaultRelayMinerServerAddress: {
			ServerType:    config.RelayMinerServerTypeHTTP,
			ListenAddress: defaultRelayMinerServerAddress,
			SupplierConfigsMap: map[string]*config.RelayMinerSupplierConfig{
				defaultService: {
					ServiceId:  defaultService,
					ServerType: config.RelayMinerServerTypeHTTP,
					ServiceConfig: &config.RelayMinerSupplierServiceConfig{
						BackendUrl: &url.URL{
							Scheme: "http",
							Host:   "127.0.0.1:8645",
							Path:   "/",
						},
					},
					PubliclyExposedEndpoints: []string{
						"supplier1pingall",
					},
				},
			},
		},
	}

	t.relayerProxyBehavior = []func(*testproxy.TestBehavior){
		testproxy.WithRelayerProxyDependenciesForBlockHeight(supplierOperatorKeyName, 1),
		testproxy.WithServicesConfigMap(t.servicesConfigMap),
		testproxy.WithDefaultSupplier(supplierOperatorKeyName, supplierEndpoints),
		testproxy.WithDefaultApplication(appPrivateKey),
		testproxy.WithDefaultSessionSupplier(supplierOperatorKeyName, defaultService, appPrivateKey),
		testproxy.WithRelayMeter(),
	}
}

// TestOKPingAllWithSingleRelayServer reuses the default relayminer with one
// supplier to test the relayproxy.PingAll method.
func (t *RelayProxyPingAllSuite) TestOKPingAllWithSingleRelayServer() {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	testBehavoirs := testproxy.NewRelayerProxyTestBehavior(ctx, t.T(), t.relayerProxyBehavior...)

	rp, err := proxy.NewRelayerProxy(
		testBehavoirs.Deps,
		proxy.WithSigningKeyNames([]string{supplierOperatorKeyName}),
		proxy.WithServicesConfigMap(t.servicesConfigMap),
	)
	require.NoError(t.T(), err)

	go func() {
		err := rp.Start(ctx)
		if http.ErrServerClosed != err {
			require.NoError(t.T(), err)
		}
	}()

	// waiting for relayer proxy to start
	// and perform ping request.
	time.Sleep(100 * time.Millisecond)

	err = rp.PingAll(ctx)
	require.NoError(t.T(), err)

	err = rp.Stop(ctx)
	require.NoError(t.T(), err)
}

// TestOKPingAllWithMultipleRelayServers reuses default relayminer and
// instantiates an additional one to test the connectivity.
func (t *RelayProxyPingAllSuite) TestOKPingAllWithMultipleRelayServers() {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	secondRelayMinerAddr := "127.0.0.1:8246"
	secondServiceName := "secondservice"

	// adding supplier endpoint.
	supplierEndpoints := map[string][]*sharedtypes.SupplierEndpoint{
		secondServiceName: {
			{
				Url:     "http://secondservice:8646",
				RpcType: sharedtypes.RPCType_JSON_RPC,
			},
		},
	}

	servicesConfigMap := map[string]*config.RelayMinerServerConfig{
		secondRelayMinerAddr: {
			ServerType:    config.RelayMinerServerTypeHTTP,
			ListenAddress: secondRelayMinerAddr,
			SupplierConfigsMap: map[string]*config.RelayMinerSupplierConfig{
				secondServiceName: {
					ServiceId:  secondServiceName,
					ServerType: config.RelayMinerServerTypeHTTP,
					ServiceConfig: &config.RelayMinerSupplierServiceConfig{
						BackendUrl: &url.URL{
							Scheme: "http",
							Host:   "127.0.0.1:8646",
							Path:   "/",
						},
					},
					PubliclyExposedEndpoints: []string{
						"secondservice",
					},
				},
			},
		},
	}

	relayProxyBehavior := append(t.relayerProxyBehavior, []func(*testproxy.TestBehavior){
		testproxy.WithDefaultSupplier(supplierOperatorKeyName, supplierEndpoints),
		testproxy.WithServicesConfigMap(servicesConfigMap),
	}...)

	testBehavoirs := testproxy.NewRelayerProxyTestBehavior(ctx, t.T(), relayProxyBehavior...)

	// copying the default relayminer in the test service config
	// map for the relay proxy.
	for k, v := range t.servicesConfigMap {
		servicesConfigMap[k] = v
	}

	rp, err := proxy.NewRelayerProxy(
		testBehavoirs.Deps,
		proxy.WithSigningKeyNames([]string{supplierOperatorKeyName}),
		proxy.WithServicesConfigMap(servicesConfigMap),
	)
	require.NoError(t.T(), err)

	go func() {
		err := rp.Start(ctx)
		if err != http.ErrServerClosed {
			require.NoError(t.T(), err)
		}
	}()

	// waiting for relayer proxy to start
	// and perform ping request.
	time.Sleep(100 * time.Millisecond)

	err = rp.PingAll(ctx)
	require.NoError(t.T(), err)

	err = rp.Stop(ctx)
	require.NoError(t.T(), err)
}

// TestNOKPingAllWithPartialFailureAtStartup test the connectivity for multiple
// relayminer with a partial sucess/failure at startup.
func (t *RelayProxyPingAllSuite) TestNOKPingAllWithPartialFailureAtStartup() {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	failingRelayMinerAddr := "127.0.0.1:8247"
	failingServiceName := "failingservice"

	supplierEndpoints := map[string][]*sharedtypes.SupplierEndpoint{
		failingServiceName: {
			{
				Url:     "http://failingservice:8647",
				RpcType: sharedtypes.RPCType_JSON_RPC,
			},
		},
	}

	servicesConfigMap := map[string]*config.RelayMinerServerConfig{
		failingRelayMinerAddr: &config.RelayMinerServerConfig{
			ListenAddress: failingRelayMinerAddr,
			ServerType:    config.RelayMinerServerTypeHTTP,
			SupplierConfigsMap: map[string]*config.RelayMinerSupplierConfig{
				failingServiceName: &config.RelayMinerSupplierConfig{
					ServerType: config.RelayMinerServerTypeHTTP,
					ServiceId:  failingServiceName,
					ServiceConfig: &config.RelayMinerSupplierServiceConfig{
						BackendUrl: &url.URL{
							Scheme: "http",
							Host:   "127.0.0.1:8647",
							Path:   "/",
						},
					},
					PubliclyExposedEndpoints: []string{
						"failingservice",
					},
				},
			},
		},
	}

	relayProxyBehavior := append(t.relayerProxyBehavior, []func(*testproxy.TestBehavior){
		testproxy.WithDefaultSupplier(supplierOperatorKeyName, supplierEndpoints),
		testproxy.WithServicesConfigMap(servicesConfigMap),
	}...)

	test := testproxy.NewRelayerProxyTestBehavior(ctx, t.T(), relayProxyBehavior...)

	// copying the default relayminer in the test service config
	// map for the relay proxy.
	for k, v := range t.servicesConfigMap {
		servicesConfigMap[k] = v
	}

	rp, err := proxy.NewRelayerProxy(
		test.Deps,
		proxy.WithSigningKeyNames([]string{supplierOperatorKeyName}),
		proxy.WithServicesConfigMap(servicesConfigMap),
	)
	require.NoError(t.T(), err)

	// we are explicitly shutting down a supplier to simulate a
	// failure while testing the connectivity at startup.
	err = test.ShutdownServiceID(failingServiceName)
	require.NoError(t.T(), err)

	// testing connectivity at startup
	err = rp.Start(ctx)
	require.Error(t.T(), err)
	require.True(t.T(), strings.Contains(err.Error(), "connection refused"))

	err = rp.Stop(ctx)
	require.NoError(t.T(), err)
}

// TestNOKPingAllWithPartialFailureAfterStartup test the connectivity for multiple
// relayminer with a partial sucess/failure at runtime.
func (t *RelayProxyPingAllSuite) TestNOKPingAllWithPartialFailureAfterStartup() {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	failingRelayMinerAddr := "127.0.0.1:8248"
	failingServiceName := "faillingservice"

	supplierEndpoints := map[string][]*sharedtypes.SupplierEndpoint{
		failingServiceName: {
			{
				Url:     "http://failingservice:8648",
				RpcType: sharedtypes.RPCType_JSON_RPC,
			},
		},
	}

	servicesConfigMap := map[string]*config.RelayMinerServerConfig{
		failingRelayMinerAddr: &config.RelayMinerServerConfig{
			ListenAddress: failingRelayMinerAddr,
			ServerType:    config.RelayMinerServerTypeHTTP,
			SupplierConfigsMap: map[string]*config.RelayMinerSupplierConfig{
				failingServiceName: &config.RelayMinerSupplierConfig{
					ServerType: config.RelayMinerServerTypeHTTP,
					ServiceId:  failingServiceName,
					ServiceConfig: &config.RelayMinerSupplierServiceConfig{
						BackendUrl: &url.URL{
							Scheme: "http",
							Host:   "127.0.0.1:8648",
							Path:   "/",
						},
					},
					PubliclyExposedEndpoints: []string{
						"failingservice",
					},
				},
			},
		},
	}

	relayProxyBehavior := append(t.relayerProxyBehavior, []func(*testproxy.TestBehavior){
		testproxy.WithDefaultSupplier(supplierOperatorKeyName, supplierEndpoints),
		testproxy.WithServicesConfigMap(servicesConfigMap),
	}...)

	test := testproxy.NewRelayerProxyTestBehavior(ctx, t.T(), relayProxyBehavior...)

	// copying the default relayminer in the test service config
	// map for the relay proxy.
	for k, v := range t.servicesConfigMap {
		servicesConfigMap[k] = v
	}

	rp, err := proxy.NewRelayerProxy(
		test.Deps,
		proxy.WithSigningKeyNames([]string{supplierOperatorKeyName}),
		proxy.WithServicesConfigMap(servicesConfigMap),
	)
	require.NoError(t.T(), err)

	go func() {
		err := rp.Start(ctx)
		if err != http.ErrServerClosed {
			require.NoError(t.T(), err)
		}
	}()

	// waiting for relayer proxy to start
	// and perform ping request.
	time.Sleep(100 * time.Millisecond)

	// we are explicitly shutting down a supplier to simulate an error
	// while testing the connectivity.
	err = test.ShutdownServiceID(failingServiceName)
	require.NoError(t.T(), err)

	err = rp.PingAll(ctx)
	require.Error(t.T(), err)
	require.True(t.T(), strings.Contains(err.Error(), "connection refused"))

	err = rp.Stop(ctx)
	require.NoError(t.T(), err)
}

// TestOKPingAllDifferentEndpoint test the connectivity with different type of
// endpoints (ipv6, domain name).
func (t *RelayProxyPingAllSuite) TestOKPingAllDifferentEndpoint() {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	srv, err := mockdns.NewServer(map[string]mockdns.Zone{
		"exampleservice.org.": {
			A: []string{"127.0.0.1"},
		},
	}, false)
	require.NoError(t.T(), err)
	defer srv.Close()

	srv.PatchNet(net.DefaultResolver)
	defer mockdns.UnpatchNet(net.DefaultResolver)

	relayminerDomainNameAddr := "exampleservice.org:8249"
	domainNameServiceName := "exampleservice.org"

	relayminerIPV6ServiceAddr := "localhost:8250"
	IPV6ServiceName := "ipv6service"

	supplierEndpoints := map[string][]*sharedtypes.SupplierEndpoint{
		domainNameServiceName: {
			{
				Url:     "http://exampleservice.org:8649",
				RpcType: sharedtypes.RPCType_JSON_RPC,
			},
		},
		IPV6ServiceName: {
			{
				Url:     "http://ipv6service:8650",
				RpcType: sharedtypes.RPCType_JSON_RPC,
			},
		},
	}

	servicesConfigMap := map[string]*config.RelayMinerServerConfig{
		relayminerDomainNameAddr: &config.RelayMinerServerConfig{
			ListenAddress: relayminerDomainNameAddr,
			ServerType:    config.RelayMinerServerTypeHTTP,
			SupplierConfigsMap: map[string]*config.RelayMinerSupplierConfig{
				domainNameServiceName: &config.RelayMinerSupplierConfig{
					ServerType: config.RelayMinerServerTypeHTTP,
					ServiceId:  domainNameServiceName,
					ServiceConfig: &config.RelayMinerSupplierServiceConfig{
						BackendUrl: &url.URL{
							Scheme: "http",
							Host:   "exampleservice.org:8649",
							Path:   "/",
						},
					},
					PubliclyExposedEndpoints: []string{
						"exampleservice.org",
					},
				},
			},
		},
		relayminerIPV6ServiceAddr: &config.RelayMinerServerConfig{
			ListenAddress: relayminerIPV6ServiceAddr,
			ServerType:    config.RelayMinerServerTypeHTTP,
			SupplierConfigsMap: map[string]*config.RelayMinerSupplierConfig{
				IPV6ServiceName: &config.RelayMinerSupplierConfig{
					ServerType: config.RelayMinerServerTypeHTTP,
					ServiceId:  IPV6ServiceName,
					ServiceConfig: &config.RelayMinerSupplierServiceConfig{
						BackendUrl: &url.URL{
							Scheme: "http",
							Host:   "[::1]:8650",
							Path:   "/",
						},
					},
					PubliclyExposedEndpoints: []string{
						"ipv6service",
					},
				},
			},
		},
	}

	relayProxyBehavior := append(t.relayerProxyBehavior, []func(*testproxy.TestBehavior){
		testproxy.WithDefaultSupplier(supplierOperatorKeyName, supplierEndpoints),
		testproxy.WithServicesConfigMap(servicesConfigMap),
	}...)

	test := testproxy.NewRelayerProxyTestBehavior(ctx, t.T(), relayProxyBehavior...)

	// copying the default relayminer in the test service config
	// map for the relay proxy.
	for k, v := range t.servicesConfigMap {
		servicesConfigMap[k] = v
	}

	rp, err := proxy.NewRelayerProxy(
		test.Deps,
		proxy.WithSigningKeyNames([]string{supplierOperatorKeyName}),
		proxy.WithServicesConfigMap(servicesConfigMap),
	)
	require.NoError(t.T(), err)

	go func() {
		err := rp.Start(ctx)
		if err != http.ErrServerClosed {
			require.NoError(t.T(), err)
		}
	}()

	// waiting for relayer proxy to start
	// and perform ping request.
	time.Sleep(100 * time.Millisecond)

	err = rp.PingAll(ctx)
	require.NoError(t.T(), err)

	err = rp.Stop(ctx)
	require.NoError(t.T(), err)
}

func sendRequestWithUnparsableBody(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errorCode int32, errorMessage string) {
	// Send non JSONRpc payload when the post request specifies json
	reader := io.NopCloser(bytes.NewReader([]byte("invalid request")))

	res, err := http.DefaultClient.Post(
		fmt.Sprintf("http://%s", servicesConfigMap[defaultRelayMinerServer].ListenAddress),
		"application/json",
		reader,
	)
	require.NoError(t, err)
	require.NotNil(t, res)

	return testproxy.GetRelayResponseError(t, res)
}

func sendRequestWithMissingSignature(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errorCode int32, errorMessage string) {
	req := testproxy.GenerateRelayRequest(
		test,
		appPrivateKey,
		defaultService,
		blockHeight,
		supplierOperatorKeyName,
		testproxy.PrepareJSONRPCRequest(t),
	)
	req.Meta.Signature = nil
	return testproxy.MarshalAndSend(test, servicesConfigMap, defaultRelayMinerServer, defaultService, req)
}

func sendRequestWithInvalidSignature(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errorCode int32, errorMessage string) {
	req := testproxy.GenerateRelayRequest(
		test,
		appPrivateKey,
		defaultService,
		blockHeight,
		supplierOperatorKeyName,
		testproxy.PrepareJSONRPCRequest(t),
	)
	req.Meta.Signature = []byte("invalid signature")

	return testproxy.MarshalAndSend(test, servicesConfigMap, defaultRelayMinerServer, defaultService, req)
}

func sendRequestWithMissingSessionHeaderApplicationAddress(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errCode int32, errorMessage string) {
	randomPrivKey := secp256k1.GenPrivKey()
	req := testproxy.GenerateRelayRequest(
		test,
		randomPrivKey,
		defaultService,
		blockHeight,
		supplierOperatorKeyName,
		testproxy.PrepareJSONRPCRequest(t),
	)

	// The application address is missing from the session header
	req.Meta.SessionHeader.ApplicationAddress = ""

	// Assign a valid but random ring signature so that the request is not rejected
	// before looking at the application address
	req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, randomPrivKey)

	return testproxy.MarshalAndSend(test, servicesConfigMap, defaultRelayMinerServer, defaultService, req)
}

func sendRequestWithNonStakedApplicationAddress(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errCode int32, errorMessage string) {
	randomPrivKey := secp256k1.GenPrivKey()
	req := testproxy.GenerateRelayRequest(
		test,
		randomPrivKey,
		defaultService,
		blockHeight,
		supplierOperatorKeyName,
		testproxy.PrepareJSONRPCRequest(t),
	)

	// Have a valid signature from the non staked key
	req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, randomPrivKey)

	return testproxy.MarshalAndSend(test, servicesConfigMap, defaultRelayMinerServer, defaultService, req)
}

func sendRequestWithRingSignatureMismatch(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errCode int32, errorMessage string) {
	req := testproxy.GenerateRelayRequest(
		test,
		appPrivateKey,
		defaultService,
		blockHeight,
		supplierOperatorKeyName,
		testproxy.PrepareJSONRPCRequest(t),
	)

	// The signature is valid but does not match the ring for the application address
	randomPrivKey := secp256k1.GenPrivKey()
	req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, randomPrivKey)

	return testproxy.MarshalAndSend(test, servicesConfigMap, defaultRelayMinerServer, defaultService, req)
}

func sendRequestWithDifferentSession(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errCode int32, errorMessage string) {
	// Use a block height that generates a different session ID
	sharedParams := sharedtypes.DefaultParams()
	blockHeightAfterSessionGracePeriod := int64(blockHeight + sharedParams.NumBlocksPerSession + 1)
	req := testproxy.GenerateRelayRequest(
		test,
		appPrivateKey,
		defaultService,
		blockHeightAfterSessionGracePeriod,
		supplierOperatorKeyName,
		testproxy.PrepareJSONRPCRequest(t),
	)
	req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, appPrivateKey)

	return testproxy.MarshalAndSend(test, servicesConfigMap, defaultRelayMinerServer, defaultService, req)
}

func sendRequestWithInvalidRelaySupplier(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errCode int32, errorMessage string) {
	req := testproxy.GenerateRelayRequest(
		test,
		appPrivateKey,
		defaultService,
		blockHeight,
		supplierOperatorKeyName,
		testproxy.PrepareJSONRPCRequest(t),
	)
	req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, appPrivateKey)

	return testproxy.MarshalAndSend(test, servicesConfigMap, defaultRelayMinerServer, defaultService, req)
}

func sendRequestWithSignatureForDifferentPayload(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errCode int32, errorMessage string) {
	req := testproxy.GenerateRelayRequest(
		test, appPrivateKey,
		defaultService,
		blockHeight,
		supplierOperatorKeyName,
		testproxy.PrepareJSONRPCRequest(t),
	)
	req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, appPrivateKey)
	bodyBz := []byte(`{"method":"someMethod","id":1,"jsonrpc":"2.0","params":["alteredParam"]}`)
	request := &http.Request{
		Method: http.MethodPost,
		URL:    &url.URL{},
		Header: http.Header{},
		Body:   io.NopCloser(bytes.NewReader(bodyBz)),
	}
	request.Header.Set("Content-Type", "application/json")

	_, requestBz, err := sdktypes.SerializeHTTPRequest(request)
	require.NoError(t, err)

	// Alter the request payload so the hash doesn't match the one used by the signature
	req.Payload = requestBz

	return testproxy.MarshalAndSend(test, servicesConfigMap, defaultRelayMinerServer, defaultService, req)
}

func sendRequestWithSuccessfulReply(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errCode int32, errorMessage string) {
	req := testproxy.GenerateRelayRequest(
		test,
		appPrivateKey,
		defaultService,
		blockHeight,
		supplierOperatorKeyName,
		testproxy.PrepareJSONRPCRequest(t),
	)
	req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, appPrivateKey)

	return testproxy.MarshalAndSend(test, servicesConfigMap, defaultRelayMinerServer, defaultService, req)
}

// sendRequestWithCustomSessionHeight is a helper function that generates a `RelayRequest`
// with a `Session` that contains the given `requestSessionBlockHeight` and sends it to the
// `RelayerProxy`.
func sendRequestWithCustomSessionHeight(
	requestSessionBlockHeight int64,
) func(t *testing.T, test *testproxy.TestBehavior) (errCode int32, errorMessage string) {
	return func(t *testing.T, test *testproxy.TestBehavior) (errCode int32, errorMessage string) {
		req := testproxy.GenerateRelayRequest(
			test,
			appPrivateKey,
			defaultService,
			requestSessionBlockHeight,
			supplierOperatorKeyName,
			testproxy.PrepareJSONRPCRequest(t),
		)
		req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, appPrivateKey)

		return testproxy.MarshalAndSend(test, servicesConfigMap, defaultRelayMinerServer, defaultService, req)
	}
}
