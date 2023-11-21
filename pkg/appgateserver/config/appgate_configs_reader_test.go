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
		desc     string
		err      *sdkerrors.Error
		expected *config.AppGateConfig
		config   string
	}{
		// Valid Configs
		{
			desc: "appgate_config_test: valid app gate config",
			err:  nil,
			expected: &config.AppGateConfig{
				SelfSigning:       true,
				SigningKey:        "app1",
				ListeningEndpoint: &url.URL{Scheme: "http", Host: "localhost:42069"},
				QueryNodeUrl:      &url.URL{Scheme: "tcp", Host: "127.0.0.1:36657"},
			},
			config: `
				self_signing: true
				signing_key: app1
				listening_endpoint: http://localhost:42069
				query_node_url: tcp://127.0.0.1:36657
				`,
		},
		{
			desc: "appgate_config_test: valid app gate config with undefined self signing",
			err:  nil,
			expected: &config.AppGateConfig{
				SelfSigning:       false,
				SigningKey:        "app1",
				ListeningEndpoint: &url.URL{Scheme: "http", Host: "localhost:42069"},
				QueryNodeUrl:      &url.URL{Scheme: "tcp", Host: "127.0.0.1:36657"},
			},
			config: `
				signing_key: app1
				listening_endpoint: http://localhost:42069
				query_node_url: tcp://127.0.0.1:36657
				`,
		},
		// Invalid Configs
		{
			desc:   "appgate_config_test: invalid appgate config",
			err:    config.ErrAppGateConfigUnmarshalYAML,
			config: ``,
		},
		{
			desc: "appgate_config_test: no signing key",
			err:  config.ErrAppGateConfigEmptySigningKey,
			config: `
				self_signing: true
				signing_key:
				listening_endpoint: http://localhost:42069
				query_node_url: tcp://127.0.0.1:36657
				`,
		},
		{
			desc: "appgate_config_test: invalid listening endpoint",
			err:  config.ErrAppGateConfigInvalidListeningEndpoint,
			config: `
				self_signing: true
				signing_key: app1
				listening_endpoint: &localhost:42069
				query_node_url: tcp://127.0.0.1:36657
				`,
		},
		{
			desc: "appgate_config_test: invalid query node url",
			err:  config.ErrAppGateConfigInvalidQueryNodeUrl,
			config: `
				self_signing: true
				signing_key: app1
				listening_endpoint: http://localhost:42069
				query_node_url: &127.0.0.1:36657
				`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			normalizedConfig := yaml.NormalizeYAMLIndentation(tt.config)
			config, err := config.ParseAppGateConfigs([]byte(normalizedConfig))

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

			require.Equal(t, tt.expected.SelfSigning, config.SelfSigning)
			require.Equal(t, tt.expected.SigningKey, config.SigningKey)
			require.Equal(t, tt.expected.ListeningEndpoint.String(), config.ListeningEndpoint.String())
			require.Equal(t, tt.expected.QueryNodeUrl.String(), config.QueryNodeUrl.String())
		})
	}
}
