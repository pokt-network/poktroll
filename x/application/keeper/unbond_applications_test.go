package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/pocket"
	testevents "github.com/pokt-network/poktroll/testutil/events"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// TestMarkBelowMinStakeApplicationsUnbonding asserts that the one-time v0.1.34
// backfill sweep marks exactly the active applications whose stake is below the
// on-chain min_stake param, while leaving at-or-above and already-unbonding
// applications untouched.
func TestMarkBelowMinStakeApplicationsUnbonding(t *testing.T) {
	applicationModuleKeepers, ctx := keepertest.NewApplicationModuleKeepers(t)

	// Raise min_stake well above DefaultMinStake (1 POKT) so the sweep is
	// exercised against a realistic on-chain value (1,000 POKT). This is the
	// scenario issue #1846 missed: apps above the default but below the real
	// min_stake were never force-unbonded.
	minStake := sdk.NewInt64Coin(pocket.DenomuPOKT, 1_000_000_000) // 1,000 POKT
	appParams := apptypes.DefaultParams()
	appParams.MinStake = &minStake
	require.NoError(t, applicationModuleKeepers.SetParams(ctx, appParams))

	sharedParams := applicationModuleKeepers.SharedKeeper.GetParams(ctx)
	expectedSessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, sdk.UnwrapSDKContext(ctx).BlockHeight())

	newApp := func(stakeAmount int64, unstakeSessionEndHeight uint64) apptypes.Application {
		stake := sdk.NewInt64Coin(pocket.DenomuPOKT, stakeAmount)
		return apptypes.Application{
			Address:                 sample.AccAddressBech32(),
			Stake:                   &stake,
			ServiceConfigs:          []*sharedtypes.ApplicationServiceConfig{{ServiceId: "svc1"}},
			UnstakeSessionEndHeight: unstakeSessionEndHeight,
		}
	}

	// Below min_stake, active -> should be marked unbonding.
	belowApp := newApp(500_000_000, apptypes.ApplicationNotUnstaking) // 500 POKT
	// Exactly at min_stake -> NOT below (LT), should be left active.
	atApp := newApp(1_000_000_000, apptypes.ApplicationNotUnstaking) // 1,000 POKT
	// Above min_stake -> should be left active.
	aboveApp := newApp(2_000_000_000, apptypes.ApplicationNotUnstaking) // 2,000 POKT
	// Below min_stake but already unbonding -> should be skipped, timeline preserved.
	const presetUnstakeHeight = uint64(123456)
	alreadyUnbondingApp := newApp(100_000_000, presetUnstakeHeight) // 100 POKT

	for _, app := range []apptypes.Application{belowApp, atApp, aboveApp, alreadyUnbondingApp} {
		applicationModuleKeepers.SetApplication(ctx, app)
	}

	// Reset events so only the sweep's emissions are observed.
	ctx, _ = testevents.ResetEventManager(ctx)

	marked, err := applicationModuleKeepers.MarkBelowMinStakeApplicationsUnbonding(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, marked, "only the below-min_stake active application should be marked")

	// belowApp: now unbonding, with UnstakeSessionEndHeight set to the current session end.
	gotBelow, found := applicationModuleKeepers.GetApplication(ctx, belowApp.Address)
	require.True(t, found)
	require.True(t, gotBelow.IsUnbonding())
	require.Equal(t, uint64(expectedSessionEndHeight), gotBelow.GetUnstakeSessionEndHeight())

	// atApp and aboveApp: untouched (still active).
	for _, addr := range []string{atApp.Address, aboveApp.Address} {
		got, gotFound := applicationModuleKeepers.GetApplication(ctx, addr)
		require.True(t, gotFound)
		require.False(t, got.IsUnbonding(), "application at/above min_stake must not be marked unbonding")
	}

	// alreadyUnbondingApp: timeline preserved, not reset to the current session end.
	gotAlready, found := applicationModuleKeepers.GetApplication(ctx, alreadyUnbondingApp.Address)
	require.True(t, found)
	require.True(t, gotAlready.IsUnbonding())
	require.Equal(t, presetUnstakeHeight, gotAlready.GetUnstakeSessionEndHeight(), "already-unbonding timeline must be preserved")

	// Exactly one EventApplicationUnbondingBegin with the BELOW_MIN_STAKE reason.
	events := sdk.UnwrapSDKContext(ctx).EventManager().Events()
	unbondingBeginEvents := testevents.FilterEvents[*apptypes.EventApplicationUnbondingBegin](t, events)
	require.Len(t, unbondingBeginEvents, 1)
	require.Equal(t, belowApp.Address, unbondingBeginEvents[0].GetApplication().GetAddress())
	require.Equal(t,
		apptypes.ApplicationUnbondingReason_APPLICATION_UNBONDING_REASON_BELOW_MIN_STAKE,
		unbondingBeginEvents[0].GetReason(),
	)
	require.Equal(t, expectedSessionEndHeight, unbondingBeginEvents[0].GetSessionEndHeight())
}
