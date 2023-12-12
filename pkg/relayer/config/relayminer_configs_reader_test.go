package config_test

import (
	"net/url"
	"testing"

	sdkerrors "cosmossdk.io/errors"
	"github.com/gogo/status"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/relayer/config"
	"github.com/pokt-network/poktroll/testutil/yaml"
)

func Test_ParseRelayMinerConfigs(t *testing.T) {
	tests := []struct {
		desc string

		inputConfig string

		expectedError  *sdkerrors.Error
		expectedConfig *config.RelayMinerConfig
	}{
		// Valid Configs
		{
			desc: "valid: relay miner config",

			inputConfig: `
				query_node_grpc_url: tcp://127.0.0.1:36658
				tx_node_grpc_url: tcp://127.0.0.1:36658
				query_node_rpc_url: tcp://127.0.0.1:36657
				signing_key_name: servicer1
				proxied_service_endpoints:
				  anvil: http://anvil:8080
				  svc1: http://svc1:8080
				smt_store_path: smt_stores
				`,

			expectedError: nil,
			expectedConfig: &config.RelayMinerConfig{
				QueryNodeGRPCUrl: &url.URL{Scheme: "tcp", Host: "127.0.0.1:36658"},
				TxNodeGRPCUrl:    &url.URL{Scheme: "tcp", Host: "127.0.0.1:36658"},
				QueryNodeRPCUrl:  &url.URL{Scheme: "tcp", Host: "127.0.0.1:36657"},
				SigningKeyName:   "servicer1",
				ProxiedServiceEndpoints: map[string]*url.URL{
					"anvil": {Scheme: "http", Host: "anvil:8080"},
					"svc1":  {Scheme: "http", Host: "svc1:8080"},
				},
				SmtStorePath: "smt_stores",
			},
		},
		{
			desc: "valid: relay miner config with query node grpc url defaulting to tx node grpc url",

			inputConfig: `
				tx_node_grpc_url: tcp://127.0.0.1:36658
				query_node_rpc_url: tcp://127.0.0.1:36657
				signing_key_name: servicer1
				proxied_service_endpoints:
				  anvil: http://anvil:8080
				  svc1: http://svc1:8080
				smt_store_path: smt_stores
				`,

			expectedError: nil,
			expectedConfig: &config.RelayMinerConfig{
				QueryNodeGRPCUrl: &url.URL{Scheme: "tcp", Host: "127.0.0.1:36658"},
				TxNodeGRPCUrl:    &url.URL{Scheme: "tcp", Host: "127.0.0.1:36658"},
				QueryNodeRPCUrl:  &url.URL{Scheme: "tcp", Host: "127.0.0.1:36657"},
				SigningKeyName:   "servicer1",
				ProxiedServiceEndpoints: map[string]*url.URL{
					"anvil": {Scheme: "http", Host: "anvil:8080"},
					"svc1":  {Scheme: "http", Host: "svc1:8080"},
				},
				SmtStorePath: "smt_stores",
			},
		},
		// Invalid Configs
		{
			desc: "invalid: invalid tx node grpc url",

			inputConfig: `
				query_node_grpc_url: tcp://127.0.0.1:36658
				tx_node_grpc_url: &tcp://127.0.0.1:36658
				query_node_rpc_url: tcp://127.0.0.1:36657
				signing_key_name: servicer1
				proxied_service_endpoints:
				  anvil: http://anvil:8080
				  svc1: http://svc1:8080
				smt_store_path: smt_stores
				`,

			expectedError: config.ErrRelayMinerConfigInvalidTxNodeGRPCUrl,
		},
		{
			desc: "invalid: missing tx node grpc url",

			inputConfig: `
				query_node_grpc_url: tcp://127.0.0.1:36658
				query_node_rpc_url: tcp://127.0.0.1:36657
				signing_key_name: servicer1
				proxied_service_endpoints:
				  anvil: http://anvil:8080
				  svc1: http://svc1:8080
				smt_store_path: smt_stores
				`,

			expectedError: config.ErrRelayMinerConfigInvalidTxNodeGRPCUrl,
		},
		{
			desc: "invalid: invalid query node grpc url",

			inputConfig: `
				query_node_grpc_url: &tcp://127.0.0.1:36658
				tx_node_grpc_url: tcp://127.0.0.1:36658
				query_node_rpc_url: tcp://127.0.0.1:36657
				signing_key_name: servicer1
				proxied_service_endpoints:
				  anvil: http://anvil:8080
				  svc1: http://svc1:8080
				smt_store_path: smt_stores
				`,

			expectedError: config.ErrRelayMinerConfigInvalidTxNodeGRPCUrl,
		},
		{
			desc: "invalid: missing query node rpc url",

			inputConfig: `
				query_node_grpc_url: tcp://128.0.0.1:36658
				tx_node_grpc_url: tcp://128.0.0.1:36658
				// explicitly missing query_node_rpc_url
				signing_key_name: servicer1
				proxied_service_endpoints:
				  anvil: http://anvil:8080
				  svc1: http://svc1:8080
				smt_store_path: smt_stores
				`,

			expectedError: config.ErrRelayMinerConfigInvalidTxNodeGRPCUrl,
		},
		{
			desc: "invalid: missing signing key name",

			inputConfig: `
				query_node_grpc_url: tcp://127.0.0.1:36658
				tx_node_grpc_url: &tcp://127.0.0.1:36658
				query_node_rpc_url: tcp://127.0.0.1:36657
				signing_key_name:
				proxied_service_endpoints:
				  anvil: http://anvil:8080
				  svc1: http://svc1:8080
				smt_store_path: smt_stores
				`,

			expectedError: config.ErrRelayMinerConfigInvalidTxNodeGRPCUrl,
		},
		{
			desc: "invalid: missing smt store path",

			inputConfig: `
				query_node_grpc_url: tcp://127.0.0.1:36658
				tx_node_grpc_url: &tcp://127.0.0.1:36658
				query_node_rpc_url: tcp://127.0.0.1:36657
				signing_key_name: servicer1
				proxied_service_endpoints:
				  anvil: http://anvil:8080
				  svc1: http://svc1:8080
				`,

			expectedError: config.ErrRelayMinerConfigInvalidTxNodeGRPCUrl,
		},
		{
			desc: "invalid: empty proxied service endpoints",

			inputConfig: `
				query_node_grpc_url: tcp://127.0.0.1:36658
				tx_node_grpc_url: &tcp://127.0.0.1:36658
				query_node_rpc_url: tcp://127.0.0.1:36657
				signing_key_name: servicer1
				proxied_service_endpoints:
				smt_store_path: smt_stores
				`,

			expectedError: config.ErrRelayMinerConfigInvalidTxNodeGRPCUrl,
		},
		{
			desc: "invalid: invalid proxied service endpoint",

			inputConfig: `
				query_node_grpc_url: tcp://127.0.0.1:36658
				tx_node_grpc_url: &tcp://127.0.0.1:36658
				query_node_rpc_url: tcp://127.0.0.1:36657
				signing_key_name: servicer1
				proxied_service_endpoints:
				  anvil: &http://anvil:8080
				  svc1: http://svc1:8080
				smt_store_path: smt_stores
				`,

			expectedError: config.ErrRelayMinerConfigInvalidTxNodeGRPCUrl,
		},
		{
			desc: "invalid: invalid tx node grpc url",

			inputConfig: `
				query_node_grpc_url: tcp://127.0.0.1:36658
				tx_node_grpc_url: &tcp://127.0.0.1:36658
				query_node_rpc_url: tcp://127.0.0.1:36657
				signing_key_name: servicer1
				smt_store_path: smt_stores
				`,

			expectedError: config.ErrRelayMinerConfigUnmarshalYAML,
		},
		{
			desc: "invalid: empty RelayMiner config file",

			inputConfig: ``,

			expectedError: config.ErrRelayMinerConfigUnmarshalYAML,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			normalizedConfig := yaml.NormalizeYAMLIndentation(tt.inputConfig)
			config, err := config.ParseRelayMinerConfigs([]byte(normalizedConfig))

			if tt.expectedError != nil {
				require.Error(t, err)
				require.Nil(t, config)
				stat, ok := status.FromError(tt.expectedError)
				require.True(t, ok)
				require.Contains(t, stat.Message(), tt.expectedError.Error())
				require.Nil(t, config)
				return
			}

			require.NoError(t, err)

			require.Equal(t, tt.expectedConfig.QueryNodeRPCUrl.String(), config.QueryNodeRPCUrl.String())
			require.Equal(t, tt.expectedConfig.QueryNodeGRPCUrl.String(), config.QueryNodeGRPCUrl.String())
			require.Equal(t, tt.expectedConfig.TxNodeGRPCUrl.String(), config.TxNodeGRPCUrl.String())
			require.Equal(t, tt.expectedConfig.SigningKeyName, config.SigningKeyName)
			require.Equal(t, tt.expectedConfig.SmtStorePath, config.SmtStorePath)
			require.Equal(t, len(tt.expectedConfig.ProxiedServiceEndpoints), len(config.ProxiedServiceEndpoints))
			for serviceId, endpoint := range tt.expectedConfig.ProxiedServiceEndpoints {
				require.Equal(t, endpoint.String(), config.ProxiedServiceEndpoints[serviceId].String())
			}
		})
	}
}
