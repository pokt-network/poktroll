package types

import (
	"fmt"
	"math/rand"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
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

// A change activating before an already pending entry (the next session start
// moved earlier) replaces every pending entry at or after its activation height.
func TestApplication_RecordPerSessionSpendLimitChange_EarlierActivationReplacesPending(t *testing.T) {
	a := sdk.NewInt64Coin("upokt", 1_000_000)
	b := sdk.NewInt64Coin("upokt", 2_000_000)
	c := sdk.NewInt64Coin("upokt", 3_000_000)
	d := sdk.NewInt64Coin("upokt", 4_000_000)
	stake := sdk.NewInt64Coin("upokt", 100_000_000)
	app := &Application{
		Address:              "pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4",
		Stake:                &stake,
		PerSessionSpendLimit: &a,
		ServiceConfigs:       []*sharedtypes.ApplicationServiceConfig{svc("svc")},
	}

	app.PerSessionSpendLimit = &b
	app.RecordPerSessionSpendLimitChange(&a, 60)
	app.PerSessionSpendLimit = &c
	app.RecordPerSessionSpendLimitChange(&b, 70)
	app.PerSessionSpendLimit = &d
	app.RecordPerSessionSpendLimitChange(&c, 65)

	require.Equal(t, []*ApplicationPerSessionSpendLimitUpdate{
		{PerSessionSpendLimit: &a, ActivationHeight: 1},
		{PerSessionSpendLimit: &b, ActivationHeight: 60},
		{PerSessionSpendLimit: &d, ActivationHeight: 65},
	}, app.PerSessionSpendLimitHistory)

	app.PerSessionSpendLimit = &a
	app.RecordPerSessionSpendLimitChange(&d, 60)
	require.Equal(t, []*ApplicationPerSessionSpendLimitUpdate{
		{PerSessionSpendLimit: &a, ActivationHeight: 1},
	}, app.PerSessionSpendLimitHistory)
	require.Equal(t, &a, app.GetPerSessionSpendLimitAtHeight(70))

	gs := DefaultGenesis()
	gs.ApplicationList = []Application{*app}
	require.NoError(t, gs.Validate())
}

// Random limit changes over a session grid with varying session lengths: the
// limit resolved at every started session start never changes afterwards, equals
// the last value set before that session started, and the history stays
// strictly increasing, also when activation heights are not monotonic.
func TestApplication_RecordPerSessionSpendLimitChange_SessionStartModel(t *testing.T) {
	values := []*sdk.Coin{nil}
	for _, amt := range []int64{1_000_000, 2_000_000, 5_000_000} {
		c := sdk.NewInt64Coin("upokt", amt)
		values = append(values, &c)
	}

	resolvedAtStarts := func(app *Application, starts []int64, h int64) []string {
		var out []string
		for _, s := range starts {
			if s > h {
				break
			}
			out = append(out, fmt.Sprint(app.GetPerSessionSpendLimitAtHeight(s)))
		}
		return out
	}

	for seed := int64(0); seed < 200; seed++ {
		r := rand.New(rand.NewSource(seed))
		// When set, the activation height passed in is not always the next
		// session start, as for a grid computed from params that change later.
		irregularActivation := seed%2 == 1

		starts := []int64{1}
		for len(starts) < 40 {
			starts = append(starts, starts[len(starts)-1]+int64(1+r.Intn(6)))
		}
		nextStart := func(h int64) int64 {
			for _, s := range starts {
				if s > h {
					return s
				}
			}
			t.Fatalf("height %d beyond grid", h)
			return 0
		}

		initial := values[r.Intn(len(values))]
		app := &Application{PerSessionSpendLimit: initial}
		type change struct {
			height, activation int64
			limit              *sdk.Coin
		}
		var changes []change

		maxHeight := starts[len(starts)-2]
		for h := int64(1); h < maxHeight; h++ {
			// Several changes may land in the same block or session.
			for r.Intn(3) == 0 {
				before := resolvedAtStarts(app, starts, h)
				prev := app.PerSessionSpendLimit
				app.PerSessionSpendLimit = values[r.Intn(len(values))]
				activation := nextStart(h)
				if irregularActivation {
					activation += int64(r.Intn(3))
				}
				app.RecordPerSessionSpendLimitChange(prev, activation)
				require.Equalf(t, before, resolvedAtStarts(app, starts, h),
					"seed %d h %d: a started session's limit changed (history %v)", seed, h, app.PerSessionSpendLimitHistory)
				changes = append(changes, change{h, activation, app.PerSessionSpendLimit})
			}

			for i, e := range app.PerSessionSpendLimitHistory {
				if i > 0 {
					require.Greaterf(t, e.ActivationHeight, app.PerSessionSpendLimitHistory[i-1].ActivationHeight,
						"seed %d h %d: history %v", seed, h, app.PerSessionSpendLimitHistory)
				}
			}
			if irregularActivation {
				continue
			}

			// Every started session resolves to the last value set before it started.
			for _, s := range starts {
				if s > h {
					break
				}
				want := initial
				for _, c := range changes {
					if c.height < s {
						want = c.limit
					}
				}
				got := app.GetPerSessionSpendLimitAtHeight(s)
				require.Truef(t, spendLimitsEqual(want, got),
					"seed %d h %d session start %d: want %v got %v (history %v)", seed, h, s, want, got, app.PerSessionSpendLimitHistory)
			}

			future := 0
			for _, e := range app.PerSessionSpendLimitHistory {
				if e.ActivationHeight > h {
					future++
				}
			}
			require.LessOrEqual(t, future, 1, "seed %d h %d", seed, h)
		}
	}
}
