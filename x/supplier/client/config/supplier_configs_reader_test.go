package config_test

import (
	"testing"

	sdkerrors "cosmossdk.io/errors"
	"github.com/gogo/status"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/yaml"
	"github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/client/config"
)

func Test_ParseSupplierConfigs(t *testing.T) {
	tests := []struct {
		desc     string
		err      *sdkerrors.Error
		expected []*types.SupplierServiceConfig
		config   string
	}{
		// Valid Configs
		{
			desc: "services_test: valid full service config",
			err:  nil,
			expected: []*types.SupplierServiceConfig{
				{
					Service: &types.Service{Id: "svc"},
					Endpoints: []*types.SupplierEndpoint{
						{
							Url:     "http://pokt.network:8081",
							RpcType: types.RPCType_JSON_RPC,
							Configs: []*types.ConfigOption{
								{
									Key:   types.ConfigOptions_TIMEOUT,
									Value: "10",
								},
							},
						},
					},
				},
			},
			config: `
				- service_id: svc
				  endpoints:
				  - url: http://pokt.network:8081
				    rpc_type: json_rpc
				    config:
				      timeout: 10
				`,
		},
		{
			desc: "services_test: valid service config without endpoint config",
			err:  nil,
			expected: []*types.SupplierServiceConfig{
				{
					Service: &types.Service{Id: "svc"},
					Endpoints: []*types.SupplierEndpoint{
						{
							Url:     "http://pokt.network:8081",
							RpcType: types.RPCType_JSON_RPC,
						},
					},
				},
			},
			config: `
				- service_id: svc
				  endpoints:
				  - url: http://pokt.network:8081
				    rpc_type: json_rpc
				`,
		},
		{
			desc: "services_test: valid service config with empty endpoint config",
			err:  nil,
			expected: []*types.SupplierServiceConfig{
				{
					Service: &types.Service{Id: "svc"},
					Endpoints: []*types.SupplierEndpoint{
						{
							Url:     "http://pokt.network:8081",
							RpcType: types.RPCType_JSON_RPC,
							Configs: []*types.ConfigOption{},
						},
					},
				},
			},
			config: `
				- service_id: svc
				  endpoints:
				  - url: http://pokt.network:8081
				    rpc_type: json_rpc
				    config:
				`,
		},
		{
			desc: "services_test: valid service config with multiple endpoints",
			err:  nil,
			expected: []*types.SupplierServiceConfig{
				{
					Service: &types.Service{Id: "svc"},
					Endpoints: []*types.SupplierEndpoint{
						{
							Url:     "http://pokt.network:8081",
							RpcType: types.RPCType_JSON_RPC,
							Configs: []*types.ConfigOption{
								{
									Key:   types.ConfigOptions_TIMEOUT,
									Value: "10",
								},
							},
						},
						{
							Url:     "http://pokt.network:8082",
							RpcType: types.RPCType_JSON_RPC,
							Configs: []*types.ConfigOption{
								{
									Key:   types.ConfigOptions_TIMEOUT,
									Value: "11",
								},
							},
						},
					},
				},
			},
			config: `
				- service_id: svc
				  endpoints:
				  - url: http://pokt.network:8081
				    rpc_type: json_rpc
				    config:
				      timeout: 10
				  - url: http://pokt.network:8082
				    rpc_type: json_rpc
				    config:
				      timeout: 11
				`,
		},
		{
			desc: "services_test: valid service config with multiple services",
			err:  nil,
			expected: []*types.SupplierServiceConfig{
				{
					Service: &types.Service{Id: "svc1"},
					Endpoints: []*types.SupplierEndpoint{
						{
							Url:     "http://pokt.network:8081",
							RpcType: types.RPCType_JSON_RPC,
							Configs: []*types.ConfigOption{
								{
									Key:   types.ConfigOptions_TIMEOUT,
									Value: "10",
								},
							},
						},
					},
				},
				{
					Service: &types.Service{Id: "svc2"},
					Endpoints: []*types.SupplierEndpoint{
						{
							Url:     "http://pokt.network:8081",
							RpcType: types.RPCType_JSON_RPC,
							Configs: []*types.ConfigOption{
								{
									Key:   types.ConfigOptions_TIMEOUT,
									Value: "10",
								},
							},
						},
					},
				},
			},
			config: `
				- service_id: svc1
				  endpoints:
				  - url: http://pokt.network:8081
				    rpc_type: json_rpc
				    config:
				      timeout: 10
				- service_id: svc2
				  endpoints:
				  - url: http://pokt.network:8081
				    rpc_type: json_rpc
				    config:
				      timeout: 10
				`,
		},
		// Invalid Configs
		{
			desc: "services_test: invalid service config without service ID",
			err:  config.ErrSupplierConfigUnmarshalYAML,
			config: `
				  endpoints:
				  - url: http://pokt.network:8081
				    rpc_type: json_rpc
				    config:
				      timeout: 10
				`,
		},
		{
			desc: "services_test: invalid service config with empty service ID",
			err:  config.ErrSupplierConfigInvalidServiceId,
			config: `
				- service_id:
				  endpoints:
				  - url: http://pokt.network:8081
				    rpc_type: json_rpc
				    config:
				      timeout: 10
				`,
		},
		{
			desc: "services_test: invalid service config without endpoints",
			err:  config.ErrSupplierConfigNoEndpoints,
			config: `
				- service_id: svc
			`,
		},
		{
			desc: "services_test: invalid service config with empty endpoints",
			err:  config.ErrSupplierConfigNoEndpoints,
			config: `
				- service_id: svc
				  endpoints:
				`,
		},
		{
			desc: "services_test: invalid service config with unknown endpoint config key",
			err:  config.ErrSupplierConfigInvalidEndpointConfig,
			config: `
				- service_id: svc
				  endpoints:
				  - url: http://pokt.network:8081
				    rpc_type: json_rpc
				    config:
				      somekey: 10
				`,
		},
		{
			desc: "services_test: invalid service config with unknown endpoint rpc type",
			err:  config.ErrSupplierConfigInvalidRPCType,
			config: `
				- service_id: svc
				  endpoints:
				  - url: http://pokt.network:8081
				    rpc_type: somerpc
				    config:
				      timeout: 10
				`,
		},
		{
			desc: "services_test: invalid service config with invalid endpoint url",
			err:  config.ErrSupplierConfigInvalidURL,
			config: `
				- service_id: svc
				  endpoints:
				  - url: ::invalid_url
				    rpc_type: json_rpc
				    config:
				      timeout: 10
				`,
		},
		{
			desc:   "services_test: invalid service config with empty content",
			err:    config.ErrSupplierConfigInvalidURL,
			config: ``,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			normalizedConfig := yaml.NormalizeYAMLIndentation(tt.config)
			supplierServiceConfig, err := config.ParseSupplierConfigs([]byte(normalizedConfig))

			if tt.err != nil {
				require.Error(t, err)
				require.Nil(t, supplierServiceConfig)
				stat, ok := status.FromError(tt.err)
				require.True(t, ok)
				require.Contains(t, stat.Message(), tt.err.Error())
				require.Nil(t, supplierServiceConfig)
				return
			}

			require.NoError(t, err)

			require.Equal(t, len(tt.expected), len(supplierServiceConfig))
			for i, expected := range tt.expected {

				require.Equal(t, expected.Service.Id, supplierServiceConfig[i].Service.Id)

				require.Equal(t, len(expected.Endpoints), len(supplierServiceConfig[i].Endpoints))
				for j, expectedEndpoint := range expected.Endpoints {

					require.Equal(t, expectedEndpoint.Url, supplierServiceConfig[i].Endpoints[j].Url)
					require.Equal(t, expectedEndpoint.RpcType, supplierServiceConfig[i].Endpoints[j].RpcType)

					require.Equal(t, len(expectedEndpoint.Configs), len(supplierServiceConfig[i].Endpoints[j].Configs))
					for k, expectedConfig := range expectedEndpoint.Configs {

						require.Equal(t, expectedConfig.Key, supplierServiceConfig[i].Endpoints[j].Configs[k].Key)
						require.Equal(t, expectedConfig.Value, supplierServiceConfig[i].Endpoints[j].Configs[k].Value)
					}
				}
			}
		})
	}
}
