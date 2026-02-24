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

	agg := aggregateMints(results)
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

	agg := aggregateMints(results)
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

	agg := aggregateBurns(results)
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

	agg := aggregateModToModTransfers(results)
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

	agg := aggregateModToAcctTransfers(results)
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

	agg := aggregateModToAcctTransfers(results)
	require.Len(t, agg, 2, "same recipient with different OpReasons should produce separate entries")
}

func TestAggregate_EmptyInput(t *testing.T) {
	var emptyResults tlm.ClaimSettlementResults

	require.Empty(t, aggregateMints(emptyResults))
	require.Empty(t, aggregateBurns(emptyResults))
	require.Empty(t, aggregateModToModTransfers(emptyResults))
	require.Empty(t, aggregateModToAcctTransfers(emptyResults))
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

	agg := aggregateMints(results)
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
	reference := aggregateModToAcctTransfers(results)
	require.Len(t, reference, 3)

	for i := 0; i < 100; i++ {
		agg := aggregateModToAcctTransfers(results)
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

	agg := aggregateModToModTransfers(results)
	require.Len(t, agg, 2, "different module pairs should produce separate entries")
}
