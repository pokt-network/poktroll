package keeper_test

import (
	"testing"

	cosmoslog "cosmossdk.io/log"

	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	testproof "github.com/pokt-network/poktroll/testutil/proof"
	testsession "github.com/pokt-network/poktroll/testutil/session"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	tokenomicskeeper "github.com/pokt-network/poktroll/x/tokenomics/keeper"
)

// BenchmarkGetClaimeduPOKT measures the single arithmetic operation that Phase 1.5 performs
// one additional time per claim (relative to pre-change settlement) to price each claim for
// the budget accumulation. It does NO store reads — it is pure big.Rat arithmetic over
// already-loaded shared params + relay-mining difficulty. Multiply ns/op by the per-block
// claim count (~2,700 on mainnet) to bound the added CPU cost of the extra pass.
func BenchmarkGetClaimeduPOKT(b *testing.B) {
	numRelays := uint64(1000)
	cupr := uint64(1)
	service := prepareTestService(cupr)

	difficulty := servicetypes.RelayMiningDifficulty{
		ServiceId:  service.Id,
		TargetHash: protocol.BaseRelayDifficultyHashBz, // difficulty 1: no scaling
	}

	sharedParams := sharedtypes.DefaultParams()
	sharedParams.ComputeUnitsToTokensMultiplier = 1_000_000
	sharedParams.ComputeUnitCostGranularity = 1_000_000

	claim := prooftypes.Claim{
		SessionHeader: &sessiontypes.SessionHeader{
			ApplicationAddress:      "pokt1benchapp",
			ServiceId:               service.Id,
			SessionId:               "bench_session",
			SessionStartBlockHeight: 1,
			SessionEndBlockHeight:   testsession.GetSessionEndHeightWithDefaultParams(1),
		},
		RootHash: testproof.SmstRootWithSumAndCount(numRelays*cupr, numRelays),
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := claim.GetClaimeduPOKT(sharedParams, difficulty); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPhase15_AccumulateClaimBudget measures the true marginal cost Phase 1.5 adds per
// claim in steady state: a difficulty map lookup + GetClaimeduPOKT + global-inflation
// arithmetic + a memoized session-budget lookup + two math.Int adds. The session budget is
// pre-initialized (getOrInitSessionBudget's one store read, GetParamsAtHeight, is amortized
// across the group's claims and excluded here), so this reflects the per-claim overhead for
// every claim after the first in each (app, session) group.
//
// The claim is VALIDATED, so claimWillSettle short-circuits without resolving the proof
// requirement. That is the steady-state case worth measuring: without the short-circuit this
// benchmark reports ~28.5us/claim instead of ~1.7us, because ProofRequirementForClaim runs
// three history iterators that Phase 2 then repeats for the same claim.
func BenchmarkPhase15_AccumulateClaimBudget(b *testing.B) {
	f := newBudgetRedistributionFixture(b, 1000, []uint64{1500})

	sctx := tokenomicskeeper.NewSettlementContext(f.ctx, f.keepers.Keeper, cosmoslog.NewNopLogger())
	claim := prepareTestClaim(f.relays[0], f.service, &f.app, &f.suppliers[0])
	if err := sctx.ClaimCacheWarmUp(f.ctx, &claim); err != nil {
		b.Fatal(err)
	}
	sctx.IncrementSupplierCount(claim.SessionHeader.ApplicationAddress, claim.SessionHeader.SessionId)
	// Prime the group so getOrInitSessionBudget's store read is amortized away.
	if err := sctx.AccumulateClaimBudget(f.ctx, &claim); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := sctx.AccumulateClaimBudget(f.ctx, &claim); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPhase15_Group50 measures the full Phase 1.5 wall cost for one realistic 50-claim
// (app, session) group: a fresh settlement context, warm every claim, then accumulate. Note
// the warmup store reads counted here are MOVED from Phase 2 (not added) — Phase 2's warmup
// becomes a cache hit — so this over-counts the change's true cost.
func BenchmarkPhase15_Group50(b *testing.B) {
	const groupSize = 50
	relays := make([]uint64, groupSize)
	for i := range relays {
		relays[i] = uint64(300 + i*40) // spread around the floor: some below, some above
	}
	f := newBudgetRedistributionFixture(b, 1000, relays)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sctx := tokenomicskeeper.NewSettlementContext(f.ctx, f.keepers.Keeper, cosmoslog.NewNopLogger())
		claims := make([]prooftypes.Claim, groupSize)
		for j := range f.suppliers {
			claims[j] = prepareTestClaim(f.relays[j], f.service, &f.app, &f.suppliers[j])
			if err := sctx.ClaimCacheWarmUp(f.ctx, &claims[j]); err != nil {
				b.Fatal(err)
			}
			sctx.IncrementSupplierCount(claims[j].SessionHeader.ApplicationAddress, claims[j].SessionHeader.SessionId)
		}
		for j := range claims {
			if err := sctx.AccumulateClaimBudget(f.ctx, &claims[j]); err != nil {
				b.Fatal(err)
			}
		}
	}
}
