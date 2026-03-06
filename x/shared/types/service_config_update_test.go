package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestServiceConfigUpdate_IsActive(t *testing.T) {
	tests := []struct {
		name               string
		activationHeight   int64
		deactivationHeight int64
		queryHeight        int64
		expectedActive     bool
	}{
		{
			name:               "active: query at activation height (boundary)",
			activationHeight:   10,
			deactivationHeight: 20,
			queryHeight:        10,
			expectedActive:     true,
		},
		{
			name:               "active: query between activation and deactivation",
			activationHeight:   10,
			deactivationHeight: 20,
			queryHeight:        15,
			expectedActive:     true,
		},
		{
			name:               "active: query one before deactivation",
			activationHeight:   10,
			deactivationHeight: 20,
			queryHeight:        19,
			expectedActive:     true,
		},
		{
			name:               "inactive: query at deactivation height (boundary, <= means inactive)",
			activationHeight:   10,
			deactivationHeight: 20,
			queryHeight:        20,
			expectedActive:     false,
		},
		{
			name:               "inactive: query after deactivation",
			activationHeight:   10,
			deactivationHeight: 20,
			queryHeight:        25,
			expectedActive:     false,
		},
		{
			name:               "inactive: query before activation",
			activationHeight:   10,
			deactivationHeight: 20,
			queryHeight:        5,
			expectedActive:     false,
		},
		{
			name:               "inactive: query one before activation",
			activationHeight:   10,
			deactivationHeight: 20,
			queryHeight:        9,
			expectedActive:     false,
		},
		{
			name:               "active indefinitely: no deactivation (0)",
			activationHeight:   10,
			deactivationHeight: sharedtypes.NoDeactivationHeight,
			queryHeight:        1000,
			expectedActive:     true,
		},
		{
			name:               "active indefinitely: query at activation",
			activationHeight:   10,
			deactivationHeight: sharedtypes.NoDeactivationHeight,
			queryHeight:        10,
			expectedActive:     true,
		},
		{
			name:               "inactive: no deactivation but before activation",
			activationHeight:   10,
			deactivationHeight: sharedtypes.NoDeactivationHeight,
			queryHeight:        5,
			expectedActive:     false,
		},
		{
			name:               "active: activation at height 0, no deactivation",
			activationHeight:   0,
			deactivationHeight: sharedtypes.NoDeactivationHeight,
			queryHeight:        0,
			expectedActive:     true,
		},
		{
			name:               "active: activation and query both at 1",
			activationHeight:   1,
			deactivationHeight: sharedtypes.NoDeactivationHeight,
			queryHeight:        1,
			expectedActive:     true,
		},
		{
			name:               "inactive: deactivation equals activation (zero-width window)",
			activationHeight:   10,
			deactivationHeight: 10,
			queryHeight:        10,
			expectedActive:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scu := &sharedtypes.ServiceConfigUpdate{
				ActivationHeight:   tt.activationHeight,
				DeactivationHeight: tt.deactivationHeight,
			}
			result := scu.IsActive(tt.queryHeight)
			require.Equal(t, tt.expectedActive, result)
		})
	}
}

func TestSupplier_IsActive(t *testing.T) {
	serviceId := "svc1"

	tests := []struct {
		name           string
		history        []*sharedtypes.ServiceConfigUpdate
		queryHeight    int64
		expectedActive bool
	}{
		{
			name: "active: single active config",
			history: []*sharedtypes.ServiceConfigUpdate{
				{
					Service:            &sharedtypes.SupplierServiceConfig{ServiceId: serviceId},
					ActivationHeight:   1,
					DeactivationHeight: sharedtypes.NoDeactivationHeight,
				},
			},
			queryHeight:    10,
			expectedActive: true,
		},
		{
			name: "inactive: single deactivated config",
			history: []*sharedtypes.ServiceConfigUpdate{
				{
					Service:            &sharedtypes.SupplierServiceConfig{ServiceId: serviceId},
					ActivationHeight:   1,
					DeactivationHeight: 5,
				},
			},
			queryHeight:    10,
			expectedActive: false,
		},
		{
			name: "inactive: wrong service ID",
			history: []*sharedtypes.ServiceConfigUpdate{
				{
					Service:            &sharedtypes.SupplierServiceConfig{ServiceId: "other-svc"},
					ActivationHeight:   1,
					DeactivationHeight: sharedtypes.NoDeactivationHeight,
				},
			},
			queryHeight:    10,
			expectedActive: false,
		},
		{
			name:           "inactive: empty history",
			history:        []*sharedtypes.ServiceConfigUpdate{},
			queryHeight:    10,
			expectedActive: false,
		},
		{
			name: "active: multiple configs, second one active",
			history: []*sharedtypes.ServiceConfigUpdate{
				{
					Service:            &sharedtypes.SupplierServiceConfig{ServiceId: serviceId},
					ActivationHeight:   1,
					DeactivationHeight: 5,
				},
				{
					Service:            &sharedtypes.SupplierServiceConfig{ServiceId: serviceId},
					ActivationHeight:   10,
					DeactivationHeight: sharedtypes.NoDeactivationHeight,
				},
			},
			queryHeight:    15,
			expectedActive: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			supplier := &sharedtypes.Supplier{
				ServiceConfigHistory: tt.history,
			}
			result := supplier.IsActive(tt.queryHeight, serviceId)
			require.Equal(t, tt.expectedActive, result)
		})
	}
}
