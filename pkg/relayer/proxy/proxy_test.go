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

var defaultConfig = testproxy.RelayerProxyConfig{
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

func TestRelayerProxy_StartAndStop(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	test := testproxy.NewRelayerProxyTestBehavior(ctx, t, ctrl, defaultConfig, defaultBehavior...)

	rp, err := proxy.NewRelayerProxy(
		test.Deps,
		proxy.WithSigningKeyName(test.SupplierKeyName),
		proxy.WithProxiedServicesEndpoints(test.ProxiedServicesEndpoints),
	)
	require.NoError(t, err)

	go rp.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	res1, err := http.DefaultClient.Get(test.ProvidedServices["service1"].Url)
	require.NoError(t, err)
	require.NotNil(t, res1)

	res2, err := http.DefaultClient.Get(test.ProvidedServices["service2"].Url)
	require.NoError(t, err)
	require.NotNil(t, res2)

	err = rp.Stop(ctx)
	require.NoError(t, err)
}

func TestRelayerProxy_InvalidSupplierKeyName(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	test := testproxy.NewRelayerProxyTestBehavior(ctx, t, ctrl, defaultConfig, defaultBehavior...)

	rp, err := proxy.NewRelayerProxy(
		test.Deps,
		proxy.WithSigningKeyName("wrongKeyName"),
		proxy.WithProxiedServicesEndpoints(test.ProxiedServicesEndpoints),
	)
	require.NoError(t, err)

	err = rp.Start(ctx)
	require.Error(t, err)
}

func TestRelayerProxy_MissingSupplierKeyName(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	test := testproxy.NewRelayerProxyTestBehavior(ctx, t, ctrl, defaultConfig, defaultBehavior...)

	_, err := proxy.NewRelayerProxy(
		test.Deps,
		proxy.WithSigningKeyName(""),
		proxy.WithProxiedServicesEndpoints(test.ProxiedServicesEndpoints),
	)
	require.Error(t, err)
}

func TestRelayerProxy_NoProxiedServices(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := testproxy.RelayerProxyConfig{
		SupplierKeyName:       "supplierKeyName",
		ProxiedServicesConfig: nil,
		ProvidedServices:      defaultConfig.ProvidedServices,
	}

	test := testproxy.NewRelayerProxyTestBehavior(ctx, t, ctrl, cfg, defaultBehavior...)

	_, err := proxy.NewRelayerProxy(
		test.Deps,
		proxy.WithSigningKeyName(cfg.SupplierKeyName),
		proxy.WithProxiedServicesEndpoints(test.ProxiedServicesEndpoints),
	)
	require.Error(t, err)
}

func TestRelayerProxy_UnsupportedRpcType(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := testproxy.RelayerProxyConfig{
		SupplierKeyName:       "supplierKeyName",
		ProxiedServicesConfig: defaultConfig.ProxiedServicesConfig,
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
func TestRelayerProxy_Relays(t *testing.T) {
	tests := []struct {
		name                 string
		relayerProxyBehavior []func(*testproxy.TestBehavior)
		inputScenario        func(t *testing.T, test *testproxy.TestBehavior) (errCode int32, errMsg string)

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
			name: "Invalid session header application address",

			relayerProxyBehavior: defaultBehavior,
			inputScenario:        sendRequestWithNonStakedApplicationAddress,

			expectedErrCode: -32000,
			expectedErrMsg:  "error getting ring for application address",
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

		test := testproxy.NewRelayerProxyTestBehavior(ctx, t, ctrl, defaultConfig, tt.relayerProxyBehavior...)

		rp, err := proxy.NewRelayerProxy(
			test.Deps,
			proxy.WithSigningKeyName(defaultConfig.SupplierKeyName),
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
	reader := io.NopCloser(bytes.NewReader([]byte("invalid request")))

	res, err := http.DefaultClient.Post(test.ProvidedServices["service1"].Url, "application/json", reader)
	require.NoError(t, err)
	require.NotNil(t, res)

	return testproxy.GetRelayResponseError(t, res)
}

func sendRequestWithMissingMeta(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errorCode int32, errorMessage string) {

	req := &servicetypes.RelayRequest{
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
			Signature:     []byte("invalidSignature"),
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
			SessionHeader: &sessiontypes.SessionHeader{},
		},
		Payload: &servicetypes.RelayRequest_JsonRpcPayload{
			JsonRpcPayload: testproxy.ValidPayload,
		},
	}

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
				ApplicationAddress: testproxy.GetAddressFromPrivateKey(test, randomPrivKey),
			},
		},
		Payload: &servicetypes.RelayRequest_JsonRpcPayload{
			JsonRpcPayload: testproxy.ValidPayload,
		},
	}
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
				ApplicationAddress: testproxy.GetAddressFromPrivateKey(test, test.ApplicationPrivateKey),
			},
		},
		Payload: &servicetypes.RelayRequest_JsonRpcPayload{
			JsonRpcPayload: testproxy.ValidPayload,
		},
	}
	randomPrivKey := secp256k1.GenPrivKey()
	req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, randomPrivKey)

	return testproxy.MarshalAndSend(test, req)
}

func sendRequestWithInvalidRelaySupplier(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errCode int32, errorMessage string) {
	req := &servicetypes.RelayRequest{
		Meta: &servicetypes.RelayRequestMetadata{
			SessionHeader: &sessiontypes.SessionHeader{
				ApplicationAddress: testproxy.GetAddressFromPrivateKey(test, test.ApplicationPrivateKey),
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
				ApplicationAddress: testproxy.GetAddressFromPrivateKey(test, test.ApplicationPrivateKey),
			},
		},
		Payload: &servicetypes.RelayRequest_JsonRpcPayload{
			JsonRpcPayload: jsonRpcPayload,
		},
	}
	req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, test.ApplicationPrivateKey)
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
				ApplicationAddress: testproxy.GetAddressFromPrivateKey(test, test.ApplicationPrivateKey),
				Service:            &sharedtypes.Service{Id: "service1"},
				SessionId:          "",
			},
		},
		Payload: &servicetypes.RelayRequest_JsonRpcPayload{
			JsonRpcPayload: testproxy.ValidPayload,
		},
	}
	req.Meta.Signature = testproxy.GetApplicationRingSignature(t, req, test.ApplicationPrivateKey)

	return testproxy.MarshalAndSend(test, req)
}
