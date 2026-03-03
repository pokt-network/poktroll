package config_test

import (
	"testing"

	sdkerrors "cosmossdk.io/errors"
	"github.com/gogo/status"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/yaml"
	"github.com/pokt-network/poktroll/x/service/config"
)

func Test_ParseEditServiceConfig(t *testing.T) {
	tests := []struct {
		desc        string
		inputConfig string

		expectedError    *sdkerrors.Error
		expectedServices []*config.YAMLServiceEntry
	}{
		// Valid configs
		{
			desc: "valid single service",
			inputConfig: `
				services:
				  - service_id: svc1
				    service_name: "Service One"
				    compute_units_per_relay: 10
			`,
			expectedError: nil,
			expectedServices: []*config.YAMLServiceEntry{
				{
					ServiceId:            "svc1",
					ServiceName:          "Service One",
					ComputeUnitsPerRelay: 10,
				},
			},
		},
		{
			desc: "valid multiple services",
			inputConfig: `
				services:
				  - service_id: svc1
				    service_name: "Service One"
				    compute_units_per_relay: 10
				  - service_id: svc2
				    service_name: "Service Two"
				    compute_units_per_relay: 20
			`,
			expectedError: nil,
			expectedServices: []*config.YAMLServiceEntry{
				{
					ServiceId:            "svc1",
					ServiceName:          "Service One",
					ComputeUnitsPerRelay: 10,
				},
				{
					ServiceId:            "svc2",
					ServiceName:          "Service Two",
					ComputeUnitsPerRelay: 20,
				},
			},
		},
		// Invalid configs
		{
			desc:          "empty content",
			inputConfig:   ``,
			expectedError: config.ErrServiceConfigEmptyContent,
		},
		{
			desc: "missing services key",
			inputConfig: `
				not_services:
				  - service_id: svc1
			`,
			expectedError: config.ErrServiceConfigNoServices,
		},
		{
			desc: "empty services list",
			inputConfig: `
				services:
			`,
			expectedError: config.ErrServiceConfigNoServices,
		},
		{
			desc: "missing service_id",
			inputConfig: `
				services:
				  - service_name: "Service One"
				    compute_units_per_relay: 10
			`,
			expectedError: config.ErrServiceConfigInvalidServiceId,
		},
		{
			desc: "empty service_id",
			inputConfig: `
				services:
				  - service_id:
				    service_name: "Service One"
				    compute_units_per_relay: 10
			`,
			expectedError: config.ErrServiceConfigInvalidServiceId,
		},
		{
			desc: "valid - omitted service_name (optional for edit-service)",
			inputConfig: `
				services:
				  - service_id: svc1
				    compute_units_per_relay: 10
			`,
			expectedError: nil,
			expectedServices: []*config.YAMLServiceEntry{
				{
					ServiceId:            "svc1",
					ComputeUnitsPerRelay: 10,
				},
			},
		},
		{
			desc: "zero compute_units_per_relay",
			inputConfig: `
				services:
				  - service_id: svc1
				    service_name: "Service One"
				    compute_units_per_relay: 0
			`,
			expectedError: config.ErrServiceConfigInvalidComputeUnits,
		},
		{
			desc: "missing compute_units_per_relay defaults to zero",
			inputConfig: `
				services:
				  - service_id: svc1
				    service_name: "Service One"
			`,
			expectedError: config.ErrServiceConfigInvalidComputeUnits,
		},
		{
			desc: "malformed YAML",
			inputConfig: `
				services:
				  - service_id: svc1
				  service_name: "bad indent
			`,
			expectedError: config.ErrServiceConfigUnmarshalYAML,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			normalizedConfig := yaml.NormalizeYAMLIndentation(tt.inputConfig)
			editConfig, err := config.ParseEditServiceConfig([]byte(normalizedConfig))

			if tt.expectedError != nil {
				require.ErrorIs(t, err, tt.expectedError)
				stat, ok := status.FromError(tt.expectedError)
				require.True(t, ok)
				require.Contains(t, stat.Message(), tt.expectedError.Error())
				require.Nil(t, editConfig)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, editConfig)
			require.Equal(t, len(tt.expectedServices), len(editConfig.Services))
			for i, expectedSvc := range tt.expectedServices {
				require.Equal(t, expectedSvc.ServiceId, editConfig.Services[i].ServiceId)
				require.Equal(t, expectedSvc.ServiceName, editConfig.Services[i].ServiceName)
				require.Equal(t, expectedSvc.ComputeUnitsPerRelay, editConfig.Services[i].ComputeUnitsPerRelay)
			}
		})
	}
}
