package config_test

import (
	"net/url"
	"os"
	"testing"

	sdkerrors "cosmossdk.io/errors"
	"github.com/gogo/status"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/relayer/config"
	"github.com/pokt-network/poktroll/testutil/yaml"
)

func Test_ParseRelayMinerConfig_ReferenceExample(t *testing.T) {
	configContent, err := os.ReadFile("../../../localnet/pocketd/config/relayminer_config_full_example.yaml")
	require.NoError(t, err)

	_, err = config.ParseRelayMinerConfigs(configContent)
	require.NoError(t, err)
}

func Test_ParseRelayMinerConfigs(t *testing.T) {
	tests := []struct {
		desc            string
		inputConfigYAML string

		expectedErr    *sdkerrors.Error
		expectedConfig *config.RelayMinerConfig
	}{
		// Valid Configs
		{
			desc: "valid: relay miner config",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:26657
				  query_node_grpc_url: tcp://127.0.0.1:9090
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				default_signing_key_names: [ supplier1 ]
				default_request_timeout_seconds: 60
				smt_store_path: smt_stores
				suppliers:
				  - service_id: ethereum
				    listen_url: http://127.0.0.1:8080
				    service_config:
				      backend_url: http://anvil.servicer:8545
				      authentication:
				        username: user
				        password: pwd
				      headers: {}
				    request_timeout_seconds: 20
				`,

			expectedErr: nil,
			expectedConfig: &config.RelayMinerConfig{
				PocketNode: &config.RelayMinerPocketNodeConfig{
					QueryNodeRPCUrl:  &url.URL{Scheme: "tcp", Host: "127.0.0.1:26657"},
					QueryNodeGRPCUrl: &url.URL{Scheme: "tcp", Host: "127.0.0.1:9090"},
					TxNodeRPCUrl:     &url.URL{Scheme: "tcp", Host: "127.0.0.1:36659"},
				},
				DefaultSigningKeyNames:       []string{"supplier1"},
				DefaultRequestTimeoutSeconds: uint64(60),
				SmtStorePath:                 "smt_stores",
				Servers: map[string]*config.RelayMinerServerConfig{
					"http://127.0.0.1:8080": {
						ListenAddress:        "127.0.0.1:8080",
						ServerType:           config.RelayMinerServerTypeHTTP,
						XForwardedHostLookup: false,
						SupplierConfigsMap: map[string]*config.RelayMinerSupplierConfig{
							"ethereum": {
								ServiceId:  "ethereum",
								ServerType: config.RelayMinerServerTypeHTTP,
								DefaultServiceConfig: &config.RelayMinerSupplierServiceConfig{
									BackendUrl: &url.URL{Scheme: "http", Host: "anvil.servicer:8545"},
									Authentication: &config.RelayMinerSupplierServiceAuthentication{
										Username: "user",
										Password: "pwd",
									},
									Headers: map[string]string{},
								},
								RequestTimeoutSeconds: uint64(20),
							},
						},
					},
				},
			},
		},
		{
			desc: "valid: relay miner config with signing key configured on supplier level",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:36657
				  query_node_grpc_url: tcp://127.0.0.1:36658
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				smt_store_path: smt_stores
				suppliers:
				  - service_id: ethereum
				    listen_url: http://127.0.0.1:8080
				    signing_key_names: [ supplier1 ]
				    service_config:
				      backend_url: http://anvil.servicer:8545
				      authentication:
				        username: user
				        password: pwd
				      headers: {}
				    request_timeout_seconds: 20
				`,

			expectedErr: nil,
			expectedConfig: &config.RelayMinerConfig{
				PocketNode: &config.RelayMinerPocketNodeConfig{
					QueryNodeRPCUrl:  &url.URL{Scheme: "tcp", Host: "127.0.0.1:36657"},
					QueryNodeGRPCUrl: &url.URL{Scheme: "tcp", Host: "127.0.0.1:36658"},
					TxNodeRPCUrl:     &url.URL{Scheme: "tcp", Host: "127.0.0.1:36659"},
				},
				SmtStorePath:                 "smt_stores",
				DefaultRequestTimeoutSeconds: config.DefaultRequestTimeoutSeconds,
				Servers: map[string]*config.RelayMinerServerConfig{
					"http://127.0.0.1:8080": {
						ListenAddress:        "127.0.0.1:8080",
						ServerType:           config.RelayMinerServerTypeHTTP,
						XForwardedHostLookup: false,
						SupplierConfigsMap: map[string]*config.RelayMinerSupplierConfig{
							"ethereum": {
								ServiceId:  "ethereum",
								ServerType: config.RelayMinerServerTypeHTTP,
								DefaultServiceConfig: &config.RelayMinerSupplierServiceConfig{
									BackendUrl: &url.URL{Scheme: "http", Host: "anvil.servicer:8545"},
									Authentication: &config.RelayMinerSupplierServiceAuthentication{
										Username: "user",
										Password: "pwd",
									},
									Headers: map[string]string{},
								},
								SigningKeyNames:       []string{"supplier1"},
								RequestTimeoutSeconds: 20,
							},
						},
					},
				},
			},
		},
		{
			desc: "valid: relay miner config with signing keys configured on both global and supplier level",

			inputConfigYAML: `
			pocket_node:
			  query_node_rpc_url: tcp://127.0.0.1:36657
			  query_node_grpc_url: tcp://127.0.0.1:36658
			  tx_node_rpc_url: tcp://127.0.0.1:36659
			smt_store_path: smt_stores
			default_signing_key_names: [supplier1]
			default_request_timeout_seconds: 120
			suppliers:
			  - service_id: ethereum
			    listen_url: http://127.0.0.1:8080
			    signing_key_names: []
			    service_config:
			      backend_url: http://anvil.servicer:8545
			      authentication:
			        username: user
			        password: pwd
			      headers: {}
			  - service_id: ollama
			    listen_url: http://127.0.0.1:8080
			    signing_key_names: [supplier2]
			    service_config:
			      backend_url: http://ollama.servicer:8545
			      authentication:
			        username: user
			        password: pwd
			      headers: {}
			`,

			expectedErr: nil,
			expectedConfig: &config.RelayMinerConfig{
				PocketNode: &config.RelayMinerPocketNodeConfig{
					QueryNodeRPCUrl:  &url.URL{Scheme: "tcp", Host: "127.0.0.1:36657"},
					QueryNodeGRPCUrl: &url.URL{Scheme: "tcp", Host: "127.0.0.1:36658"},
					TxNodeRPCUrl:     &url.URL{Scheme: "tcp", Host: "127.0.0.1:36659"},
				},
				SmtStorePath:                 "smt_stores",
				DefaultSigningKeyNames:       []string{"supplier1"},
				DefaultRequestTimeoutSeconds: 120,
				Servers: map[string]*config.RelayMinerServerConfig{
					"http://127.0.0.1:8080": {
						ListenAddress:        "127.0.0.1:8080",
						ServerType:           config.RelayMinerServerTypeHTTP,
						XForwardedHostLookup: false,
						SupplierConfigsMap: map[string]*config.RelayMinerSupplierConfig{
							"ethereum": {
								ServiceId:  "ethereum",
								ServerType: config.RelayMinerServerTypeHTTP,
								DefaultServiceConfig: &config.RelayMinerSupplierServiceConfig{
									BackendUrl: &url.URL{Scheme: "http", Host: "anvil.servicer:8545"},
									Authentication: &config.RelayMinerSupplierServiceAuthentication{
										Username: "user",
										Password: "pwd",
									},
									Headers: map[string]string{},
								},
								// Note the supplier is missing in the yaml, but it is populated from
								// the global `default_signing_key_names`
								SigningKeyNames:       []string{"supplier1"},
								RequestTimeoutSeconds: 120,
							},
							"ollama": {
								ServiceId:  "ollama",
								ServerType: config.RelayMinerServerTypeHTTP,
								DefaultServiceConfig: &config.RelayMinerSupplierServiceConfig{
									BackendUrl: &url.URL{Scheme: "http", Host: "ollama.servicer:8545"},
									Authentication: &config.RelayMinerSupplierServiceAuthentication{
										Username: "user",
										Password: "pwd",
									},
									Headers: map[string]string{},
								},
								SigningKeyNames:       []string{"supplier2"},
								RequestTimeoutSeconds: 120,
							},
						},
					},
				},
			},
		},
		{
			desc: "valid: multiple suppliers, single server",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:26657
				  query_node_grpc_url: tcp://127.0.0.1:9090
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				default_signing_key_names: [ supplier1 ]
				smt_store_path: smt_stores
				suppliers:
				  - service_id: ethereum
				    listen_url: http://127.0.0.1:8080
				    service_config:
				      backend_url: http://anvil.servicer:8545
				      authentication:
				        username: user
				        password: pwd
				      headers: {}
				  - service_id: 7b-llm-model
				    listen_url: http://127.0.0.1:8080
				    service_config:
				      backend_url: http://llama-endpoint
				`,

			expectedErr: nil,
			expectedConfig: &config.RelayMinerConfig{
				PocketNode: &config.RelayMinerPocketNodeConfig{
					QueryNodeRPCUrl:  &url.URL{Scheme: "tcp", Host: "127.0.0.1:26657"},
					QueryNodeGRPCUrl: &url.URL{Scheme: "tcp", Host: "127.0.0.1:9090"},
					TxNodeRPCUrl:     &url.URL{Scheme: "tcp", Host: "127.0.0.1:36659"},
				},
				DefaultSigningKeyNames:       []string{"supplier1"},
				SmtStorePath:                 "smt_stores",
				DefaultRequestTimeoutSeconds: config.DefaultRequestTimeoutSeconds,
				Servers: map[string]*config.RelayMinerServerConfig{
					"http://127.0.0.1:8080": {
						ListenAddress:        "127.0.0.1:8080",
						ServerType:           config.RelayMinerServerTypeHTTP,
						XForwardedHostLookup: false,
						SupplierConfigsMap: map[string]*config.RelayMinerSupplierConfig{
							"ethereum": {
								ServiceId:  "ethereum",
								ServerType: config.RelayMinerServerTypeHTTP,
								DefaultServiceConfig: &config.RelayMinerSupplierServiceConfig{
									BackendUrl: &url.URL{Scheme: "http", Host: "anvil.servicer:8545"},
									Authentication: &config.RelayMinerSupplierServiceAuthentication{
										Username: "user",
										Password: "pwd",
									},
									Headers: map[string]string{},
								},
								RequestTimeoutSeconds: config.DefaultRequestTimeoutSeconds,
							},
							"7b-llm-model": {
								ServiceId:  "7b-llm-model",
								ServerType: config.RelayMinerServerTypeHTTP,
								DefaultServiceConfig: &config.RelayMinerSupplierServiceConfig{
									BackendUrl: &url.URL{Scheme: "http", Host: "llama-endpoint"},
								},
								RequestTimeoutSeconds: config.DefaultRequestTimeoutSeconds,
							},
						},
					},
				},
			},
		},
		{
			desc: "valid: relay miner config with query node rpc url defaulting to tx node rpc url",

			inputConfigYAML: `
				pocket_node:
				  query_node_grpc_url: tcp://127.0.0.1:9090
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				default_signing_key_names: [ supplier1 ]
				smt_store_path: smt_stores
				suppliers:
				  - service_id: ethereum
				    listen_url: http://127.0.0.1:8080
				    service_config:
				      backend_url: http://anvil.servicer:8545
				`,

			expectedErr: nil,
			expectedConfig: &config.RelayMinerConfig{
				PocketNode: &config.RelayMinerPocketNodeConfig{
					QueryNodeRPCUrl:  &url.URL{Scheme: "tcp", Host: "127.0.0.1:36659"},
					QueryNodeGRPCUrl: &url.URL{Scheme: "tcp", Host: "127.0.0.1:9090"},
					TxNodeRPCUrl:     &url.URL{Scheme: "tcp", Host: "127.0.0.1:36659"},
				},
				DefaultSigningKeyNames:       []string{"supplier1"},
				SmtStorePath:                 "smt_stores",
				DefaultRequestTimeoutSeconds: config.DefaultRequestTimeoutSeconds,
				Servers: map[string]*config.RelayMinerServerConfig{
					"http://127.0.0.1:8080": {
						ListenAddress:        "127.0.0.1:8080",
						ServerType:           config.RelayMinerServerTypeHTTP,
						XForwardedHostLookup: false,
						SupplierConfigsMap: map[string]*config.RelayMinerSupplierConfig{
							"ethereum": {
								ServiceId:  "ethereum",
								ServerType: config.RelayMinerServerTypeHTTP,
								DefaultServiceConfig: &config.RelayMinerSupplierServiceConfig{
									BackendUrl: &url.URL{Scheme: "http", Host: "anvil.servicer:8545"},
								},
								RequestTimeoutSeconds: config.DefaultRequestTimeoutSeconds,
							},
						},
					},
				},
			},
		},
		{
			desc: "valid: relay miner config with x_forwarded_host_lookup set to true",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:26657
				  query_node_grpc_url: tcp://127.0.0.1:9090
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				default_signing_key_names: [ supplier1 ]
				smt_store_path: smt_stores
				suppliers:
				  - service_id: ethereum
				    x_forwarded_host_lookup: true
				    listen_url: http://127.0.0.1:8080
				    service_config:
				      backend_url: http://anvil.servicer:8545
				`,

			expectedErr: nil,
			expectedConfig: &config.RelayMinerConfig{
				PocketNode: &config.RelayMinerPocketNodeConfig{
					QueryNodeRPCUrl:  &url.URL{Scheme: "tcp", Host: "127.0.0.1:26657"},
					QueryNodeGRPCUrl: &url.URL{Scheme: "tcp", Host: "127.0.0.1:9090"},
					TxNodeRPCUrl:     &url.URL{Scheme: "tcp", Host: "127.0.0.1:36659"},
				},
				DefaultSigningKeyNames:       []string{"supplier1"},
				SmtStorePath:                 "smt_stores",
				DefaultRequestTimeoutSeconds: config.DefaultRequestTimeoutSeconds,
				Servers: map[string]*config.RelayMinerServerConfig{
					"http://127.0.0.1:8080": {
						ListenAddress:        "127.0.0.1:8080",
						ServerType:           config.RelayMinerServerTypeHTTP,
						XForwardedHostLookup: true,
						SupplierConfigsMap: map[string]*config.RelayMinerSupplierConfig{
							"ethereum": {
								ServiceId:  "ethereum",
								ServerType: config.RelayMinerServerTypeHTTP,
								DefaultServiceConfig: &config.RelayMinerSupplierServiceConfig{
									BackendUrl: &url.URL{Scheme: "http", Host: "anvil.servicer:8545"},
								},
								RequestTimeoutSeconds: config.DefaultRequestTimeoutSeconds,
							},
						},
					},
				},
			},
		},
		{
			desc: "valid: relay miner config with rpc_type_service_configs",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:26657
				  query_node_grpc_url: tcp://127.0.0.1:9090
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				default_signing_key_names: [ supplier1 ]
				smt_store_path: smt_stores
				suppliers:
				  - service_id: ethereum
				    listen_url: http://127.0.0.1:8080
				    service_config:
				      backend_url: http://anvil.servicer:8545
				      headers:
				        X-Default: default-value
				    rpc_type_service_configs:
				      json_rpc:
				        backend_url: http://json_rpc.servicer:8545
				        headers:
				          X-Type: json-rpc
				      rest:
				        backend_url: http://rest.servicer:8545
				        headers:
				          X-Type: rest
				`,

			expectedErr: nil,
			expectedConfig: &config.RelayMinerConfig{
				PocketNode: &config.RelayMinerPocketNodeConfig{
					QueryNodeRPCUrl:  &url.URL{Scheme: "tcp", Host: "127.0.0.1:26657"},
					QueryNodeGRPCUrl: &url.URL{Scheme: "tcp", Host: "127.0.0.1:9090"},
					TxNodeRPCUrl:     &url.URL{Scheme: "tcp", Host: "127.0.0.1:36659"},
				},
				DefaultSigningKeyNames:       []string{"supplier1"},
				SmtStorePath:                 "smt_stores",
				DefaultRequestTimeoutSeconds: config.DefaultRequestTimeoutSeconds,
				Servers: map[string]*config.RelayMinerServerConfig{
					"http://127.0.0.1:8080": {
						ListenAddress:        "127.0.0.1:8080",
						ServerType:           config.RelayMinerServerTypeHTTP,
						XForwardedHostLookup: false,
						SupplierConfigsMap: map[string]*config.RelayMinerSupplierConfig{
							"ethereum": {
								ServiceId:  "ethereum",
								ServerType: config.RelayMinerServerTypeHTTP,
								DefaultServiceConfig: &config.RelayMinerSupplierServiceConfig{
									BackendUrl: &url.URL{Scheme: "http", Host: "anvil.servicer:8545"},
									Headers: map[string]string{
										"X-Default": "default-value",
									},
								},
								RPCTypeServiceConfigs: map[config.RPCType]*config.RelayMinerSupplierServiceConfig{
									config.RPCTypeJSONRPC: {
										BackendUrl: &url.URL{Scheme: "http", Host: "json_rpc.servicer:8545"},
										Headers: map[string]string{
											"X-Type": "json-rpc",
										},
									},
									config.RPCTypeREST: {
										BackendUrl: &url.URL{Scheme: "http", Host: "rest.servicer:8545"},
										Headers: map[string]string{
											"X-Type": "rest",
										},
									},
								},
								RequestTimeoutSeconds: config.DefaultRequestTimeoutSeconds,
							},
						},
					},
				},
			},
		},
		// Invalid Configs
		{
			desc: "invalid: invalid tx node grpc url",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:26657
				  query_node_grpc_url: tcp://127.0.0.1:9090
				  tx_node_rpc_url: &tcp://127.0.0.1:36659
				default_signing_key_names: [ supplier1 ]
				smt_store_path: smt_stores
				suppliers:
				  - service_id: ethereum
				    listen_url: http://127.0.0.1:8080
				    service_config:
				      backend_url: http://anvil.servicer:8545
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidNodeUrl,
		},
		{
			desc: "invalid: missing tx node grpc url",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:26657
				  # explicitly omitted tx node grpc url
				  query_node_grpc_url: tcp://127.0.0.1:9090
				default_signing_key_names: [ supplier1 ]
				smt_store_path: smt_stores
				suppliers:
				  - service_id: ethereum
				    service_config:
				      backend_url: http://anvil.servicer:8545
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidNodeUrl,
		},
		{
			desc: "invalid: invalid query node grpc url",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:26657
				  query_node_grpc_url: &tcp://127.0.0.1:9090
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				default_signing_key_names: [ supplier1 ]
				smt_store_path: smt_stores
				suppliers:
				  - service_id: ethereum
				    listen_url: http://127.0.0.1:8080
				    service_config:
				      backend_url: http://anvil.servicer:8545
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidNodeUrl,
		},
		{
			desc: "invalid: invalid query node rpc url",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: &tcp://127.0.0.1:26657
				  query_node_grpc_url: tcp://127.0.0.1:9090
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				default_signing_key_names: [ supplier1 ]
				smt_store_path: smt_stores
				suppliers:
				  - service_id: ethereum
				    listen_url: http://127.0.0.1:8080
				    service_config:
				      backend_url: http://anvil.servicer:8545
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidNodeUrl,
		},
		{
			desc: "invalid: missing query node grpc url",

			inputConfigYAML: `
				pocket_node:
				  # explicitly omitted query node rpc url
				  query_node_rpc_url: tcp://127.0.0.1:26657
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				default_signing_key_names: [ supplier1 ]
				smt_store_path: smt_stores
				suppliers:
				  - service_id: ethereum
				    listen_url: http://127.0.0.1:8080
				    service_config:
				      backend_url: http://anvil.servicer:8545
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidNodeUrl,
		},
		{
			desc: "invalid: missing both default and supplier signing key names",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:26657
				  query_node_grpc_url: tcp://127.0.0.1:9090
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				# explicitly omitted signing key name
				smt_store_path: smt_stores
				suppliers:
				  - service_id: ethereum
				    listen_url: http://127.0.0.1:8080
				    service_config:
				      backend_url: http://anvil.servicer:8545
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidSigningKeyName,
		},
		{
			desc: "invalid: missing smt store path",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:26657
				  query_node_grpc_url: tcp://127.0.0.1:9090
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				default_signing_key_names: [ supplier1 ]
				# explicitly omitted smt store path
				suppliers:
				  - service_id: ethereum
				    listen_url: http://127.0.0.1:8080
				    service_config:
				      backend_url: http://anvil.servicer:8545
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidSmtStorePath,
		},
		{
			desc: "invalid: empty listen address",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:26657
				  query_node_grpc_url: tcp://127.0.0.1:9090
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				default_signing_key_names: [ supplier1 ]
				smt_store_path: smt_stores
				suppliers:
				  - service_id: ethereum
				    listen_url: http:// # explicitly empty listen url
				    service_config:
				      backend_url: http://anvil.servicer:8545
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidServer,
		},
		{
			desc: "invalid: unsupported server type",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:26657
				  query_node_grpc_url: tcp://127.0.0.1:9090
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				default_signing_key_names: [ supplier1 ]
				smt_store_path: smt_stores
				suppliers:
				  - service_id: ethereum
				    listen_url: unsupported://127.0.0.1:8080
				    service_config:
				      backend_url: http://anvil.servicer:8545
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidServer,
		},
		{
			desc: "invalid: missing supplier name",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:26657
				  query_node_grpc_url: tcp://127.0.0.1:9090
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				default_signing_key_names: [ supplier1 ]
				smt_store_path: smt_stores
				suppliers:
				  - listen_url: http://127.0.0.1:8080
				    # explicitly missing supplier name
				    service_config:
				      backend_url: http://anvil.servicer:8545
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidSupplier,
		},
		{
			desc: "invalid: empty supplier name",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:26657
				  query_node_grpc_url: tcp://127.0.0.1:9090
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				default_signing_key_names: [ supplier1 ]
				smt_store_path: smt_stores
				suppliers:
				  - service_id: # explicitly empty supplier name
				    listen_url: http://127.0.0.1:8080
				    service_config:
				      backend_url: http://anvil.servicer:8545
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidSupplier,
		},
		{
			desc: "invalid: bad supplier service config backend url",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:26657
				  query_node_grpc_url: tcp://127.0.0.1:9090
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				default_signing_key_names: [ supplier1 ]
				smt_store_path: smt_stores
				suppliers:
				  - service_id: ethereum
				    listen_url: http://127.0.0.1:8080
				    service_config:
				      backend_url: &http://anvil.servicer:8545
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidSupplier,
		},
		{
			desc: "invalid: empty supplier service config backend url",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:26657
				  query_node_grpc_url: tcp://127.0.0.1:9090
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				default_signing_key_names: [ supplier1 ]
				smt_store_path: smt_stores
				suppliers:
				  - service_id: ethereum
				    listen_url: http://127.0.0.1:8080
				    service_config:
				      backend_url: # explicitly empty supplier service config backend url
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidSupplier,
		},
		{
			desc: "invalid: missing supplier service config backend url",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:26657
				  query_node_grpc_url: tcp://127.0.0.1:9090
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				default_signing_key_names: [ supplier1 ]
				smt_store_path: smt_stores
				suppliers:
				  - service_id: ethereum
				    listen_url: http://127.0.0.1:8080
				    service_config:
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidSupplier,
		},
		{
			desc: "invalid: empty RelayMiner config file",

			inputConfigYAML: ``,

			expectedErr: config.ErrRelayMinerConfigEmpty,
		},
		// TODO_NB: Test for supplier and server types mismatch once we have more
		// than one server type.
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			normalizedConfig := yaml.NormalizeYAMLIndentation(test.inputConfigYAML)
			config, err := config.ParseRelayMinerConfigs([]byte(normalizedConfig))

			// Invalid configuration
			if test.expectedErr != nil {
				require.ErrorIs(t, err, test.expectedErr)
				require.Nil(t, config)
				stat, ok := status.FromError(test.expectedErr)
				require.True(t, ok)
				require.Contains(t, stat.Message(), test.expectedErr.Error())
				require.Nil(t, config)
				return
			}

			// Valid configuration
			require.NoError(t, err)

			require.Equal(
				t,
				test.expectedConfig.DefaultSigningKeyNames,
				config.DefaultSigningKeyNames,
			)

			require.Equal(
				t,
				test.expectedConfig.SmtStorePath,
				config.SmtStorePath,
			)

			require.Equal(
				t,
				test.expectedConfig.PocketNode.QueryNodeGRPCUrl.String(),
				config.PocketNode.QueryNodeGRPCUrl.String(),
			)

			require.Equal(
				t,
				test.expectedConfig.PocketNode.QueryNodeRPCUrl.String(),
				config.PocketNode.QueryNodeRPCUrl.String(),
			)

			require.Equal(
				t,
				test.expectedConfig.PocketNode.TxNodeRPCUrl.String(),
				config.PocketNode.TxNodeRPCUrl.String(),
			)

			require.Equal(
				t,
				test.expectedConfig.DefaultRequestTimeoutSeconds,
				config.DefaultRequestTimeoutSeconds,
			)

			for listenAddress, server := range test.expectedConfig.Servers {
				require.Equal(
					t,
					server.ListenAddress,
					config.Servers[listenAddress].ListenAddress,
				)

				require.Equal(
					t,
					server.ServerType,
					config.Servers[listenAddress].ServerType,
				)

				for supplierOperatorName, supplier := range server.SupplierConfigsMap {
					require.Equal(
						t,
						supplier.ServiceId,
						config.Servers[listenAddress].SupplierConfigsMap[supplierOperatorName].ServiceId,
					)

					require.Equal(
						t,
						supplier.ServerType,
						config.Servers[listenAddress].SupplierConfigsMap[supplierOperatorName].ServerType,
					)

					require.Equal(
						t,
						supplier.DefaultServiceConfig.BackendUrl.String(),
						config.Servers[listenAddress].SupplierConfigsMap[supplierOperatorName].DefaultServiceConfig.BackendUrl.String(),
					)

					require.Equal(
						t,
						supplier.RequestTimeoutSeconds,
						config.Servers[listenAddress].SupplierConfigsMap[supplierOperatorName].RequestTimeoutSeconds,
					)

					if supplier.DefaultServiceConfig.Authentication != nil {
						require.NotNil(
							t,
							config.Servers[listenAddress].SupplierConfigsMap[supplierOperatorName].DefaultServiceConfig.Authentication,
						)

						require.Equal(
							t,
							supplier.DefaultServiceConfig.Authentication.Username,
							config.Servers[listenAddress].SupplierConfigsMap[supplierOperatorName].DefaultServiceConfig.Authentication.Username,
						)

						require.Equal(
							t,
							supplier.DefaultServiceConfig.Authentication.Password,
							config.Servers[listenAddress].SupplierConfigsMap[supplierOperatorName].DefaultServiceConfig.Authentication.Password,
						)
					}

					for headerKey, headerValue := range supplier.DefaultServiceConfig.Headers {
						require.Equal(
							t,
							headerValue,
							config.Servers[listenAddress].SupplierConfigsMap[supplierOperatorName].DefaultServiceConfig.Headers[headerKey],
						)
					}

					// Test RPCTypeServiceConfigs if they exist
					if len(supplier.RPCTypeServiceConfigs) > 0 {
						require.Equal(
							t,
							len(supplier.RPCTypeServiceConfigs),
							len(config.Servers[listenAddress].SupplierConfigsMap[supplierOperatorName].RPCTypeServiceConfigs),
						)

						for rpcType, rpcServiceConfig := range supplier.RPCTypeServiceConfigs {
							actualRpcServiceConfig := config.Servers[listenAddress].SupplierConfigsMap[supplierOperatorName].RPCTypeServiceConfigs[rpcType]
							require.NotNil(t, actualRpcServiceConfig)

							require.Equal(
								t,
								rpcServiceConfig.BackendUrl.String(),
								actualRpcServiceConfig.BackendUrl.String(),
							)

							if rpcServiceConfig.Authentication != nil {
								require.NotNil(t, actualRpcServiceConfig.Authentication)
								require.Equal(
									t,
									rpcServiceConfig.Authentication.Username,
									actualRpcServiceConfig.Authentication.Username,
								)
								require.Equal(
									t,
									rpcServiceConfig.Authentication.Password,
									actualRpcServiceConfig.Authentication.Password,
								)
							}

							for headerKey, headerValue := range rpcServiceConfig.Headers {
								require.Equal(
									t,
									headerValue,
									actualRpcServiceConfig.Headers[headerKey],
								)
							}
						}
					}
				}
			}
		})
	}
}
