package keeper_test

import (
	"testing"

	cosmoslog "cosmossdk.io/log"
	cosmosmath "cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/pocket"
	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtest "github.com/pokt-network/poktroll/testutil/shared"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	tokenomicskeeper "github.com/pokt-network/poktroll/x/tokenomics/keeper"
	tlm "github.com/pokt-network/poktroll/x/tokenomics/token_logic_module"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// budgetRedistributionFixture holds a fully wired multi-supplier settlement scenario for a
// single (application, session) group, plus the derived floor/budget, so individual tests
// can run it under different overservicing_bonus_multiplier values.
type budgetRedistributionFixture struct {
	keepers   testkeeper.TokenomicsModuleKeepers
	ctx       cosmostypes.Context
	service   *sharedtypes.Service
	app       apptypes.Application
	suppliers []sharedtypes.Supplier
	relays    []uint64

	floor  int64 // per-supplier guaranteed floor B/N (== appStake/numPendingSessions/N)
	budget int64 // B == appStake/numPendingSessions == N*floor
}

// newBudgetRedistributionFixture builds an application with a stake sized so that the
// per-supplier floor equals `floorTarget`, plus one supplier per entry in `relayCounts`.
// Params are chosen so that claimed uPOKT == numRelays exactly (CUPR=1,
// CUTTM==granularity, difficulty=1, global inflation=0), and the mint=burn distribution
// pays the supplier 100% so ProcessTokenLogicModules returns exactly the post-cap amount.
func newBudgetRedistributionFixture(t testing.TB, floorTarget int64, relayCounts []uint64) *budgetRedistributionFixture {
	t.Helper()

	granularity := uint64(1_000_000)
	cuttm := uint64(1) * granularity
	cupr := uint64(1)
	service := prepareTestService(cupr)

	keepers, ctx := testkeeper.NewTokenomicsModuleKeepers(t,
		cosmoslog.NewNopLogger(),
		testkeeper.WithService(*service),
		testkeeper.WithDefaultModuleBalances(),
		testkeeper.WithTokenLogicModules([]tlm.TokenLogicModule{
			tlm.NewRelayBurnEqualsMintTLM(),
		}),
	)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(1)
	keepers.SetService(sdkCtx, *service)

	// Pin the compute-unit → uPOKT conversion so claimed uPOKT == numRelays.
	sharedParams := keepers.SharedKeeper.GetParams(sdkCtx)
	sharedParams.ComputeUnitsToTokensMultiplier = cuttm
	sharedParams.ComputeUnitCostGranularity = granularity
	require.NoError(t, keepers.SharedKeeper.SetParams(sdkCtx, sharedParams))

	// No global inflation (stake terms == settlement terms) and pay the supplier 100% so the
	// TLM output equals the cap decided by ensureClaimAmountLimits.
	tokenomicsParams := keepers.Keeper.GetParams(sdkCtx)
	tokenomicsParams.GlobalInflationPerClaim = 0
	tokenomicsParams.MintEqualsBurnClaimDistribution = tokenomicstypes.MintEqualsBurnClaimDistribution{
		Supplier: 1,
	}
	require.NoError(t, keepers.Keeper.SetParams(sdkCtx, tokenomicsParams))

	// floor = appStake / numPendingSessions / N  ⇒  appStake = floor * N * numPendingSessions.
	numPendingSessions := sharedtypes.GetNumPendingSessions(&sharedParams)
	numSuppliers := int64(len(relayCounts))
	appStakeAmt := cosmosmath.NewInt(floorTarget * numSuppliers * numPendingSessions)

	appStake := cosmostypes.NewCoin(pocket.DenomuPOKT, appStakeAmt)
	app := apptypes.Application{
		Address:        sample.AccAddressBech32(),
		Stake:          &appStake,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{{ServiceId: service.Id}},
	}
	keepers.SetApplication(sdkCtx, app)

	suppliers := make([]sharedtypes.Supplier, numSuppliers)
	for i := range relayCounts {
		supplierAddr := sample.AccAddressBech32()
		services := []*sharedtypes.SupplierServiceConfig{{
			ServiceId: service.Id,
			RevShare: []*sharedtypes.ServiceRevenueShare{{
				Address:            supplierAddr,
				RevSharePercentage: 100,
			}},
		}}
		supplierStake := cosmostypes.NewCoin(pocket.DenomuPOKT, cosmosmath.NewInt(1_000_000))
		suppliers[i] = sharedtypes.Supplier{
			OwnerAddress:         supplierAddr,
			OperatorAddress:      supplierAddr,
			Stake:                &supplierStake,
			Services:             services,
			ServiceConfigHistory: sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(supplierAddr, services, 1, 0),
		}
		keepers.SetAndIndexDehydratedSupplier(sdkCtx, suppliers[i])
	}

	return &budgetRedistributionFixture{
		keepers:   keepers,
		ctx:       sdkCtx,
		service:   service,
		app:       app,
		suppliers: suppliers,
		relays:    relayCounts,
		floor:     floorTarget,
		budget:    floorTarget * numSuppliers,
	}
}

// run executes the full two-phase settlement for the fixture under the given
// overservicing_bonus_multiplier and returns the post-cap settlement amount per supplier
// (indexed the same as f.suppliers / f.relays). `order` is the sequence of supplier indices
// to process in; pass nil for natural order. It resets the application stake first so the
// scenario can be replayed multiple times against the same keepers.
func (f *budgetRedistributionFixture) run(t testing.TB, m uint64, order []int) []int64 {
	t.Helper()

	if order == nil {
		order = make([]int, len(f.suppliers))
		for i := range order {
			order[i] = i
		}
	}

	// Reset app stake (Phase 2 burns it, and dropping below min-stake persists the app).
	freshStake := *f.app.Stake
	f.app.Stake = &freshStake
	f.keepers.SetApplication(f.ctx, f.app)

	tokenomicsParams := f.keepers.Keeper.GetParams(f.ctx)
	tokenomicsParams.OverservicingBonusMultiplier = m
	require.NoError(t, f.keepers.Keeper.SetParams(f.ctx, tokenomicsParams))

	sctx := tokenomicskeeper.NewSettlementContext(f.ctx, f.keepers.Keeper, f.keepers.Logger())

	// Phase 1: warm caches and count every claimant BEFORE any budget is computed.
	claims := make([]prooftypes.Claim, len(f.suppliers))
	for _, i := range order {
		claims[i] = prepareTestClaim(f.relays[i], f.service, &f.app, &f.suppliers[i])
		require.NoError(t, sctx.ClaimCacheWarmUp(f.ctx, &claims[i]))
		sctx.IncrementSupplierCount(claims[i].SessionHeader.ApplicationAddress, claims[i].SessionHeader.SessionId)
	}

	// Phase 1.5: accumulate each claim into its group's floor/unused/excess totals.
	for _, i := range order {
		require.NoError(t, sctx.AccumulateClaimBudget(f.ctx, &claims[i]))
	}

	// Phase 2: settle each claim against the precomputed budget.
	settled := make([]int64, len(f.suppliers))
	for _, i := range order {
		result := tlm.NewClaimSettlementResult(claims[i])
		coin, err := f.keepers.ProcessTokenLogicModules(f.ctx, sctx, result)
		require.NoError(t, err)
		settled[i] = coin.Amount.Int64()
	}
	return settled
}

// TestBudgetRedistribution_FloorVsRedistribute is the core test: two heavy overservicers
// and two idle/light suppliers share one application budget. With m=1 the heavy suppliers
// are capped at the floor (legacy head-split), stranding budget; with m=0 the unused budget
// is redistributed to the heavy suppliers, who are paid in full — and Σsettled == B.
func TestBudgetRedistribution_FloorVsRedistribute(t *testing.T) {
	const floor = int64(1000)
	// heavy = 1.5x floor (500 excess each), light = 0.5x floor (500 unused each).
	f := newBudgetRedistributionFixture(t, floor, []uint64{1500, 1500, 500, 500})

	// m=1: legacy head-split. Heavy capped at floor; light paid its (below-floor) claim.
	legacy := f.run(t, 1, nil)
	require.Equal(t, []int64{1000, 1000, 500, 500}, legacy,
		"m=1 must reproduce the legacy floor cap exactly")
	require.Equal(t, int64(3000), sum(legacy),
		"m=1 strands 1000 uPOKT of budget (2x500 heavy excess unpaid)")

	// m=0 must be BENIGN — identical to m=1 (legacy), never "unlimited". This is the safety
	// property: an unset/clobbered zero can never silently enable redistribution.
	require.Equal(t, legacy, f.run(t, 0, nil),
		"m=0 must be treated as the legacy floor cap (benign zero-value)")

	// m >= numSuppliers lifts the cap so the app budget B binds: unused (2x500=1000) is split
	// across the heavy suppliers in proportion to excess (2x500) → +500 each → both paid in
	// full at 1500.
	redistributed := f.run(t, 100, nil)
	require.Equal(t, []int64{1500, 1500, 500, 500}, redistributed,
		"a high multiplier must pay heavy overservicers in full from the unused budget")
	require.Equal(t, f.budget, sum(redistributed),
		"full redistribution recovers 100% of the budget: Σsettled == B")

	// Monotonicity (spec §2.1.4): no supplier is ever worse off than under m=1.
	for i := range legacy {
		require.GreaterOrEqual(t, redistributed[i], legacy[i],
			"supplier %d must not be paid less under redistribution", i)
	}
}

// TestBudgetRedistribution_BoundedByMultiplier verifies that a finite multiplier caps the
// bonus at m*floor even when more unused budget is available.
func TestBudgetRedistribution_BoundedByMultiplier(t *testing.T) {
	const floor = int64(1000)
	// One heavy (3x floor) and three idle-ish suppliers leaving plenty of unused budget.
	f := newBudgetRedistributionFixture(t, floor, []uint64{3000, 100, 100, 100})

	// m=2 caps the heavy supplier at 2*floor even though its full claim (3000) would fit in
	// the unused budget.
	settled := f.run(t, 2, nil)
	require.Equal(t, int64(2000), settled[0], "heavy supplier capped at m*floor = 2000")

	// A high multiplier pays it in full (well within B).
	require.Equal(t, int64(3000), f.run(t, 100, nil)[0], "high multiplier pays the heavy supplier in full")
}

// TestBudgetRedistribution_OrderIndependent asserts the consensus-critical property: the
// per-supplier settlement is invariant to the order claims are processed in (§2.1.5). The
// accumulation is a commutative integer sum, so a reversed order must yield identical
// results.
func TestBudgetRedistribution_OrderIndependent(t *testing.T) {
	const floor = int64(1000)
	f := newBudgetRedistributionFixture(t, floor, []uint64{1500, 1500, 500, 500})

	forward := f.run(t, 100, []int{0, 1, 2, 3})
	reversed := f.run(t, 100, []int{3, 2, 1, 0})
	require.Equal(t, forward, reversed,
		"per-supplier settlement must be independent of processing order")
}

// TestBudgetRedistribution_AllBelowFloor covers totalExcess == 0: when nobody exceeds the
// floor there is no division by zero and every claim is paid in full.
func TestBudgetRedistribution_AllBelowFloor(t *testing.T) {
	const floor = int64(1000)
	f := newBudgetRedistributionFixture(t, floor, []uint64{100, 200, 300, 400})

	settled := f.run(t, 100, nil)
	require.Equal(t, []int64{100, 200, 300, 400}, settled,
		"all-below-floor claims are paid in full with no bonus and no div-by-zero")
}

// TestBudgetRedistribution_SingleSupplier covers N==1: the floor equals the whole budget B,
// so a single supplier can be paid up to B.
func TestBudgetRedistribution_SingleSupplier(t *testing.T) {
	const floor = int64(1000) // with N=1, floor == B == appStake/numPendingSessions
	f := newBudgetRedistributionFixture(t, floor, []uint64{800})

	// Claim below the floor: paid in full.
	require.Equal(t, []int64{800}, f.run(t, 100, nil))

	// A claim above the floor is capped at the floor (== B); there is no other supplier to
	// donate unused budget, so redistribution cannot raise it even with a high multiplier.
	over := newBudgetRedistributionFixture(t, floor, []uint64{1500})
	require.Equal(t, []int64{1000}, over.run(t, 100, nil),
		"a lone supplier is bounded by B even with a high multiplier")
}

func sum(xs []int64) int64 {
	var total int64
	for _, x := range xs {
		total += x
	}
	return total
}
