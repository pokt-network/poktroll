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

func Test_ParseAppGateConfigs(t *testing.T) {
	tests := []struct {
		desc     string
		err      *sdkerrors.Error
		expected *config.RelayMinerConfig
		config   string
	}{
		// Valid Configs
		{
			desc: "relayminer_config_test: valid relay miner config",
			err:  nil,
			expected: &config.RelayMinerConfig{
				QueryNodeUrl:   &url.URL{Scheme: "tcp", Host: "localhost:26657"},
				NetworkNodeUrl: &url.URL{Scheme: "tcp", Host: "127.0.0.1:36657"},
				SigningKeyName: "servicer1",
				ProxiedServiceEndpoints: map[string]*url.URL{
					"anvil": {Scheme: "http", Host: "anvil:8080"},
					"svc1":  {Scheme: "http", Host: "svc1:8080"},
				},
				SmtStorePath: "smt_stores",
			},
			config: `
				query_node_url: tcp://localhost:26657
				network_node_url: tcp://127.0.0.1:36657
				signing_key_name: servicer1
				proxied_service_endpoints:
				  anvil: http://anvil:8080
				  svc1: http://svc1:8080
				smt_store_path: smt_stores
				`,
		},
		// Invalid Configs
		{
			desc: "relayminer_config_test: invalid network node url",
			err:  config.ErrRelayMinerConfigInvalidNetworkNodeUrl,
			config: `
				query_node_url: tcp://localhost:26657
				network_node_url: &tcp://127.0.0.1:36657
				signing_key_name: servicer1
				proxied_service_endpoints:
				  anvil: http://anvil:8080
				  svc1: http://svc1:8080
				smt_store_path: smt_stores
				`,
		},
		{
			desc: "relayminer_config_test: missing network node url",
			err:  config.ErrRelayMinerConfigInvalidNetworkNodeUrl,
			config: `
				query_node_url: tcp://localhost:26657
				signing_key_name: servicer1
				proxied_service_endpoints:
				  anvil: http://anvil:8080
				  svc1: http://svc1:8080
				smt_store_path: smt_stores
				`,
		},
		{
			desc: "relayminer_config_test: invalid query node url",
			err:  config.ErrRelayMinerConfigInvalidNetworkNodeUrl,
			config: `
				query_node_url: &tcp://localhost:26657
				network_node_url: tcp://127.0.0.1:36657
				signing_key_name: servicer1
				proxied_service_endpoints:
				  anvil: http://anvil:8080
				  svc1: http://svc1:8080
				smt_store_path: smt_stores
				`,
		},
		{
			desc: "relayminer_config_test: missing query node url",
			err:  config.ErrRelayMinerConfigInvalidNetworkNodeUrl,
			config: `
				network_node_url: tcp://127.0.0.1:36657
				signing_key_name: servicer1
				proxied_service_endpoints:
				  anvil: http://anvil:8080
				  svc1: http://svc1:8080
				smt_store_path: smt_stores
				`,
		},
		{
			desc: "relayminer_config_test: missing signing key name",
			err:  config.ErrRelayMinerConfigInvalidNetworkNodeUrl,
			config: `
				query_node_url: tcp://localhost:26657
				network_node_url: &tcp://127.0.0.1:36657
				signing_key_name:
				proxied_service_endpoints:
				  anvil: http://anvil:8080
				  svc1: http://svc1:8080
				smt_store_path: smt_stores
				`,
		},
		{
			desc: "relayminer_config_test: missing smt store path",
			err:  config.ErrRelayMinerConfigInvalidNetworkNodeUrl,
			config: `
				query_node_url: tcp://localhost:26657
				network_node_url: &tcp://127.0.0.1:36657
				signing_key_name: servicer1
				proxied_service_endpoints:
				  anvil: http://anvil:8080
				  svc1: http://svc1:8080
				`,
		},
		{
			desc: "relayminer_config_test: empty proxied service endpoints",
			err:  config.ErrRelayMinerConfigInvalidNetworkNodeUrl,
			config: `
				query_node_url: tcp://localhost:26657
				network_node_url: &tcp://127.0.0.1:36657
				signing_key_name: servicer1
				proxied_service_endpoints:
				smt_store_path: smt_stores
				`,
		},
		{
			desc: "relayminer_config_test: invalid proxied service endpoint",
			err:  config.ErrRelayMinerConfigInvalidNetworkNodeUrl,
			config: `
				query_node_url: tcp://localhost:26657
				network_node_url: &tcp://127.0.0.1:36657
				signing_key_name: servicer1
				proxied_service_endpoints:
				  anvil: &http://anvil:8080
				  svc1: http://svc1:8080
				smt_store_path: smt_stores
				`,
		},
		{
			desc: "relayminer_config_test: invalid network node url",
			err:  config.ErrRelayMinerConfigUnmarshalYAML,
			config: `
				query_node_url: tcp://localhost:26657
				network_node_url: &tcp://127.0.0.1:36657
				signing_key_name: servicer1
				smt_store_path: smt_stores
				`,
		},
		{
			desc:   "relayminer_config_test: invalid relay miner config file",
			err:    config.ErrRelayMinerConfigUnmarshalYAML,
			config: ``,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			normalizedConfig := yaml.NormalizeYAMLIndentation(tt.config)
			config, err := config.ParseRelayMinerConfigs([]byte(normalizedConfig))

			if tt.err != nil {
				require.Error(t, err)
				require.Nil(t, config)
				stat, ok := status.FromError(tt.err)
				require.True(t, ok)
				require.Contains(t, stat.Message(), tt.err.Error())
				require.Nil(t, config)
				return
			}

			require.NoError(t, err)

			require.Equal(t, tt.expected.QueryNodeUrl.String(), config.QueryNodeUrl.String())
			require.Equal(t, tt.expected.NetworkNodeUrl.String(), config.NetworkNodeUrl.String())
			require.Equal(t, tt.expected.SigningKeyName, config.SigningKeyName)
			require.Equal(t, tt.expected.SmtStorePath, config.SmtStorePath)
			require.Equal(t, len(tt.expected.ProxiedServiceEndpoints), len(config.ProxiedServiceEndpoints))
			for serviceId, endpoint := range tt.expected.ProxiedServiceEndpoints {
				require.Equal(t, endpoint.String(), config.ProxiedServiceEndpoints[serviceId].String())
			}
		})
	}
}
