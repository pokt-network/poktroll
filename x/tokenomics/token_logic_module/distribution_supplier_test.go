package token_logic_module

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/pocket"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

func TestGetSupplierShareholderAmountMap_UniqueAddresses(t *testing.T) {
	revShare := []*sharedtypes.ServiceRevenueShare{
		{Address: "pokt1addr1", RevSharePercentage: 30},
		{Address: "pokt1addr2", RevSharePercentage: 70},
	}
	amountToDistribute := math.NewInt(1000)

	shareMap, err := GetSupplierShareholderAmountMap(revShare, amountToDistribute)
	require.NoError(t, err)

	require.Len(t, shareMap, 2)
	require.Equal(t, math.NewInt(300), shareMap["pokt1addr1"])
	require.Equal(t, math.NewInt(700), shareMap["pokt1addr2"])

	// Verify total distributed equals input
	total := math.NewInt(0)
	for _, amt := range shareMap {
		total = total.Add(amt)
	}
	require.Equal(t, amountToDistribute, total)
}

// TestGetSupplierShareholderAmountMap_DuplicateAddresses asserts that the
// dedupe-rejection layer added in v0.1.34 catches duplicate addresses BEFORE
// the math runs. Pre-v0.1.34 the function silently overwrote the first
// occurrence in the result map (data loss); this test pins the new contract:
// duplicates return an error so the caller can route to the owner-fallback.
func TestGetSupplierShareholderAmountMap_DuplicateAddresses(t *testing.T) {
	// Same address appears twice. Sum is 100 — would have masked the data-loss
	// bug pre-v0.1.34, which is exactly the regression we are guarding against.
	revShare := []*sharedtypes.ServiceRevenueShare{
		{Address: "pokt1duplicate", RevSharePercentage: 10},
		{Address: "pokt1duplicate", RevSharePercentage: 90},
	}
	amountToDistribute := math.NewInt(1000)

	shareMap, err := GetSupplierShareholderAmountMap(revShare, amountToDistribute)
	require.Error(t, err)
	require.Nil(t, shareMap)
	require.Contains(t, err.Error(), "duplicate revshare recipient address")
}

func TestGetSupplierShareholderAmountMap_Remainder(t *testing.T) {
	// 3 shareholders splitting 100 uPOKT at 33/33/34 — tests remainder allocation
	revShare := []*sharedtypes.ServiceRevenueShare{
		{Address: "pokt1a", RevSharePercentage: 33},
		{Address: "pokt1b", RevSharePercentage: 33},
		{Address: "pokt1c", RevSharePercentage: 34},
	}
	amountToDistribute := math.NewInt(100)

	shareMap, err := GetSupplierShareholderAmountMap(revShare, amountToDistribute)
	require.NoError(t, err)

	require.Len(t, shareMap, 3)

	// Verify total distributed equals input (remainder goes to first shareholder)
	total := math.NewInt(0)
	for _, amt := range shareMap {
		total = total.Add(amt)
	}
	require.Equal(t, amountToDistribute, total)
}

// TestGetSupplierShareholderAmountMap_SumOver100 pins the over-100% rejection
// path. This is the audit's defining scenario: migration `mergeRevShareDuplicates`
// can produce sum > 100 from pre-v0.1.34 duplicate-revshare state, and the old
// settlement path computed a NEGATIVE remainder that NewCoin panicked on.
func TestGetSupplierShareholderAmountMap_SumOver100(t *testing.T) {
	revShare := []*sharedtypes.ServiceRevenueShare{
		{Address: "pokt1a", RevSharePercentage: 110}, // post-merge: dup of 60 + 50
		{Address: "pokt1b", RevSharePercentage: 30},
	}
	amountToDistribute := math.NewInt(100)

	shareMap, err := GetSupplierShareholderAmountMap(revShare, amountToDistribute)
	require.Error(t, err)
	require.Nil(t, shareMap)
	require.Contains(t, err.Error(), "sum 140 != required 100")
}

// TestGetSupplierShareholderAmountMap_SumUnder100 ensures sum < 100 also
// triggers rejection. Symmetric defense; covers hand-edited or partially
// migrated state.
func TestGetSupplierShareholderAmountMap_SumUnder100(t *testing.T) {
	revShare := []*sharedtypes.ServiceRevenueShare{
		{Address: "pokt1a", RevSharePercentage: 50},
		{Address: "pokt1b", RevSharePercentage: 30},
	}
	amountToDistribute := math.NewInt(100)

	shareMap, err := GetSupplierShareholderAmountMap(revShare, amountToDistribute)
	require.Error(t, err)
	require.Nil(t, shareMap)
	require.Contains(t, err.Error(), "sum 80 != required 100")
}

// TestGetSupplierShareholderAmountMap_NilEntry asserts that any nil entry in
// the revshare slice causes a clean rejection (vs. a nil-deref panic on
// `rs.Address`).
func TestGetSupplierShareholderAmountMap_NilEntry(t *testing.T) {
	revShare := []*sharedtypes.ServiceRevenueShare{
		{Address: "pokt1a", RevSharePercentage: 50},
		nil,
		{Address: "pokt1b", RevSharePercentage: 50},
	}
	amountToDistribute := math.NewInt(100)

	shareMap, err := GetSupplierShareholderAmountMap(revShare, amountToDistribute)
	require.Error(t, err)
	require.Nil(t, shareMap)
	require.Contains(t, err.Error(), "nil revshare entry")
}

// TestGetSupplierShareholderAmountMap_EmptyList covers the empty-list edge
// case. A zero-length slice is rejected to prevent an out-of-bounds index
// access on serviceRevShare[0] when adding the remainder.
func TestGetSupplierShareholderAmountMap_EmptyList(t *testing.T) {
	shareMap, err := GetSupplierShareholderAmountMap(nil, math.NewInt(100))
	require.Error(t, err)
	require.Nil(t, shareMap)
	require.Contains(t, err.Error(), "empty revshare list")
}

// newFallbackTestCtx returns a minimal sdk.Context configured with a fresh
// EventManager so emitted events can be inspected with em.Events().
func newFallbackTestCtx(t *testing.T) (cosmostypes.Context, *cosmostypes.EventManager) {
	t.Helper()
	em := cosmostypes.NewEventManager()
	sdkCtx := cosmostypes.Context{}.
		WithBlockHeader(cmtproto.Header{Height: 1}).
		WithEventManager(em)
	return sdkCtx, em
}

// newFallbackTestResult returns a minimally-populated ClaimSettlementResult so
// `result.GetSessionEndHeight()` returns the configured height for the emitted
// event's `session_end_block_height` field.
func newFallbackTestResult(sessionEndHeight int64) *tokenomicstypes.ClaimSettlementResult {
	return &tokenomicstypes.ClaimSettlementResult{
		Claim: prooftypes.Claim{
			SupplierOperatorAddress: "pokt1opdummy",
			SessionHeader: &sessiontypes.SessionHeader{
				SessionId:             "test-session",
				ServiceId:             "svc1",
				SessionEndBlockHeight: sessionEndHeight,
				ApplicationAddress:    "pokt1app",
			},
		},
	}
}

// countFallbackEvents returns the number of EventSupplierRevShareFallbackDistribution
// typed events recorded on em. Useful because EmitTypedEvent records events with
// the message proto FQN as the event type — counting by string keeps the test
// resilient to ordering/attribute changes.
func countFallbackEvents(t *testing.T, em *cosmostypes.EventManager) int {
	t.Helper()
	count := 0
	const eventType = "pocket.tokenomics.EventSupplierRevShareFallbackDistribution"
	for _, ev := range em.Events() {
		if ev.Type == eventType {
			count++
		}
	}
	return count
}

// TestDistributeSupplierRewardsToShareholders_HappyPath asserts the normal
// distribution path still works after the validation refactor: sum=100, no
// duplicates, no nils — the 3 shareholders each get their proportional
// amount and NO fallback event is emitted.
func TestDistributeSupplierRewardsToShareholders_HappyPath(t *testing.T) {
	sdkCtx, em := newFallbackTestCtx(t)
	result := newFallbackTestResult(100)
	supplier := &sharedtypes.Supplier{
		OperatorAddress: "pokt1opNORMAL",
		OwnerAddress:    "pokt1ownerNORMAL",
		Services: []*sharedtypes.SupplierServiceConfig{{
			ServiceId: "svc1",
			RevShare: []*sharedtypes.ServiceRevenueShare{
				{Address: "pokt1a", RevSharePercentage: 30},
				{Address: "pokt1b", RevSharePercentage: 70},
			},
		}},
	}

	err := distributeSupplierRewardsToShareholders(
		sdkCtx,
		log.NewNopLogger(),
		result,
		tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SUPPLIER_SHAREHOLDER_REWARD_DISTRIBUTION,
		supplier,
		"svc1",
		math.NewInt(1000),
	)
	require.NoError(t, err)

	// Both configured shareholders received their proportional cut.
	require.Len(t, result.ModToAcctTransfers, 2)
	amounts := map[string]int64{}
	for _, tr := range result.ModToAcctTransfers {
		require.Equal(t, suppliertypes.ModuleName, tr.SenderModule)
		require.Equal(t, pocket.DenomuPOKT, tr.Coin.Denom)
		amounts[tr.RecipientAddress] = tr.Coin.Amount.Int64()
	}
	require.Equal(t, int64(300), amounts["pokt1a"])
	require.Equal(t, int64(700), amounts["pokt1b"])

	// No fallback engaged → no fallback event emitted.
	require.Equal(t, 0, countFallbackEvents(t, em),
		"happy-path distribution must NOT emit EventSupplierRevShareFallbackDistribution")
}

// TestDistributeSupplierRewardsToShareholders_SumOver100_OwnerFallback is the
// audit's defining scenario. The supplier's RevShare is left in an invalid
// post-migration state (sum=140). The fallback must:
//
//	(a) NOT halt the chain (no panic),
//	(b) queue exactly ONE mod-to-acct transfer paying the full amount to the
//	    supplier's owner_address,
//	(c) emit EventSupplierRevShareFallbackDistribution carrying the operator,
//	    owner, service_id, session_end_block_height, the full amount, op_reason,
//	    observed_sum_percentage=140, and the rejection reason.
//
// None of the original revshare recipients should receive any uPOKT.
func TestDistributeSupplierRewardsToShareholders_SumOver100_OwnerFallback(t *testing.T) {
	sdkCtx, em := newFallbackTestCtx(t)
	result := newFallbackTestResult(123)
	const (
		operatorAddr = "pokt1opBAD"
		ownerAddr    = "pokt1ownerBAD"
		bystanderA   = "pokt1payee_a"
		bystanderB   = "pokt1payee_b"
	)
	supplier := &sharedtypes.Supplier{
		OperatorAddress: operatorAddr,
		OwnerAddress:    ownerAddr,
		Services: []*sharedtypes.SupplierServiceConfig{{
			ServiceId: "svc1",
			RevShare: []*sharedtypes.ServiceRevenueShare{
				// Simulates migration-time merge of duplicate revshares.
				// Original: [(a,60),(b,30),(a,50)] -> merged: [(a,110),(b,30)] -> sum=140.
				{Address: bystanderA, RevSharePercentage: 110},
				{Address: bystanderB, RevSharePercentage: 30},
			},
		}},
	}

	const opReason = tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SUPPLIER_SHAREHOLDER_REWARD_DISTRIBUTION
	require.NotPanics(t, func() {
		require.NoError(t, distributeSupplierRewardsToShareholders(
			sdkCtx,
			log.NewNopLogger(),
			result,
			opReason,
			supplier,
			"svc1",
			math.NewInt(1000),
		))
	})

	// Exactly one transfer: full amount to owner. The bystanderA/B "configured"
	// recipients must NOT receive anything — their config was rejected.
	require.Len(t, result.ModToAcctTransfers, 1, "fallback must produce exactly ONE mod-to-acct transfer")
	tr := result.ModToAcctTransfers[0]
	require.Equal(t, ownerAddr, tr.RecipientAddress, "fallback recipient must be the supplier's owner_address")
	require.Equal(t, int64(1000), tr.Coin.Amount.Int64(), "fallback transfer must be the FULL amount")
	require.Equal(t, pocket.DenomuPOKT, tr.Coin.Denom)
	require.Equal(t, suppliertypes.ModuleName, tr.SenderModule)
	require.Equal(t, opReason, tr.OpReason)

	require.Equal(t, 1, countFallbackEvents(t, em),
		"fallback path must emit exactly one EventSupplierRevShareFallbackDistribution")
}

// TestDistributeSupplierRewardsToShareholders_SumUnder100_OwnerFallback covers
// the symmetric undershoot case (sum < 100). Same fallback behavior.
func TestDistributeSupplierRewardsToShareholders_SumUnder100_OwnerFallback(t *testing.T) {
	sdkCtx, em := newFallbackTestCtx(t)
	result := newFallbackTestResult(123)
	supplier := &sharedtypes.Supplier{
		OperatorAddress: "pokt1opU",
		OwnerAddress:    "pokt1ownerU",
		Services: []*sharedtypes.SupplierServiceConfig{{
			ServiceId: "svc1",
			RevShare: []*sharedtypes.ServiceRevenueShare{
				{Address: "pokt1payee_a", RevSharePercentage: 50},
				{Address: "pokt1payee_b", RevSharePercentage: 30},
			},
		}},
	}

	const opReason = tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_SUPPLIER_SHAREHOLDER_REWARD_DISTRIBUTION
	require.NoError(t, distributeSupplierRewardsToShareholders(
		sdkCtx,
		log.NewNopLogger(),
		result,
		opReason,
		supplier,
		"svc1",
		math.NewInt(500),
	))

	require.Len(t, result.ModToAcctTransfers, 1)
	require.Equal(t, "pokt1ownerU", result.ModToAcctTransfers[0].RecipientAddress)
	require.Equal(t, int64(500), result.ModToAcctTransfers[0].Coin.Amount.Int64())
	require.Equal(t, 1, countFallbackEvents(t, em))
}

// TestDistributeSupplierRewardsToShareholders_DuplicateAddresses_OwnerFallback
// asserts that even when sum==100 (e.g. raw [(a,30),(a,70)]), duplicate addresses
// in the configured list ALSO trigger the owner-fallback rather than allowing
// silent map-overwrite data loss.
func TestDistributeSupplierRewardsToShareholders_DuplicateAddresses_OwnerFallback(t *testing.T) {
	sdkCtx, em := newFallbackTestCtx(t)
	result := newFallbackTestResult(456)
	supplier := &sharedtypes.Supplier{
		OperatorAddress: "pokt1opD",
		OwnerAddress:    "pokt1ownerD",
		Services: []*sharedtypes.SupplierServiceConfig{{
			ServiceId: "svc1",
			RevShare: []*sharedtypes.ServiceRevenueShare{
				{Address: "pokt1same", RevSharePercentage: 30},
				{Address: "pokt1same", RevSharePercentage: 70}, // same address again, sum=100
			},
		}},
	}

	require.NoError(t, distributeSupplierRewardsToShareholders(
		sdkCtx,
		log.NewNopLogger(),
		result,
		tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SUPPLIER_SHAREHOLDER_REWARD_DISTRIBUTION,
		supplier,
		"svc1",
		math.NewInt(1000),
	))

	require.Len(t, result.ModToAcctTransfers, 1)
	require.Equal(t, "pokt1ownerD", result.ModToAcctTransfers[0].RecipientAddress,
		"duplicate-address revshare must route to owner fallback, NOT pay the duplicate twice or once")
	require.Equal(t, int64(1000), result.ModToAcctTransfers[0].Coin.Amount.Int64())
	require.Equal(t, 1, countFallbackEvents(t, em))
}

// TestDistributeSupplierRewardsToShareholders_EmptyOwner_FaultyClaim asserts
// the defensive branch where the supplier somehow has an empty owner address
// (should be impossible after stake-time validation). The function must return
// an error WITHOUT panicking — settlement treats this as a faulty claim and
// continues with other claims.
func TestDistributeSupplierRewardsToShareholders_EmptyOwner_FaultyClaim(t *testing.T) {
	sdkCtx, _ := newFallbackTestCtx(t)
	result := newFallbackTestResult(789)
	supplier := &sharedtypes.Supplier{
		OperatorAddress: "pokt1opE",
		OwnerAddress:    "", // empty owner blocks the fallback recipient
		Services: []*sharedtypes.SupplierServiceConfig{{
			ServiceId: "svc1",
			RevShare: []*sharedtypes.ServiceRevenueShare{
				{Address: "pokt1payee_a", RevSharePercentage: 110}, // sum != 100 → triggers fallback path
				{Address: "pokt1payee_b", RevSharePercentage: 30},
			},
		}},
	}

	err := distributeSupplierRewardsToShareholders(
		sdkCtx,
		log.NewNopLogger(),
		result,
		tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SUPPLIER_SHAREHOLDER_REWARD_DISTRIBUTION,
		supplier,
		"svc1",
		math.NewInt(1000),
	)
	require.Error(t, err, "empty owner with invalid revshare must surface a settlement-side error (faulty claim), not panic")
	require.Len(t, result.ModToAcctTransfers, 0, "no transfer should be queued when fallback recipient is missing")
}
