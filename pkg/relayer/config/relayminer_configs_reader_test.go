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
	configContent, err := os.ReadFile("../../../localnet/poktrolld/config/relayminer_config_full_example.yaml")
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
							publicly_exposed_endpoints:
								- ethereum.devnet1.poktroll.com
								- ethereum
				`,

			expectedErr: nil,
			expectedConfig: &config.RelayMinerConfig{
				PocketNode: &config.RelayMinerPocketNodeConfig{
					QueryNodeRPCUrl:  &url.URL{Scheme: "tcp", Host: "127.0.0.1:26657"},
					QueryNodeGRPCUrl: &url.URL{Scheme: "tcp", Host: "127.0.0.1:9090"},
					TxNodeRPCUrl:     &url.URL{Scheme: "tcp", Host: "127.0.0.1:36659"},
				},
				DefaultSigningKeyNames: []string{"supplier1"},
				SmtStorePath:           "smt_stores",
				Servers: map[string]*config.RelayMinerServerConfig{
					"http://127.0.0.1:8080": {
						ListenAddress:        "127.0.0.1:8080",
						ServerType:           config.RelayMinerServerTypeHTTP,
						XForwardedHostLookup: false,
						SupplierConfigsMap: map[string]*config.RelayMinerSupplierConfig{
							"ethereum": {
								ServiceId:  "ethereum",
								ServerType: config.RelayMinerServerTypeHTTP,
								ServiceConfig: &config.RelayMinerSupplierServiceConfig{
									BackendUrl: &url.URL{Scheme: "http", Host: "anvil.servicer:8545"},
									Authentication: &config.RelayMinerSupplierServiceAuthentication{
										Username: "user",
										Password: "pwd",
									},
									Headers: map[string]string{},
								},
								PubliclyExposedEndpoints: []string{
									"ethereum.devnet1.poktroll.com",
									"ethereum",
								},
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
                      publicly_exposed_endpoints:
                        - ethereum.devnet1.poktroll.com
                        - ethereum
                `,

			expectedErr: nil,
			expectedConfig: &config.RelayMinerConfig{
				PocketNode: &config.RelayMinerPocketNodeConfig{
					QueryNodeRPCUrl:  &url.URL{Scheme: "tcp", Host: "127.0.0.1:36657"},
					QueryNodeGRPCUrl: &url.URL{Scheme: "tcp", Host: "127.0.0.1:36658"},
					TxNodeRPCUrl:     &url.URL{Scheme: "tcp", Host: "127.0.0.1:36659"},
				},
				SmtStorePath: "smt_stores",
				Servers: map[string]*config.RelayMinerServerConfig{
					"http://127.0.0.1:8080": {
						ListenAddress:        "127.0.0.1:8080",
						ServerType:           config.RelayMinerServerTypeHTTP,
						XForwardedHostLookup: false,
						SupplierConfigsMap: map[string]*config.RelayMinerSupplierConfig{
							"ethereum": {
								ServiceId:  "ethereum",
								ServerType: config.RelayMinerServerTypeHTTP,
								ServiceConfig: &config.RelayMinerSupplierServiceConfig{
									BackendUrl: &url.URL{Scheme: "http", Host: "anvil.servicer:8545"},
									Authentication: &config.RelayMinerSupplierServiceAuthentication{
										Username: "user",
										Password: "pwd",
									},
									Headers: map[string]string{},
								},
								PubliclyExposedEndpoints: []string{
									"ethereum.devnet1.poktroll.com",
									"ethereum",
								},
								SigningKeyNames: []string{"supplier1"},
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
					publicly_exposed_endpoints:
						- ethereum.devnet1.poktroll.com
						- ethereum
			- service_id: ollama
				listen_url: http://127.0.0.1:8080
				signing_key_names: [supplier2]
				service_config:
					backend_url: http://ollama.servicer:8545
					authentication:
						username: user
						password: pwd
					headers: {}
					publicly_exposed_endpoints:
						- ollama.devnet1.poktroll.com
						- ollama
                `,

			expectedErr: nil,
			expectedConfig: &config.RelayMinerConfig{
				PocketNode: &config.RelayMinerPocketNodeConfig{
					QueryNodeRPCUrl:  &url.URL{Scheme: "tcp", Host: "127.0.0.1:36657"},
					QueryNodeGRPCUrl: &url.URL{Scheme: "tcp", Host: "127.0.0.1:36658"},
					TxNodeRPCUrl:     &url.URL{Scheme: "tcp", Host: "127.0.0.1:36659"},
				},
				SmtStorePath:           "smt_stores",
				DefaultSigningKeyNames: []string{"supplier1"},
				Servers: map[string]*config.RelayMinerServerConfig{
					"http://127.0.0.1:8080": {
						ListenAddress:        "127.0.0.1:8080",
						ServerType:           config.RelayMinerServerTypeHTTP,
						XForwardedHostLookup: false,
						SupplierConfigsMap: map[string]*config.RelayMinerSupplierConfig{
							"ethereum": {
								ServiceId:  "ethereum",
								ServerType: config.RelayMinerServerTypeHTTP,
								ServiceConfig: &config.RelayMinerSupplierServiceConfig{
									BackendUrl: &url.URL{Scheme: "http", Host: "anvil.servicer:8545"},
									Authentication: &config.RelayMinerSupplierServiceAuthentication{
										Username: "user",
										Password: "pwd",
									},
									Headers: map[string]string{},
								},
								PubliclyExposedEndpoints: []string{
									"ethereum.devnet1.poktroll.com",
									"ethereum",
								},
								// Note the supplier is missing in the yaml, but it is populated from
								// the global `default_signing_key_names`
								SigningKeyNames: []string{"supplier1"},
							},
							"ollama": {
								ServiceId:  "ollama",
								ServerType: config.RelayMinerServerTypeHTTP,
								ServiceConfig: &config.RelayMinerSupplierServiceConfig{
									BackendUrl: &url.URL{Scheme: "http", Host: "ollama.servicer:8545"},
									Authentication: &config.RelayMinerSupplierServiceAuthentication{
										Username: "user",
										Password: "pwd",
									},
									Headers: map[string]string{},
								},
								PubliclyExposedEndpoints: []string{
									"ollama.devnet1.poktroll.com",
									"ollama",
								},
								SigningKeyNames: []string{"supplier2"},
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
							publicly_exposed_endpoints:
								- ethereum.devnet1.poktroll.com
				  - service_id: 7b-llm-model
				    listen_url: http://127.0.0.1:8080
				    service_config:
				      backend_url: http://llama-endpoint
							publicly_exposed_endpoints:
								- 7b-llm-model.devnet1.poktroll.com
								- 7b-llm-model

				`,

			expectedErr: nil,
			expectedConfig: &config.RelayMinerConfig{
				PocketNode: &config.RelayMinerPocketNodeConfig{
					QueryNodeRPCUrl:  &url.URL{Scheme: "tcp", Host: "127.0.0.1:26657"},
					QueryNodeGRPCUrl: &url.URL{Scheme: "tcp", Host: "127.0.0.1:9090"},
					TxNodeRPCUrl:     &url.URL{Scheme: "tcp", Host: "127.0.0.1:36659"},
				},
				DefaultSigningKeyNames: []string{"supplier1"},
				SmtStorePath:           "smt_stores",
				Servers: map[string]*config.RelayMinerServerConfig{
					"http://127.0.0.1:8080": {
						ListenAddress:        "127.0.0.1:8080",
						ServerType:           config.RelayMinerServerTypeHTTP,
						XForwardedHostLookup: false,
						SupplierConfigsMap: map[string]*config.RelayMinerSupplierConfig{
							"ethereum": {
								ServiceId:  "ethereum",
								ServerType: config.RelayMinerServerTypeHTTP,
								ServiceConfig: &config.RelayMinerSupplierServiceConfig{
									BackendUrl: &url.URL{Scheme: "http", Host: "anvil.servicer:8545"},
									Authentication: &config.RelayMinerSupplierServiceAuthentication{
										Username: "user",
										Password: "pwd",
									},
									Headers: map[string]string{},
								},
								PubliclyExposedEndpoints: []string{
									"ethereum.devnet1.poktroll.com",
									"ethereum",
								},
							},
							"7b-llm-model": {
								ServiceId:  "7b-llm-model",
								ServerType: config.RelayMinerServerTypeHTTP,
								ServiceConfig: &config.RelayMinerSupplierServiceConfig{
									BackendUrl: &url.URL{Scheme: "http", Host: "llama-endpoint"},
								},
								PubliclyExposedEndpoints: []string{
									"7b-llm-model.devnet1.poktroll.com",
									"7b-llm-model",
								},
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
							publicly_exposed_endpoints:
								- ethereum.devnet1.poktroll.com
								- ethereum

				`,

			expectedErr: nil,
			expectedConfig: &config.RelayMinerConfig{
				PocketNode: &config.RelayMinerPocketNodeConfig{
					QueryNodeRPCUrl:  &url.URL{Scheme: "tcp", Host: "127.0.0.1:36659"},
					QueryNodeGRPCUrl: &url.URL{Scheme: "tcp", Host: "127.0.0.1:9090"},
					TxNodeRPCUrl:     &url.URL{Scheme: "tcp", Host: "127.0.0.1:36659"},
				},
				DefaultSigningKeyNames: []string{"supplier1"},
				SmtStorePath:           "smt_stores",
				Servers: map[string]*config.RelayMinerServerConfig{
					"http://127.0.0.1:8080": {
						ListenAddress:        "127.0.0.1:8080",
						ServerType:           config.RelayMinerServerTypeHTTP,
						XForwardedHostLookup: false,
						SupplierConfigsMap: map[string]*config.RelayMinerSupplierConfig{
							"ethereum": {
								ServiceId:  "ethereum",
								ServerType: config.RelayMinerServerTypeHTTP,
								ServiceConfig: &config.RelayMinerSupplierServiceConfig{
									BackendUrl: &url.URL{Scheme: "http", Host: "anvil.servicer:8545"},
								},
								PubliclyExposedEndpoints: []string{
									"ethereum.devnet1.poktroll.com",
									"ethereum",
								},
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
							publicly_exposed_endpoints:
								- ethereum.devnet1.poktroll.com
				`,

			expectedErr: nil,
			expectedConfig: &config.RelayMinerConfig{
				PocketNode: &config.RelayMinerPocketNodeConfig{
					QueryNodeRPCUrl:  &url.URL{Scheme: "tcp", Host: "127.0.0.1:26657"},
					QueryNodeGRPCUrl: &url.URL{Scheme: "tcp", Host: "127.0.0.1:9090"},
					TxNodeRPCUrl:     &url.URL{Scheme: "tcp", Host: "127.0.0.1:36659"},
				},
				DefaultSigningKeyNames: []string{"supplier1"},
				SmtStorePath:           "smt_stores",
				Servers: map[string]*config.RelayMinerServerConfig{
					"http://127.0.0.1:8080": {
						ListenAddress:        "127.0.0.1:8080",
						ServerType:           config.RelayMinerServerTypeHTTP,
						XForwardedHostLookup: true,
						SupplierConfigsMap: map[string]*config.RelayMinerSupplierConfig{
							"ethereum": {
								ServiceId:  "ethereum",
								ServerType: config.RelayMinerServerTypeHTTP,
								ServiceConfig: &config.RelayMinerSupplierServiceConfig{
									BackendUrl: &url.URL{Scheme: "http", Host: "anvil.servicer:8545"},
								},
								PubliclyExposedEndpoints: []string{
									"ethereum.devnet1.poktroll.com",
									"ethereum",
								},
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
							publicly_exposed_endpoints:
								- ethereum.devnet1.poktroll.com
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
							publicly_exposed_endpoints:
								- ethereum.devnet1.poktroll.com
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
							publicly_exposed_endpoints:
								- ethereum.devnet1.poktroll.com

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
							publicly_exposed_endpoints:
								- ethereum.devnet1.poktroll.com

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
							publicly_exposed_endpoints:
								- ethereum.devnet1.poktroll.com

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
							publicly_exposed_endpoints:
								- ethereum.devnet1.poktroll.com

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
							publicly_exposed_endpoints:
								- ethereum.devnet1.poktroll.com

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
							publicly_exposed_endpoints:
								- ethereum.devnet1.poktroll.com

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
							publicly_exposed_endpoints:
								- ethereum.devnet1.poktroll.com
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
							publicly_exposed_endpoints:
								- ethereum.devnet1.poktroll.com
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
							publicly_exposed_endpoints:
								- ethereum.devnet1.poktroll.com

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
							publicly_exposed_endpoints:
								- ethereum.devnet1.poktroll.com

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
							publicly_exposed_endpoints:
								- ethereum.devnet1.poktroll.com

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
					  publicly_exposed_endpoints:
						- ethereum.devnet1.poktroll.com
					  # explicitly missing supplier service config backend url
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidSupplier,
		},
		{
			desc: "invalid: blank supplier exposed endpoint",

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
				      publicly_exposed_endpoints:
				        - # explicitly blank supplier host
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
						supplier.ServiceConfig.BackendUrl.String(),
						config.Servers[listenAddress].SupplierConfigsMap[supplierOperatorName].ServiceConfig.BackendUrl.String(),
					)

					if supplier.ServiceConfig.Authentication != nil {
						require.NotNil(
							t,
							config.Servers[listenAddress].SupplierConfigsMap[supplierOperatorName].ServiceConfig.Authentication,
						)

						require.Equal(
							t,
							supplier.ServiceConfig.Authentication.Username,
							config.Servers[listenAddress].SupplierConfigsMap[supplierOperatorName].ServiceConfig.Authentication.Username,
						)

						require.Equal(
							t,
							supplier.ServiceConfig.Authentication.Password,
							config.Servers[listenAddress].SupplierConfigsMap[supplierOperatorName].ServiceConfig.Authentication.Password,
						)
					}

					for headerKey, headerValue := range supplier.ServiceConfig.Headers {
						require.Equal(
							t,
							headerValue,
							config.Servers[listenAddress].SupplierConfigsMap[supplierOperatorName].ServiceConfig.Headers[headerKey],
						)
					}

					for i, host := range supplier.PubliclyExposedEndpoints {
						require.Contains(
							t,
							host,
							config.Servers[listenAddress].SupplierConfigsMap[supplierOperatorName].PubliclyExposedEndpoints[i],
						)
					}
				}
			}
		})
	}
}
