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
		desc string

		inputConfig string

		expectedError  *sdkerrors.Error
		expectedConfig []*sharedtypes.ApplicationServiceConfig
	}{
		// Valid Configs
		{
			desc: "valid: service staking config",

			inputConfig: `
				service_ids:
				  - svc1
				  - svc2
				`,

			expectedError: nil,
			expectedConfig: []*sharedtypes.ApplicationServiceConfig{
				{
					Service: &sharedtypes.Service{Id: "svc1"},
				},
				{
					Service: &sharedtypes.Service{Id: "svc2"},
				},
			},
		},
		// Invalid Configs
		{
			desc: "invalid: empty service staking config",

			inputConfig: ``,

			expectedError: config.ErrApplicationConfigUnmarshalYAML,
		},
		{
			desc: "invalid: no service ids",

			inputConfig: `
				service_ids:
				`,

			expectedError: config.ErrApplicationConfigInvalidServiceId,
		},
		{
			desc: "invalid: invalid serviceId",

			inputConfig: `
				service_ids:
				  - sv c1
				`,

			expectedError: config.ErrApplicationConfigInvalidServiceId,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			normalizedConfig := yaml.NormalizeYAMLIndentation(tt.inputConfig)
			appServiceConfig, err := config.ParseApplicationConfigs([]byte(normalizedConfig))

			if tt.expectedError != nil {
				require.Error(t, err)
				require.Nil(t, appServiceConfig)
				stat, ok := status.FromError(tt.expectedError)
				require.True(t, ok)
				require.Contains(t, stat.Message(), tt.expectedError.Error())
				require.Nil(t, appServiceConfig)
				return
			}

			require.NoError(t, err)

			log.Printf("serviceIds: %v", appServiceConfig)
			require.Equal(t, len(tt.expectedConfig), len(appServiceConfig))
			for i, expected := range tt.expectedConfig {
				require.Equal(t, expected.Service.Id, appServiceConfig[i].Service.Id)
			}
		})
	}
}
