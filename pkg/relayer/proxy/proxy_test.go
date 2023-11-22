package proxy_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	ring_secp256k1 "github.com/athanorlabs/go-dleq/secp256k1"
	ringtypes "github.com/athanorlabs/go-dleq/types"
	"github.com/cometbft/cometbft/crypto"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/golang/mock/gomock"
	"github.com/noot/ring-go"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/relayer/proxy"
	"github.com/pokt-network/poktroll/pkg/signer"
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

func TestRelayerProxy_UnparsableRequest(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	test := testproxy.NewRelayerProxyTestBehavior(ctx, t, ctrl, defaultConfig, defaultBehavior...)

	rp, err := proxy.NewRelayerProxy(
		test.Deps,
		proxy.WithSigningKeyName(defaultConfig.SupplierKeyName),
		proxy.WithProxiedServicesEndpoints(test.ProxiedServicesEndpoints),
	)
	require.NoError(t, err)

	go rp.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	errCode, errMsg := sendRequestWithUnparsableBody(t, test)
	require.Equal(t, int32(-32000), errCode)
	require.True(t, strings.HasPrefix(errMsg, "proto: RelayRequest"))
}

func TestRelayerProxy_MissingSessionMeta(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	test := testproxy.NewRelayerProxyTestBehavior(ctx, t, ctrl, defaultConfig, defaultBehavior...)

	rp, err := proxy.NewRelayerProxy(
		test.Deps,
		proxy.WithSigningKeyName(defaultConfig.SupplierKeyName),
		proxy.WithProxiedServicesEndpoints(test.ProxiedServicesEndpoints),
	)
	require.NoError(t, err)

	go rp.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	errCode, errMsg := sendRequestWithMissingMeta(t, test)
	require.Equal(t, int32(-32000), errCode)
	require.True(t, strings.HasPrefix(errMsg, "missing meta from relay request"))
}

func TestRelayerProxy_MissingSignature(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	test := testproxy.NewRelayerProxyTestBehavior(ctx, t, ctrl, defaultConfig, defaultBehavior...)

	rp, err := proxy.NewRelayerProxy(
		test.Deps,
		proxy.WithSigningKeyName(defaultConfig.SupplierKeyName),
		proxy.WithProxiedServicesEndpoints(test.ProxiedServicesEndpoints),
	)
	require.NoError(t, err)

	go rp.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	errCode, errMsg := sendRequestWithMissingSignature(t, test)
	require.Equal(t, int32(-32000), errCode)
	require.True(t, strings.HasPrefix(errMsg, "missing signature from relay request"))
}

func TestRelayerProxy_InvalidRingSignature(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	test := testproxy.NewRelayerProxyTestBehavior(ctx, t, ctrl, defaultConfig, defaultBehavior...)

	rp, err := proxy.NewRelayerProxy(
		test.Deps,
		proxy.WithSigningKeyName(defaultConfig.SupplierKeyName),
		proxy.WithProxiedServicesEndpoints(test.ProxiedServicesEndpoints),
	)
	require.NoError(t, err)

	go rp.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	errCode, errMsg := sendRequestWithInvalidRingSignature(t, test)
	require.Equal(t, int32(-32000), errCode)
	require.True(t, strings.HasPrefix(errMsg, "error deserializing ring signature"))
}

func TestRelayerProxy_MissingSessionHeaderApplicationAddress(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	test := testproxy.NewRelayerProxyTestBehavior(ctx, t, ctrl, defaultConfig, defaultBehavior...)

	rp, err := proxy.NewRelayerProxy(
		test.Deps,
		proxy.WithSigningKeyName(defaultConfig.SupplierKeyName),
		proxy.WithProxiedServicesEndpoints(test.ProxiedServicesEndpoints),
	)
	require.NoError(t, err)

	go rp.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	errCode, errMsg := sendRequestWithMissingSessionHeaderApplicationAddress(t, test)
	require.Equal(t, int32(-32000), errCode)
	require.True(t, strings.HasPrefix(errMsg, "missing application address from relay request"))
}

func TestRelayerProxy_InvalidSessionHeaderApplicationAddress(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	test := testproxy.NewRelayerProxyTestBehavior(ctx, t, ctrl, defaultConfig, defaultBehavior...)

	rp, err := proxy.NewRelayerProxy(
		test.Deps,
		proxy.WithSigningKeyName(defaultConfig.SupplierKeyName),
		proxy.WithProxiedServicesEndpoints(test.ProxiedServicesEndpoints),
	)
	require.NoError(t, err)

	go rp.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	errCode, errMsg := sendRequestWithMissingSessionHeaderApplicationAddress(t, test)
	require.Equal(t, int32(-32000), errCode)
	require.True(t, strings.HasPrefix(errMsg, "missing application address from relay request"))
}

func TestRelayerProxy_NonStakedApplicationAddress(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	test := testproxy.NewRelayerProxyTestBehavior(ctx, t, ctrl, defaultConfig, defaultBehavior...)

	rp, err := proxy.NewRelayerProxy(
		test.Deps,
		proxy.WithSigningKeyName(defaultConfig.SupplierKeyName),
		proxy.WithProxiedServicesEndpoints(test.ProxiedServicesEndpoints),
	)
	require.NoError(t, err)

	go rp.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	errCode, errMsg := sendRequestWithNonStakedApplicationAddress(t, test)
	require.Equal(t, int32(-32000), errCode)
	require.True(t, strings.HasPrefix(errMsg, "error getting ring for application address"))
}

func TestRelayerProxy_RingSignatureMismatch(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	test := testproxy.NewRelayerProxyTestBehavior(ctx, t, ctrl, defaultConfig, defaultBehavior...)

	rp, err := proxy.NewRelayerProxy(
		test.Deps,
		proxy.WithSigningKeyName(defaultConfig.SupplierKeyName),
		proxy.WithProxiedServicesEndpoints(test.ProxiedServicesEndpoints),
	)
	require.NoError(t, err)

	go rp.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	errCode, errMsg := sendRequestWithRingSignatureMismatch(t, test)
	require.Equal(t, int32(-32000), errCode)
	require.True(t, strings.HasPrefix(errMsg, "ring signature does not match ring for application address"))
}

//func TestRelayerProxy_InvalidSignature(t *testing.T) {
//	ctx := context.Background()
//	ctx, cancel := context.WithCancel(ctx)
//	defer cancel()
//	ctrl := gomock.NewController(t)
//	defer ctrl.Finish()
//
//	test := testproxy.NewRelayerProxyTestBehavior(ctx, t, ctrl, defaultConfig, defaultBehavior...)
//
//	rp, err := proxy.NewRelayerProxy(
//		test.Deps,
//		proxy.WithSigningKeyName(defaultConfig.SupplierKeyName),
//		proxy.WithProxiedServicesEndpoints(test.ProxiedServicesEndpoints),
//	)
//	require.NoError(t, err)
//
//	go rp.Start(ctx)
//	time.Sleep(100 * time.Millisecond)
//
//	errCode, errMsg := sendRequestWithInvalidSignature(t, test)
//	require.Equal(t, int32(-32000), errCode)
//	require.True(t, strings.HasPrefix(errMsg, "invalid ring signature"))
//}

func extractError(t *testing.T, res *http.Response) (errorCode int32, errorMessage string) {
	responseBody, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	relayResponse := &servicetypes.RelayResponse{}
	err = relayResponse.Unmarshal(responseBody)
	require.NoError(t, err)

	payload := relayResponse.Payload.(*servicetypes.RelayResponse_JsonRpcPayload).JsonRpcPayload

	return payload.Error.Code, payload.Error.Message
}

func sendRequestWithUnparsableBody(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errorCode int32, errorMessage string) {
	reader := io.NopCloser(bytes.NewReader([]byte("invalid request")))

	res, err := http.DefaultClient.Post(test.ProvidedServices["service1"].Url, "application/json", reader)
	require.NoError(t, err)
	require.NotNil(t, res)

	return extractError(t, res)
}

func sendRequestWithMissingMeta(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errorCode int32, errorMessage string) {
	jsonRpcPayload := &servicetypes.JSONRPCRequestPayload{
		Method:  "someMethod",
		Id:      1,
		Jsonrpc: "2.0",
		Params:  []string{"someParam"},
	}

	req := &servicetypes.RelayRequest{
		Payload: &servicetypes.RelayRequest_JsonRpcPayload{
			JsonRpcPayload: jsonRpcPayload,
		},
	}
	reqBz, err := req.Marshal()
	require.NoError(t, err)
	reader := io.NopCloser(bytes.NewReader(reqBz))

	res, err := http.DefaultClient.Post(test.ProvidedServices["service1"].Url, "application/json", reader)
	require.NoError(t, err)
	require.NotNil(t, res)

	return extractError(t, res)
}

func sendRequestWithMissingSignature(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errorCode int32, errorMessage string) {
	jsonRpcPayload := &servicetypes.JSONRPCRequestPayload{
		Method:  "someMethod",
		Id:      1,
		Jsonrpc: "2.0",
		Params:  []string{"someParam"},
	}

	req := &servicetypes.RelayRequest{
		Meta: &servicetypes.RelayRequestMetadata{
			SessionHeader: &sessiontypes.SessionHeader{},
		},
		Payload: &servicetypes.RelayRequest_JsonRpcPayload{
			JsonRpcPayload: jsonRpcPayload,
		},
	}
	reqBz, err := req.Marshal()
	require.NoError(t, err)
	reader := io.NopCloser(bytes.NewReader(reqBz))

	res, err := http.DefaultClient.Post(test.ProvidedServices["service1"].Url, "application/json", reader)
	require.NoError(t, err)
	require.NotNil(t, res)

	return extractError(t, res)
}

func sendRequestWithInvalidRingSignature(
	t *testing.T,
	test *testproxy.TestBehavior,
) (errorCode int32, errorMessage string) {
	jsonRpcPayload := &servicetypes.JSONRPCRequestPayload{
		Method:  "someMethod",
		Id:      1,
		Jsonrpc: "2.0",
		Params:  []string{"someParam"},
	}

	req := &servicetypes.RelayRequest{
		Meta: &servicetypes.RelayRequestMetadata{
			SessionHeader: &sessiontypes.SessionHeader{},
			Signature:     []byte("invalidSignature"),
		},
		Payload: &servicetypes.RelayRequest_JsonRpcPayload{
			JsonRpcPayload: jsonRpcPayload,
		},
	}
	reqBz, err := req.Marshal()
	require.NoError(t, err)
	reader := io.NopCloser(bytes.NewReader(reqBz))

	res, err := http.DefaultClient.Post(test.ProvidedServices["service1"].Url, "application/json", reader)
	require.NoError(t, err)
	require.NotNil(t, res)

	return extractError(t, res)
}

func sendRequestWithMissingSessionHeaderApplicationAddress(
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
			SessionHeader: &sessiontypes.SessionHeader{},
		},
		Payload: &servicetypes.RelayRequest_JsonRpcPayload{
			JsonRpcPayload: jsonRpcPayload,
		},
	}

	randomPrivKey := secp256k1.GenPrivKey()
	req.Meta.Signature = getApplicationRingSignature(t, test, req, randomPrivKey)

	reqBz, err := req.Marshal()
	require.NoError(t, err)

	reader := io.NopCloser(bytes.NewReader(reqBz))
	res, err := http.DefaultClient.Post(test.ProvidedServices["service1"].Url, "application/json", reader)
	require.NoError(t, err)
	require.NotNil(t, res)

	return extractError(t, res)
}

func sendRequestWithNonStakedApplicationAddress(
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
				ApplicationAddress: "invalidAddress",
			},
		},
		Payload: &servicetypes.RelayRequest_JsonRpcPayload{
			JsonRpcPayload: jsonRpcPayload,
		},
	}
	randomPrivKey := secp256k1.GenPrivKey()
	req.Meta.Signature = getApplicationRingSignature(t, test, req, randomPrivKey)

	reqBz, err := req.Marshal()
	require.NoError(t, err)

	reader := io.NopCloser(bytes.NewReader(reqBz))
	res, err := http.DefaultClient.Post(test.ProvidedServices["service1"].Url, "application/json", reader)
	require.NoError(t, err)
	require.NotNil(t, res)

	return extractError(t, res)
}

func sendRequestWithRingSignatureMismatch(
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
				ApplicationAddress: test.GetApplicationAddress(),
			},
		},
		Payload: &servicetypes.RelayRequest_JsonRpcPayload{
			JsonRpcPayload: jsonRpcPayload,
		},
	}
	randomPrivKey := secp256k1.GenPrivKey()
	req.Meta.Signature = getApplicationRingSignature(t, test, req, randomPrivKey)

	reqBz, err := req.Marshal()
	require.NoError(t, err)

	reader := io.NopCloser(bytes.NewReader(reqBz))
	res, err := http.DefaultClient.Post(test.ProvidedServices["service1"].Url, "application/json", reader)
	require.NoError(t, err)
	require.NotNil(t, res)

	return extractError(t, res)
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
				ApplicationAddress: test.GetApplicationAddress(),
			},
		},
		Payload: &servicetypes.RelayRequest_JsonRpcPayload{
			JsonRpcPayload: jsonRpcPayload,
		},
	}
	req.Meta.Signature = getApplicationRingSignature(t, test, req, test.ApplicationPrivateKey)

	reqBz, err := req.Marshal()
	require.NoError(t, err)

	reader := io.NopCloser(bytes.NewReader(reqBz))
	res, err := http.DefaultClient.Post(test.ProvidedServices["service1"].Url, "application/json", reader)
	require.NoError(t, err)
	require.NotNil(t, res)

	return extractError(t, res)
}

func getApplicationRingSignature(
	t *testing.T,
	test *testproxy.TestBehavior,
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
