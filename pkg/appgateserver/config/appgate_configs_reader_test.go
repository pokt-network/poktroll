package config_test

import (
	"net/url"
	"testing"

	sdkerrors "cosmossdk.io/errors"
	"github.com/gogo/status"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/appgateserver/config"
	"github.com/pokt-network/poktroll/testutil/yaml"
)

func Test_ParseAppGateConfigs(t *testing.T) {
	tests := []struct {
		desc string

		inputConfigYAML string

		expectedErr    *sdkerrors.Error
		expectedConfig *config.AppGateServerConfig
	}{
		// Valid Configs
		{
			desc: "valid: AppGateServer config",

			inputConfigYAML: `
				query_node_rpc_url: tcp://127.0.0.1:26657
				query_node_grpc_url: tcp://127.0.0.1:9090
				signing_key: app1
				self_signing: true
				listening_endpoint: http://localhost:42069
				`,

			expectedErr: nil,
			expectedConfig: &config.AppGateServerConfig{
				QueryNodeRPCUrl:   &url.URL{Scheme: "tcp", Host: "127.0.0.1:26657"},
				QueryNodeGRPCUrl:  &url.URL{Scheme: "tcp", Host: "127.0.0.1:9090"},
				SigningKey:        "app1",
				SelfSigning:       true,
				ListeningEndpoint: &url.URL{Scheme: "http", Host: "localhost:42069"},
			},
		},
		{
			desc: "valid: AppGateServer config with undefined self signing",

			inputConfigYAML: `
				query_node_rpc_url: tcp://127.0.0.1:26657
				query_node_grpc_url: tcp://127.0.0.1:9090
				signing_key: app1
				listening_endpoint: http://localhost:42069
				`,

			expectedErr: nil,
			expectedConfig: &config.AppGateServerConfig{
				QueryNodeRPCUrl:   &url.URL{Scheme: "tcp", Host: "127.0.0.1:26657"},
				QueryNodeGRPCUrl:  &url.URL{Scheme: "tcp", Host: "127.0.0.1:9090"},
				SigningKey:        "app1",
				SelfSigning:       false,
				ListeningEndpoint: &url.URL{Scheme: "http", Host: "localhost:42069"},
			},
		},
		// Invalid Configs
		{
			desc: "invalid: empty AppGateServer config",

			inputConfigYAML: ``,

			expectedErr: config.ErrAppGateConfigEmpty,
		},
		{
			desc: "invalid: no signing key",

			inputConfigYAML: `
				query_node_rpc_url: tcp://127.0.0.1:26657
				query_node_grpc_url: tcp://127.0.0.1:9090
				# NB: explicitly missing signing key
				self_signing: true
				listening_endpoint: http://localhost:42069
				`,

			expectedErr: config.ErrAppGateConfigEmptySigningKey,
		},
		{
			desc: "invalid: invalid listening endpoint",

			inputConfigYAML: `
				query_node_rpc_url: tcp://127.0.0.1:26657
				query_node_grpc_url: tcp://127.0.0.1:9090
				signing_key: app1
				self_signing: true
				listening_endpoint: l&ocalhost:42069
				`,

			expectedErr: config.ErrAppGateConfigInvalidListeningEndpoint,
		},
		{
			desc: "invalid: invalid query node grpc url",

			inputConfigYAML: `
				query_node_rpc_url: tcp://127.0.0.1:26657
				query_node_grpc_url: 1&27.0.0.1:9090
				signing_key: app1
				self_signing: true
				listening_endpoint: http://localhost:42069
				`,

			expectedErr: config.ErrAppGateConfigInvalidQueryNodeGRPCUrl,
		},
		{
			desc: "invalid: missing query node grpc url",

			inputConfigYAML: `
				query_node_rpc_url: tcp://127.0.0.1:26657
				# NB: explicitly missing query_node_grpc_url
				signing_key: app1
				self_signing: true
				listening_endpoint: http://localhost:42069
				`,

			expectedErr: config.ErrAppGateConfigInvalidQueryNodeGRPCUrl,
		},
		{
			desc: "invalid: invalid query node rpc url",

			inputConfigYAML: `
				query_node_rpc_url: 1&27.0.0.1:26657
				query_node_grpc_url: tcp://127.0.0.1:9090
				signing_key: app1
				self_signing: true
				listening_endpoint: http://localhost:42069
				`,

			expectedErr: config.ErrAppGateConfigInvalidQueryNodeRPCUrl,
		},
		{
			desc: "invalid: missing query node rpc url",

			inputConfigYAML: `
				# NB: explicitly missing query_node_rpc_url
				query_node_grpc_url: tcp://127.0.0.1:9090
				signing_key: app1
				self_signing: true
				listening_endpoint: http://localhost:42069
				`,

			expectedErr: config.ErrAppGateConfigInvalidQueryNodeRPCUrl,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			normalizedConfig := yaml.NormalizeYAMLIndentation(test.inputConfigYAML)
			config, err := config.ParseAppGateServerConfigs([]byte(normalizedConfig))

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

			require.Equal(t, test.expectedConfig.SelfSigning, config.SelfSigning)
			require.Equal(t, test.expectedConfig.SigningKey, config.SigningKey)
			require.Equal(t, test.expectedConfig.ListeningEndpoint.String(), config.ListeningEndpoint.String())
			require.Equal(t, test.expectedConfig.QueryNodeGRPCUrl.String(), config.QueryNodeGRPCUrl.String())
			require.Equal(t, test.expectedConfig.QueryNodeGRPCUrl.String(), config.QueryNodeGRPCUrl.String())
		})
	}
}
