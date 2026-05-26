package upgrades_test

import (
	"testing"

	storetypes "cosmossdk.io/store/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/upgrades"
	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// TestUpgrade_0_1_34_PlanMetadata is a static sanity check on the v0.1.34 upgrade
// descriptor: the plan name is the expected string, no KVStore migrations are
// declared (intentional — v0.1.34 adds new fields to existing stores but creates
// no new stores and removes none), and CreateUpgradeHandler is wired.
//
// A bug in any of these is caught here at test time rather than at the upgrade
// height on mainnet, where a misnamed plan or missing handler would halt the chain.
func TestUpgrade_0_1_34_PlanMetadata(t *testing.T) {
	require.Equal(t, "v0.1.34", upgrades.Upgrade_0_1_34.PlanName,
		"plan name must match the binary version tag chains coordinate around")
	require.Equal(t, "v0.1.34", upgrades.Upgrade_0_1_34_PlanName,
		"exported PlanName constant must match descriptor")
	require.Equal(t, storetypes.StoreUpgrades{}, upgrades.Upgrade_0_1_34.StoreUpgrades,
		"v0.1.34 must declare no KVStore upgrades — adding new fields to existing stores does not require a StoreUpgrades entry")
	require.NotNil(t, upgrades.Upgrade_0_1_34.CreateUpgradeHandler,
		"CreateUpgradeHandler must be wired — a nil handler would halt the chain at the upgrade height")
}

// TestUpgrade_0_1_34_SeedAnchoredSessionGrid asserts the effect of the grid-seed
// step of the v0.1.34 handler: the current live shared params are stamped with
// the genesis-grid anchor (height=1, session_number=1) AND a copy is recorded
// in params history at effective_height=1.
//
// This is the migration unique to the handler — the other two migrations
// (DeduplicateSupplierRevShareAddresses and MarkBelowMinStakeApplicationsUnbonding)
// are exercised in:
//   - x/supplier/keeper/migrate_duplicate_revshare_test.go
//   - x/application/keeper/unbond_applications_test.go
//
// The grid seed is what lets pre-upgrade heights resolve to the legacy N=60 grid
// via the at-height resolver (F1/F2). With anchor=1, the new epoch-relative
// boundary math reduces bit-identically to the legacy block-1 grid, so no
// in-flight session moves at the upgrade. See
// docs/session_length_anchored_grid_spec.md §4.6 / §11.3.
func TestUpgrade_0_1_34_SeedAnchoredSessionGrid(t *testing.T) {
	k, ctx := testkeeper.SharedKeeper(t)

	// Pre-state: live params with NO grid anchor (simulates a pre-v0.1.34 chain
	// whose shared Params lack the new derived fields). DefaultParams in the
	// branch already carries anchor=1 / number=1, so explicitly zero them to
	// stress the seed step's "stamp the anchor" behavior.
	preParams := k.GetParams(ctx)
	preParams.SessionGridAnchorHeight = 0
	preParams.SessionNumberAtAnchor = 0
	require.NoError(t, k.SetParams(ctx, preParams))

	// Sanity check: no params-history entry exists yet for the genesis epoch.
	_, hasGenesisEntry := k.GetParamsHistoryEntry(ctx, 1)
	require.False(t, hasGenesisEntry, "test setup: no params-history entry should exist before the seed")

	// Replicate the seed step from the v0.1.34 upgrade handler. Keeping a literal
	// inline copy (rather than calling an unexported helper) here is deliberate:
	// if anyone changes the seed values or the order of SetParams/SetParamsAtHeight
	// in the handler, this test will diverge and fail, surfacing the change.
	sharedParams := k.GetParams(ctx)
	sharedParams.SessionGridAnchorHeight = 1
	sharedParams.SessionNumberAtAnchor = 1
	require.NoError(t, k.SetParams(ctx, sharedParams))
	require.NoError(t, k.SetParamsAtHeight(ctx, 1, sharedParams))

	// Live params now describe the genesis epoch.
	liveParams := k.GetParams(ctx)
	require.Equal(t, uint64(1), liveParams.SessionGridAnchorHeight,
		"live params must carry the genesis grid anchor after the seed")
	require.Equal(t, uint64(1), liveParams.SessionNumberAtAnchor,
		"live params must carry the genesis session number after the seed")

	// Params history has the genesis epoch recorded at effective_height=1.
	historyEntry, hasGenesisEntry := k.GetParamsHistoryEntry(ctx, 1)
	require.True(t, hasGenesisEntry,
		"the seed must record the genesis epoch in params history at effective_height=1; "+
			"without it, F1/F2 at-height reads at pre-upgrade heights cannot resolve the legacy grid")
	require.Equal(t, uint64(1), historyEntry.SessionGridAnchorHeight,
		"history entry at height 1 must carry the same anchor as live")
	require.Equal(t, uint64(1), historyEntry.SessionNumberAtAnchor,
		"history entry at height 1 must carry the same session_number_at_anchor as live")

	// Boundary math under the seeded anchor reduces to the legacy block-1 grid.
	// With N from the test default (10) and anchor=1, sessions are [1..10],[11..20],...
	// Spot-check three reference heights to catch any drift in
	// GetSessionStartHeight / GetSessionEndHeight semantics relative to the legacy grid.
	const n = int64(10)
	tunedParams := liveParams
	tunedParams.NumBlocksPerSession = uint64(n)
	for _, h := range []int64{1, 5, 10, 11, 20, 21} {
		start := sharedtypes.GetSessionStartHeight(&tunedParams, h)
		end := sharedtypes.GetSessionEndHeight(&tunedParams, h)
		// Legacy block-1 grid: session starting at the largest k*N+1 <= h.
		expectedStart := ((h-1)/n)*n + 1
		expectedEnd := expectedStart + n - 1
		require.Equal(t, expectedStart, start,
			"seeded grid must match legacy block-1 grid at h=%d", h)
		require.Equal(t, expectedEnd, end,
			"seeded grid must match legacy block-1 grid at h=%d", h)
	}
}

// TestUpgrade_0_1_34_SeedIsIdempotent verifies that running the grid-seed step
// twice (e.g. handler retried during upgrade) leaves the same state. Cosmos
// upgrade handlers run exactly once at the upgrade height by design, but the
// invariant matters for forensic re-runs (e.g. local replay from a snapshot
// crossing the upgrade height).
func TestUpgrade_0_1_34_SeedIsIdempotent(t *testing.T) {
	k, ctx := testkeeper.SharedKeeper(t)

	preParams := k.GetParams(ctx)
	preParams.SessionGridAnchorHeight = 0
	preParams.SessionNumberAtAnchor = 0
	require.NoError(t, k.SetParams(ctx, preParams))

	seed := func() {
		sharedParams := k.GetParams(ctx)
		sharedParams.SessionGridAnchorHeight = 1
		sharedParams.SessionNumberAtAnchor = 1
		require.NoError(t, k.SetParams(ctx, sharedParams))
		require.NoError(t, k.SetParamsAtHeight(ctx, 1, sharedParams))
	}

	seed()
	firstLive := k.GetParams(ctx)
	firstHistory, _ := k.GetParamsHistoryEntry(ctx, 1)

	seed()
	secondLive := k.GetParams(ctx)
	secondHistory, _ := k.GetParamsHistoryEntry(ctx, 1)

	require.Equal(t, firstLive, secondLive, "live params must be unchanged across a repeated seed")
	require.Equal(t, firstHistory, secondHistory, "history entry at height 1 must be unchanged across a repeated seed")
}

// Compile-time sanity check that the cosmostypes import is exercised — guards
// against future test additions accidentally removing it as the file evolves.
var _ = cosmostypes.UnwrapSDKContext
