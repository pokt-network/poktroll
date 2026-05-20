package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func svc(id string) *sharedtypes.ApplicationServiceConfig {
	return &sharedtypes.ApplicationServiceConfig{ServiceId: id}
}

func TestApplicationServiceConfigUpdate_IsActive(t *testing.T) {
	tests := []struct {
		name               string
		activationHeight   int64
		deactivationHeight int64
		queryHeight        int64
		expectedActive     bool
	}{
		{name: "before activation", activationHeight: 10, deactivationHeight: 0, queryHeight: 9, expectedActive: false},
		{name: "at activation", activationHeight: 10, deactivationHeight: 0, queryHeight: 10, expectedActive: true},
		{name: "after activation, no deactivation", activationHeight: 10, deactivationHeight: 0, queryHeight: 100, expectedActive: true},
		{name: "before deactivation", activationHeight: 10, deactivationHeight: 20, queryHeight: 19, expectedActive: true},
		{name: "at deactivation", activationHeight: 10, deactivationHeight: 20, queryHeight: 20, expectedActive: false},
		{name: "after deactivation", activationHeight: 10, deactivationHeight: 20, queryHeight: 21, expectedActive: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			update := &ApplicationServiceConfigUpdate{
				ActivationHeight:   tt.activationHeight,
				DeactivationHeight: tt.deactivationHeight,
			}
			require.Equal(t, tt.expectedActive, update.IsActive(tt.queryHeight))
		})
	}
}

// TestApplication_GetActiveServiceConfigs_EmptyHistoryFallsBackToFlat verifies the
// "never changed" contract: an application with no service_config_history returns
// its flat ServiceConfigs as active for any height.
func TestApplication_GetActiveServiceConfigs_EmptyHistoryFallsBackToFlat(t *testing.T) {
	app := Application{
		Address:        "app1",
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{svc("svcA")},
	}

	for _, height := range []int64{1, 100, 1_000_000} {
		active := app.GetActiveServiceConfigs(height)
		require.Len(t, active, 1)
		require.Equal(t, "svcA", active[0].ServiceId)
	}
}

// TestApplication_GetActiveServiceConfigs_HistoryFiltersByHeight verifies that once
// history exists it is authoritative and filters by the query height.
func TestApplication_GetActiveServiceConfigs_HistoryFiltersByHeight(t *testing.T) {
	// svcA active [1, 100), svcB active [100, +inf).
	app := Application{
		Address:        "app1",
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{svc("svcB")},
		ServiceConfigHistory: []*ApplicationServiceConfigUpdate{
			{ApplicationAddress: "app1", Service: svc("svcA"), ActivationHeight: 1, DeactivationHeight: 100},
			{ApplicationAddress: "app1", Service: svc("svcB"), ActivationHeight: 100, DeactivationHeight: 0},
		},
	}

	// Before the swap: svcA only.
	active := app.GetActiveServiceConfigs(50)
	require.Len(t, active, 1)
	require.Equal(t, "svcA", active[0].ServiceId)

	// At the boundary: svcB (svcA deactivated exactly at 100).
	active = app.GetActiveServiceConfigs(100)
	require.Len(t, active, 1)
	require.Equal(t, "svcB", active[0].ServiceId)

	// After the swap: svcB only.
	active = app.GetActiveServiceConfigs(500)
	require.Len(t, active, 1)
	require.Equal(t, "svcB", active[0].ServiceId)
}

func TestApplication_BackfillServiceConfigHistory(t *testing.T) {
	t.Run("backfills empty history from flat configs", func(t *testing.T) {
		app := Application{
			Address:        "app1",
			ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{svc("svcA")},
		}
		modified := app.BackfillServiceConfigHistory()
		require.True(t, modified)
		require.Len(t, app.ServiceConfigHistory, 1)
		require.Equal(t, "svcA", app.ServiceConfigHistory[0].Service.ServiceId)
		require.Equal(t, int64(1), app.ServiceConfigHistory[0].ActivationHeight)
		require.Equal(t, int64(0), app.ServiceConfigHistory[0].DeactivationHeight)
	})

	t.Run("idempotent when history already present", func(t *testing.T) {
		app := Application{
			Address:        "app1",
			ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{svc("svcA")},
			ServiceConfigHistory: []*ApplicationServiceConfigUpdate{
				{ApplicationAddress: "app1", Service: svc("svcA"), ActivationHeight: 1},
			},
		}
		modified := app.BackfillServiceConfigHistory()
		require.False(t, modified)
		require.Len(t, app.ServiceConfigHistory, 1)
	})

	t.Run("no-op with no flat configs", func(t *testing.T) {
		app := Application{Address: "app1"}
		modified := app.BackfillServiceConfigHistory()
		require.False(t, modified)
		require.Empty(t, app.ServiceConfigHistory)
	})
}
