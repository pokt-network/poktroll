package keeper

import (
	"fmt"
	"runtime"
	"testing"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	gogoproto "github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"

	tlm "github.com/pokt-network/poktroll/x/tokenomics/token_logic_module"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// Event profile per SDK bank operation (from cosmos-sdk v0.53.0):
//
//	MintCoins:                    2 events (coin_received + coin_mint)
//	BurnCoins:                    2 events (coin_spent + coin_burn)
//	SendCoins (mod→mod, mod→acct): 4 events (coin_spent + coin_received + transfer + message)
//
// Plus, each EventClaimSettled emitted per claim adds 1 more event.
// On v0.1.31 mainnet with 2,551 claims: 8 ops/claim × 2,551 = 20,408 bank ops → ~61K SDK events.
// CometBFT indexes ALL events into LevelDB → 5.8M entries at block 651093.

const (
	benchDenom          = "upokt"
	numMainnetSuppliers = 200
)

// generateRealisticResults creates N ClaimSettlementResults modeling mainnet's
// per-claim operation profile: 2 mints, 1 burn, 2 mod-to-mod, 3 mod-to-acct.
func generateRealisticResults(n int) tlm.ClaimSettlementResults {
	results := make(tlm.ClaimSettlementResults, n)

	for i := range results {
		supplierOwner := fmt.Sprintf("pokt1supplier%04d", i%numMainnetSuppliers)
		daoAddr := "pokt1dao_address_mainnet"
		amt := int64(1000 + (i % 5000))

		results[i] = &tokenomicstypes.ClaimSettlementResult{
			Mints: []tokenomicstypes.MintBurnOp{
				{DestinationModule: "supplier", OpReason: tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SUPPLIER_STAKE_MINT, Coin: cosmostypes.NewCoin(benchDenom, math.NewInt(amt))},
				{DestinationModule: "tokenomics", OpReason: tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_INFLATION, Coin: cosmostypes.NewCoin(benchDenom, math.NewInt(amt/1000+1))},
			},
			Burns: []tokenomicstypes.MintBurnOp{
				{DestinationModule: "application", OpReason: tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_APPLICATION_STAKE_BURN, Coin: cosmostypes.NewCoin(benchDenom, math.NewInt(amt))},
			},
			ModToModTransfers: []tokenomicstypes.ModToModTransfer{
				{SenderModule: "tokenomics", RecipientModule: "supplier", OpReason: tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_SUPPLIER_SHAREHOLDER_REWARD_MODULE_TRANSFER, Coin: cosmostypes.NewCoin(benchDenom, math.NewInt(amt/1000+1))},
				{SenderModule: "tokenomics", RecipientModule: "application", OpReason: tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_REIMBURSEMENT_REQUEST_ESCROW_MODULE_TRANSFER, Coin: cosmostypes.NewCoin(benchDenom, math.NewInt(amt/1000+1))},
			},
			ModToAcctTransfers: []tokenomicstypes.ModToAcctTransfer{
				{SenderModule: "supplier", RecipientAddress: supplierOwner, OpReason: tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SUPPLIER_SHAREHOLDER_REWARD_DISTRIBUTION, Coin: cosmostypes.NewCoin(benchDenom, math.NewInt(amt*80/100))},
				{SenderModule: "supplier", RecipientAddress: daoAddr, OpReason: tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_DAO_REWARD_DISTRIBUTION, Coin: cosmostypes.NewCoin(benchDenom, math.NewInt(amt*20/100))},
				{SenderModule: "supplier", RecipientAddress: supplierOwner, OpReason: tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_SUPPLIER_SHAREHOLDER_REWARD_DISTRIBUTION, Coin: cosmostypes.NewCoin(benchDenom, math.NewInt(amt/1000+1))},
			},
		}
	}

	return results
}

// simulateBankEvents mimics the exact events the SDK bank keeper emits for each
// bank operation type. This is what accumulates in the EventManager during a block
// and gets indexed by CometBFT — the root cause of the 11.4 GB memory leak.
//
// Events per operation (from cosmos-sdk v0.53.0 bank keeper source):
//   - MintCoins:  coin_received (addCoins) + coin_mint = 2 events
//   - BurnCoins:  coin_spent (subUnlockedCoins) + coin_burn = 2 events
//   - SendCoins:  coin_spent + coin_received + transfer + message = 4 events
func simulateBankEvents(em *cosmostypes.EventManager, numMints, numBurns, numSends int) {
	coins := cosmostypes.NewCoins(cosmostypes.NewCoin(benchDenom, math.NewInt(1000)))
	addr := cosmostypes.AccAddress("pokt1testaddr000000000")

	for range numMints {
		em.EmitEvent(banktypes.NewCoinReceivedEvent(addr, coins))
		em.EmitEvent(banktypes.NewCoinMintEvent(addr, coins))
	}

	for range numBurns {
		em.EmitEvent(banktypes.NewCoinSpentEvent(addr, coins))
		em.EmitEvent(banktypes.NewCoinBurnEvent(addr, coins))
	}

	for range numSends {
		em.EmitEvent(banktypes.NewCoinSpentEvent(addr, coins))
		em.EmitEvent(banktypes.NewCoinReceivedEvent(addr, coins))
		em.EmitEvent(cosmostypes.NewEvent(
			banktypes.EventTypeTransfer,
			cosmostypes.NewAttribute(banktypes.AttributeKeyRecipient, addr.String()),
			cosmostypes.NewAttribute(banktypes.AttributeKeySender, addr.String()),
			cosmostypes.NewAttribute(cosmostypes.AttributeKeyAmount, coins.String()),
		))
		em.EmitEvent(cosmostypes.NewEvent(
			cosmostypes.EventTypeMessage,
			cosmostypes.NewAttribute(banktypes.AttributeKeySender, addr.String()),
		))
	}
}

// TestSettlementEventMemoryComparison simulates the full downstream memory impact
// of v0.1.31 (per-claim bank ops) vs v0.1.33 (aggregated bank ops).
//
// It creates real SDK EventManager instances, emits the exact events the bank
// keeper would emit, and measures actual heap growth.
//
// Run with: go test -run TestSettlementEventMemoryComparison -v -count=1 -tags test ./x/tokenomics/keeper/
func TestSettlementEventMemoryComparison(t *testing.T) {
	for _, numClaims := range []int{2500, 5000, 10000} {
		t.Run(fmt.Sprintf("claims_%d", numClaims), func(t *testing.T) {
			results := generateRealisticResults(numClaims)

			// Count pre-aggregation operations.
			var preMints, preBurns, preSends int
			for _, r := range results {
				preMints += len(r.GetMints())
				preBurns += len(r.GetBurns())
				preSends += len(r.GetModToModTransfers()) + len(r.GetModToAcctTransfers())
			}
			preTotalOps := preMints + preBurns + preSends
			preEventCount := preMints*2 + preBurns*2 + preSends*4
			// Add 1 EventClaimSettled per claim (v0.1.31 behavior).
			preEventCount += numClaims

			// === Simulate v0.1.31: per-claim bank ops ===
			runtime.GC()
			var memBefore runtime.MemStats
			runtime.ReadMemStats(&memBefore)

			emOld := cosmostypes.NewEventManager()
			simulateBankEvents(emOld, preMints, preBurns, preSends)
			// Simulate per-claim EventClaimSettled emission.
			for range numClaims {
				emOld.EmitEvent(cosmostypes.NewEvent(
					"pocket.tokenomics.EventClaimSettled",
					cosmostypes.NewAttribute("supplier_operator_address", "pokt1supplier0000"),
					cosmostypes.NewAttribute("num_relays", "100"),
					cosmostypes.NewAttribute("num_claimed_compute_units", "1000"),
					cosmostypes.NewAttribute("claimed_upokt", "5000upokt"),
					cosmostypes.NewAttribute("service_id", "svc01"),
					cosmostypes.NewAttribute("session_end_block_height", "1000"),
				))
			}

			runtime.GC()
			var memAfterOld runtime.MemStats
			runtime.ReadMemStats(&memAfterOld)
			oldHeapBytes := memAfterOld.HeapInuse - memBefore.HeapInuse
			oldAllocBytes := memAfterOld.TotalAlloc - memBefore.TotalAlloc
			oldEventCount := len(emOld.Events())

			// === Simulate v0.1.33: aggregated bank ops ===
			aggMints, err := aggregateMints(results)
			require.NoError(t, err)
			aggBurns, err := aggregateBurns(results)
			require.NoError(t, err)
			aggModToMod, err := aggregateModToModTransfers(results)
			require.NoError(t, err)
			aggModToAcct, err := aggregateModToAcctTransfers(results)
			require.NoError(t, err)

			postMints := len(aggMints)
			postBurns := len(aggBurns)
			postSends := len(aggModToMod) + len(aggModToAcct)
			postTotalOps := postMints + postBurns + postSends
			// v0.1.33: still emits per-claim EventClaimSettled + per-aggregated-op EventSettlementBatch.
			postBatchEvents := postTotalOps // 1 EventSettlementBatch per aggregated op

			runtime.GC()
			var memBefore2 runtime.MemStats
			runtime.ReadMemStats(&memBefore2)

			emNew := cosmostypes.NewEventManager()
			simulateBankEvents(emNew, postMints, postBurns, postSends)
			// Per-claim EventClaimSettled still emitted in v0.1.33.
			for range numClaims {
				emNew.EmitEvent(cosmostypes.NewEvent(
					"pocket.tokenomics.EventClaimSettled",
					cosmostypes.NewAttribute("supplier_operator_address", "pokt1supplier0000"),
					cosmostypes.NewAttribute("num_relays", "100"),
					cosmostypes.NewAttribute("num_claimed_compute_units", "1000"),
					cosmostypes.NewAttribute("claimed_upokt", "5000upokt"),
					cosmostypes.NewAttribute("service_id", "svc01"),
					cosmostypes.NewAttribute("session_end_block_height", "1000"),
				))
			}
			// EventSettlementBatch events (one per aggregated op).
			for range postBatchEvents {
				emNew.EmitEvent(cosmostypes.NewEvent(
					"pocket.tokenomics.EventSettlementBatch",
					cosmostypes.NewAttribute("op_type", "mint"),
					cosmostypes.NewAttribute("total_amount", "1000000upokt"),
					cosmostypes.NewAttribute("num_claims", "2500"),
				))
			}

			runtime.GC()
			var memAfterNew runtime.MemStats
			runtime.ReadMemStats(&memAfterNew)
			newHeapBytes := memAfterNew.HeapInuse - memBefore2.HeapInuse
			newAllocBytes := memAfterNew.TotalAlloc - memBefore2.TotalAlloc
			newEventCount := len(emNew.Events())

			// === Report ===
			opReductionPct := float64(preTotalOps-postTotalOps) / float64(preTotalOps) * 100
			eventReductionPct := float64(oldEventCount-newEventCount) / float64(oldEventCount) * 100

			t.Logf("\n╔══════════════════════════════════════════════════════════════╗")
			t.Logf("║  Settlement Memory Simulation: %d claims                    ║", numClaims)
			t.Logf("╠══════════════════════════════════════════════════════════════╣")
			t.Logf("║                                                              ║")
			t.Logf("║  BANK OPERATIONS                                             ║")
			t.Logf("║    v0.1.31 (per-claim):  %6d ops  (mint=%d burn=%d send=%d)", preTotalOps, preMints, preBurns, preSends)
			t.Logf("║    v0.1.33 (aggregated): %6d ops  (mint=%d burn=%d send=%d)", postTotalOps, postMints, postBurns, postSends)
			t.Logf("║    Reduction:            %6.2f%%", opReductionPct)
			t.Logf("║                                                              ║")
			t.Logf("║  SDK EVENTS (accumulated in EventManager)                    ║")
			t.Logf("║    v0.1.31: %6d events", oldEventCount)
			t.Logf("║    v0.1.33: %6d events", newEventCount)
			t.Logf("║    Reduction: %6.2f%%", eventReductionPct)
			t.Logf("║                                                              ║")
			t.Logf("║  HEAP MEMORY (EventManager + events in memory)               ║")
			t.Logf("║    v0.1.31: %6.1f MB heap-in-use, %6.1f MB total alloc", float64(oldHeapBytes)/(1024*1024), float64(oldAllocBytes)/(1024*1024))
			t.Logf("║    v0.1.33: %6.1f MB heap-in-use, %6.1f MB total alloc", float64(newHeapBytes)/(1024*1024), float64(newAllocBytes)/(1024*1024))
			t.Logf("║    Heap saved: ~%.1f MB", float64(oldHeapBytes-newHeapBytes)/(1024*1024))
			t.Logf("║                                                              ║")
			t.Logf("║  EXTRAPOLATION TO MAINNET (block 651093 had 5.8M events)     ║")
			t.Logf("║    Mainnet pre-agg events per 2551 claims: ~%d", preEventCount*2551/numClaims)
			t.Logf("║    Mainnet post-agg events: ~%d", newEventCount*2551/numClaims)
			mainnetHeapSavedMB := float64(oldAllocBytes-newAllocBytes) * float64(2551) / float64(numClaims) / (1024 * 1024)
			t.Logf("║    Estimated mainnet alloc saved: ~%.0f MB (%.1f GB)", mainnetHeapSavedMB, mainnetHeapSavedMB/1024)
			t.Logf("╚══════════════════════════════════════════════════════════════╝")

			// Assertions.
			require.Less(t, postTotalOps, preTotalOps)
			require.Less(t, newEventCount, oldEventCount)
			if numClaims >= 2500 {
				require.Greater(t, opReductionPct, 95.0, "expected >95%% op reduction at mainnet scale")
				require.Greater(t, eventReductionPct, 80.0, "expected >80%% event reduction at mainnet scale")
			}
		})
	}
}

// BenchmarkAggregation_MainnetScale benchmarks the full aggregation pipeline
// at mainnet scale (2,500 claims).
func BenchmarkAggregation_MainnetScale(b *testing.B) {
	results := generateRealisticResults(2500)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		aggregateMints(results)
		aggregateBurns(results)
		aggregateModToModTransfers(results)
		aggregateModToAcctTransfers(results)
	}
}

// BenchmarkEventManager_PreVsPostAggregation benchmarks the cost of emitting
// events into the SDK EventManager — the direct cause of the memory leak.
func BenchmarkEventManager_PreVsPostAggregation(b *testing.B) {
	results := generateRealisticResults(2500)

	// Count ops.
	var preMints, preBurns, preSends int
	for _, r := range results {
		preMints += len(r.GetMints())
		preBurns += len(r.GetBurns())
		preSends += len(r.GetModToModTransfers()) + len(r.GetModToAcctTransfers())
	}

	aggMints, _ := aggregateMints(results)
	aggBurns, _ := aggregateBurns(results)
	aggModToMod, _ := aggregateModToModTransfers(results)
	aggModToAcct, _ := aggregateModToAcctTransfers(results)

	postMints := len(aggMints)
	postBurns := len(aggBurns)
	postSends := len(aggModToMod) + len(aggModToAcct)

	b.Run("v0.1.31_per_claim", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			em := cosmostypes.NewEventManager()
			simulateBankEvents(em, preMints, preBurns, preSends)
		}
	})

	b.Run("v0.1.33_aggregated", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			em := cosmostypes.NewEventManager()
			simulateBankEvents(em, postMints, postBurns, postSends)
		}
	})
}

// measureABCIEventOverhead converts SDK events to ABCI events (what CometBFT sees),
// then measures:
//   - Proto marshal size: bytes written to block storage and returned by block_results RPC
//   - CometBFT index key size: each attribute becomes a LevelDB entry with key format
//     "{type}.{attr_key}/{attr_value}/{height}/{event_idx}" (~100-200 bytes per attribute)
//   - Total attribute count: drives LevelDB write amplification
func measureABCIEventOverhead(em *cosmostypes.EventManager) (protoBytes int, indexKeyBytes int, totalAttrs int) {
	abciEvents := em.ABCIEvents()

	for i, event := range abciEvents {
		// Proto marshal size: marshal each event individually.
		bz, err := gogoproto.Marshal(&event)
		if err == nil {
			protoBytes += len(bz)
		}

		// CometBFT BlockIndexer creates one LevelDB entry per attribute.
		// Key format: "{event_type}.{attr_key}/{attr_value}/{height}/{event_index}"
		// Value: empty (index-only).
		for _, attr := range event.Attributes {
			totalAttrs++
			// Estimate key size: type + "." + key + "/" + value + "/" + height(~7 digits) + "/" + index
			indexKeyBytes += len(event.Type) + 1 + len(attr.Key) + 1 + len(attr.Value) + 1 + 7 + 1 + len(fmt.Sprintf("%d", i))
		}
	}

	return protoBytes, indexKeyBytes, totalAttrs
}

// TestSettlementDownstreamOverhead measures the three downstream memory consumers
// that caused the 11.4 GB leak on mainnet:
//
//  1. Proto marshal size → block storage + block_results RPC wire size
//  2. CometBFT index key size → LevelDB batch buffer (the 2.6 GB from pprof)
//  3. Total attributes → drives the 5.8M index entries seen at block 651093
//
// Run with: go test -run TestSettlementDownstreamOverhead -v -count=1 -tags test ./x/tokenomics/keeper/
func TestSettlementDownstreamOverhead(t *testing.T) {
	for _, numClaims := range []int{2500, 5000, 10000} {
		t.Run(fmt.Sprintf("claims_%d", numClaims), func(t *testing.T) {
			results := generateRealisticResults(numClaims)

			// Count pre-aggregation operations.
			var preMints, preBurns, preSends int
			for _, r := range results {
				preMints += len(r.GetMints())
				preBurns += len(r.GetBurns())
				preSends += len(r.GetModToModTransfers()) + len(r.GetModToAcctTransfers())
			}

			// Aggregate.
			aggMints, err := aggregateMints(results)
			require.NoError(t, err)
			aggBurns, err := aggregateBurns(results)
			require.NoError(t, err)
			aggModToMod, err := aggregateModToModTransfers(results)
			require.NoError(t, err)
			aggModToAcct, err := aggregateModToAcctTransfers(results)
			require.NoError(t, err)

			postMints := len(aggMints)
			postBurns := len(aggBurns)
			postSends := len(aggModToMod) + len(aggModToAcct)

			// === Build v0.1.31 EventManager ===
			emOld := cosmostypes.NewEventManager()
			simulateBankEvents(emOld, preMints, preBurns, preSends)
			for range numClaims {
				emOld.EmitEvent(cosmostypes.NewEvent(
					"pocket.tokenomics.EventClaimSettled",
					cosmostypes.NewAttribute("supplier_operator_address", "pokt1supplier0000"),
					cosmostypes.NewAttribute("num_relays", "100"),
					cosmostypes.NewAttribute("num_claimed_compute_units", "1000"),
					cosmostypes.NewAttribute("claimed_upokt", "5000upokt"),
					cosmostypes.NewAttribute("service_id", "svc01"),
					cosmostypes.NewAttribute("session_end_block_height", "1000"),
				))
			}

			// === Build v0.1.33 EventManager ===
			emNew := cosmostypes.NewEventManager()
			simulateBankEvents(emNew, postMints, postBurns, postSends)
			for range numClaims {
				emNew.EmitEvent(cosmostypes.NewEvent(
					"pocket.tokenomics.EventClaimSettled",
					cosmostypes.NewAttribute("supplier_operator_address", "pokt1supplier0000"),
					cosmostypes.NewAttribute("num_relays", "100"),
					cosmostypes.NewAttribute("num_claimed_compute_units", "1000"),
					cosmostypes.NewAttribute("claimed_upokt", "5000upokt"),
					cosmostypes.NewAttribute("service_id", "svc01"),
					cosmostypes.NewAttribute("session_end_block_height", "1000"),
				))
			}
			for range postMints + postBurns + postSends {
				emNew.EmitEvent(cosmostypes.NewEvent(
					"pocket.tokenomics.EventSettlementBatch",
					cosmostypes.NewAttribute("op_type", "mint"),
					cosmostypes.NewAttribute("total_amount", "1000000upokt"),
					cosmostypes.NewAttribute("num_claims", fmt.Sprintf("%d", numClaims)),
				))
			}

			// === Measure downstream overhead ===
			oldProtoBytes, oldIndexKeyBytes, oldTotalAttrs := measureABCIEventOverhead(emOld)
			newProtoBytes, newIndexKeyBytes, newTotalAttrs := measureABCIEventOverhead(emNew)

			protoReduction := float64(oldProtoBytes-newProtoBytes) / float64(oldProtoBytes) * 100
			indexReduction := float64(oldIndexKeyBytes-newIndexKeyBytes) / float64(oldIndexKeyBytes) * 100
			attrReduction := float64(oldTotalAttrs-newTotalAttrs) / float64(oldTotalAttrs) * 100

			t.Logf("\n╔══════════════════════════════════════════════════════════════════╗")
			t.Logf("║  Downstream Overhead Simulation: %d claims                      ║", numClaims)
			t.Logf("╠══════════════════════════════════════════════════════════════════╣")
			t.Logf("║                                                                  ║")
			t.Logf("║  1. PROTO MARSHAL SIZE (block storage + block_results RPC)        ║")
			t.Logf("║     v0.1.31: %8.1f MB", float64(oldProtoBytes)/(1024*1024))
			t.Logf("║     v0.1.33: %8.1f MB", float64(newProtoBytes)/(1024*1024))
			t.Logf("║     Reduction: %.1f%%", protoReduction)
			t.Logf("║                                                                  ║")
			t.Logf("║  2. COMETBFT INDEX KEY SIZE (LevelDB batch buffer)               ║")
			t.Logf("║     v0.1.31: %8.1f MB", float64(oldIndexKeyBytes)/(1024*1024))
			t.Logf("║     v0.1.33: %8.1f MB", float64(newIndexKeyBytes)/(1024*1024))
			t.Logf("║     Reduction: %.1f%%", indexReduction)
			t.Logf("║                                                                  ║")
			t.Logf("║  3. TOTAL ATTRIBUTES (LevelDB entries = index writes)            ║")
			t.Logf("║     v0.1.31: %8d attrs", oldTotalAttrs)
			t.Logf("║     v0.1.33: %8d attrs", newTotalAttrs)
			t.Logf("║     Reduction: %.1f%%", attrReduction)
			t.Logf("║                                                                  ║")
			t.Logf("║  MAINNET EXTRAPOLATION (2,551 claims, block 651093)              ║")
			scale := float64(2551) / float64(numClaims)
			oldMainnetProtoMB := float64(oldProtoBytes) * scale / (1024 * 1024)
			newMainnetProtoMB := float64(newProtoBytes) * scale / (1024 * 1024)
			oldMainnetIndexMB := float64(oldIndexKeyBytes) * scale / (1024 * 1024)
			newMainnetIndexMB := float64(newIndexKeyBytes) * scale / (1024 * 1024)
			oldMainnetAttrs := float64(oldTotalAttrs) * scale
			newMainnetAttrs := float64(newTotalAttrs) * scale
			t.Logf("║     Proto: %.1f MB → %.1f MB (saved %.1f MB)", oldMainnetProtoMB, newMainnetProtoMB, oldMainnetProtoMB-newMainnetProtoMB)
			t.Logf("║     Index: %.1f MB → %.1f MB (saved %.1f MB)", oldMainnetIndexMB, newMainnetIndexMB, oldMainnetIndexMB-newMainnetIndexMB)
			t.Logf("║     Attrs: %.0f → %.0f (saved %.0f entries)", oldMainnetAttrs, newMainnetAttrs, oldMainnetAttrs-newMainnetAttrs)
			t.Logf("╚══════════════════════════════════════════════════════════════════╝")

			require.Greater(t, protoReduction, 50.0, "proto bytes should reduce significantly")
			require.Greater(t, indexReduction, 50.0, "index key bytes should reduce significantly")
		})
	}
}

// BenchmarkProtoMarshal_PreVsPost benchmarks the cost of proto-marshaling all
// ABCI events — the operation that causes the 5.8 GB allocation in block_results.
func BenchmarkProtoMarshal_PreVsPost(b *testing.B) {
	results := generateRealisticResults(2500)

	var preMints, preBurns, preSends int
	for _, r := range results {
		preMints += len(r.GetMints())
		preBurns += len(r.GetBurns())
		preSends += len(r.GetModToModTransfers()) + len(r.GetModToAcctTransfers())
	}

	aggMints, _ := aggregateMints(results)
	aggBurns, _ := aggregateBurns(results)
	aggModToMod, _ := aggregateModToModTransfers(results)
	aggModToAcct, _ := aggregateModToAcctTransfers(results)

	// Build event sets once.
	emOld := cosmostypes.NewEventManager()
	simulateBankEvents(emOld, preMints, preBurns, preSends)
	oldEvents := emOld.ABCIEvents()

	emNew := cosmostypes.NewEventManager()
	simulateBankEvents(emNew, len(aggMints), len(aggBurns), len(aggModToMod)+len(aggModToAcct))
	newEvents := emNew.ABCIEvents()

	b.Run("v0.1.31_marshal_all_events", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			for i := range oldEvents {
				gogoproto.Marshal(&oldEvents[i])
			}
		}
	})

	b.Run("v0.1.33_marshal_all_events", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			for i := range newEvents {
				gogoproto.Marshal(&newEvents[i])
			}
		}
	})
}
