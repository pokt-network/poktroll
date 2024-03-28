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

		expectedErr    *sdkerrors.Error
		expectedConfig *config.RelayMinerConfig
	}{
		// Valid Configs
		{
			desc: "valid: relay miner config",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:36657
				  query_node_grpc_url: tcp://127.0.0.1:36658
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				signing_key_name: supplier1
				smt_store_path: smt_stores
				proxies:
				  - proxy_name: http-example
				    host: 127.0.0.1:8080
				    type: http
				suppliers:
				  - service_id: ethereum
				    type: http
				    service_config:
				      url: http://anvil.servicer:8545
				      authentication:
				        username: user
				        password: pwd
				      headers: {}
				    hosts:
				      - tcp://ethereum.devnet1.poktroll.com
				      - tcp://ethereum
				    proxy_names:
				      - http-example
				`,

			expectedErr: nil,
			expectedConfig: &config.RelayMinerConfig{
				PocketNode: &config.RelayMinerPocketNodeConfig{
					QueryNodeRPCUrl:  &url.URL{Scheme: "tcp", Host: "127.0.0.1:36657"},
					QueryNodeGRPCUrl: &url.URL{Scheme: "tcp", Host: "127.0.0.1:36658"},
					TxNodeRPCUrl:     &url.URL{Scheme: "tcp", Host: "127.0.0.1:36659"},
				},
				SigningKeyName: "supplier1",
				SmtStorePath:   "smt_stores",
				Proxies: map[string]*config.RelayMinerProxyConfig{
					"http-example": {
						ProxyName:            "http-example",
						Host:                 "127.0.0.1:8080",
						Type:                 config.ProxyTypeHTTP,
						XForwardedHostLookup: false,
						Suppliers: map[string]*config.RelayMinerSupplierConfig{
							"ethereum": {
								ServiceId: "ethereum",
								Type:      config.ProxyTypeHTTP,
								ServiceConfig: &config.RelayMinerSupplierServiceConfig{
									Url: &url.URL{Scheme: "http", Host: "anvil.servicer:8545"},
									Authentication: &config.RelayMinerSupplierServiceAuthentication{
										Username: "user",
										Password: "pwd",
									},
									Headers: map[string]string{},
								},
								Hosts: []string{
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
			desc: "valid: multiple suppliers, single proxy",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:36657
				  query_node_grpc_url: tcp://127.0.0.1:36658
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				signing_key_name: supplier1
				smt_store_path: smt_stores
				proxies:
				  - proxy_name: http-example
				    host: 127.0.0.1:8080
				    type: http
				suppliers:
				  - service_id: ethereum
				    type: http
				    service_config:
				      url: http://anvil.servicer:8545
				      authentication:
				        username: user
				        password: pwd
				      headers: {}
				    hosts:
				      - tcp://ethereum.devnet1.poktroll.com
				    proxy_names:
				      - http-example
				  - service_id: 7b-llm-model
				    type: http
				    service_config:
				      url: http://llama-endpoint
				    hosts:
				      - tcp://7b-llm-model.devnet1.poktroll.com
				      - tcp://7b-llm-model
				    proxy_names:
				      - http-example
				`,

			expectedErr: nil,
			expectedConfig: &config.RelayMinerConfig{
				PocketNode: &config.RelayMinerPocketNodeConfig{
					QueryNodeRPCUrl:  &url.URL{Scheme: "tcp", Host: "127.0.0.1:36657"},
					QueryNodeGRPCUrl: &url.URL{Scheme: "tcp", Host: "127.0.0.1:36658"},
					TxNodeRPCUrl:     &url.URL{Scheme: "tcp", Host: "127.0.0.1:36659"},
				},
				SigningKeyName: "supplier1",
				SmtStorePath:   "smt_stores",
				Proxies: map[string]*config.RelayMinerProxyConfig{
					"http-example": {
						ProxyName:            "http-example",
						Host:                 "127.0.0.1:8080",
						Type:                 config.ProxyTypeHTTP,
						XForwardedHostLookup: false,
						Suppliers: map[string]*config.RelayMinerSupplierConfig{
							"ethereum": {
								ServiceId: "ethereum",
								Type:      config.ProxyTypeHTTP,
								ServiceConfig: &config.RelayMinerSupplierServiceConfig{
									Url: &url.URL{Scheme: "http", Host: "anvil.servicer:8545"},
									Authentication: &config.RelayMinerSupplierServiceAuthentication{
										Username: "user",
										Password: "pwd",
									},
									Headers: map[string]string{},
								},
								Hosts: []string{
									"ethereum.devnet1.poktroll.com",
									"ethereum",
								},
							},
							"7b-llm-model": {
								ServiceId: "7b-llm-model",
								Type:      config.ProxyTypeHTTP,
								ServiceConfig: &config.RelayMinerSupplierServiceConfig{
									Url: &url.URL{Scheme: "http", Host: "llama-endpoint"},
								},
								Hosts: []string{
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
			desc: "valid: multiple proxies for a single supplier, no auth",

			inputConfigYAML: `
			  pocket_node:
			    query_node_rpc_url: tcp://127.0.0.1:36657
			    query_node_grpc_url: tcp://127.0.0.1:36658
			    tx_node_rpc_url: tcp://127.0.0.1:36659
			  signing_key_name: supplier1
			  smt_store_path: smt_stores
			  proxies:
			    - proxy_name: first-proxy
			      host: 127.0.0.1:8080
			      type: http
			    - proxy_name: second-proxy
			      host: 127.0.0.1:8081
			      type: http
			  suppliers:
			    - service_id: ethereum
			      type: http
			      service_config:
			        url: http://anvil.servicer:8545
			      hosts:
			        - tcp://ethereum.devnet1.poktroll.com
			      proxy_names:
			        - first-proxy
			        - second-proxy
			  `,

			expectedErr: nil,
			expectedConfig: &config.RelayMinerConfig{
				PocketNode: &config.RelayMinerPocketNodeConfig{
					QueryNodeRPCUrl:  &url.URL{Scheme: "tcp", Host: "127.0.0.1:36657"},
					QueryNodeGRPCUrl: &url.URL{Scheme: "tcp", Host: "127.0.0.1:36658"},
					TxNodeRPCUrl:     &url.URL{Scheme: "tcp", Host: "127.0.0.1:36659"},
				},
				SigningKeyName: "supplier1",
				SmtStorePath:   "smt_stores",
				Proxies: map[string]*config.RelayMinerProxyConfig{
					"first-proxy": {
						ProxyName:            "first-proxy",
						Host:                 "127.0.0.1:8080",
						Type:                 config.ProxyTypeHTTP,
						XForwardedHostLookup: false,
						Suppliers: map[string]*config.RelayMinerSupplierConfig{
							"ethereum": {
								ServiceId: "ethereum",
								Type:      config.ProxyTypeHTTP,
								ServiceConfig: &config.RelayMinerSupplierServiceConfig{
									Url: &url.URL{Scheme: "http", Host: "anvil.servicer:8545"},
								},
								Hosts: []string{
									"ethereum.devnet1.poktroll.com",
								},
							},
						},
					},
					"second-proxy": {
						ProxyName:            "second-proxy",
						Host:                 "127.0.0.1:8081",
						Type:                 config.ProxyTypeHTTP,
						XForwardedHostLookup: false,
						Suppliers: map[string]*config.RelayMinerSupplierConfig{
							"ethereum": {
								ServiceId: "ethereum",
								Type:      config.ProxyTypeHTTP,
								ServiceConfig: &config.RelayMinerSupplierServiceConfig{
									Url: &url.URL{Scheme: "http", Host: "anvil.servicer:8545"},
								},
								Hosts: []string{
									"ethereum.devnet1.poktroll.com",
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
				  query_node_grpc_url: tcp://127.0.0.1:36658
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				signing_key_name: supplier1
				smt_store_path: smt_stores
				proxies:
				  - proxy_name: http-example
				    host: 127.0.0.1:8080
				    type: http
				suppliers:
				  - service_id: ethereum
				    type: http
				    service_config:
				      url: http://anvil.servicer:8545
				    hosts:
				      - tcp://ethereum.devnet1.poktroll.com
				      - tcp://ethereum
				    proxy_names:
				      - http-example
				`,

			expectedErr: nil,
			expectedConfig: &config.RelayMinerConfig{
				PocketNode: &config.RelayMinerPocketNodeConfig{
					QueryNodeRPCUrl:  &url.URL{Scheme: "tcp", Host: "127.0.0.1:36659"},
					QueryNodeGRPCUrl: &url.URL{Scheme: "tcp", Host: "127.0.0.1:36658"},
					TxNodeRPCUrl:     &url.URL{Scheme: "tcp", Host: "127.0.0.1:36659"},
				},
				SigningKeyName: "supplier1",
				SmtStorePath:   "smt_stores",
				Proxies: map[string]*config.RelayMinerProxyConfig{
					"http-example": {
						ProxyName:            "http-example",
						Host:                 "127.0.0.1:8080",
						Type:                 config.ProxyTypeHTTP,
						XForwardedHostLookup: false,
						Suppliers: map[string]*config.RelayMinerSupplierConfig{
							"ethereum": {
								ServiceId: "ethereum",
								Type:      config.ProxyTypeHTTP,
								ServiceConfig: &config.RelayMinerSupplierServiceConfig{
									Url: &url.URL{Scheme: "http", Host: "anvil.servicer:8545"},
								},
								Hosts: []string{
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
				  query_node_rpc_url: tcp://127.0.0.1:36657
				  query_node_grpc_url: tcp://127.0.0.1:36658
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				signing_key_name: supplier1
				smt_store_path: smt_stores
				proxies:
				  - proxy_name: http-example
				    host: 127.0.0.1:8080
				    type: http
				    x_forwarded_host_lookup: true
				suppliers:
				  - service_id: ethereum
				    type: http
				    service_config:
				      url: http://anvil.servicer:8545
				    hosts:
				      - tcp://ethereum.devnet1.poktroll.com
				      - tcp://ethereum
				    proxy_names:
				      - http-example
				`,

			expectedErr: nil,
			expectedConfig: &config.RelayMinerConfig{
				PocketNode: &config.RelayMinerPocketNodeConfig{
					QueryNodeRPCUrl:  &url.URL{Scheme: "tcp", Host: "127.0.0.1:36657"},
					QueryNodeGRPCUrl: &url.URL{Scheme: "tcp", Host: "127.0.0.1:36658"},
					TxNodeRPCUrl:     &url.URL{Scheme: "tcp", Host: "127.0.0.1:36659"},
				},
				SigningKeyName: "supplier1",
				SmtStorePath:   "smt_stores",
				Proxies: map[string]*config.RelayMinerProxyConfig{
					"http-example": {
						ProxyName:            "http-example",
						Host:                 "127.0.0.1:8080",
						Type:                 config.ProxyTypeHTTP,
						XForwardedHostLookup: true,
						Suppliers: map[string]*config.RelayMinerSupplierConfig{
							"ethereum": {
								ServiceId: "ethereum",
								Type:      config.ProxyTypeHTTP,
								ServiceConfig: &config.RelayMinerSupplierServiceConfig{
									Url: &url.URL{Scheme: "http", Host: "anvil.servicer:8545"},
								},
								Hosts: []string{
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
				  query_node_rpc_url: tcp://127.0.0.1:36657
				  query_node_grpc_url: tcp://127.0.0.1:36658
				  tx_node_rpc_url: &tcp://127.0.0.1:36659
				signing_key_name: supplier1
				smt_store_path: smt_stores
				proxies:
				  - proxy_name: http-example
				    host: 127.0.0.1:8080
				    type: http
				suppliers:
				  - service_id: ethereum
				    type: http
				    service_config:
				      url: http://anvil.servicer:8545
				    hosts:
				      - tcp://ethereum.devnet1.poktroll.com
				    proxy_names:
				      - http-example
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidNodeUrl,
		},
		{
			desc: "invalid: missing tx node grpc url",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:36657
				  # explicitly omitted tx node grpc url
				  query_node_grpc_url: tcp://127.0.0.1:36658
				signing_key_name: supplier1
				smt_store_path: smt_stores
				proxies:
				  - proxy_name: http-example
				    host: 127.0.0.1:8080
				    type: http
				suppliers:
				  - service_id: ethereum
				    type: http
				    service_config:
				      url: http://anvil.servicer:8545
				    hosts:
				      - tcp://ethereum.devnet1.poktroll.com
				    proxy_names:
				      - http-example
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidNodeUrl,
		},
		{
			desc: "invalid: invalid query node grpc url",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:36657
				  query_node_grpc_url: &tcp://127.0.0.1:36658
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				signing_key_name: supplier1
				smt_store_path: smt_stores
				proxies:
				  - proxy_name: http-example
				    host: 127.0.0.1:8080
				    type: http
				suppliers:
				  - service_id: ethereum
				    type: http
				    service_config:
				      url: http://anvil.servicer:8545
				    hosts:
				      - tcp://ethereum.devnet1.poktroll.com
				    proxy_names:
				      - http-example
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidNodeUrl,
		},
		{
			desc: "invalid: invalid query node rpc url",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: &tcp://127.0.0.1:36657
				  query_node_grpc_url: tcp://127.0.0.1:36658
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				signing_key_name: supplier1
				smt_store_path: smt_stores
				proxies:
				  - proxy_name: http-example
				    host: 127.0.0.1:8080
				    type: http
				suppliers:
				  - service_id: ethereum
				    type: http
				    service_config:
				      url: http://anvil.servicer:8545
				    hosts:
				      - tcp://ethereum.devnet1.poktroll.com
				    proxy_names:
				      - http-example
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidNodeUrl,
		},
		{
			desc: "invalid: missing query node grpc url",

			inputConfigYAML: `
				pocket_node:
				  # explicitly omitted query node rpc url
				  query_node_rpc_url: tcp://127.0.0.1:36657
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				signing_key_name: supplier1
				smt_store_path: smt_stores
				proxies:
				  - proxy_name: http-example
				    host: 127.0.0.1:8080
				    type: http
				suppliers:
				  - service_id: ethereum
				    type: http
				    service_config:
				      url: http://anvil.servicer:8545
				    hosts:
				      - tcp://ethereum.devnet1.poktroll.com
				    proxy_names:
				      - http-example
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidNodeUrl,
		},
		{
			desc: "invalid: missing signing key name",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:36657
				  query_node_grpc_url: tcp://127.0.0.1:36658
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				# explicitly omitted signing key name
				smt_store_path: smt_stores
				proxies:
				  - proxy_name: http-example
				    host: 127.0.0.1:8080
				    type: http
				suppliers:
				  - service_id: ethereum
				    type: http
				    service_config:
				      url: http://anvil.servicer:8545
				    hosts:
				      - tcp://ethereum.devnet1.poktroll.com
				    proxy_names:
				      - http-example
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidSigningKeyName,
		},
		{
			desc: "invalid: missing smt store path",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:36657
				  query_node_grpc_url: tcp://127.0.0.1:36658
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				signing_key_name: supplier1
				# explicitly omitted smt store path
				proxies:
				  - proxy_name: http-example
				    host: 127.0.0.1:8080
				    type: http
				suppliers:
				  - service_id: ethereum
				    type: http
				    service_config:
				      url: http://anvil.servicer:8545
				    hosts:
				      - tcp://ethereum.devnet1.poktroll.com
				    proxy_names:
				      - http-example
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidSmtStorePath,
		},
		{
			desc: "invalid: missing proxies section",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:36657
				  query_node_grpc_url: tcp://127.0.0.1:36658
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				signing_key_name: supplier1
				smt_store_path: smt_stores
				# explicitly omitted proxies section
				suppliers:
				  - proxy_name: ethereum
				    type: http
				    service_config:
				      url: http://anvil.servicer:8545
				    hosts:
				      - tcp://ethereum.devnet1.poktroll.com
				    proxy_names:
				      - http-example
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidProxy,
		},
		{
			desc: "invalid: empty proxies section",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:36657
				  query_node_grpc_url: tcp://127.0.0.1:36658
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				signing_key_name: supplier1
				smt_store_path: smt_stores
				proxies: # explicitly empty proxies section
				suppliers:
				  - service_id: ethereum
				    type: http
				    service_config:
				      url: http://anvil.servicer:8545
				    hosts:
				      - tcp://ethereum.devnet1.poktroll.com
				    proxy_names:
				      - http-example
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidProxy,
		},
		{
			desc: "invalid: omitted proxy name",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:36657
				  query_node_grpc_url: tcp://127.0.0.1:36658
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				signing_key_name: supplier1
				smt_store_path: smt_stores
				proxies:
				  # explicitly omitted proxy name
				  - host: 127.0.0.1:8080
				    type: http
				suppliers:
				  - service_id: ethereum
				    type: http
				    service_config:
				      url: http://anvil.servicer:8545
				    hosts:
				      - tcp://ethereum.devnet1.poktroll.com
				    proxy_names:
				      - http-example
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidProxy,
		},
		{
			desc: "invalid: empty proxy name",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:36657
				  query_node_grpc_url: tcp://127.0.0.1:36658
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				signing_key_name: supplier1
				smt_store_path: smt_stores
				proxies:
				  - proxy_name: # explicitly empty proxy name
				    host: 127.0.0.1:8080
				    type: http
				suppliers:
				  - service_id: ethereum
				    type: http
				    service_config:
				      url: http://anvil.servicer:8545
				    hosts:
				      - tcp://ethereum.devnet1.poktroll.com
				    proxy_names:
				      - http-example
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidProxy,
		},
		{
			desc: "invalid: missing http proxy host",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:36657
				  query_node_grpc_url: tcp://127.0.0.1:36658
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				signing_key_name: supplier1
				smt_store_path: smt_stores
				proxies:
				  - proxy_name: http-example
				    # explicitly missing proxy host
				    type: http
				suppliers:
				  - service_id: ethereum
				    type: http
				    service_config:
				      url: http://anvil.servicer:8545
				    hosts:
				      - tcp://ethereum.devnet1.poktroll.com
				    proxy_names:
				      - http-example
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidProxy,
		},
		{
			desc: "invalid: empty http proxy host",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:36657
				  query_node_grpc_url: tcp://127.0.0.1:36658
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				signing_key_name: supplier1
				smt_store_path: smt_stores
				proxies:
				  - proxy_name: http-example
				    host: # explicitly empty proxy host
				    type: http
				suppliers:
				  - service_id: ethereum
				    type: http
				    service_config:
				      url: http://anvil.servicer:8545
				    hosts:
				      - tcp://ethereum.devnet1.poktroll.com
				    proxy_names:
				      - http-example
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidProxy,
		},
		{
			desc: "invalid: missing proxy type",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:36657
				  query_node_grpc_url: tcp://127.0.0.1:36658
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				signing_key_name: supplier1
				smt_store_path: smt_stores
				proxies:
				  - proxy_name: http-example
				    host: 127.0.0.1:8080
				    # explicitly missing proxy type
				suppliers:
				  - service_id: ethereum
				    type: http
				    service_config:
				      url: http://anvil.servicer:8545
				    hosts:
				      - tcp://ethereum.devnet1.poktroll.com
				    proxy_names:
				      - http-example
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidProxy,
		},
		{
			desc: "invalid: empty proxy type",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:36657
				  query_node_grpc_url: tcp://127.0.0.1:36658
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				signing_key_name: supplier1
				smt_store_path: smt_stores
				proxies:
				  - proxy_name: http-example
				    host: 127.0.0.1:8080
				    type: # explicitly empty proxy type
				suppliers:
				  - service_id: ethereum
				    type: http
				    service_config:
				      url: http://anvil.servicer:8545
				    hosts:
				      - tcp://ethereum.devnet1.poktroll.com
				    proxy_names:
				      - http-example
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidProxy,
		},
		{
			desc: "invalid: unsupported proxy type",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:36657
				  query_node_grpc_url: tcp://127.0.0.1:36658
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				signing_key_name: supplier1
				smt_store_path: smt_stores
				proxies:
				  - proxy_name: http-example
				    host: 127.0.0.1:8080
				    type: unsupported
				suppliers:
				  - service_id: ethereum
				    type: http
				    service_config:
				      url: http://anvil.servicer:8545
				    hosts:
				      - tcp://ethereum.devnet1.poktroll.com
				    proxy_names:
				      - http-example
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidProxy,
		},
		{
			desc: "invalid: missing supplier name",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:36657
				  query_node_grpc_url: tcp://127.0.0.1:36658
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				signing_key_name: supplier1
				smt_store_path: smt_stores
				proxies:
				  - proxy_name: http-example
				    host: 127.0.0.1:8080
				    type: http
				suppliers:
				  # explicitly missing supplier name
				  - type: http
				    service_config:
				      url: http://anvil.servicer:8545
				    hosts:
				      - tcp://ethereum.devnet1.poktroll.com
				    proxy_names:
				      - http-example
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidSupplier,
		},
		{
			desc: "invalid: empty supplier name",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:36657
				  query_node_grpc_url: tcp://127.0.0.1:36658
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				signing_key_name: supplier1
				smt_store_path: smt_stores
				proxies:
				  - proxy_name: http-example
				    host: 127.0.0.1:8080
				    type: http
				suppliers:
				  - service_id: # explicitly empty supplier name
				    type: http
				    service_config:
				      url: http://anvil.servicer:8545
				    hosts:
				      - tcp://ethereum.devnet1.poktroll.com
				    proxy_names:
				      - http-example
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidSupplier,
		},
		{
			desc: "invalid: unsupported supplier type",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:36657
				  query_node_grpc_url: tcp://127.0.0.1:36658
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				signing_key_name: supplier1
				smt_store_path: smt_stores
				proxies:
				  - proxy_name: http-example
				    host: 127.0.0.1:8080
				    type: http
				suppliers:
				  - service_id: ethereum
				    type: unsupported
				    service_config:
				      url: http://anvil.servicer:8545
				    hosts:
				      - tcp://ethereum.devnet1.poktroll.com
				    proxy_names:
				      - http-example
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidSupplier,
		},
		{
			desc: "invalid: missing supplier type",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:36657
				  query_node_grpc_url: tcp://127.0.0.1:36658
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				signing_key_name: supplier1
				smt_store_path: smt_stores
				proxies:
				  - proxy_name: http-example
				    host: 127.0.0.1:8080
				    type: http
				suppliers:
				  - service_id: ethereum
				    # explicitly missing supplier type
				    service_config:
				      url: http://anvil.servicer:8545
				    hosts:
				      - tcp://ethereum.devnet1.poktroll.com
				    proxy_names:
				      - http-example
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidSupplier,
		},
		{
			desc: "invalid: empty supplier type",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:36657
				  query_node_grpc_url: tcp://127.0.0.1:36658
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				signing_key_name: supplier1
				smt_store_path: smt_stores
				proxies:
				  - proxy_name: http-example
				    host: 127.0.0.1:8080
				    type: http
				suppliers:
				  - service_id: ethereum
				    type: # explicitly empty supplier type
				    service_config:
				      url: http://anvil.servicer:8545
				    hosts:
				      - tcp://ethereum.devnet1.poktroll.com
				    proxy_names:
				      - http-example
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidSupplier,
		},
		{
			desc: "invalid: bad supplier service config url",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:36657
				  query_node_grpc_url: tcp://127.0.0.1:36658
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				signing_key_name: supplier1
				smt_store_path: smt_stores
				proxies:
				  - proxy_name: http-example
				    host: 127.0.0.1:8080
				    type: http
				suppliers:
				  - service_id: ethereum
				    type: http
				    service_config:
				      url: &http://anvil.servicer:8545
				    hosts:
				      - tcp://ethereum.devnet1.poktroll.com
				    proxy_names:
				      - http-example
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidSupplier,
		},
		{
			desc: "invalid: empty supplier service config url",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:36657
				  query_node_grpc_url: tcp://127.0.0.1:36658
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				signing_key_name: supplier1
				smt_store_path: smt_stores
				proxies:
				  - proxy_name: http-example
				    host: 127.0.0.1:8080
				    type: http
				suppliers:
				  - service_id: ethereum
				    type: http
				    service_config:
				      url: # explicitly empty supplier service config url
				    hosts:
				      - tcp://ethereum.devnet1.poktroll.com
				    proxy_names:
				      - http-example
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidSupplier,
		},
		{
			desc: "invalid: missing supplier service config url",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:36657
				  query_node_grpc_url: tcp://127.0.0.1:36658
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				signing_key_name: supplier1
				smt_store_path: smt_stores
				proxies:
				  - proxy_name: http-example
				    host: 127.0.0.1:8080
				    type: http
				suppliers:
				  - service_id: ethereum
				    type: http
				    service_config:
				      # explicitly missing supplier service config url
				    hosts:
				      - tcp://ethereum.devnet1.poktroll.com
				    proxy_names:
				      - http-example
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidSupplier,
		},
		{
			desc: "invalid: bad supplier host",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:36657
				  query_node_grpc_url: tcp://127.0.0.1:36658
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				signing_key_name: supplier1
				smt_store_path: smt_stores
				proxies:
				  - proxy_name: http-example
				    host: 127.0.0.1:8080
				    type: http
				suppliers:
				  - service_id: ethereum
				    type: http
				    service_config:
				      url: http://anvil.servicer:8545
				    hosts:
				      - &tcp://ethereum.devnet1.poktroll.com
				    proxy_names:
				      - http-example
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidSupplier,
		},
		{
			desc: "invalid: blank supplier host",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:36657
				  query_node_grpc_url: tcp://127.0.0.1:36658
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				signing_key_name: supplier1
				smt_store_path: smt_stores
				proxies:
				  - proxy_name: http-example
				    host: 127.0.0.1:8080
				    type: http
				suppliers:
				  - service_id: ethereum
				    type: http
				    service_config:
				      url: http://anvil.servicer:8545
				    hosts:
				      - # explicitly blank supplier host
				    proxy_names:
				      - http-example
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidSupplier,
		},
		{
			desc: "invalid: empty supplier proxy references",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:36657
				  query_node_grpc_url: tcp://127.0.0.1:36658
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				signing_key_name: supplier1
				smt_store_path: smt_stores
				proxies:
				  - proxy_name: http-example
				    host: 127.0.0.1:8080
				    type: http
				suppliers:
				  - service_id: ethereum
				    type: http
				    service_config:
				      url: http://anvil.servicer:8545
				    hosts:
				      - tcp://ethereum.devnet1.poktroll.com
				    proxy_names:
				      - bad-proxy-name
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidSupplier,
		},
		{
			desc: "invalid: empty supplier proxy references",

			inputConfigYAML: `
				pocket_node:
				  query_node_rpc_url: tcp://127.0.0.1:36657
				  query_node_grpc_url: tcp://127.0.0.1:36658
				  tx_node_rpc_url: tcp://127.0.0.1:36659
				signing_key_name: supplier1
				smt_store_path: smt_stores
				proxies:
				  - proxy_name: http-example
				    host: 127.0.0.1:8080
				    type: http
				suppliers:
				  - service_id: ethereum
				    type: http
				    service_config:
				      url: http://anvil.servicer:8545
				    hosts:
				      - tcp://devnet1.poktroll.com # hosts for both suppliers are the same
				    proxy_names:
				      - http-example
				  - service_id: avax
				    type: http
				    service_config:
				      url: http://avax.servicer:8545
				    hosts:
				      - tcp://devnet1.poktroll.com # hosts for both suppliers are the same
				    proxy_names:
				      - http-example
				`,

			expectedErr: config.ErrRelayMinerConfigInvalidProxy,
		},
		{
			desc: "invalid: empty RelayMiner config file",

			inputConfigYAML: ``,

			expectedErr: config.ErrRelayMinerConfigEmpty,
		},
		// TODO_NB: Test for supplier and proxy types mismatch once we have more
		// than one proxy type.
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			normalizedConfig := yaml.NormalizeYAMLIndentation(test.inputConfigYAML)
			config, err := config.ParseRelayMinerConfigs([]byte(normalizedConfig))

			if test.expectedErr != nil {
				require.ErrorIs(t, err, test.expectedErr)
				require.Nil(t, config)
				stat, ok := status.FromError(test.expectedErr)
				require.True(t, ok)
				require.Contains(t, stat.Message(), test.expectedErr.Error())
				require.Nil(t, config)
				return
			}

			require.NoError(t, err)

			require.Equal(
				t,
				test.expectedConfig.SigningKeyName,
				config.SigningKeyName,
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

			for proxyName, proxy := range test.expectedConfig.Proxies {
				require.Equal(
					t,
					proxy.ProxyName,
					config.Proxies[proxyName].ProxyName,
				)

				require.Equal(
					t,
					proxy.Host,
					config.Proxies[proxyName].Host,
				)

				require.Equal(
					t,
					proxy.Type,
					config.Proxies[proxyName].Type,
				)

				for supplierName, supplier := range proxy.Suppliers {
					require.Equal(
						t,
						supplier.ServiceId,
						config.Proxies[proxyName].Suppliers[supplierName].ServiceId,
					)

					require.Equal(
						t,
						supplier.Type,
						config.Proxies[proxyName].Suppliers[supplierName].Type,
					)

					require.Equal(
						t,
						supplier.ServiceConfig.Url.String(),
						config.Proxies[proxyName].Suppliers[supplierName].ServiceConfig.Url.String(),
					)

					if supplier.ServiceConfig.Authentication != nil {
						require.NotNil(
							t,
							config.Proxies[proxyName].Suppliers[supplierName].ServiceConfig.Authentication,
						)

						require.Equal(
							t,
							supplier.ServiceConfig.Authentication.Username,
							config.Proxies[proxyName].Suppliers[supplierName].ServiceConfig.Authentication.Username,
						)

						require.Equal(
							t,
							supplier.ServiceConfig.Authentication.Password,
							config.Proxies[proxyName].Suppliers[supplierName].ServiceConfig.Authentication.Password,
						)
					}

					for headerKey, headerValue := range supplier.ServiceConfig.Headers {
						require.Equal(
							t,
							headerValue,
							config.Proxies[proxyName].Suppliers[supplierName].ServiceConfig.Headers[headerKey],
						)
					}

					for i, host := range supplier.Hosts {
						require.Contains(
							t,
							host,
							config.Proxies[proxyName].Suppliers[supplierName].Hosts[i],
						)
					}
				}
			}
		})
	}
}
