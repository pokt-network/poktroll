package proxy_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/relayer/proxy"
	"github.com/pokt-network/poktroll/testutil/testproxy"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var relayerProxyConfig = testproxy.RelayerProxyConfig{
	SupplierKeyName: "supplierKeyName",
	ProxiedServicesConfig: map[string]string{
		"service1": "http://localhost:8080",
		"service2": "http://localhost:8081",
	},
	ProvidedServices: map[string]testproxy.ProvidedServiceConfig{
		"service1": {Url: "http://localhost:8180", RpcType: sharedtypes.RPCType_JSON_RPC},
		"service2": {Url: "http://localhost:8181", RpcType: sharedtypes.RPCType_JSON_RPC},
	},
}

var defaultBehavior = []func(*testproxy.TestBehavior){
	testproxy.WithRelayerProxyMocks,
	testproxy.WithRelayerProxyDependencies,
	testproxy.WithKeyringDefaultBehavior,
	testproxy.WithSupplierDefaultBehavior,
	testproxy.WithRelayerProxiedServices,
	testproxy.WithProxiedServiceDefaultBehavior,
	testproxy.WithApplicationDefaultBehavior,
	testproxy.WithAccountsDefaultBehavior,
	testproxy.WithBlockClientDefaultBehavior,
	testproxy.WithSessionDefaultBehavior,
}

// RelayerProxy should start and stop without errors
func TestRelayerProxy_StartAndStop(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup the RelayerProxy instrumented behavior
	test := testproxy.NewRelayerProxyTestBehavior(ctx, t, ctrl, relayerProxyConfig, defaultBehavior...)

	// Create a RelayerProxy
	rp, err := proxy.NewRelayerProxy(
		test.Deps,
		proxy.WithSigningKeyName(test.SupplierKeyName),
		proxy.WithProxiedServicesEndpoints(test.ProxiedServicesEndpoints),
	)
	require.NoError(t, err)

	// Start RelayerProxy
	go rp.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	// Test that RelayerProxy is handling requests, don't care about the response here
	res1, err := http.DefaultClient.Get(test.ProvidedServices["service1"].Url)
	require.NoError(t, err)
	require.NotNil(t, res1)

	res2, err := http.DefaultClient.Get(test.ProvidedServices["service2"].Url)
	require.NoError(t, err)
	require.NotNil(t, res2)

	// Stop RelayerProxy
	err = rp.Stop(ctx)
	require.NoError(t, err)
}

// RelayerProxy should fail to start if the signing key is not found in the keyring
func TestRelayerProxy_InvalidSupplierKeyName(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	test := testproxy.NewRelayerProxyTestBehavior(ctx, t, ctrl, relayerProxyConfig, defaultBehavior...)

	rp, err := proxy.NewRelayerProxy(
		test.Deps,
		proxy.WithSigningKeyName("wrongKeyName"),
		proxy.WithProxiedServicesEndpoints(test.ProxiedServicesEndpoints),
	)
	require.NoError(t, err)

	err = rp.Start(ctx)
	require.Error(t, err)
}

// RelayerProxy should fail to build if the signing key name is not provided
func TestRelayerProxy_MissingSupplierKeyName(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	test := testproxy.NewRelayerProxyTestBehavior(ctx, t, ctrl, relayerProxyConfig, defaultBehavior...)

	_, err := proxy.NewRelayerProxy(
		test.Deps,
		proxy.WithSigningKeyName(""),
		proxy.WithProxiedServicesEndpoints(test.ProxiedServicesEndpoints),
	)
	require.Error(t, err)
}

// RelayerProxy should fail to build if the proxied services endpoints are not provided
func TestRelayerProxy_NoProxiedServices(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := testproxy.RelayerProxyConfig{
		SupplierKeyName: relayerProxyConfig.SupplierKeyName,
		// Do not provide proxied services
		ProxiedServicesConfig: nil,
		ProvidedServices:      relayerProxyConfig.ProvidedServices,
	}

	test := testproxy.NewRelayerProxyTestBehavior(ctx, t, ctrl, cfg, defaultBehavior...)

	_, err := proxy.NewRelayerProxy(
		test.Deps,
		proxy.WithSigningKeyName(cfg.SupplierKeyName),
		proxy.WithProxiedServicesEndpoints(test.ProxiedServicesEndpoints),
	)
	require.Error(t, err)
}

// RelayerProxy should fail to start if it cannot spawn a server for the
// services it advertized on-chain
func TestRelayerProxy_UnsupportedRpcType(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := testproxy.RelayerProxyConfig{
		SupplierKeyName:       relayerProxyConfig.SupplierKeyName,
		ProxiedServicesConfig: relayerProxyConfig.ProxiedServicesConfig,
		// Supplier has advertised providing a GRPC service but does not support it
		ProvidedServices: map[string]testproxy.ProvidedServiceConfig{
			"service1": {Url: "http://localhost:8180", RpcType: sharedtypes.RPCType_GRPC},
		},
	}

	test := testproxy.NewRelayerProxyTestBehavior(ctx, t, ctrl, cfg, defaultBehavior...)

	rp, err := proxy.NewRelayerProxy(
		test.Deps,
		proxy.WithSigningKeyName(cfg.SupplierKeyName),
		proxy.WithProxiedServicesEndpoints(test.ProxiedServicesEndpoints),
	)
	require.NoError(t, err)

	err = rp.Start(ctx)
	require.Error(t, err)
}

// Test different RelayRequest scenarios
func TestRelayerProxy_Relays(t *testing.T) {
	tests := []struct {
		name string

		// RelayerProxy instrumented behavior
		relayerProxyBehavior []func(*testproxy.TestBehavior)
		// Input scenario builds a RelayRequest, marshals it and sends it to the RelayerProxy
		inputScenario func(
			t *testing.T,
			test *testproxy.TestBehavior,
		) (errCode int32, errMsg string)

		// The request result should not contain any error returned by
		// the http.DefaultClient.Do call.
		// We infer the behavior from the response's code and message prefix
		expectedErrCode int32
		expectedErrMsg  string
	}{
		{
			name: "Unparsable request",

			relayerProxyBehavior: defaultBehavior,
			inputScenario:        sendRequestWithUnparsableBody,

			expectedErrCode: -32000,
			expectedErrMsg:  "proto: RelayRequest",
		},
		{
			name: "Missing session meta",

			relayerProxyBehavior: defaultBehavior,
			inputScenario:        sendRequestWithMissingMeta,

			expectedErrCode: -32000,
			expectedErrMsg:  "missing meta from relay request",
		},
		{
			name: "Missing signature",

			relayerProxyBehavior: defaultBehavior,
			inputScenario:        sendRequestWithMissingSignature,

			expectedErrCode: -32000,
			expectedErrMsg:  "missing signature from relay request",
		},
		{
			name: "Invalid ring signature",

			relayerProxyBehavior: defaultBehavior,
			inputScenario:        sendRequestWithInvalidRingSignature,

			expectedErrCode: -32000,
			expectedErrMsg:  "error deserializing ring signature",
		},
		{
			name: "Missing session header application address",

			relayerProxyBehavior: defaultBehavior,
			inputScenario:        sendRequestWithMissingSessionHeaderApplicationAddress,

			expectedErrCode: -32000,
			expectedErrMsg:  "missing application address from relay request",
		},
		{
			name: "Non staked application address",

			relayerProxyBehavior: defaultBehavior,
			inputScenario:        sendRequestWithNonStakedApplicationAddress,

			expectedErrCode: -32000,
			expectedErrMsg:  "error getting ring for application address",
		},
		{
			name: "Ring signature mismatch",

			relayerProxyBehavior: defaultBehavior,
			inputScenario:        sendRequestWithRingSignatureMismatch,

			expectedErrCode: -32000,
			expectedErrMsg:  "ring signature does not match ring for application address",
		},
		{
			name: "Invalid relay supplier",

			relayerProxyBehavior: []func(*testproxy.TestBehavior){
				testproxy.WithRelayerProxyMocks,
				testproxy.WithRelayerProxyDependencies,
				testproxy.WithKeyringDefaultBehavior,
				testproxy.WithSupplierDefaultBehavior,
				testproxy.WithRelayerProxiedServices,
				testproxy.WithProxiedServiceDefaultBehavior,
				testproxy.WithApplicationDefaultBehavior,
				testproxy.WithAccountsDefaultBehavior,
				testproxy.WithBlockClientDefaultBehavior,

				// The client requested a relay from us but we don't belong to its session
				testproxy.WithSessionSupplierMismatchBehavior,
			},
			inputScenario: sendRequestWithInvalidRelaySupplier,

			expectedErrCode: -32000,
			expectedErrMsg:  "invalid relayer proxy supplier",
		},
		{
			name: "Invalid signature",

			relayerProxyBehavior: defaultBehavior,
			inputScenario:        sendRequestWithInvalidSignature,

			expectedErrCode: -32000,
			expectedErrMsg:  "invalid ring signature",
		},
		{
			name: "Successful relay",

			relayerProxyBehavior: defaultBehavior,
			inputScenario:        sendRequestWithSuccessfulReply,

			expectedErrCode: 0,
			expectedErrMsg:  "",
		},
	}

	for _, tt := range tests {
		ctx := context.TODO()
		ctx, cancel := context.WithCancel(ctx)
		ctrl := gomock.NewController(t)

		test := testproxy.NewRelayerProxyTestBehavior(
			ctx,
			t,
			ctrl,
			relayerProxyConfig,
			tt.relayerProxyBehavior...,
		)

		rp, err := proxy.NewRelayerProxy(
			test.Deps,
			proxy.WithSigningKeyName(relayerProxyConfig.SupplierKeyName),
			proxy.WithProxiedServicesEndpoints(test.ProxiedServicesEndpoints),
		)
		require.NoError(t, err)

		go rp.Start(ctx)
		time.Sleep(100 * time.Millisecond)

		errCode, errMsg := tt.inputScenario(t, test)
		require.Equal(t, tt.expectedErrCode, errCode)
		require.True(t, strings.HasPrefix(errMsg, tt.expectedErrMsg))

		ctrl.Finish()
		cancel()
	}
}

func sendRequestWithUnparsableBody(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errorCode int32, errorMessage string) {
	// Send non JSONRpc payload
	reader := io.NopCloser(bytes.NewReader([]byte("invalid request")))

	res, err := http.DefaultClient.Post(
		test.ProvidedServices["service1"].Url,
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
		Payload: &servicetypes.RelayRequest_JsonRpcPayload{
			JsonRpcPayload: testproxy.ValidPayload,
		},
	}

	return testproxy.MarshalAndSend(test, req)
}

func sendRequestWithMissingSignature(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errorCode int32, errorMessage string) {
	req := &servicetypes.RelayRequest{
		Meta: &servicetypes.RelayRequestMetadata{
			// RelayRequest metadata is missing the signature
			SessionHeader: &sessiontypes.SessionHeader{},
		},
		Payload: &servicetypes.RelayRequest_JsonRpcPayload{
			JsonRpcPayload: testproxy.ValidPayload,
		},
	}

	return testproxy.MarshalAndSend(test, req)
}

func sendRequestWithInvalidRingSignature(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errorCode int32, errorMessage string) {
	req := &servicetypes.RelayRequest{
		Meta: &servicetypes.RelayRequestMetadata{
			SessionHeader: &sessiontypes.SessionHeader{},
			// RelayRequest metadata has an invalid signature
			Signature: []byte("invalidSignature"),
		},
		Payload: &servicetypes.RelayRequest_JsonRpcPayload{
			JsonRpcPayload: testproxy.ValidPayload,
		},
	}

	return testproxy.MarshalAndSend(test, req)
}

func sendRequestWithMissingSessionHeaderApplicationAddress(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errCode int32, errorMessage string) {
	req := &servicetypes.RelayRequest{
		Meta: &servicetypes.RelayRequestMetadata{
			SessionHeader: &sessiontypes.SessionHeader{
				// RelayRequest session header is missing the application address
			},
		},
		Payload: &servicetypes.RelayRequest_JsonRpcPayload{
			JsonRpcPayload: testproxy.ValidPayload,
		},
	}

	// Assign a valid but random ring signature so that the request is not rejected
	// before looking at the application address
	randomPrivKey := secp256k1.GenPrivKey()
	req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, randomPrivKey)

	return testproxy.MarshalAndSend(test, req)
}

func sendRequestWithNonStakedApplicationAddress(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errCode int32, errorMessage string) {
	randomPrivKey := secp256k1.GenPrivKey()

	req := &servicetypes.RelayRequest{
		Meta: &servicetypes.RelayRequestMetadata{
			SessionHeader: &sessiontypes.SessionHeader{
				// The key used to sign the request is not staked
				ApplicationAddress: testproxy.GetAddressFromPrivateKey(
					test,
					randomPrivKey,
				),
			},
		},
		Payload: &servicetypes.RelayRequest_JsonRpcPayload{
			JsonRpcPayload: testproxy.ValidPayload,
		},
	}
	// Have a legit signature from the non staked key
	req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, randomPrivKey)

	return testproxy.MarshalAndSend(test, req)
}

func sendRequestWithRingSignatureMismatch(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errCode int32, errorMessage string) {
	req := &servicetypes.RelayRequest{
		Meta: &servicetypes.RelayRequestMetadata{
			SessionHeader: &sessiontypes.SessionHeader{
				ApplicationAddress: testproxy.GetAddressFromPrivateKey(
					test,
					test.ApplicationPrivateKey,
				),
			},
		},
		Payload: &servicetypes.RelayRequest_JsonRpcPayload{
			JsonRpcPayload: testproxy.ValidPayload,
		},
	}
	// The signature is valid but does not match the ring for the application address
	randomPrivKey := secp256k1.GenPrivKey()
	req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, randomPrivKey)

	return testproxy.MarshalAndSend(test, req)
}

func sendRequestWithInvalidRelaySupplier(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errCode int32, errorMessage string) {
	// The RelayRequest is correctly formatted but the supplier does not belong to the session
	req := &servicetypes.RelayRequest{
		Meta: &servicetypes.RelayRequestMetadata{
			SessionHeader: &sessiontypes.SessionHeader{
				ApplicationAddress: testproxy.GetAddressFromPrivateKey(
					test,
					test.ApplicationPrivateKey,
				),
			},
		},
		Payload: &servicetypes.RelayRequest_JsonRpcPayload{
			JsonRpcPayload: testproxy.ValidPayload,
		},
	}
	req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, test.ApplicationPrivateKey)

	return testproxy.MarshalAndSend(test, req)
}

func sendRequestWithInvalidSignature(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errCode int32, errorMessage string) {
	jsonRpcPayload := &servicetypes.JSONRPCRequestPayload{
		Method:  "someMethod",
		Id:      1,
		Jsonrpc: "2.0",
		Params:  []string{"someParam"},
	}

	req := &servicetypes.RelayRequest{
		Meta: &servicetypes.RelayRequestMetadata{
			SessionHeader: &sessiontypes.SessionHeader{
				ApplicationAddress: testproxy.GetAddressFromPrivateKey(
					test,
					test.ApplicationPrivateKey,
				),
			},
		},
		Payload: &servicetypes.RelayRequest_JsonRpcPayload{
			JsonRpcPayload: jsonRpcPayload,
		},
	}
	req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, test.ApplicationPrivateKey)

	// Alter the reuqest payload so the hash doesn't match the signature
	jsonRpcPayload.Params = []string{"alteredParam"}

	return testproxy.MarshalAndSend(test, req)
}

func sendRequestWithSuccessfulReply(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errCode int32, errorMessage string) {
	req := &servicetypes.RelayRequest{
		Meta: &servicetypes.RelayRequestMetadata{
			SessionHeader: &sessiontypes.SessionHeader{
				ApplicationAddress: testproxy.GetAddressFromPrivateKey(
					test,
					test.ApplicationPrivateKey,
				),
				Service:   &sharedtypes.Service{Id: "service1"},
				SessionId: "",
			},
		},
		Payload: &servicetypes.RelayRequest_JsonRpcPayload{
			JsonRpcPayload: testproxy.ValidPayload,
		},
	}
	req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, test.ApplicationPrivateKey)

	return testproxy.MarshalAndSend(test, req)
}
