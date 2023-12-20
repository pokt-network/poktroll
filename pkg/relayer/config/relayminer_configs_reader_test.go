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

		inputConfigYAML string

		expectedError  *sdkerrors.Error
		expectedConfig *config.RelayMinerConfig
	}{
		// Valid Configs
		{
			desc: "valid: relay miner config",

			inputConfigYAML: `
				query_node_rpc_url: tcp://127.0.0.1:36657
				query_node_grpc_url: tcp://127.0.0.1:36658
				tx_node_grpc_url: tcp://127.0.0.1:36658
				signing_key_name: servicer1
				smt_store_path: smt_stores
				proxied_service_endpoints:
				  anvil: http://anvil:8080
				  svc1: http://svc1:8080
				`,

			expectedError: nil,
			expectedConfig: &config.RelayMinerConfig{
				QueryNodeRPCUrl:  &url.URL{Scheme: "tcp", Host: "127.0.0.1:36657"},
				QueryNodeGRPCUrl: &url.URL{Scheme: "tcp", Host: "127.0.0.1:36658"},
				TxNodeGRPCUrl:    &url.URL{Scheme: "tcp", Host: "127.0.0.1:36658"},
				SigningKeyName:   "servicer1",
				SmtStorePath:     "smt_stores",
				ProxiedServiceEndpoints: map[string]*url.URL{
					"anvil": {Scheme: "http", Host: "anvil:8080"},
					"svc1":  {Scheme: "http", Host: "svc1:8080"},
				},
			},
		},
		{
			desc: "valid: relay miner config with query node grpc url defaulting to tx node grpc url",

			inputConfigYAML: `
				query_node_rpc_url: tcp://127.0.0.1:36657
				tx_node_grpc_url: tcp://127.0.0.1:36658
				signing_key_name: servicer1
				smt_store_path: smt_stores
				proxied_service_endpoints:
				  anvil: http://anvil:8080
				  svc1: http://svc1:8080
				`,

			expectedError: nil,
			expectedConfig: &config.RelayMinerConfig{
				QueryNodeRPCUrl:  &url.URL{Scheme: "tcp", Host: "127.0.0.1:36657"},
				QueryNodeGRPCUrl: &url.URL{Scheme: "tcp", Host: "127.0.0.1:36658"},
				TxNodeGRPCUrl:    &url.URL{Scheme: "tcp", Host: "127.0.0.1:36658"},
				SigningKeyName:   "servicer1",
				SmtStorePath:     "smt_stores",
				ProxiedServiceEndpoints: map[string]*url.URL{
					"anvil": {Scheme: "http", Host: "anvil:8080"},
					"svc1":  {Scheme: "http", Host: "svc1:8080"},
				},
			},
		},
		// Invalid Configs
		{
			desc: "invalid: invalid tx node grpc url",

			inputConfigYAML: `
				query_node_rpc_url: tcp://127.0.0.1:36657
				query_node_grpc_url: tcp://127.0.0.1:36658
				tx_node_grpc_url: &tcp://127.0.0.1:36658
				signing_key_name: servicer1
				smt_store_path: smt_stores
				proxied_service_endpoints:
				  anvil: http://anvil:8080
				  svc1: http://svc1:8080
				`,

			expectedError: config.ErrRelayMinerConfigInvalidTxNodeGRPCUrl,
		},
		{
			desc: "invalid: missing tx node grpc url",

			inputConfigYAML: `
				query_node_rpc_url: tcp://127.0.0.1:36657
				query_node_grpc_url: tcp://127.0.0.1:36658
				signing_key_name: servicer1
				smt_store_path: smt_stores
				proxied_service_endpoints:
				  anvil: http://anvil:8080
				  svc1: http://svc1:8080
				`,

			expectedError: config.ErrRelayMinerConfigInvalidTxNodeGRPCUrl,
		},
		{
			desc: "invalid: invalid query node grpc url",

			inputConfigYAML: `
				query_node_rpc_url: tcp://127.0.0.1:36657
				query_node_grpc_url: &tcp://127.0.0.1:36658
				tx_node_grpc_url: tcp://127.0.0.1:36658
				signing_key_name: servicer1
				smt_store_path: smt_stores
				proxied_service_endpoints:
				  anvil: http://anvil:8080
				  svc1: http://svc1:8080
				`,

			expectedError: config.ErrRelayMinerConfigInvalidQueryNodeGRPCUrl,
		},
		{
			desc: "invalid: missing query node rpc url",

			inputConfigYAML: `
				# NB: explicitly missing query_node_rpc_url
				query_node_grpc_url: tcp://128.0.0.1:36658
				tx_node_grpc_url: tcp://128.0.0.1:36658
				signing_key_name: servicer1
				smt_store_path: smt_stores
				proxied_service_endpoints:
				  anvil: http://anvil:8080
				  svc1: http://svc1:8080
				`,

			expectedError: config.ErrRelayMinerConfigInvalidQueryNodeRPCUrl,
		},
		{
			desc: "invalid: missing signing key name",

			inputConfigYAML: `
				query_node_rpc_url: tcp://127.0.0.1:36657
				query_node_grpc_url: tcp://127.0.0.1:36658
				tx_node_grpc_url: tcp://127.0.0.1:36658
				# NB: explicitly missing signing_key_name
				smt_store_path: smt_stores
				proxied_service_endpoints:
				  anvil: http://anvil:8080
				  svc1: http://svc1:8080
				`,

			expectedError: config.ErrRelayMinerConfigInvalidSigningKeyName,
		},
		{
			desc: "invalid: missing smt store path",

			inputConfigYAML: `
				query_node_rpc_url: tcp://127.0.0.1:36657
				query_node_grpc_url: tcp://127.0.0.1:36658
				tx_node_grpc_url: tcp://127.0.0.1:36658
				signing_key_name: servicer1
				proxied_service_endpoints:
				  anvil: http://anvil:8080
				  svc1: http://svc1:8080
				`,

			expectedError: config.ErrRelayMinerConfigInvalidSmtStorePath,
		},
		{
			desc: "invalid: empty proxied service endpoints",

			inputConfigYAML: `
				query_node_rpc_url: tcp://127.0.0.1:36657
				query_node_grpc_url: tcp://127.0.0.1:36658
				tx_node_grpc_url: tcp://127.0.0.1:36658
				signing_key_name: servicer1
				smt_store_path: smt_stores
				proxied_service_endpoints:
				`,

			expectedError: config.ErrRelayMinerConfigInvalidServiceEndpoint,
		},
		{
			desc: "invalid: invalid proxied service endpoint",

			inputConfigYAML: `
				query_node_rpc_url: tcp://127.0.0.1:36657
				query_node_grpc_url: tcp://127.0.0.1:36658
				tx_node_grpc_url: tcp://127.0.0.1:36658
				signing_key_name: servicer1
				smt_store_path: smt_stores
				proxied_service_endpoints:
				  anvil: &http://anvil:8080
				  svc1: http://svc1:8080
				`,

			expectedError: config.ErrRelayMinerConfigInvalidServiceEndpoint,
		},
		{
			desc: "invalid: invalid tx node grpc url",

			inputConfigYAML: `
				query_node_rpc_url: tcp://127.0.0.1:36657
				query_node_grpc_url: tcp://127.0.0.1:36658
				tx_node_grpc_url: &tcp://127.0.0.1:36658
				signing_key_name: servicer1
				smt_store_path: smt_stores
				`,

			expectedError: config.ErrRelayMinerConfigInvalidTxNodeGRPCUrl,
		},
		{
			desc: "invalid: empty RelayMiner config file",

			inputConfigYAML: ``,

			expectedError: config.ErrRelayMinerConfigEmpty,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			normalizedConfig := yaml.NormalizeYAMLIndentation(tt.inputConfigYAML)
			config, err := config.ParseRelayMinerConfigs([]byte(normalizedConfig))

			if tt.expectedError != nil {
				require.ErrorIs(t, err, tt.expectedError)
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
