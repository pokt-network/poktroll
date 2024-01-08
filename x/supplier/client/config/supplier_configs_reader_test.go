package config_test

import (
	"testing"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/status"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/yaml"
	"github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/client/config"
)

func Test_ParseSupplierConfigs(t *testing.T) {
	tests := []struct {
		desc           string
		expectedError  *sdkerrors.Error
		expectedConfig *config.SupplierStakeConfig
		inputConfig    string
	}{
		// Valid Configs
		{
			desc: "services_test: valid full service config",
			inputConfig: `
				stake_amount: 1000upokt
				services:
				  - service_id: svc
				    endpoints:
				    - url: http://pokt.network:8081
				      rpc_type: json_rpc
				      config:
				        timeout: 10
				`,
			expectedError: nil,
			expectedConfig: &config.SupplierStakeConfig{
				StakeAmount: sdk.NewCoin("upokt", sdk.NewInt(1000)),
				Services: []*types.SupplierServiceConfig{
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
			},
		},
		{
			desc: "services_test: valid service config without endpoint config",
			inputConfig: `
				stake_amount: 1000upokt
				services:
				  - service_id: svc
				    endpoints:
				    - url: http://pokt.network:8081
				      rpc_type: json_rpc
				`,
			expectedError: nil,
			expectedConfig: &config.SupplierStakeConfig{
				StakeAmount: sdk.NewCoin("upokt", sdk.NewInt(1000)),
				Services: []*types.SupplierServiceConfig{
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
			},
		},
		{
			desc: "services_test: valid service config with empty endpoint config",
			inputConfig: `
				stake_amount: 1000upokt
				services:
				  - service_id: svc
				    endpoints:
				    - url: http://pokt.network:8081
				      rpc_type: json_rpc
				      config:
				`,
			expectedError: nil,
			expectedConfig: &config.SupplierStakeConfig{
				StakeAmount: sdk.NewCoin("upokt", sdk.NewInt(1000)),
				Services: []*types.SupplierServiceConfig{
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
			},
		},
		{
			desc: "services_test: valid service config with multiple endpoints",
			inputConfig: `
				stake_amount: 1000upokt
				services:
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
			expectedError: nil,
			expectedConfig: &config.SupplierStakeConfig{
				StakeAmount: sdk.NewCoin("upokt", sdk.NewInt(1000)),
				Services: []*types.SupplierServiceConfig{
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
			},
		},
		{
			desc:          "services_test: valid service config with multiple services",
			expectedError: nil,
			inputConfig: `
				stake_amount: 1000upokt
				services:
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
			expectedConfig: &config.SupplierStakeConfig{
				StakeAmount: sdk.NewCoin("upokt", sdk.NewInt(1000)),
				Services: []*types.SupplierServiceConfig{
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
			},
		},
		// Invalid Configs
		{
			desc: "services_test: invalid service config without service ID",
			inputConfig: `
				stake_amount: 1000upokt
				services:
				  - endpoints:
				    - url: http://pokt.network:8081
				      rpc_type: json_rpc
				      config:
				        timeout: 10
				`,
			expectedError: config.ErrSupplierConfigInvalidServiceId,
		},
		{
			desc: "services_test: invalid service config with empty service ID",
			inputConfig: `
				stake_amount: 1000upokt
				services:
				  - service_id:
				    endpoints:
				    - url: http://pokt.network:8081
				      rpc_type: json_rpc
				      config:
				        timeout: 10
				`,
			expectedError: config.ErrSupplierConfigInvalidServiceId,
		},
		{
			desc: "services_test: invalid service config without endpoints",
			inputConfig: `
				stake_amount: 1000upokt
				services:
				  - service_id: svc
			`,
			expectedError: config.ErrSupplierConfigNoEndpoints,
		},
		{
			desc: "services_test: invalid service config with empty endpoints",
			inputConfig: `
				stake_amount: 1000upokt
				services:
				  - service_id: svc
				    endpoints:
				`,
			expectedError: config.ErrSupplierConfigNoEndpoints,
		},
		{
			desc: "services_test: invalid service config with unknown endpoint config key",
			inputConfig: `
				stake_amount: 1000upokt
				services:
				  - service_id: svc
				    endpoints:
				    - url: http://pokt.network:8081
				      rpc_type: json_rpc
				      config:
				        somekey: 10
				`,
			expectedError: config.ErrSupplierConfigInvalidEndpointConfig,
		},
		{
			desc: "services_test: invalid service config with unknown endpoint rpc type",
			inputConfig: `
				stake_amount: 1000upokt
				services:
				  - service_id: svc
				    endpoints:
				    - url: http://pokt.network:8081
				      rpc_type: somerpc
				      config:
				        timeout: 10
				`,
			expectedError: config.ErrSupplierConfigInvalidRPCType,
		},
		{
			desc: "services_test: invalid service config with invalid endpoint url",
			inputConfig: `
				stake_amount: 1000upokt
				services:
				  - service_id: svc
				    endpoints:
				    - url: ::invalid_url
				      rpc_type: json_rpc
				      config:
				        timeout: 10
				`,
			expectedError: config.ErrSupplierConfigInvalidURL,
		},
		{
			desc:          "services_test: invalid service config with empty content",
			expectedError: config.ErrSupplierConfigEmptyContent,
			inputConfig:   ``,
		},
		{
			desc: "services_test: missing stake amount",
			inputConfig: `
				services:
				  - service_id: svc
				    endpoints:
				    - url: http://pokt.network:8081
				      rpc_type: json_rpc
				      config:
				        timeout: 10
				`,
			expectedError: config.ErrSupplierConfigInvalidStake,
		},
		{
			desc: "services_test: invalid stake denom",
			inputConfig: `
				stake_amount: 1000invalid
				services:
				  - service_id: svc
				    endpoints:
				    - url: http://pokt.network:8081
				      rpc_type: json_rpc
				      config:
				        timeout: 10
				`,
			expectedError: config.ErrSupplierConfigInvalidStake,
		},
		{
			desc: "services_test: negative stake amount",
			inputConfig: `
				stake_amount: -1000upokt
				services:
				  - service_id: svc
				    endpoints:
				    - url: http://pokt.network:8081
				      rpc_type: json_rpc
				      config:
				        timeout: 10
				`,
			expectedError: config.ErrSupplierConfigInvalidStake,
		},
		{
			desc: "services_test: zero stake amount",
			inputConfig: `
				stake_amount: 0upokt
				services:
				  - service_id: svc
				    endpoints:
				    - url: http://pokt.network:8081
				      rpc_type: json_rpc
				      config:
				        timeout: 10
				`,
			expectedError: config.ErrSupplierConfigInvalidStake,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			normalizedConfig := yaml.NormalizeYAMLIndentation(tt.inputConfig)
			supplierServiceConfig, err := config.ParseSupplierConfigs([]byte(normalizedConfig))

			if tt.expectedError != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedError)
				stat, ok := status.FromError(tt.expectedError)
				require.True(t, ok)
				require.Contains(t, stat.Message(), tt.expectedError.Error())
				require.Nil(t, supplierServiceConfig)
				return
			}

			require.NoError(t, err)

			require.Equal(t, tt.expectedConfig.StakeAmount, supplierServiceConfig.StakeAmount)
			require.Equal(t, tt.expectedConfig.StakeAmount.Denom, supplierServiceConfig.StakeAmount.Denom)

			require.Equal(t, len(tt.expectedConfig.Services), len(supplierServiceConfig.Services))
			for i, expected := range tt.expectedConfig.Services {

				require.Equal(t, expected.Service.Id, supplierServiceConfig.Services[i].Service.Id)

				require.Equal(t, len(expected.Endpoints), len(supplierServiceConfig.Services[i].Endpoints))
				for j, expectedEndpoint := range expected.Endpoints {

					require.Equal(t, expectedEndpoint.Url, supplierServiceConfig.Services[i].Endpoints[j].Url)
					require.Equal(t, expectedEndpoint.RpcType, supplierServiceConfig.Services[i].Endpoints[j].RpcType)

					require.Equal(t, len(expectedEndpoint.Configs), len(supplierServiceConfig.Services[i].Endpoints[j].Configs))
					for k, expectedConfig := range expectedEndpoint.Configs {

						require.Equal(t, expectedConfig.Key, supplierServiceConfig.Services[i].Endpoints[j].Configs[k].Key)
						require.Equal(t, expectedConfig.Value, supplierServiceConfig.Services[i].Endpoints[j].Configs[k].Value)
					}
				}
			}
		})
	}
}
