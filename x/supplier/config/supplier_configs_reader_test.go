package config_test

import (
	"context"
	"fmt"
	"slices"
	"testing"

	sdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/status"
	"github.com/stretchr/testify/require"

	_ "github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/yaml"
	"github.com/pokt-network/poktroll/x/shared/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/config"
)

func Test_ParseSupplierConfigs_Services(t *testing.T) {
	operatorAddress := sample.AccAddress()
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
				operator_address: %s
				stake_amount: 1000upokt
				services:
				  - service_id: svc
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				      config:
				        timeout: 10
				`, ownerAddress, operatorAddress),
			expectedError: nil,
			expectedConfig: &config.SupplierStakeConfig{
				OwnerAddress:    ownerAddress,
				OperatorAddress: operatorAddress,
				StakeAmount:     sdk.NewCoin("upokt", math.NewInt(1000)),
				Services: []*types.SupplierServiceConfig{
					{
						ServiceId: "svc",
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
						RevShare: []*types.ServiceRevenueShare{
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
				operator_address: %s
				stake_amount: 1000upokt
				services:
				  - service_id: svc
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				`, ownerAddress, operatorAddress),
			expectedError: nil,
			expectedConfig: &config.SupplierStakeConfig{
				OwnerAddress:    ownerAddress,
				OperatorAddress: operatorAddress,
				StakeAmount:     sdk.NewCoin("upokt", math.NewInt(1000)),
				Services: []*types.SupplierServiceConfig{
					{
						ServiceId: "svc",
						Endpoints: []*types.SupplierEndpoint{
							{
								Url:     "http://pokt.network:8081",
								RpcType: types.RPCType_JSON_RPC,
							},
						},
						RevShare: []*types.ServiceRevenueShare{
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
				operator_address: %s
				stake_amount: 1000upokt
				services:
				  - service_id: svc
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				      config:
				`, ownerAddress, operatorAddress),
			expectedError: nil,
			expectedConfig: &config.SupplierStakeConfig{
				OwnerAddress:    ownerAddress,
				OperatorAddress: operatorAddress,
				StakeAmount:     sdk.NewCoin("upokt", math.NewInt(1000)),
				Services: []*types.SupplierServiceConfig{
					{
						ServiceId: "svc",
						Endpoints: []*types.SupplierEndpoint{
							{
								Url:     "http://pokt.network:8081",
								RpcType: types.RPCType_JSON_RPC,
								Configs: []*types.ConfigOption{},
							},
						},
						RevShare: []*types.ServiceRevenueShare{
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
				operator_address: %s
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
				`, ownerAddress, operatorAddress),
			expectedError: nil,
			expectedConfig: &config.SupplierStakeConfig{
				OwnerAddress:    ownerAddress,
				OperatorAddress: operatorAddress,
				StakeAmount:     sdk.NewCoin("upokt", math.NewInt(1000)),
				Services: []*types.SupplierServiceConfig{
					{
						ServiceId: "svc",
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
						RevShare: []*types.ServiceRevenueShare{
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
				operator_address: %s
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
				`, ownerAddress, operatorAddress),
			expectedConfig: &config.SupplierStakeConfig{
				OwnerAddress:    ownerAddress,
				OperatorAddress: operatorAddress,
				StakeAmount:     sdk.NewCoin("upokt", math.NewInt(1000)),
				Services: []*types.SupplierServiceConfig{
					{
						ServiceId: "svc1",
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
						RevShare: []*types.ServiceRevenueShare{
							{
								Address:            ownerAddress,
								RevSharePercentage: 100,
							},
						},
					},
					{
						ServiceId: "svc2",
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
						RevShare: []*types.ServiceRevenueShare{
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
			desc: "valid full service config with both default and service specific rev share",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				operator_address: %s
				default_rev_share_percent:
					%s: 51
					%s: 49
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
				`, ownerAddress, operatorAddress, firstShareHolderAddress, secondShareHolderAddress, ownerAddress, firstShareHolderAddress),
			expectedError: nil,
			expectedConfig: &config.SupplierStakeConfig{
				OwnerAddress:    ownerAddress,
				OperatorAddress: operatorAddress,
				StakeAmount:     sdk.NewCoin("upokt", math.NewInt(1000)),
				Services: []*types.SupplierServiceConfig{
					{
						ServiceId: "svc",
						Endpoints: []*types.SupplierEndpoint{
							{
								Url:     "http://pokt.network:8081",
								RpcType: types.RPCType_JSON_RPC,
							},
						},
						RevShare: []*types.ServiceRevenueShare{
							{
								Address:            firstShareHolderAddress,
								RevSharePercentage: 51,
							},
							{
								Address:            secondShareHolderAddress,
								RevSharePercentage: 49,
							},
						},
					},
					{
						ServiceId: "svc2",
						Endpoints: []*types.SupplierEndpoint{
							{
								Url:     "http://pokt.network:8082",
								RpcType: types.RPCType_JSON_RPC,
							},
						},
						RevShare: []*types.ServiceRevenueShare{
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
		{
			desc: "valid with only default rev share",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				operator_address: %s
				default_rev_share_percent:
					%s: 51
					%s: 49
				stake_amount: 1000upokt
				services:
					# Service with default rev share
				  - service_id: svc
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				`, ownerAddress, operatorAddress, firstShareHolderAddress, secondShareHolderAddress),
			expectedError: nil,
			expectedConfig: &config.SupplierStakeConfig{
				OwnerAddress:    ownerAddress,
				OperatorAddress: operatorAddress,
				StakeAmount:     sdk.NewCoin("upokt", math.NewInt(1000)),
				Services: []*types.SupplierServiceConfig{
					{
						ServiceId: "svc",
						Endpoints: []*types.SupplierEndpoint{
							{
								Url:     "http://pokt.network:8081",
								RpcType: types.RPCType_JSON_RPC,
							},
						},
						RevShare: []*types.ServiceRevenueShare{
							{
								Address:            firstShareHolderAddress,
								RevSharePercentage: 51,
							},
							{
								Address:            secondShareHolderAddress,
								RevSharePercentage: 49,
							},
						},
					},
				},
			},
		},
		{
			desc: "valid omitted operator address",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				# omitted operator address
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
				OwnerAddress:    ownerAddress,
				OperatorAddress: ownerAddress,
				StakeAmount:     sdk.NewCoin("upokt", math.NewInt(1000)),
				Services: []*types.SupplierServiceConfig{
					{
						ServiceId: "svc",
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
						RevShare: []*types.ServiceRevenueShare{
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
			desc: "valid missing default rev share config",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				operator_address: %s
				stake_amount: 1000upokt
				services:
				  - service_id: svc
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				      config:
				        timeout: 10
				`, ownerAddress, ownerAddress),
			expectedError: nil,
			expectedConfig: &config.SupplierStakeConfig{
				OwnerAddress:    ownerAddress,
				OperatorAddress: ownerAddress,
				StakeAmount:     sdk.NewCoin("upokt", math.NewInt(1000)),
				Services: []*types.SupplierServiceConfig{
					{
						ServiceId: "svc",
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
						RevShare: []*types.ServiceRevenueShare{
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
			desc: "valid missing default and specific service rev share config defaults to 100% owner rev share",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				operator_address: %s
				stake_amount: 1000upokt
				services:
				  - service_id: svc
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				`, ownerAddress, ownerAddress),
			expectedError: nil,
			expectedConfig: &config.SupplierStakeConfig{
				OwnerAddress:    ownerAddress,
				OperatorAddress: operatorAddress,
				StakeAmount:     sdk.NewCoin("upokt", math.NewInt(1000)),
				Services: []*types.SupplierServiceConfig{
					{
						ServiceId: "svc",
						Endpoints: []*types.SupplierEndpoint{
							{
								Url:     "http://pokt.network:8081",
								RpcType: types.RPCType_JSON_RPC,
							},
						},
						RevShare: []*types.ServiceRevenueShare{
							{
								Address:            ownerAddress,
								RevSharePercentage: 100,
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
				operator_address: %s
				stake_amount: 1000upokt
				services:
				  - endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				      config:
				        timeout: 10
				`, ownerAddress, operatorAddress),
			expectedError: config.ErrSupplierConfigInvalidServiceId,
		},
		{
			desc: "invalid service config with empty service ID",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				operator_address: %s
				stake_amount: 1000upokt
				services:
				  - service_id:
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				      config:
				        timeout: 10
				`, ownerAddress, operatorAddress),
			expectedError: config.ErrSupplierConfigInvalidServiceId,
		},
		{
			desc: "invalid service config without endpoints",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				operator_address: %s
				stake_amount: 1000upokt
				services:
				  - service_id: svc
			`, ownerAddress, operatorAddress),
			expectedError: config.ErrSupplierConfigNoEndpoints,
		},
		{
			desc: "invalid service config with empty endpoints",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				operator_address: %s
				stake_amount: 1000upokt
				services:
				  - service_id: svc
				    endpoints:
				`, ownerAddress, operatorAddress),
			expectedError: config.ErrSupplierConfigNoEndpoints,
		},
		{
			desc: "invalid service config with unknown endpoint config key",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				operator_address: %s
				stake_amount: 1000upokt
				services:
				  - service_id: svc
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				      config:
				        somekey: 10
				`, ownerAddress, operatorAddress),
			expectedError: config.ErrSupplierConfigInvalidEndpointConfig,
		},
		{
			desc: "invalid service config with unknown endpoint rpc type",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				operator_address: %s
				stake_amount: 1000upokt
				services:
				  - service_id: svc
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: somerpc
				      config:
				        timeout: 10
				`, ownerAddress, operatorAddress),
			expectedError: config.ErrSupplierConfigInvalidRPCType,
		},
		{
			desc: "invalid service config with invalid endpoint url",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				operator_address: %s
				stake_amount: 1000upokt
				services:
				  - service_id: svc
				    endpoints:
				    - publicly_exposed_url: ::invalid_url
				      rpc_type: json_rpc
				      config:
				        timeout: 10
				`, ownerAddress, operatorAddress),
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
				operator_address: %s
				services:
				  - service_id: svc
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				      config:
				        timeout: 10
				`, ownerAddress, operatorAddress),
			expectedError: config.ErrSupplierConfigInvalidStake,
		},
		{
			desc: "invalid stake denom",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				operator_address: %s
				stake_amount: 1000invalid
				services:
				  - service_id: svc
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				      config:
				        timeout: 10
				`, ownerAddress, operatorAddress),
			expectedError: config.ErrSupplierConfigInvalidStake,
		},
		{
			desc: "negative stake amount",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				operator_address: %s
				stake_amount: -1000upokt
				services:
				  - service_id: svc
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				      config:
				        timeout: 10
				`, ownerAddress, operatorAddress),
			expectedError: config.ErrSupplierConfigInvalidStake,
		},
		{
			desc: "zero stake amount",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				operator_address: %s
				stake_amount: 0upokt
				services:
				  - service_id: svc
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				      config:
				        timeout: 10
				`, ownerAddress, operatorAddress),
			expectedError: config.ErrSupplierConfigInvalidStake,
		},
		{
			desc: "missing owner address",
			inputConfig: fmt.Sprintf(`
				# explicitly omitted owner address
				operator_address: %s
				stake_amount: 1000upokt
				services:
				  - service_id: svc
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				      config:
				        timeout: 10
				`, operatorAddress),
			expectedError: config.ErrSupplierConfigInvalidOwnerAddress,
		},
		{
			desc: "invalid owner address",
			inputConfig: fmt.Sprintf(`
				owner_address: invalid_address
				operator_address: %s
				stake_amount: 1000upokt
				services:
				  - service_id: svc
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				      config:
				        timeout: 10
				`, operatorAddress),
			expectedError: config.ErrSupplierConfigInvalidOwnerAddress,
		},
		{
			desc: "invalid operator address",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				operator_address: invalid_address
				stake_amount: 1000upokt
				services:
				  - service_id: svc
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				      config:
				        timeout: 10
				`, ownerAddress),
			expectedError: config.ErrSupplierConfigInvalidOperatorAddress,
		},
		{
			desc: "default rev share does not sum to 100",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				operator_address: %s
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
				`, ownerAddress, operatorAddress, firstShareHolderAddress, secondShareHolderAddress),
			expectedError: sharedtypes.ErrSharedInvalidRevShare,
		},
		{
			desc: "service specific rev share does not sum up to 100",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				operator_address: %s
				stake_amount: 1000upokt
				services:
				  - service_id: svc
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
						rev_share_percent:
							%s: 50
							%s: 49
				`, ownerAddress, operatorAddress, firstShareHolderAddress, secondShareHolderAddress),
			expectedError: sharedtypes.ErrSharedInvalidRevShare,
		},
		{
			desc: "invalid revenue share address",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				operator_address: %s
				stake_amount: 1000upokt
				services:
				  - service_id: svc
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
						rev_share_percent:
							%s: 50
							%s: 49
				`, ownerAddress, operatorAddress, firstShareHolderAddress, "invalid_address"),
			expectedError: sharedtypes.ErrSharedInvalidRevShare,
		},
		{
			desc: "empty revenue share address",
			inputConfig: fmt.Sprintf(`
				owner_address: %s
				operator_address: %s
				stake_amount: 1000upokt
				services:
				  - service_id: svc
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
						rev_share_percent:
							%s: 50
							%s: 49
				`, ownerAddress, operatorAddress, firstShareHolderAddress, ""),
			expectedError: config.ErrSupplierConfigUnmarshalYAML,
		},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			normalizedConfig := yaml.NormalizeYAMLIndentation(tt.inputConfig)
			supplierServiceConfig, err := config.ParseSupplierConfigs(ctx, []byte(normalizedConfig))

			if tt.expectedError != nil {
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

				require.Equal(t, expectedService.ServiceId, service.ServiceId)

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

				require.Equal(t, len(expectedService.RevShare), len(service.RevShare))
				for _, expectedRevShare := range expectedService.RevShare {
					revShareIdx := slices.IndexFunc(service.RevShare, func(revShare *sharedtypes.ServiceRevenueShare) bool {
						return revShare.Address == expectedRevShare.Address
					})
					require.NotEqualf(t, -1, revShareIdx, "expected a revshare entry with address %s for service ID %s", expectedRevShare.Address, service.ServiceId)

					require.Equal(t, expectedRevShare.Address, service.RevShare[revShareIdx].Address)
					require.Equal(t, expectedRevShare.RevSharePercentage, service.RevShare[revShareIdx].RevSharePercentage)
				}
			}
		})
	}
}
