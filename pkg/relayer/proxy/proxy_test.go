package proxy_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/relayer/config"
	"github.com/pokt-network/poktroll/pkg/relayer/proxy"
	"github.com/pokt-network/poktroll/testutil/testproxy"
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const (
	blockHeight          = 1
	defaultService       = "service1"
	secondaryService     = "service2"
	thirdService         = "service3"
	defaultProxyServer   = "server1"
	secondaryProxyServer = "server2"
)

var (
	// helpers used for tests that are initialized in init()
	supplierKeyName string

	// supplierEndpoints is the map of serviceName -> []SupplierEndpoint
	// where serviceName is the name of the service the supplier staked for
	// and SupplierEndpoint is the endpoint of the service advertised on-chain
	// by the supplier
	supplierEndpoints map[string][]*sharedtypes.SupplierEndpoint

	// appPrivateKey is the private key of the application that is used to sign
	// relay responses.
	// It is also used in these tests to derive the public key and address of the
	// application.
	appPrivateKey *secp256k1.PrivKey

	// proxiedServices is the parsed configuration of the RelayMinerProxyConfig
	proxiedServices map[string]*config.RelayMinerProxyConfig

	// defaultRelayerProxyBehavior is the list of functions that are used to
	// define the behavior of the RelayerProxy in the tests.
	defaultRelayerProxyBehavior []func(*testproxy.TestBehavior)
)

func init() {
	supplierKeyName = "supplierKeyName"
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
				Url:     "http://supplier:8546/",
				RpcType: sharedtypes.RPCType_GRPC,
			},
		},
		thirdService: {
			{
				Url:     "http://supplier:8547/",
				RpcType: sharedtypes.RPCType_GRPC,
			},
		},
	}

	proxiedServices = map[string]*config.RelayMinerProxyConfig{
		defaultProxyServer: {
			ProxyName: defaultProxyServer,
			Type:      config.ProxyTypeHTTP,
			Host:      "127.0.0.1:8080",
			Suppliers: map[string]*config.RelayMinerSupplierConfig{
				defaultService: {
					ServiceId: defaultService,
					Type:      config.ProxyTypeHTTP,
					Hosts:     []string{"supplier:8545"},
					ServiceConfig: &config.RelayMinerSupplierServiceConfig{
						Url: &url.URL{Scheme: "http", Host: "127.0.0.1:8545", Path: "/"},
					},
				},
				secondaryService: {
					ServiceId: secondaryService,
					Type:      config.ProxyTypeHTTP,
					Hosts:     []string{"supplier:8546"},
					ServiceConfig: &config.RelayMinerSupplierServiceConfig{
						Url: &url.URL{Scheme: "http", Host: "127.0.0.1:8546", Path: "/"},
					},
				},
			},
		},
		secondaryProxyServer: {
			ProxyName: secondaryProxyServer,
			Type:      config.ProxyTypeHTTP,
			Host:      "127.0.0.1:8081",
			Suppliers: map[string]*config.RelayMinerSupplierConfig{
				thirdService: {
					ServiceId: thirdService,
					Type:      config.ProxyTypeHTTP,
					Hosts:     []string{"supplier:8547"},
					ServiceConfig: &config.RelayMinerSupplierServiceConfig{
						Url: &url.URL{Scheme: "http", Host: "127.0.0.1:8547", Path: "/"},
					},
				},
			},
		},
	}

	defaultRelayerProxyBehavior = []func(*testproxy.TestBehavior){
		testproxy.WithRelayerProxyDependenciesForBlockHeight(supplierKeyName, blockHeight),
		testproxy.WithRelayerProxiedServices(proxiedServices),
		testproxy.WithDefaultSupplier(supplierKeyName, supplierEndpoints),
		testproxy.WithDefaultApplication(appPrivateKey),
		testproxy.WithDefaultSessionSupplier(supplierKeyName, defaultService, appPrivateKey),
	}
}

// RelayerProxy should start and stop without errors
func TestRelayerProxy_StartAndStop(t *testing.T) {
	ctx := context.TODO()
	// Setup the RelayerProxy instrumented behavior
	test := testproxy.NewRelayerProxyTestBehavior(ctx, t, defaultRelayerProxyBehavior...)

	// Create a RelayerProxy
	rp, err := proxy.NewRelayerProxy(
		test.Deps,
		proxy.WithSigningKeyName(supplierKeyName),
		proxy.WithProxiedServicesEndpoints(proxiedServices),
	)
	require.NoError(t, err)

	// Start RelayerProxy
	go rp.Start(ctx)
	// Block so relayerProxy has sufficient time to start
	time.Sleep(100 * time.Millisecond)

	// Test that RelayerProxy is handling requests (ignoring the actual response content)
	res, err := http.DefaultClient.Get(fmt.Sprintf("http://%s/", proxiedServices[defaultProxyServer].Host))
	require.NoError(t, err)
	require.NotNil(t, res)

	// Test that RelayerProxy is handling requests from the other server
	res, err = http.DefaultClient.Get(fmt.Sprintf("http://%s/", proxiedServices[secondaryProxyServer].Host))
	require.NoError(t, err)
	require.NotNil(t, res)

	// Stop RelayerProxy
	err = rp.Stop(ctx)
	require.NoError(t, err)
}

// RelayerProxy should fail to start if the signing key is not found in the keyring
func TestRelayerProxy_InvalidSupplierKeyName(t *testing.T) {
	ctx := context.TODO()
	test := testproxy.NewRelayerProxyTestBehavior(ctx, t, defaultRelayerProxyBehavior...)

	rp, err := proxy.NewRelayerProxy(
		test.Deps,
		proxy.WithSigningKeyName("wrongKeyName"),
		proxy.WithProxiedServicesEndpoints(proxiedServices),
	)
	require.NoError(t, err)

	err = rp.Start(ctx)
	require.Error(t, err)
}

// RelayerProxy should fail to build if the signing key name is not provided
func TestRelayerProxy_MissingSupplierKeyName(t *testing.T) {
	ctx := context.TODO()
	test := testproxy.NewRelayerProxyTestBehavior(ctx, t, defaultRelayerProxyBehavior...)

	_, err := proxy.NewRelayerProxy(
		test.Deps,
		proxy.WithSigningKeyName(""),
		proxy.WithProxiedServicesEndpoints(proxiedServices),
	)
	require.Error(t, err)
}

// RelayerProxy should fail to build if the proxied services endpoints are not provided
func TestRelayerProxy_NoProxiedServices(t *testing.T) {
	ctx := context.TODO()

	test := testproxy.NewRelayerProxyTestBehavior(ctx, t, defaultRelayerProxyBehavior...)

	_, err := proxy.NewRelayerProxy(
		test.Deps,
		proxy.WithSigningKeyName(supplierKeyName),
		proxy.WithProxiedServicesEndpoints(make(map[string]*config.RelayMinerProxyConfig)),
	)
	require.Error(t, err)
}

// RelayerProxy should fail to start if it cannot spawn a server for the
// services it advertized on-chain.
func TestRelayerProxy_UnsupportedRpcType(t *testing.T) {
	ctx := context.TODO()

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
		testproxy.WithRelayerProxyDependenciesForBlockHeight(supplierKeyName, blockHeight),
		testproxy.WithRelayerProxiedServices(proxiedServices),

		// The supplier is staked on-chain but the service it provides is not supported by the proxy
		testproxy.WithDefaultSupplier(supplierKeyName, unsupportedSupplierEndpoint),
		testproxy.WithDefaultApplication(appPrivateKey),
		testproxy.WithDefaultSessionSupplier(supplierKeyName, defaultService, appPrivateKey),
	}

	test := testproxy.NewRelayerProxyTestBehavior(ctx, t, unsupportedRPCTypeBehavior...)

	rp, err := proxy.NewRelayerProxy(
		test.Deps,
		proxy.WithSigningKeyName(supplierKeyName),
		proxy.WithProxiedServicesEndpoints(proxiedServices),
	)
	require.NoError(t, err)

	err = rp.Start(ctx)
	require.Error(t, err)
}

func TestRelayerProxy_UnsupportedTransportType(t *testing.T) {
	ctx := context.TODO()

	badTransportSupplierEndpoints := map[string][]*sharedtypes.SupplierEndpoint{
		defaultService: {
			{
				Url:     "xttp://supplier:8545/",
				RpcType: sharedtypes.RPCType_JSON_RPC,
			},
		},
	}

	unsupportedTransportProxy := map[string]*config.RelayMinerProxyConfig{
		defaultProxyServer: {
			ProxyName: defaultProxyServer,
			// The proxy is configured with an unsupported transport type
			Type: config.ProxyType(100),
			Host: "127.0.0.1:8080",
			Suppliers: map[string]*config.RelayMinerSupplierConfig{
				defaultService: {
					ServiceId: defaultService,
					// The proxy is configured with an unsupported transport type
					Type:  config.ProxyType(100),
					Hosts: []string{"supplier:8545"},
					ServiceConfig: &config.RelayMinerSupplierServiceConfig{
						Url: &url.URL{Scheme: "http", Host: "127.0.0.1:8545", Path: "/"},
					},
				},
			},
		},
	}

	unsupportedTransportTypeBehavior := []func(*testproxy.TestBehavior){
		testproxy.WithRelayerProxyDependenciesForBlockHeight(supplierKeyName, blockHeight),

		// The proxy is configured with an unsupported transport type for the proxy
		testproxy.WithRelayerProxiedServices(unsupportedTransportProxy),
		testproxy.WithDefaultSupplier(supplierKeyName, badTransportSupplierEndpoints),
		testproxy.WithDefaultApplication(appPrivateKey),
		testproxy.WithDefaultSessionSupplier(supplierKeyName, defaultService, appPrivateKey),
	}

	test := testproxy.NewRelayerProxyTestBehavior(ctx, t, unsupportedTransportTypeBehavior...)

	rp, err := proxy.NewRelayerProxy(
		test.Deps,
		proxy.WithSigningKeyName(supplierKeyName),
		proxy.WithProxiedServicesEndpoints(unsupportedTransportProxy),
	)
	require.NoError(t, err)

	err = rp.Start(ctx)
	require.ErrorIs(t, err, proxy.ErrRelayerProxyUnsupportedTransportType)
}

func TestRelayerProxy_NonConfiguredSupplierServices(t *testing.T) {
	ctx := context.TODO()

	missingServicesProxy := map[string]*config.RelayMinerProxyConfig{
		defaultProxyServer: {
			ProxyName: defaultProxyServer,
			Type:      config.ProxyTypeHTTP,
			Host:      "127.0.0.1:8080",
			Suppliers: map[string]*config.RelayMinerSupplierConfig{
				defaultService: {
					ServiceId: defaultService,
					Type:      config.ProxyTypeHTTP,
					Hosts:     []string{"supplier:8545"},
					ServiceConfig: &config.RelayMinerSupplierServiceConfig{
						Url: &url.URL{Scheme: "http", Host: "127.0.0.1:8545", Path: "/"},
					},
				},
			},
		},
	}

	unsupportedTransportTypeBehavior := []func(*testproxy.TestBehavior){
		testproxy.WithRelayerProxyDependenciesForBlockHeight(supplierKeyName, blockHeight),

		// The proxy is configured with an unsupported transport type for the proxy
		testproxy.WithRelayerProxiedServices(missingServicesProxy),
		testproxy.WithDefaultSupplier(supplierKeyName, supplierEndpoints),
		testproxy.WithDefaultApplication(appPrivateKey),
		testproxy.WithDefaultSessionSupplier(supplierKeyName, defaultService, appPrivateKey),
	}

	test := testproxy.NewRelayerProxyTestBehavior(ctx, t, unsupportedTransportTypeBehavior...)

	rp, err := proxy.NewRelayerProxy(
		test.Deps,
		proxy.WithSigningKeyName(supplierKeyName),
		proxy.WithProxiedServicesEndpoints(missingServicesProxy),
	)
	require.NoError(t, err)

	err = rp.Start(ctx)
	require.ErrorIs(t, err, proxy.ErrRelayerProxyServiceEndpointNotHandled)
}

// Test different RelayRequest scenarios
func TestRelayerProxy_Relays(t *testing.T) {
	// blockOutsideSessionGracePeriod is the block height that is after the first
	// session's grace period and within the second session's grace period,
	// meaning a relay should not be handled at this block height.
	blockOutsideSessionGracePeriod := blockHeight +
		sessionkeeper.NumBlocksPerSession +
		sessionkeeper.GetSessionGracePeriodBlockCount()

	// blockWithinSessionGracePeriod is the block height that is after the first
	// session but within its session's grace period, meaning a relay should be
	// handled at this block height.
	blockWithinSessionGracePeriod := blockHeight + sessionkeeper.GetSessionGracePeriodBlockCount()

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
			expectedErrCode: 0,
			expectedErrMsg:  "cannot unmarshal request payload",
		},
		{
			desc: "Missing signature from relay request",

			relayerProxyBehavior: defaultRelayerProxyBehavior,
			inputScenario:        sendRequestWithMissingSignature,

			expectedErrCode: -32000,
			expectedErrMsg:  "missing signature from relay request",
		},
		{
			desc: "Invalid signature associated with relay request",

			relayerProxyBehavior: defaultRelayerProxyBehavior,
			inputScenario:        sendRequestWithInvalidSignature,

			expectedErrCode: -32000,
			expectedErrMsg:  "error deserializing ring signature",
		},
		{
			desc: "Missing session header application address associated with relay request",

			relayerProxyBehavior: defaultRelayerProxyBehavior,
			inputScenario:        sendRequestWithMissingSessionHeaderApplicationAddress,

			expectedErrCode: -32000,
			expectedErrMsg:  "invalid session header: invalid application address",
		},
		{
			desc: "Non staked application address",

			relayerProxyBehavior: defaultRelayerProxyBehavior,
			inputScenario:        sendRequestWithNonStakedApplicationAddress,

			expectedErrCode: -32000,
			expectedErrMsg:  "error getting ring for application address",
		},
		{
			desc: "Ring signature mismatch",

			relayerProxyBehavior: defaultRelayerProxyBehavior,
			inputScenario:        sendRequestWithRingSignatureMismatch,

			expectedErrCode: -32000,
			expectedErrMsg:  "ring signature in the relay request does not match the expected one for the app",
		},
		{
			desc: "Session mismatch",

			relayerProxyBehavior: defaultRelayerProxyBehavior,
			inputScenario:        sendRequestWithDifferentSession,

			expectedErrCode: -32000,
			expectedErrMsg:  "session mismatch",
		},
		{
			desc: "Invalid relay supplier",

			relayerProxyBehavior: []func(*testproxy.TestBehavior){
				testproxy.WithRelayerProxyDependenciesForBlockHeight(supplierKeyName, blockHeight),
				testproxy.WithRelayerProxiedServices(proxiedServices),
				testproxy.WithDefaultSupplier(supplierKeyName, supplierEndpoints),
				testproxy.WithDefaultApplication(appPrivateKey),
				// Missing session supplier
				testproxy.WithDefaultSessionSupplier("", defaultService, appPrivateKey),
			},
			inputScenario: sendRequestWithInvalidRelaySupplier,

			expectedErrCode: -32000,
			expectedErrMsg:  "error while trying to retrieve a session",
		},
		{
			desc: "Relay request signature does not match the request payload",

			relayerProxyBehavior: defaultRelayerProxyBehavior,
			inputScenario:        sendRequestWithSignatureForDifferentPayload,

			expectedErrCode: -32000,
			expectedErrMsg:  "invalid relay request signature or bytes",
		},
		{
			desc: "Successful relay",

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
					supplierKeyName,
					blockWithinSessionGracePeriod,
				),
				testproxy.WithRelayerProxiedServices(proxiedServices),
				testproxy.WithDefaultSupplier(supplierKeyName, supplierEndpoints),
				testproxy.WithDefaultApplication(appPrivateKey),
				// Add 2 sessions, with the first one being within the withing grace period
				// and the second one being the current session
				testproxy.WithSuccessiveSessions(supplierKeyName, defaultService, appPrivateKey, 2),
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
					supplierKeyName,
					// Set the current block height value returned by the block provider
					blockOutsideSessionGracePeriod,
				),
				testproxy.WithRelayerProxiedServices(proxiedServices),
				testproxy.WithDefaultSupplier(supplierKeyName, supplierEndpoints),
				testproxy.WithDefaultApplication(appPrivateKey),
				// Add 3 sessions, with the first one that is no longer within its
				// session grace period
				testproxy.WithSuccessiveSessions(supplierKeyName, defaultService, appPrivateKey, 3),
			},
			// Send a request that has a late session past the grace period
			inputScenario: sendRequestWithCustomSessionHeight(blockHeight),

			expectedErrCode: -32000,
			expectedErrMsg:  "session expired", // Relay rejected by the supplier
		},
	}

	ctx := context.TODO()
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			ctx, cancel := context.WithCancel(ctx)
			testBehavior := testproxy.NewRelayerProxyTestBehavior(ctx, t, test.relayerProxyBehavior...)

			rp, err := proxy.NewRelayerProxy(
				testBehavior.Deps,
				proxy.WithSigningKeyName(supplierKeyName),
				proxy.WithProxiedServicesEndpoints(proxiedServices),
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

func sendRequestWithUnparsableBody(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errorCode int32, errorMessage string) {
	// Send non JSONRpc payload when the post request specifies json
	reader := io.NopCloser(bytes.NewReader([]byte("invalid request")))

	res, err := http.DefaultClient.Post(
		fmt.Sprintf("http://%s", proxiedServices[defaultProxyServer].Host),
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
		testproxy.PrepareJsonRPCRequestPayload(),
	)
	req.Meta.Signature = nil
	return testproxy.MarshalAndSend(test, proxiedServices, defaultProxyServer, defaultService, req)
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
		testproxy.PrepareJsonRPCRequestPayload(),
	)
	req.Meta.Signature = []byte("invalid signature")

	return testproxy.MarshalAndSend(test, proxiedServices, defaultProxyServer, defaultService, req)
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
		testproxy.PrepareJsonRPCRequestPayload(),
	)

	// The application address is missing from the session header
	req.Meta.SessionHeader.ApplicationAddress = ""

	// Assign a valid but random ring signature so that the request is not rejected
	// before looking at the application address
	req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, randomPrivKey)

	return testproxy.MarshalAndSend(test, proxiedServices, defaultProxyServer, defaultService, req)
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
		testproxy.PrepareJsonRPCRequestPayload(),
	)

	// Have a valid signature from the non staked key
	req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, randomPrivKey)

	return testproxy.MarshalAndSend(test, proxiedServices, defaultProxyServer, defaultService, req)
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
		testproxy.PrepareJsonRPCRequestPayload(),
	)

	// The signature is valid but does not match the ring for the application address
	randomPrivKey := secp256k1.GenPrivKey()
	req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, randomPrivKey)

	return testproxy.MarshalAndSend(test, proxiedServices, defaultProxyServer, defaultService, req)
}

func sendRequestWithDifferentSession(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errCode int32, errorMessage string) {
	// Use secondaryService instead of service1 so the session IDs don't match
	req := testproxy.GenerateRelayRequest(
		test,
		appPrivateKey,
		secondaryService,
		blockHeight,
		testproxy.PrepareJsonRPCRequestPayload(),
	)
	req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, appPrivateKey)

	return testproxy.MarshalAndSend(test, proxiedServices, defaultProxyServer, defaultService, req)
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
		testproxy.PrepareJsonRPCRequestPayload(),
	)
	req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, appPrivateKey)

	return testproxy.MarshalAndSend(test, proxiedServices, defaultProxyServer, defaultService, req)
}

func sendRequestWithSignatureForDifferentPayload(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errCode int32, errorMessage string) {
	req := testproxy.GenerateRelayRequest(
		test, appPrivateKey,
		defaultService,
		blockHeight,
		testproxy.PrepareJsonRPCRequestPayload(),
	)
	req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, appPrivateKey)

	// Alter the request payload so the hash doesn't match the one used by the signature
	req.Payload = []byte(`{"method":"someMethod","id":1,"jsonrpc":"2.0","params":["alteredParam"]}`)

	return testproxy.MarshalAndSend(test, proxiedServices, defaultProxyServer, defaultService, req)
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
		testproxy.PrepareJsonRPCRequestPayload(),
	)
	req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, appPrivateKey)

	return testproxy.MarshalAndSend(test, proxiedServices, defaultProxyServer, defaultService, req)
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
			testproxy.PrepareJsonRPCRequestPayload(),
		)
		req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, appPrivateKey)

		return testproxy.MarshalAndSend(test, proxiedServices, defaultProxyServer, defaultService, req)
	}
}
