package config_test

import (
	"fmt"
	"testing"

	sdkerrors "cosmossdk.io/errors"
	math "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/status"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/yaml"
	"github.com/pokt-network/poktroll/x/shared/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/config"
)

func Test_ParseSupplierConfigs_Services(t *testing.T) {
	ownerAddress := sample.AccAddress()
	firstShareHolderAddress := sample.AccAddress()
	secondShareHolderAddress := sample.AccAddress()

	tests := []struct {
		desc        string
		inputConfig string

		expectedError  *sdkerrors.Error
		expectedConfig *config.SupplierStakeConfig
	}{
		// Valid Configs
		{
			desc: "valid full service config",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				stake_amount: 1000upokt
				services:
				  - service_id: svc
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				      config:
				        timeout: 10
				`, ownerAddress),
			expectedError: nil,
			expectedConfig: &config.SupplierStakeConfig{
				OwnerAddress: ownerAddress,
				StakeAmount:  sdk.NewCoin("upokt", math.NewInt(1000)),
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
						RevShare: []*types.ServiceRevShare{
							{
								Address:            ownerAddress,
								RevSharePercentage: 100,
							},
						},
					},
				},
			},
		},
		{
			desc: "valid service config without endpoint specific config",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				stake_amount: 1000upokt
				services:
				  - service_id: svc
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				`, ownerAddress),
			expectedError: nil,
			expectedConfig: &config.SupplierStakeConfig{
				OwnerAddress: ownerAddress,
				StakeAmount:  sdk.NewCoin("upokt", math.NewInt(1000)),
				Services: []*types.SupplierServiceConfig{
					{
						Service: &types.Service{Id: "svc"},
						Endpoints: []*types.SupplierEndpoint{
							{
								Url:     "http://pokt.network:8081",
								RpcType: types.RPCType_JSON_RPC,
							},
						},
						RevShare: []*types.ServiceRevShare{
							{
								Address:            ownerAddress,
								RevSharePercentage: 100,
							},
						},
					},
				},
			},
		},
		{
			desc: "valid service config with empty endpoint specific config",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				stake_amount: 1000upokt
				services:
				  - service_id: svc
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				      config:
				`, ownerAddress),
			expectedError: nil,
			expectedConfig: &config.SupplierStakeConfig{
				OwnerAddress: ownerAddress,
				StakeAmount:  sdk.NewCoin("upokt", math.NewInt(1000)),
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
						RevShare: []*types.ServiceRevShare{
							{
								Address:            ownerAddress,
								RevSharePercentage: 100,
							},
						},
					},
				},
			},
		},
		{
			desc: "valid service config with multiple endpoints",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				stake_amount: 1000upokt
				services:
				  - service_id: svc
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				      config:
				        timeout: 10
				    - publicly_exposed_url: http://pokt.network:8082
				      rpc_type: json_rpc
				      config:
				        timeout: 11
				`, ownerAddress),
			expectedError: nil,
			expectedConfig: &config.SupplierStakeConfig{
				OwnerAddress: ownerAddress,
				StakeAmount:  sdk.NewCoin("upokt", math.NewInt(1000)),
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
						RevShare: []*types.ServiceRevShare{
							{
								Address:            ownerAddress,
								RevSharePercentage: 100,
							},
						},
					},
				},
			},
		},
		{
			desc:          "valid service config with multiple services",
			expectedError: nil,
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				stake_amount: 1000upokt
				services:
				  - service_id: svc1
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				      config:
				        timeout: 10
				  - service_id: svc2
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				      config:
				        timeout: 10
				`, ownerAddress),
			expectedConfig: &config.SupplierStakeConfig{
				OwnerAddress: ownerAddress,
				StakeAmount:  sdk.NewCoin("upokt", math.NewInt(1000)),
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
						RevShare: []*types.ServiceRevShare{
							{
								Address:            ownerAddress,
								RevSharePercentage: 100,
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
						RevShare: []*types.ServiceRevShare{
							{
								Address:            ownerAddress,
								RevSharePercentage: 100,
							},
						},
					},
				},
			},
		},
		{
			desc: "valid full service config with default and service specific rev share",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				default_rev_share_percent:
					%s: 50.5
					%s: 49.5
				stake_amount: 1000upokt
				services:
					# Service with default rev share
				  - service_id: svc
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
					# Service with custom rev share
				  - service_id: svc2
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8082
				      rpc_type: json_rpc
						rev_share_percent:
							%s: 60
							%s: 40
				`, ownerAddress, firstShareHolderAddress, secondShareHolderAddress, ownerAddress, firstShareHolderAddress),
			expectedError: nil,
			expectedConfig: &config.SupplierStakeConfig{
				OwnerAddress: ownerAddress,
				StakeAmount:  sdk.NewCoin("upokt", math.NewInt(1000)),
				Services: []*types.SupplierServiceConfig{
					{
						Service: &types.Service{Id: "svc"},
						Endpoints: []*types.SupplierEndpoint{
							{
								Url:     "http://pokt.network:8081",
								RpcType: types.RPCType_JSON_RPC,
							},
						},
						RevShare: []*types.ServiceRevShare{
							{
								Address:            firstShareHolderAddress,
								RevSharePercentage: 50.5,
							},
							{
								Address:            secondShareHolderAddress,
								RevSharePercentage: 49.5,
							},
						},
					},
					{
						Service: &types.Service{Id: "svc2"},
						Endpoints: []*types.SupplierEndpoint{
							{
								Url:     "http://pokt.network:8082",
								RpcType: types.RPCType_JSON_RPC,
							},
						},
						RevShare: []*types.ServiceRevShare{
							{
								Address:            ownerAddress,
								RevSharePercentage: 60,
							},
							{
								Address:            firstShareHolderAddress,
								RevSharePercentage: 40,
							},
						},
					},
				},
			},
		},
		// Invalid Configs
		{
			desc: "invalid service config without service ID",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				stake_amount: 1000upokt
				services:
				  - endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				      config:
				        timeout: 10
				`, ownerAddress),
			expectedError: config.ErrSupplierConfigInvalidServiceId,
		},
		{
			desc: "invalid service config with empty service ID",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				stake_amount: 1000upokt
				services:
				  - service_id:
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				      config:
				        timeout: 10
				`, ownerAddress),
			expectedError: config.ErrSupplierConfigInvalidServiceId,
		},
		{
			desc: "invalid service config without endpoints",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				stake_amount: 1000upokt
				services:
				  - service_id: svc
			`, ownerAddress),
			expectedError: config.ErrSupplierConfigNoEndpoints,
		},
		{
			desc: "invalid service config with empty endpoints",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				stake_amount: 1000upokt
				services:
				  - service_id: svc
				    endpoints:
				`, ownerAddress),
			expectedError: config.ErrSupplierConfigNoEndpoints,
		},
		{
			desc: "invalid service config with unknown endpoint config key",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				stake_amount: 1000upokt
				services:
				  - service_id: svc
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				      config:
				        somekey: 10
				`, ownerAddress),
			expectedError: config.ErrSupplierConfigInvalidEndpointConfig,
		},
		{
			desc: "invalid service config with unknown endpoint rpc type",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				stake_amount: 1000upokt
				services:
				  - service_id: svc
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: somerpc
				      config:
				        timeout: 10
				`, ownerAddress),
			expectedError: config.ErrSupplierConfigInvalidRPCType,
		},
		{
			desc: "invalid service config with invalid endpoint url",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				stake_amount: 1000upokt
				services:
				  - service_id: svc
				    endpoints:
				    - publicly_exposed_url: ::invalid_url
				      rpc_type: json_rpc
				      config:
				        timeout: 10
				`, ownerAddress),
			expectedError: config.ErrSupplierConfigInvalidURL,
		},
		{
			desc:          "invalid service config with empty content",
			expectedError: config.ErrSupplierConfigEmptyContent,
			inputConfig:   ``,
		},
		{
			desc: "missing stake amount",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				services:
				  - service_id: svc
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				      config:
				        timeout: 10
				`, ownerAddress),
			expectedError: config.ErrSupplierConfigInvalidStake,
		},
		{
			desc: "invalid stake denom",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				stake_amount: 1000invalid
				services:
				  - service_id: svc
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				      config:
				        timeout: 10
				`, ownerAddress),
			expectedError: config.ErrSupplierConfigInvalidStake,
		},
		{
			desc: "negative stake amount",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				stake_amount: -1000upokt
				services:
				  - service_id: svc
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				      config:
				        timeout: 10
				`, ownerAddress),
			expectedError: config.ErrSupplierConfigInvalidStake,
		},
		{
			desc: "zero stake amount",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				stake_amount: 0upokt
				services:
				  - service_id: svc
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				      config:
				        timeout: 10
				`, ownerAddress),
			expectedError: config.ErrSupplierConfigInvalidStake,
		},
		{
			desc: "incomplete default rev share",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				default_rev_share_percent:
					%s: 50
					%s: 49
				stake_amount: 1000upokt
				services:
					# Service with default rev share
				  - service_id: svc
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				`, ownerAddress, firstShareHolderAddress, secondShareHolderAddress),
			expectedError: sharedtypes.ErrSharedInvalidRevShare,
		},
		{
			desc: "incomplete service specific rev share",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				stake_amount: 1000upokt
				services:
				  - service_id: svc
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
						rev_share_percent:
							%s: 50
							%s: 49
				`, ownerAddress, firstShareHolderAddress, secondShareHolderAddress),
			expectedError: sharedtypes.ErrSharedInvalidRevShare,
		},
		{
			desc: "invalid share holder address",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				stake_amount: 1000upokt
				services:
				  - service_id: svc
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
						rev_share_percent:
							%s: 50
							%s: 49
				`, ownerAddress, firstShareHolderAddress, "invalid_address"),
			expectedError: sharedtypes.ErrSharedInvalidRevShare,
		},
		{
			desc: "empty share holder address",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				stake_amount: 1000upokt
				services:
				  - service_id: svc
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
						rev_share_percent:
							%s: 50
							%s: 49
				`, ownerAddress, firstShareHolderAddress, ""),
			expectedError: config.ErrSupplierConfigUnmarshalYAML,
		},
		{
			desc: "negative rev share",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				stake_amount: 1000upokt
				services:
				  - service_id: svc
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
						rev_share_percent:
							%s: 90
							%s: 11
							%s: -1
				`, ownerAddress, ownerAddress, firstShareHolderAddress, secondShareHolderAddress),
			expectedError: sharedtypes.ErrSharedInvalidRevShare,
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
			for svcIdx, expectedService := range tt.expectedConfig.Services {
				service := supplierServiceConfig.Services[svcIdx]

				require.Equal(t, expectedService.Service.Id, service.Service.Id)

				require.Equal(t, len(expectedService.Endpoints), len(service.Endpoints))
				for endpointIdx, expectedEndpoint := range expectedService.Endpoints {
					endpoint := service.Endpoints[endpointIdx]

					require.Equal(t, expectedEndpoint.Url, endpoint.Url)
					require.Equal(t, expectedEndpoint.RpcType, endpoint.RpcType)

					require.Equal(t, len(expectedEndpoint.Configs), len(endpoint.Configs))
					for configIdx, expectedConfig := range expectedEndpoint.Configs {
						config := endpoint.Configs[configIdx]

						require.Equal(t, expectedConfig.Key, config.Key)
						require.Equal(t, expectedConfig.Value, config.Value)
					}
				}
			}
		})
	}
}
