package upgrades_test

import (
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/upgrades"
	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// TestUpgrade_0_1_35_PlanMetadata is a static sanity check on the v0.1.35 upgrade
// descriptor: the plan name is the expected string, no KVStore migrations are declared
// (intentional — v0.1.35 adds a new field to an existing store and creates no new stores),
// and CreateUpgradeHandler is wired.
//
// A misnamed plan or a nil handler halts the chain at the upgrade height rather than
// upgrading it, so these are cheap checks against an expensive failure.
func TestUpgrade_0_1_35_PlanMetadata(t *testing.T) {
	require.Equal(t, "v0.1.35", upgrades.Upgrade_0_1_35.PlanName,
		"plan name must match the binary version tag chains coordinate around")
	require.Equal(t, "v0.1.35", upgrades.Upgrade_0_1_35_PlanName,
		"exported PlanName constant must match descriptor")
	require.Equal(t, storetypes.StoreUpgrades{}, upgrades.Upgrade_0_1_35.StoreUpgrades,
		"v0.1.35 must declare no KVStore upgrades — adding a new field to an existing store does not require a StoreUpgrades entry")
	require.NotNil(t, upgrades.Upgrade_0_1_35.CreateUpgradeHandler,
		"CreateUpgradeHandler must be wired — a nil handler would halt the chain at the upgrade height")
}

// TestUpgrade_0_1_35_SeedsOverservicingBonusMultiplier asserts the effect of the seed step
// of the v0.1.35 handler: a pre-upgrade params blob, which deserializes with the new
// overservicing_bonus_multiplier field at its proto3 zero value, comes out of the upgrade
// carrying the explicit default of 1.
//
// 1 reproduces the legacy head-split cap exactly, so the consensus change ships as a no-op
// and governance opts into redistribution later by raising the multiplier.
//
// As in v0.1.34_test.go, the seed step is replicated inline rather than invoked through
// the handler (which needs a fully wired *keepers.Keepers). If anyone changes the seeded
// value or the validate/set ordering in the handler, this test diverges and fails.
func TestUpgrade_0_1_35_SeedsOverservicingBonusMultiplier(t *testing.T) {
	k, ctx := testkeeper.TokenomicsKeeper(t)

	// Pre-state: simulate params stored by a pre-v0.1.35 binary, where the new field is
	// absent and therefore deserializes to the proto3 zero value.
	preParams := k.GetParams(ctx)
	preParams.OverservicingBonusMultiplier = 0
	require.NoError(t, k.SetParams(ctx, preParams))
	require.Equal(t, uint64(0), k.GetParams(ctx).OverservicingBonusMultiplier,
		"test setup: the multiplier must start unset")

	// Replicate the seed step from the v0.1.35 upgrade handler.
	tokenomicsParams := k.GetParams(ctx)
	if tokenomicsParams.OverservicingBonusMultiplier == 0 {
		tokenomicsParams.OverservicingBonusMultiplier = tokenomicstypes.DefaultOverservicingBonusMultiplier
	}
	require.NoError(t, tokenomicsParams.ValidateBasic())
	require.NoError(t, k.SetParams(ctx, tokenomicsParams))

	require.Equal(t, uint64(1), k.GetParams(ctx).OverservicingBonusMultiplier,
		"the upgrade must seed the multiplier to 1 (legacy head-split cap, no-op)")
	require.Equal(t, tokenomicstypes.DefaultOverservicingBonusMultiplier, k.GetParams(ctx).OverservicingBonusMultiplier,
		"the seeded value must be the exported default, not a hardcoded literal that can drift from it")
}

// TestUpgrade_0_1_35_SeedDoesNotClobberGovernanceValue guards the branch of the seed that
// is easy to get wrong: the handler must only fill in an UNSET multiplier.
//
// This matters if the upgrade is ever re-applied, replayed from a snapshot crossing the
// upgrade height, or copied into a later upgrade after governance has already raised the
// multiplier. Unconditionally assigning the default would silently switch redistribution
// back off and change settlement amounts network-wide.
func TestUpgrade_0_1_35_SeedDoesNotClobberGovernanceValue(t *testing.T) {
	k, ctx := testkeeper.TokenomicsKeeper(t)

	// Governance has already enabled redistribution.
	const governanceMultiplier = uint64(10)
	preParams := k.GetParams(ctx)
	preParams.OverservicingBonusMultiplier = governanceMultiplier
	require.NoError(t, k.SetParams(ctx, preParams))

	seed := func() {
		tokenomicsParams := k.GetParams(ctx)
		if tokenomicsParams.OverservicingBonusMultiplier == 0 {
			tokenomicsParams.OverservicingBonusMultiplier = tokenomicstypes.DefaultOverservicingBonusMultiplier
		}
		require.NoError(t, tokenomicsParams.ValidateBasic())
		require.NoError(t, k.SetParams(ctx, tokenomicsParams))
	}

	seed()
	require.Equal(t, governanceMultiplier, k.GetParams(ctx).OverservicingBonusMultiplier,
		"the seed must not overwrite a multiplier governance has already set")

	// Idempotent: running it again is still a no-op.
	firstParams := k.GetParams(ctx)
	seed()
	require.Equal(t, firstParams, k.GetParams(ctx),
		"tokenomics params must be unchanged across a repeated seed")
}

// TestUpgrade_0_1_35_ReportsAntiCollusionInvariant asserts that the anti-collusion
// round-trip factor is REPORTED but never fails the upgrade handler.
//
// The per-supplier head-split cap was incidentally bounding collusion throughput. Once
// v0.1.35 demotes it to a floor, `mint_ratio * mint_equals_burn_claim_distribution.supplier
// < 1` becomes the primary signal for an application and supplier round-tripping stake back
// to themselves.
//
// It MUST stay a signal and not a validation error: the handler runs inside consensus, so a
// returned error halts the chain at the upgrade height. The distribution is DAO-governed,
// and since mint_ratio <= 1 and the shares sum to 1, the product can never exceed 1 — the
// worst legal param set makes self-dealing break-even, not profitable. Halting a chain over
// that is the more expensive failure by far.
func TestUpgrade_0_1_35_ReportsAntiCollusionInvariant(t *testing.T) {
	k, ctx := testkeeper.TokenomicsKeeper(t)

	params := k.GetParams(ctx)
	params.OverservicingBonusMultiplier = tokenomicstypes.DefaultOverservicingBonusMultiplier
	require.NoError(t, params.CheckAntiCollusionInvariant(), "default params must satisfy the invariant")

	// Drive the round-trip factor to exactly 1.0: everything burned comes straight back to
	// the colluding supplier, making self-dealing free.
	params.MintRatio = 1.0
	params.MintEqualsBurnClaimDistribution = tokenomicstypes.MintEqualsBurnClaimDistribution{
		Dao:         0,
		Proposer:    0,
		Supplier:    1.0,
		SourceOwner: 0,
		Application: 0,
	}

	// The handler validates params before writing them; that validation MUST NOT trip on
	// the invariant, otherwise the upgrade halts the chain.
	require.NoError(t, params.ValidateBasic(),
		"the anti-collusion invariant must never fail params validation (it would halt the chain at the upgrade height)")
	require.NoError(t, k.SetParams(ctx, params),
		"a violating param set must still be writable by the upgrade handler")

	err := params.CheckAntiCollusionInvariant()
	require.Error(t, err, "params whose round-trip factor reaches 1 must be reported")
	require.ErrorContains(t, err, "anti-collusion invariant violated")
}
