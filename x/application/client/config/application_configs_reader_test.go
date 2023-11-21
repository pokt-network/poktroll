package config_test

import (
	"log"
	"testing"

	sdkerrors "cosmossdk.io/errors"
	"github.com/gogo/status"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/yaml"
	"github.com/pokt-network/poktroll/x/application/client/config"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func Test_ParseApplicationConfigs(t *testing.T) {
	tests := []struct {
		desc     string
		err      *sdkerrors.Error
		expected []*sharedtypes.ApplicationServiceConfig
		config   string
	}{
		// Valid Configs
		{
			desc: "application_staking_config_test: valid service staking config",
			err:  nil,
			expected: []*sharedtypes.ApplicationServiceConfig{
				{
					Service: &sharedtypes.Service{Id: "svc1"},
				},
				{
					Service: &sharedtypes.Service{Id: "svc2"},
				},
			},
			config: `
				service_ids:
				  - svc1
				  - svc2
				`,
		},
		// Invalid Configs
		{
			desc:   "application_staking_config_test: empty service staking config",
			err:    config.ErrApplicationConfigUnmarshalYAML,
			config: ``,
		},
		{
			desc: "alllication_staking_config_test: no service ids",
			err:  config.ErrApplicationConfigInvalidServiceId,
			config: `
				service_ids:
			`,
		},
		{
			desc: "application_staking_config_test: invalid serviceId",
			err:  config.ErrApplicationConfigInvalidServiceId,
			config: `
				service_ids:
				  - sv c1
				`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			normalizedConfig := yaml.NormalizeYAMLIndentation(tt.config)
			appServiceConfig, err := config.ParseApplicationConfigs([]byte(normalizedConfig))

			if tt.err != nil {
				require.Error(t, err)
				require.Nil(t, appServiceConfig)
				stat, ok := status.FromError(tt.err)
				require.True(t, ok)
				require.Contains(t, stat.Message(), tt.err.Error())
				require.Nil(t, appServiceConfig)
				return
			}

			require.NoError(t, err)

			log.Printf("serviceIds: %v", appServiceConfig)
			require.Equal(t, len(tt.expected), len(appServiceConfig))
			for i, expected := range tt.expected {
				require.Equal(t, expected.Service.Id, appServiceConfig[i].Service.Id)
			}
		})
	}
}
