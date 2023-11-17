package proxy_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/relayer/proxy"
	"github.com/pokt-network/poktroll/testutil/testproxy"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
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

func TestRelayerProxy_InvalidRequest(t *testing.T) {
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
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// No metadata
	req := &servicetypes.RelayRequest{}
	cdc := servicetypes.ModuleCdc
	reqBz, err := cdc.MarshalJSON(req)
	require.NoError(t, err)
	reader := io.NopCloser(bytes.NewBuffer(reqBz))

	res, err := http.DefaultClient.Post(test.ProvidedServices["service1"].Url, "application/json", reader)
	require.NoError(t, err)
	require.NotNil(t, res)

	require.Equal(t, int32(-32000), getErrorCode(t, res))

	jsonRpcPayload := &servicetypes.JSONRPCRequestPayload{
		Method:  "someMethod",
		Id:      1,
		Jsonrpc: "2.0",
		Params:  []string{"someParam"},
	}

	req = &servicetypes.RelayRequest{
		Meta: &servicetypes.RelayRequestMetadata{},
		Payload: &servicetypes.RelayRequest_JsonRpcPayload{
			JsonRpcPayload: jsonRpcPayload,
		},
	}
	reqBz, err = cdc.MarshalJSON(req)
	require.NoError(t, err)
	reader = io.NopCloser(bytes.NewBuffer(reqBz))

	res, err = http.DefaultClient.Post(test.ProvidedServices["service1"].Url, "application/json", reader)
	require.NoError(t, err)
	require.NotNil(t, res)

	require.Equal(t, int32(-32000), getErrorCode(t, res))

	err = rp.Stop(ctx)
	require.NoError(t, err)
}

func getErrorCode(t *testing.T, res *http.Response) int32 {
	responseBody, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	relayResponse := &servicetypes.RelayResponse{}
	err = relayResponse.Unmarshal(responseBody)
	require.NoError(t, err)

	return relayResponse.Payload.(*servicetypes.RelayResponse_JsonRpcPayload).JsonRpcPayload.Error.Code
}
