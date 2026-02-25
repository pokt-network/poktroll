package keeper

import (
	"testing"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	tlm "github.com/pokt-network/poktroll/x/tokenomics/token_logic_module"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

const testDenom = "upokt"

// newTestClaim creates a minimal claim for testing aggregation.
func newTestClaim() prooftypes.Claim {
	return prooftypes.Claim{}
}

// newTestResult creates a ClaimSettlementResult with the given operations.
func newTestResult(
	mints []tokenomicstypes.MintBurnOp,
	burns []tokenomicstypes.MintBurnOp,
	modToMod []tokenomicstypes.ModToModTransfer,
	modToAcct []tokenomicstypes.ModToAcctTransfer,
) *tokenomicstypes.ClaimSettlementResult {
	return &tokenomicstypes.ClaimSettlementResult{
		Claim:              newTestClaim(),
		Mints:              mints,
		Burns:              burns,
		ModToModTransfers:  modToMod,
		ModToAcctTransfers: modToAcct,
	}
}

func coin(amount int64) cosmostypes.Coin {
	return cosmostypes.NewCoin(testDenom, math.NewInt(amount))
}

func TestAggregateMints_BasicAggregation(t *testing.T) {
	results := tlm.ClaimSettlementResults{
		newTestResult(
			[]tokenomicstypes.MintBurnOp{
				{DestinationModule: "supplier", OpReason: tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SUPPLIER_STAKE_MINT, Coin: coin(100)},
				{DestinationModule: "tokenomics", OpReason: tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_TOKENOMICS_CLAIM_DISTRIBUTION_MINT, Coin: coin(50)},
			},
			nil, nil, nil,
		),
		newTestResult(
			[]tokenomicstypes.MintBurnOp{
				{DestinationModule: "supplier", OpReason: tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SUPPLIER_STAKE_MINT, Coin: coin(200)},
				{DestinationModule: "tokenomics", OpReason: tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_TOKENOMICS_CLAIM_DISTRIBUTION_MINT, Coin: coin(75)},
			},
			nil, nil, nil,
		),
	}

	agg, err := aggregateMints(results)
	require.NoError(t, err)
	require.Len(t, agg, 2)

	// Find entries by module name (order is deterministic by sorted key).
	var supplierMint, tokenomicsMint aggregatedMintBurnOp
	for _, m := range agg {
		switch m.DestinationModule {
		case "supplier":
			supplierMint = m
		case "tokenomics":
			tokenomicsMint = m
		}
	}

	require.Equal(t, coin(300), supplierMint.Coin)
	require.Equal(t, uint32(2), supplierMint.NumClaims)

	require.Equal(t, coin(125), tokenomicsMint.Coin)
	require.Equal(t, uint32(2), tokenomicsMint.NumClaims)
}

func TestAggregateMints_OpReasonSeparation(t *testing.T) {
	// Same module, different OpReasons — should NOT merge.
	results := tlm.ClaimSettlementResults{
		newTestResult(
			[]tokenomicstypes.MintBurnOp{
				{DestinationModule: "supplier", OpReason: tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SUPPLIER_STAKE_MINT, Coin: coin(100)},
				{DestinationModule: "supplier", OpReason: tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_INFLATION, Coin: coin(50)},
			},
			nil, nil, nil,
		),
	}

	agg, err := aggregateMints(results)
	require.NoError(t, err)
	require.Len(t, agg, 2, "different OpReasons for same module should produce separate entries")
}

func TestAggregateBurns_BasicAggregation(t *testing.T) {
	results := tlm.ClaimSettlementResults{
		newTestResult(nil,
			[]tokenomicstypes.MintBurnOp{
				{DestinationModule: "application", OpReason: tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_APPLICATION_STAKE_BURN, Coin: coin(100)},
			},
			nil, nil,
		),
		newTestResult(nil,
			[]tokenomicstypes.MintBurnOp{
				{DestinationModule: "application", OpReason: tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_APPLICATION_STAKE_BURN, Coin: coin(200)},
			},
			nil, nil,
		),
	}

	agg, err := aggregateBurns(results)
	require.NoError(t, err)
	require.Len(t, agg, 1)
	require.Equal(t, coin(300), agg[0].Coin)
	require.Equal(t, uint32(2), agg[0].NumClaims)
}

func TestAggregateModToModTransfers_BasicAggregation(t *testing.T) {
	results := tlm.ClaimSettlementResults{
		newTestResult(nil, nil,
			[]tokenomicstypes.ModToModTransfer{
				{SenderModule: "tokenomics", RecipientModule: "supplier", OpReason: tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_SUPPLIER_SHAREHOLDER_REWARD_MODULE_TRANSFER, Coin: coin(100)},
			},
			nil,
		),
		newTestResult(nil, nil,
			[]tokenomicstypes.ModToModTransfer{
				{SenderModule: "tokenomics", RecipientModule: "supplier", OpReason: tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_SUPPLIER_SHAREHOLDER_REWARD_MODULE_TRANSFER, Coin: coin(150)},
			},
			nil,
		),
	}

	agg, err := aggregateModToModTransfers(results)
	require.NoError(t, err)
	require.Len(t, agg, 1)
	require.Equal(t, coin(250), agg[0].Coin)
	require.Equal(t, uint32(2), agg[0].NumClaims)
	require.Equal(t, "tokenomics", agg[0].SenderModule)
	require.Equal(t, "supplier", agg[0].RecipientModule)
}

func TestAggregateModToAcctTransfers_BasicAggregation(t *testing.T) {
	results := tlm.ClaimSettlementResults{
		newTestResult(nil, nil, nil,
			[]tokenomicstypes.ModToAcctTransfer{
				{SenderModule: "supplier", RecipientAddress: "pokt1abc", OpReason: tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SUPPLIER_SHAREHOLDER_REWARD_DISTRIBUTION, Coin: coin(100)},
				{SenderModule: "supplier", RecipientAddress: "pokt1def", OpReason: tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_DAO_REWARD_DISTRIBUTION, Coin: coin(30)},
			},
		),
		newTestResult(nil, nil, nil,
			[]tokenomicstypes.ModToAcctTransfer{
				{SenderModule: "supplier", RecipientAddress: "pokt1abc", OpReason: tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SUPPLIER_SHAREHOLDER_REWARD_DISTRIBUTION, Coin: coin(200)},
				{SenderModule: "supplier", RecipientAddress: "pokt1def", OpReason: tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_DAO_REWARD_DISTRIBUTION, Coin: coin(70)},
			},
		),
	}

	agg, err := aggregateModToAcctTransfers(results)
	require.NoError(t, err)
	require.Len(t, agg, 2)

	var abcTransfer, defTransfer aggregatedModToAcctTransfer
	for _, a := range agg {
		switch a.RecipientAddress {
		case "pokt1abc":
			abcTransfer = a
		case "pokt1def":
			defTransfer = a
		}
	}

	require.Equal(t, coin(300), abcTransfer.Coin)
	require.Equal(t, uint32(2), abcTransfer.NumClaims)
	require.Equal(t, coin(100), defTransfer.Coin)
	require.Equal(t, uint32(2), defTransfer.NumClaims)
}

func TestAggregateModToAcctTransfers_SameRecipientDifferentReasons(t *testing.T) {
	// Same recipient + different OpReason = separate entries.
	results := tlm.ClaimSettlementResults{
		newTestResult(nil, nil, nil,
			[]tokenomicstypes.ModToAcctTransfer{
				{SenderModule: "supplier", RecipientAddress: "pokt1abc", OpReason: tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SUPPLIER_SHAREHOLDER_REWARD_DISTRIBUTION, Coin: coin(100)},
				{SenderModule: "supplier", RecipientAddress: "pokt1abc", OpReason: tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_DAO_REWARD_DISTRIBUTION, Coin: coin(50)},
			},
		),
	}

	agg, err := aggregateModToAcctTransfers(results)
	require.NoError(t, err)
	require.Len(t, agg, 2, "same recipient with different OpReasons should produce separate entries")
}

func TestAggregate_EmptyInput(t *testing.T) {
	var emptyResults tlm.ClaimSettlementResults

	aggMints, err := aggregateMints(emptyResults)
	require.NoError(t, err)
	require.Empty(t, aggMints)

	aggBurns, err := aggregateBurns(emptyResults)
	require.NoError(t, err)
	require.Empty(t, aggBurns)

	aggModToMod, err := aggregateModToModTransfers(emptyResults)
	require.NoError(t, err)
	require.Empty(t, aggModToMod)

	aggModToAcct, err := aggregateModToAcctTransfers(emptyResults)
	require.NoError(t, err)
	require.Empty(t, aggModToAcct)
}

func TestAggregate_SingleResult(t *testing.T) {
	results := tlm.ClaimSettlementResults{
		newTestResult(
			[]tokenomicstypes.MintBurnOp{
				{DestinationModule: "supplier", OpReason: tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SUPPLIER_STAKE_MINT, Coin: coin(42)},
			},
			nil, nil, nil,
		),
	}

	agg, err := aggregateMints(results)
	require.NoError(t, err)
	require.Len(t, agg, 1)
	require.Equal(t, coin(42), agg[0].Coin)
	require.Equal(t, uint32(1), agg[0].NumClaims)
}

func TestAggregate_DeterministicOrdering(t *testing.T) {
	// Run 100 times to verify the output is deterministic.
	results := tlm.ClaimSettlementResults{
		newTestResult(nil, nil, nil,
			[]tokenomicstypes.ModToAcctTransfer{
				{SenderModule: "supplier", RecipientAddress: "pokt1zzz", OpReason: tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SUPPLIER_SHAREHOLDER_REWARD_DISTRIBUTION, Coin: coin(10)},
				{SenderModule: "supplier", RecipientAddress: "pokt1aaa", OpReason: tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_DAO_REWARD_DISTRIBUTION, Coin: coin(20)},
				{SenderModule: "tokenomics", RecipientAddress: "pokt1mmm", OpReason: tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_DAO_REWARD_DISTRIBUTION, Coin: coin(30)},
			},
		),
	}

	// Get the first run's result as the reference.
	reference, err := aggregateModToAcctTransfers(results)
	require.NoError(t, err)
	require.Len(t, reference, 3)

	for i := 0; i < 100; i++ {
		agg, err := aggregateModToAcctTransfers(results)
		require.NoError(t, err)
		require.Len(t, agg, len(reference))
		for j := range reference {
			require.Equal(t, reference[j].RecipientAddress, agg[j].RecipientAddress, "iteration %d, index %d", i, j)
			require.Equal(t, reference[j].OpReason, agg[j].OpReason, "iteration %d, index %d", i, j)
			require.Equal(t, reference[j].Coin, agg[j].Coin, "iteration %d, index %d", i, j)
		}
	}
}

func TestAggregateModToModTransfers_DifferentModulePairs(t *testing.T) {
	results := tlm.ClaimSettlementResults{
		newTestResult(nil, nil,
			[]tokenomicstypes.ModToModTransfer{
				{SenderModule: "tokenomics", RecipientModule: "supplier", OpReason: tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_SUPPLIER_SHAREHOLDER_REWARD_MODULE_TRANSFER, Coin: coin(100)},
				{SenderModule: "tokenomics", RecipientModule: "application", OpReason: tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_REIMBURSEMENT_REQUEST_ESCROW_MODULE_TRANSFER, Coin: coin(50)},
			},
			nil,
		),
	}

	agg, err := aggregateModToModTransfers(results)
	require.NoError(t, err)
	require.Len(t, agg, 2, "different module pairs should produce separate entries")
}

func TestAggregate_ManyResultsSameKey_SumsCorrectly(t *testing.T) {
	// 10 results, each with 1 mint of 100 uPOKT to the same (module, reason) key.
	// Assert: 1 aggregated entry, Coin=1000, NumClaims=10.
	results := make(tlm.ClaimSettlementResults, 10)
	for i := range results {
		results[i] = newTestResult(
			[]tokenomicstypes.MintBurnOp{
				{DestinationModule: "supplier", OpReason: tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SUPPLIER_STAKE_MINT, Coin: coin(100)},
			},
			nil, nil, nil,
		)
	}

	agg, err := aggregateMints(results)
	require.NoError(t, err)
	require.Len(t, agg, 1)
	require.Equal(t, coin(1000), agg[0].Coin)
	require.Equal(t, uint32(10), agg[0].NumClaims)
	require.Equal(t, "supplier", agg[0].DestinationModule)
	require.Equal(t, tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SUPPLIER_STAKE_MINT, agg[0].OpReason)
}

func TestAggregate_MixedKeysFromMultipleResults(t *testing.T) {
	// 3 results with overlapping and distinct keys across mints.
	// Result 1: mint(supplier, RBEM_STAKE, 100) + mint(tokenomics, RBEM_DISTRIBUTION, 50)
	// Result 2: mint(supplier, RBEM_STAKE, 200) + mint(supplier, GM_INFLATION, 30)
	// Result 3: mint(tokenomics, RBEM_DISTRIBUTION, 75)
	// Assert: 3 entries — supplier|RBEM=300 (NC=2), tokenomics|RBEM=125 (NC=2), supplier|GM=30 (NC=1)
	results := tlm.ClaimSettlementResults{
		newTestResult(
			[]tokenomicstypes.MintBurnOp{
				{DestinationModule: "supplier", OpReason: tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SUPPLIER_STAKE_MINT, Coin: coin(100)},
				{DestinationModule: "tokenomics", OpReason: tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_TOKENOMICS_CLAIM_DISTRIBUTION_MINT, Coin: coin(50)},
			},
			nil, nil, nil,
		),
		newTestResult(
			[]tokenomicstypes.MintBurnOp{
				{DestinationModule: "supplier", OpReason: tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SUPPLIER_STAKE_MINT, Coin: coin(200)},
				{DestinationModule: "supplier", OpReason: tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_INFLATION, Coin: coin(30)},
			},
			nil, nil, nil,
		),
		newTestResult(
			[]tokenomicstypes.MintBurnOp{
				{DestinationModule: "tokenomics", OpReason: tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_TOKENOMICS_CLAIM_DISTRIBUTION_MINT, Coin: coin(75)},
			},
			nil, nil, nil,
		),
	}

	agg, err := aggregateMints(results)
	require.NoError(t, err)
	require.Len(t, agg, 3)

	// Build a lookup by key for easier assertion (order is deterministic by sorted key).
	aggMap := make(map[string]aggregatedMintBurnOp)
	for _, m := range agg {
		key := m.DestinationModule + "|" + m.OpReason.String()
		aggMap[key] = m
	}

	supplierRBEM := aggMap["supplier|TLM_RELAY_BURN_EQUALS_MINT_SUPPLIER_STAKE_MINT"]
	require.Equal(t, coin(300), supplierRBEM.Coin)
	require.Equal(t, uint32(2), supplierRBEM.NumClaims)

	tokenomicsRBEM := aggMap["tokenomics|TLM_RELAY_BURN_EQUALS_MINT_TOKENOMICS_CLAIM_DISTRIBUTION_MINT"]
	require.Equal(t, coin(125), tokenomicsRBEM.Coin)
	require.Equal(t, uint32(2), tokenomicsRBEM.NumClaims)

	supplierGM := aggMap["supplier|TLM_GLOBAL_MINT_INFLATION"]
	require.Equal(t, coin(30), supplierGM.Coin)
	require.Equal(t, uint32(1), supplierGM.NumClaims)
}

func TestAggregate_ZeroCoinOps_Included(t *testing.T) {
	// 2 results: first has zero-amount mint, second has 100 uPOKT mint, same key.
	// Assert: 1 entry, Coin=100, NumClaims=2 (zero ops still counted).
	results := tlm.ClaimSettlementResults{
		newTestResult(
			[]tokenomicstypes.MintBurnOp{
				{DestinationModule: "supplier", OpReason: tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SUPPLIER_STAKE_MINT, Coin: coin(0)},
			},
			nil, nil, nil,
		),
		newTestResult(
			[]tokenomicstypes.MintBurnOp{
				{DestinationModule: "supplier", OpReason: tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SUPPLIER_STAKE_MINT, Coin: coin(100)},
			},
			nil, nil, nil,
		),
	}

	agg, err := aggregateMints(results)
	require.NoError(t, err)
	require.Len(t, agg, 1)
	require.Equal(t, coin(100), agg[0].Coin)
	require.Equal(t, uint32(2), agg[0].NumClaims)
}

func TestAggregate_UnspecifiedOpReasonReturnsError(t *testing.T) {
	// Verify that ops with UNSPECIFIED reason are rejected during aggregation.
	t.Run("mints", func(t *testing.T) {
		results := tlm.ClaimSettlementResults{
			newTestResult(
				[]tokenomicstypes.MintBurnOp{
					{DestinationModule: "supplier", OpReason: tokenomicstypes.SettlementOpReason_UNSPECIFIED, Coin: coin(100)},
				},
				nil, nil, nil,
			),
		}
		_, err := aggregateMints(results)
		require.Error(t, err)
		require.ErrorIs(t, err, errUnspecifiedOpReason)
	})

	t.Run("burns", func(t *testing.T) {
		results := tlm.ClaimSettlementResults{
			newTestResult(nil,
				[]tokenomicstypes.MintBurnOp{
					{DestinationModule: "application", OpReason: tokenomicstypes.SettlementOpReason_UNSPECIFIED, Coin: coin(100)},
				},
				nil, nil,
			),
		}
		_, err := aggregateBurns(results)
		require.Error(t, err)
		require.ErrorIs(t, err, errUnspecifiedOpReason)
	})

	t.Run("mod_to_mod", func(t *testing.T) {
		results := tlm.ClaimSettlementResults{
			newTestResult(nil, nil,
				[]tokenomicstypes.ModToModTransfer{
					{SenderModule: "tokenomics", RecipientModule: "supplier", OpReason: tokenomicstypes.SettlementOpReason_UNSPECIFIED, Coin: coin(100)},
				},
				nil,
			),
		}
		_, err := aggregateModToModTransfers(results)
		require.Error(t, err)
		require.ErrorIs(t, err, errUnspecifiedOpReason)
	})

	t.Run("mod_to_acct", func(t *testing.T) {
		results := tlm.ClaimSettlementResults{
			newTestResult(nil, nil, nil,
				[]tokenomicstypes.ModToAcctTransfer{
					{SenderModule: "supplier", RecipientAddress: "pokt1abc", OpReason: tokenomicstypes.SettlementOpReason_UNSPECIFIED, Coin: coin(100)},
				},
			),
		}
		_, err := aggregateModToAcctTransfers(results)
		require.Error(t, err)
		require.ErrorIs(t, err, errUnspecifiedOpReason)
	})
}
