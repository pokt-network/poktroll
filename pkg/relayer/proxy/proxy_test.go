package proxy_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

var (
	supplierKeyName = "supplierKeyName"
	appPrivateKey   = secp256k1.GenPrivKey()
	relayerProxyUrl = "http://127.0.0.1:8545/"
	proxiedServices = map[string]*url.URL{
		"service1": {Scheme: "http", Host: "localhost:8180", Path: "/"},
		"service2": {Scheme: "http", Host: "localhost:8181", Path: "/"},
	}
)

var defaultBehavior = []func(*testproxy.TestBehavior){
	testproxy.WithRelayerProxyDependencies(supplierKeyName),
	testproxy.WithRelayerProxiedServices(proxiedServices),
	testproxy.WithDefaultActors(supplierKeyName, appPrivateKey),
	testproxy.WithDefaultSessionSupplier(supplierKeyName, appPrivateKey),
}

// RelayerProxy should start and stop without errors
func TestRelayerProxy_StartAndStop(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Setup the RelayerProxy instrumented behavior
	test := testproxy.NewRelayerProxyTestBehavior(ctx, t, defaultBehavior...)

	// Create a RelayerProxy
	rp, err := proxy.NewRelayerProxy(
		test.Deps,
		proxy.WithSigningKeyName(supplierKeyName),
		proxy.WithProxiedServicesEndpoints(proxiedServices),
	)
	require.NoError(t, err)

	// Start RelayerProxy
	go rp.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	// Test that RelayerProxy is handling requests, don't care about the response here
	res, err := http.DefaultClient.Get(relayerProxyUrl)
	require.NoError(t, err)
	require.NotNil(t, res)

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

	test := testproxy.NewRelayerProxyTestBehavior(ctx, t, defaultBehavior...)

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
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	test := testproxy.NewRelayerProxyTestBehavior(ctx, t, defaultBehavior...)

	_, err := proxy.NewRelayerProxy(
		test.Deps,
		proxy.WithSigningKeyName(""),
		proxy.WithProxiedServicesEndpoints(proxiedServices),
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

	test := testproxy.NewRelayerProxyTestBehavior(ctx, t, defaultBehavior...)

	_, err := proxy.NewRelayerProxy(
		test.Deps,
		proxy.WithSigningKeyName(supplierKeyName),
		proxy.WithProxiedServicesEndpoints(make(map[string]*url.URL)),
	)
	require.Error(t, err)
}

// RelayerProxy should fail to start if it cannot spawn a server for the
// services it advertized on-chain
// TODO_TECHDEBT: Re-enable this test once the RelayerProxy is building its
// servers from the provided config file
//func TestRelayerProxy_UnsupportedRpcType(t *testing.T) {
//	ctx := context.Background()
//	ctx, cancel := context.WithCancel(ctx)
//	defer cancel()
//	ctrl := gomock.NewController(t)
//	defer ctrl.Finish()
//
//	//cfg := testproxy.RelayerProxyConfig{
//	//	SupplierKeyName: supplierKeyName,
//	//	// Supplier has advertised providing a GRPC service but does not support it
//	//	ProvidedServices: map[string]testproxy.ProvidedServiceConfig{
//	//		"service1": {Url: "http://localhost:8180", RpcType: sharedtypes.RPCType_GRPC},
//	//	},
//	//}
//
//	test := testproxy.NewRelayerProxyTestBehavior(ctx, t, defaultBehavior...)
//
//	rp, err := proxy.NewRelayerProxy(
//		test.Deps,
//		proxy.WithSigningKeyName(supplierKeyName),
//		proxy.WithProxiedServicesEndpoints(proxiedServices),
//	)
//	require.NoError(t, err)
//
//	err = rp.Start(ctx)
//	require.Error(t, err)
//}

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

			expectedErrCode: 0,
			expectedErrMsg:  "cannot unmarshal response payload",
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
			name: "Session mismatch",

			relayerProxyBehavior: defaultBehavior,
			inputScenario:        sendRequestWithDifferentSession,

			expectedErrCode: -32000,
			expectedErrMsg:  "session mismatch",
		},
		{
			name: "Invalid relay supplier",

			relayerProxyBehavior: []func(*testproxy.TestBehavior){
				testproxy.WithRelayerProxyDependencies(supplierKeyName),
				testproxy.WithRelayerProxiedServices(proxiedServices),
				testproxy.WithDefaultActors(supplierKeyName, appPrivateKey),
				// Missing session supplier
				testproxy.WithDefaultSessionSupplier("", appPrivateKey),
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

	ctx := context.TODO()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(ctx)

			test := testproxy.NewRelayerProxyTestBehavior(ctx, t, tt.relayerProxyBehavior...)

			rp, err := proxy.NewRelayerProxy(
				test.Deps,
				proxy.WithSigningKeyName(supplierKeyName),
				proxy.WithProxiedServicesEndpoints(proxiedServices),
			)
			require.NoError(t, err)

			go rp.Start(ctx)
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
	// Send non JSONRpc payload
	reader := io.NopCloser(bytes.NewReader([]byte("invalid request")))

	res, err := http.DefaultClient.Post(relayerProxyUrl, "application/json", reader)
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
		Payload: testproxy.ValidPayload,
	}

	return testproxy.MarshalAndSend(test, relayerProxyUrl, req)
}

func sendRequestWithMissingSignature(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errorCode int32, errorMessage string) {
	appAddress := testproxy.GetAddressFromPrivateKey(test, appPrivateKey)
	sessionId := sha256.Sum256([]byte(fmt.Sprintf("%s-%s-%d", appAddress, "service1", 1)))

	req := &servicetypes.RelayRequest{
		Meta: &servicetypes.RelayRequestMetadata{
			// RelayRequest metadata is missing the signature
			SessionHeader: &sessiontypes.SessionHeader{
				ApplicationAddress: appAddress,
				SessionId:          string(sessionId[:]),
				Service:            &sharedtypes.Service{Id: "service1"},
			},
		},
		Payload: testproxy.ValidPayload,
	}

	return testproxy.MarshalAndSend(test, relayerProxyUrl, req)
}

func sendRequestWithInvalidRingSignature(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errorCode int32, errorMessage string) {
	appAddress := testproxy.GetAddressFromPrivateKey(test, appPrivateKey)
	sessionId := sha256.Sum256([]byte(fmt.Sprintf("%s-%s-%d", appAddress, "service1", 1)))

	req := &servicetypes.RelayRequest{
		Meta: &servicetypes.RelayRequestMetadata{
			SessionHeader: &sessiontypes.SessionHeader{
				ApplicationAddress: appAddress,
				SessionId:          string(sessionId[:]),
				Service:            &sharedtypes.Service{Id: "service1"},
			},
			// RelayRequest metadata has an invalid signature
			Signature: []byte("invalidSignature"),
		},
		Payload: testproxy.ValidPayload,
	}

	return testproxy.MarshalAndSend(test, relayerProxyUrl, req)
}

func sendRequestWithMissingSessionHeaderApplicationAddress(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errCode int32, errorMessage string) {
	appAddress := testproxy.GetAddressFromPrivateKey(test, appPrivateKey)
	sessionId := sha256.Sum256([]byte(fmt.Sprintf("%s-%s-%d", appAddress, "service1", 1)))

	req := &servicetypes.RelayRequest{
		Meta: &servicetypes.RelayRequestMetadata{
			SessionHeader: &sessiontypes.SessionHeader{
				// RelayRequest session header is missing the application address
				SessionId: string(sessionId[:]),
				Service:   &sharedtypes.Service{Id: "service1"},
			},
		},
		Payload: testproxy.ValidPayload,
	}

	// Assign a valid but random ring signature so that the request is not rejected
	// before looking at the application address
	randomPrivKey := secp256k1.GenPrivKey()
	req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, randomPrivKey)

	return testproxy.MarshalAndSend(test, relayerProxyUrl, req)
}

func sendRequestWithNonStakedApplicationAddress(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errCode int32, errorMessage string) {
	randomPrivKey := secp256k1.GenPrivKey()
	appAddress := testproxy.GetAddressFromPrivateKey(test, randomPrivKey)
	sessionId := sha256.Sum256([]byte(fmt.Sprintf("%s-%s-%d", appAddress, "service1", 1)))

	req := &servicetypes.RelayRequest{
		Meta: &servicetypes.RelayRequestMetadata{
			SessionHeader: &sessiontypes.SessionHeader{
				// The key used to sign the request is not staked
				ApplicationAddress: testproxy.GetAddressFromPrivateKey(test, randomPrivKey),
				SessionId:          string(sessionId[:]),
				Service:            &sharedtypes.Service{Id: "service1"},
			},
		},
		Payload: testproxy.ValidPayload,
	}
	// Have a legit signature from the non staked key
	req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, randomPrivKey)

	return testproxy.MarshalAndSend(test, relayerProxyUrl, req)
}

func sendRequestWithRingSignatureMismatch(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errCode int32, errorMessage string) {
	appAddress := testproxy.GetAddressFromPrivateKey(test, appPrivateKey)
	sessionId := sha256.Sum256([]byte(fmt.Sprintf("%s-%s-%d", appAddress, "service1", 1)))

	req := &servicetypes.RelayRequest{
		Meta: &servicetypes.RelayRequestMetadata{
			SessionHeader: &sessiontypes.SessionHeader{
				ApplicationAddress: testproxy.GetAddressFromPrivateKey(test, appPrivateKey),
				SessionId:          string(sessionId[:]),
				Service:            &sharedtypes.Service{Id: "service1"},
			},
		},
		Payload: testproxy.ValidPayload,
	}
	// The signature is valid but does not match the ring for the application address
	randomPrivKey := secp256k1.GenPrivKey()
	req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, randomPrivKey)

	return testproxy.MarshalAndSend(test, relayerProxyUrl, req)
}

func sendRequestWithDifferentSession(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errCode int32, errorMessage string) {
	appAddress := testproxy.GetAddressFromPrivateKey(test, appPrivateKey)
	sessionId := sha256.Sum256([]byte(fmt.Sprintf("%s-%s-%d", appAddress, "service2", 1)))

	// The RelayRequest is correctly formatted but the supplier does not belong to the session
	req := &servicetypes.RelayRequest{
		Meta: &servicetypes.RelayRequestMetadata{
			SessionHeader: &sessiontypes.SessionHeader{
				ApplicationAddress: appAddress,
				SessionId:          string(sessionId[:]),
				Service:            &sharedtypes.Service{Id: "service1"},
			},
		},
		Payload: testproxy.ValidPayload,
	}
	req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, appPrivateKey)

	return testproxy.MarshalAndSend(test, relayerProxyUrl, req)
}

func sendRequestWithInvalidRelaySupplier(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errCode int32, errorMessage string) {
	appAddress := testproxy.GetAddressFromPrivateKey(test, appPrivateKey)
	sessionId := sha256.Sum256([]byte(fmt.Sprintf("%s-%s-%d", appAddress, "service1", 1)))

	// The RelayRequest is correctly formatted but the supplier does not belong to the session
	req := &servicetypes.RelayRequest{
		Meta: &servicetypes.RelayRequestMetadata{
			SessionHeader: &sessiontypes.SessionHeader{
				ApplicationAddress: appAddress,
				SessionId:          string(sessionId[:]),
				Service:            &sharedtypes.Service{Id: "service1"},
			},
		},
		Payload: testproxy.ValidPayload,
	}
	req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, appPrivateKey)

	return testproxy.MarshalAndSend(test, relayerProxyUrl, req)
}

func sendRequestWithInvalidSignature(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errCode int32, errorMessage string) {
	appAddress := testproxy.GetAddressFromPrivateKey(test, appPrivateKey)
	sessionId := sha256.Sum256([]byte(fmt.Sprintf("%s-%s-%d", appAddress, "service1", 1)))
	jsonRpcPayload := []byte(`{"method":"someMethod","id":1,"jsonrpc":"2.0","params":["someParam"]}`)

	req := &servicetypes.RelayRequest{
		Meta: &servicetypes.RelayRequestMetadata{
			SessionHeader: &sessiontypes.SessionHeader{
				ApplicationAddress: testproxy.GetAddressFromPrivateKey(test, appPrivateKey),
				Service:            &sharedtypes.Service{Id: "service1"},
				SessionId:          string(sessionId[:]),
			},
		},
		Payload: jsonRpcPayload,
	}
	req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, appPrivateKey)

	// Alter the reuqest payload so the hash doesn't match the signature
	req.Payload = []byte(`{"method":"someMethod","id":1,"jsonrpc":"2.0","params":["alteredParam"]}`)

	return testproxy.MarshalAndSend(test, relayerProxyUrl, req)
}

func sendRequestWithSuccessfulReply(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errCode int32, errorMessage string) {
	appAddress := testproxy.GetAddressFromPrivateKey(test, appPrivateKey)
	sessionId := sha256.Sum256([]byte(fmt.Sprintf("%s-%s-%d", appAddress, "service1", 1)))

	req := &servicetypes.RelayRequest{
		Meta: &servicetypes.RelayRequestMetadata{
			SessionHeader: &sessiontypes.SessionHeader{
				ApplicationAddress: testproxy.GetAddressFromPrivateKey(test, appPrivateKey),
				Service:            &sharedtypes.Service{Id: "service1"},
				SessionId:          string(sessionId[:]),
			},
		},
		Payload: testproxy.ValidPayload,
	}
	req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, appPrivateKey)

	return testproxy.MarshalAndSend(test, relayerProxyUrl, req)
}
