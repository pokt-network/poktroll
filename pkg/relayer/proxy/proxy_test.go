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
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const blockHeight = 1

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
		"service1": {
			{
				Url: "http://supplier:8545/",
				// TODO_EXTEND: Consider adding support for non JSON RPC services in the future
				RpcType: sharedtypes.RPCType_JSON_RPC,
			},
		},
		"service2": {
			{
				Url:     "http://supplier:8546/",
				RpcType: sharedtypes.RPCType_GRPC,
			},
		},
		"service3": {
			{
				Url:     "http://supplier:8547/",
				RpcType: sharedtypes.RPCType_GRPC,
			},
		},
	}

	proxiedServices = map[string]*config.RelayMinerProxyConfig{
		"server1": {
			ProxyName: "server1",
			Type:      config.ProxyTypeHTTP,
			Host:      "127.0.0.1:8080",
			Suppliers: map[string]*config.RelayMinerSupplierConfig{
				"service1": {
					ServiceId: "service1",
					Type:      config.ProxyTypeHTTP,
					Hosts:     []string{"supplier:8545"},
					ServiceConfig: &config.RelayMinerSupplierServiceConfig{
						Url: &url.URL{Scheme: "http", Host: "127.0.0.1:8545", Path: "/"},
					},
				},
				"service2": {
					ServiceId: "service2",
					Type:      config.ProxyTypeHTTP,
					Hosts:     []string{"supplier:8546"},
					ServiceConfig: &config.RelayMinerSupplierServiceConfig{
						Url: &url.URL{Scheme: "http", Host: "127.0.0.1:8546", Path: "/"},
					},
				},
			},
		},
		"server2": {
			ProxyName: "server2",
			Type:      config.ProxyTypeHTTP,
			Host:      "127.0.0.1:8081",
			Suppliers: map[string]*config.RelayMinerSupplierConfig{
				"service3": {
					ServiceId: "service3",
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
		testproxy.WithRelayerProxyDependencies(supplierKeyName),
		testproxy.WithRelayerProxiedServices(proxiedServices),
		testproxy.WithDefaultSupplier(supplierKeyName, supplierEndpoints),
		testproxy.WithDefaultApplication(appPrivateKey),
		testproxy.WithDefaultSessionSupplier(supplierKeyName, "service1", appPrivateKey),
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
	res, err := http.DefaultClient.Get(fmt.Sprintf("http://%s/", proxiedServices["server1"].Host))
	require.NoError(t, err)
	require.NotNil(t, res)

	// Test that RelayerProxy is handling requests from the other server
	res, err = http.DefaultClient.Get(fmt.Sprintf("http://%s/", proxiedServices["server2"].Host))
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
		"service1": {
			{
				Url: "http://unsupported:8545/jsonrpc",
				// TODO_EXTEND: Consider adding support for non JSON RPC services in the future
				RpcType: sharedtypes.RPCType_JSON_RPC,
			},
		},
	}

	unsupportedRPCTypeBehavior := []func(*testproxy.TestBehavior){
		testproxy.WithRelayerProxyDependencies(supplierKeyName),
		testproxy.WithRelayerProxiedServices(proxiedServices),

		// The supplier is staked on-chain but the service it provides is not supported by the proxy
		testproxy.WithDefaultSupplier(supplierKeyName, unsupportedSupplierEndpoint),
		testproxy.WithDefaultApplication(appPrivateKey),
		testproxy.WithDefaultSessionSupplier(supplierKeyName, "service1", appPrivateKey),
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
		"service1": {
			{
				Url:     "xttp://supplier:8545/",
				RpcType: sharedtypes.RPCType_JSON_RPC,
			},
		},
	}

	unsupportedTransportProxy := map[string]*config.RelayMinerProxyConfig{
		"server1": {
			ProxyName: "server1",
			// The proxy is configured with an unsupported transport type
			Type: config.ProxyType(100),
			Host: "127.0.0.1:8080",
			Suppliers: map[string]*config.RelayMinerSupplierConfig{
				"service1": {
					ServiceId: "service1",
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
		testproxy.WithRelayerProxyDependencies(supplierKeyName),

		// The proxy is configured with an unsupported transport type for the proxy
		testproxy.WithRelayerProxiedServices(unsupportedTransportProxy),
		testproxy.WithDefaultSupplier(supplierKeyName, badTransportSupplierEndpoints),
		testproxy.WithDefaultApplication(appPrivateKey),
		testproxy.WithDefaultSessionSupplier(supplierKeyName, "service1", appPrivateKey),
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
		"server1": {
			ProxyName: "server1",
			Type:      config.ProxyTypeHTTP,
			Host:      "127.0.0.1:8080",
			Suppliers: map[string]*config.RelayMinerSupplierConfig{
				"service1": {
					ServiceId: "service1",
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
		testproxy.WithRelayerProxyDependencies(supplierKeyName),

		// The proxy is configured with an unsupported transport type for the proxy
		testproxy.WithRelayerProxiedServices(missingServicesProxy),
		testproxy.WithDefaultSupplier(supplierKeyName, supplierEndpoints),
		testproxy.WithDefaultApplication(appPrivateKey),
		testproxy.WithDefaultSessionSupplier(supplierKeyName, "service1", appPrivateKey),
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
			desc: "Missing session meta from relay request",

			relayerProxyBehavior: defaultRelayerProxyBehavior,
			inputScenario:        sendRequestWithMissingMeta,

			expectedErrCode: -32000,
			expectedErrMsg:  "missing meta from relay request",
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
			expectedErrMsg:  "missing application address from relay request",
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
			expectedErrMsg:  "ring signature does not match ring for application address",
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
				testproxy.WithRelayerProxyDependencies(supplierKeyName),
				testproxy.WithRelayerProxiedServices(proxiedServices),
				testproxy.WithDefaultSupplier(supplierKeyName, supplierEndpoints),
				testproxy.WithDefaultApplication(appPrivateKey),
				// Missing session supplier
				testproxy.WithDefaultSessionSupplier("", "service1", appPrivateKey),
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
			expectedErrMsg:  "invalid ring signature",
		},
		{
			desc: "Successful relay",

			relayerProxyBehavior: defaultRelayerProxyBehavior,
			inputScenario:        sendRequestWithSuccessfulReply,

			expectedErrCode: 0,
			expectedErrMsg:  "",
		},
	}

	ctx := context.TODO()
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			ctx, cancel := context.WithCancel(ctx)
			test := testproxy.NewRelayerProxyTestBehavior(ctx, t, tt.relayerProxyBehavior...)

			rp, err := proxy.NewRelayerProxy(
				test.Deps,
				proxy.WithSigningKeyName(supplierKeyName),
				proxy.WithProxiedServicesEndpoints(proxiedServices),
			)
			require.NoError(t, err)

			go rp.Start(ctx)
			// Block so relayerProxy has sufficient time to start
			time.Sleep(100 * time.Millisecond)

			errCode, errMsg := tt.inputScenario(t, test)
			require.Equal(t, tt.expectedErrCode, errCode)
			require.True(t, strings.HasPrefix(errMsg, tt.expectedErrMsg))

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
		fmt.Sprintf("http://%s", proxiedServices["server1"].Host),
		"application/json",
		reader,
	)
	require.NoError(t, err)
	require.NotNil(t, res)

	return testproxy.GetRelayResponseError(t, res)
}

func sendRequestWithMissingMeta(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errorCode int32, errorMessage string) {

	req := &servicetypes.RelayRequest{
		// RelayRequest is missing Metadata
		Payload: testproxy.PrepareJsonRPCRequestPayload(),
	}

	return testproxy.MarshalAndSend(test, proxiedServices, "server1", "service1", req)
}

func sendRequestWithMissingSignature(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errorCode int32, errorMessage string) {
	req := testproxy.GenerateRelayRequest(
		test,
		appPrivateKey,
		"service1",
		blockHeight,
		testproxy.PrepareJsonRPCRequestPayload(),
	)
	req.Meta.Signature = nil
	return testproxy.MarshalAndSend(test, proxiedServices, "server1", "service1", req)
}

func sendRequestWithInvalidSignature(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errorCode int32, errorMessage string) {
	req := testproxy.GenerateRelayRequest(
		test,
		appPrivateKey,
		"service1",
		blockHeight,
		testproxy.PrepareJsonRPCRequestPayload(),
	)
	req.Meta.Signature = []byte("invalid signature")

	return testproxy.MarshalAndSend(test, proxiedServices, "server1", "service1", req)
}

func sendRequestWithMissingSessionHeaderApplicationAddress(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errCode int32, errorMessage string) {
	randomPrivKey := secp256k1.GenPrivKey()
	req := testproxy.GenerateRelayRequest(
		test,
		randomPrivKey,
		"service1",
		blockHeight,
		testproxy.PrepareJsonRPCRequestPayload(),
	)

	// The application address is missing from the session header
	req.Meta.SessionHeader.ApplicationAddress = ""

	// Assign a valid but random ring signature so that the request is not rejected
	// before looking at the application address
	req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, randomPrivKey)

	return testproxy.MarshalAndSend(test, proxiedServices, "server1", "service1", req)
}

func sendRequestWithNonStakedApplicationAddress(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errCode int32, errorMessage string) {
	randomPrivKey := secp256k1.GenPrivKey()
	req := testproxy.GenerateRelayRequest(
		test,
		randomPrivKey,
		"service1",
		blockHeight,
		testproxy.PrepareJsonRPCRequestPayload(),
	)

	// Have a valid signature from the non staked key
	req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, randomPrivKey)

	return testproxy.MarshalAndSend(test, proxiedServices, "server1", "service1", req)
}

func sendRequestWithRingSignatureMismatch(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errCode int32, errorMessage string) {
	req := testproxy.GenerateRelayRequest(
		test,
		appPrivateKey,
		"service1",
		blockHeight,
		testproxy.PrepareJsonRPCRequestPayload(),
	)

	// The signature is valid but does not match the ring for the application address
	randomPrivKey := secp256k1.GenPrivKey()
	req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, randomPrivKey)

	return testproxy.MarshalAndSend(test, proxiedServices, "server1", "service1", req)
}

func sendRequestWithDifferentSession(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errCode int32, errorMessage string) {
	// Use service2 instead of service1 so the session IDs don't match
	req := testproxy.GenerateRelayRequest(
		test,
		appPrivateKey,
		"service2",
		blockHeight,
		testproxy.PrepareJsonRPCRequestPayload(),
	)
	req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, appPrivateKey)

	return testproxy.MarshalAndSend(test, proxiedServices, "server1", "service1", req)
}

func sendRequestWithInvalidRelaySupplier(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errCode int32, errorMessage string) {
	req := testproxy.GenerateRelayRequest(
		test,
		appPrivateKey,
		"service1",
		blockHeight,
		testproxy.PrepareJsonRPCRequestPayload(),
	)
	req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, appPrivateKey)

	return testproxy.MarshalAndSend(test, proxiedServices, "server1", "service1", req)
}

func sendRequestWithSignatureForDifferentPayload(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errCode int32, errorMessage string) {
	req := testproxy.GenerateRelayRequest(
		test, appPrivateKey,
		"service1",
		blockHeight,
		testproxy.PrepareJsonRPCRequestPayload(),
	)
	req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, appPrivateKey)

	// Alter the request payload so the hash doesn't match the one used by the signature
	req.Payload = []byte(`{"method":"someMethod","id":1,"jsonrpc":"2.0","params":["alteredParam"]}`)

	return testproxy.MarshalAndSend(test, proxiedServices, "server1", "service1", req)
}

func sendRequestWithSuccessfulReply(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errCode int32, errorMessage string) {
	req := testproxy.GenerateRelayRequest(
		test,
		appPrivateKey,
		"service1",
		blockHeight,
		testproxy.PrepareJsonRPCRequestPayload(),
	)
	req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, appPrivateKey)

	return testproxy.MarshalAndSend(test, proxiedServices, "server1", "service1", req)
}
