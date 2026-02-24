package types

import (
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/pocket"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

// TestClaimSettlementResult_GetRewardDistribution tests the GetRewardDistribution method
func TestClaimSettlementResult_GetRewardDistribution(t *testing.T) {
	tests := []struct {
		name               string
		modToAcctTransfers []ModToAcctTransfer
		expectedResult     map[string]string
		desc               string
	}{
		{
			name:               "empty transfers",
			modToAcctTransfers: []ModToAcctTransfer{},
			expectedResult:     map[string]string{},
			desc:               "should return empty map when no transfers",
		},
		{
			name: "single transfer single recipient",
			modToAcctTransfers: []ModToAcctTransfer{
				{
					RecipientAddress: "pokt1abc123",
					Coin:             cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 1000),
				},
			},
			expectedResult: map[string]string{
				"pokt1abc123": "1000upokt",
			},
			desc: "should return single recipient with correct amount",
		},
		{
			name: "multiple transfers single recipient",
			modToAcctTransfers: []ModToAcctTransfer{
				{
					RecipientAddress: "pokt1abc123",
					Coin:             cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 1000),
				},
				{
					RecipientAddress: "pokt1abc123",
					Coin:             cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 500),
				},
			},
			expectedResult: map[string]string{
				"pokt1abc123": "1500upokt",
			},
			desc: "should aggregate multiple transfers to same recipient",
		},
		{
			name: "multiple transfers multiple recipients",
			modToAcctTransfers: []ModToAcctTransfer{
				{
					RecipientAddress: "pokt1alice",
					Coin:             cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 1000),
				},
				{
					RecipientAddress: "pokt1bob",
					Coin:             cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 2000),
				},
				{
					RecipientAddress: "pokt1alice",
					Coin:             cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 500),
				},
			},
			expectedResult: map[string]string{
				"pokt1alice": "1500upokt",
				"pokt1bob":   "2000upokt",
			},
			desc: "should correctly aggregate transfers for multiple recipients",
		},
		{
			name: "large amounts",
			modToAcctTransfers: []ModToAcctTransfer{
				{
					RecipientAddress: "pokt1alice",
					Coin:             cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 1000000000000),
				},
				{
					RecipientAddress: "pokt1alice",
					Coin:             cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 999999999999),
				},
			},
			expectedResult: map[string]string{
				"pokt1alice": "1999999999999upokt",
			},
			desc: "should handle large amounts correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create settlement result with transfers
			result := &ClaimSettlementResult{
				ModToAcctTransfers: tt.modToAcctTransfers,
			}

			// Get reward distribution
			distribution := result.GetRewardDistribution()

			// Verify the result
			require.Equal(t, len(tt.expectedResult), len(distribution),
				"expected %d recipients, got %d", len(tt.expectedResult), len(distribution))

			for addr, expectedAmount := range tt.expectedResult {
				actualAmount, ok := distribution[addr]
				require.True(t, ok, "expected recipient %s not found in distribution", addr)
				require.Equal(t, expectedAmount, actualAmount,
					"incorrect amount for recipient %s: expected %s, got %s",
					addr, expectedAmount, actualAmount)
			}
		})
	}
}

// TestClaimSettlementResult_GetRewardDistribution_EmptyCase tests edge cases
func TestClaimSettlementResult_GetRewardDistribution_EmptyCase(t *testing.T) {
	result := &ClaimSettlementResult{
		ModToAcctTransfers: nil,
	}

	distribution := result.GetRewardDistribution()
	require.Empty(t, distribution, "should return empty map for nil transfers")
}

// TestClaimSettlementResult_GetRewardDistribution_ZeroAmounts tests zero amounts
func TestClaimSettlementResult_GetRewardDistribution_ZeroAmounts(t *testing.T) {
	result := &ClaimSettlementResult{
		ModToAcctTransfers: []ModToAcctTransfer{
			{
				RecipientAddress: "pokt1zero",
				Coin:             cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 0),
			},
		},
	}

	distribution := result.GetRewardDistribution()
	require.Equal(t, map[string]string{
		"pokt1zero": "0upokt",
	}, distribution, "should handle zero amounts correctly")
}

// TestClaimSettlementResult_GetRewardDistributionDetailed tests the detailed reward distribution.
func TestClaimSettlementResult_GetRewardDistributionDetailed(t *testing.T) {
	tests := []struct {
		name               string
		modToAcctTransfers []ModToAcctTransfer
		expectedDetails    []RewardDistributionDetail
	}{
		{
			name:               "empty transfers",
			modToAcctTransfers: nil,
			expectedDetails:    []RewardDistributionDetail{},
		},
		{
			name: "single transfer preserves OpReason",
			modToAcctTransfers: []ModToAcctTransfer{
				{
					RecipientAddress: "pokt1abc",
					OpReason:         SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SUPPLIER_SHAREHOLDER_REWARD_DISTRIBUTION,
					Coin:             cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 1000),
				},
			},
			expectedDetails: []RewardDistributionDetail{
				{
					RecipientAddress: "pokt1abc",
					OpReason:         SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SUPPLIER_SHAREHOLDER_REWARD_DISTRIBUTION,
					Amount:           "1000upokt",
				},
			},
		},
		{
			name: "same recipient different reasons stay separate",
			modToAcctTransfers: []ModToAcctTransfer{
				{
					RecipientAddress: "pokt1abc",
					OpReason:         SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SUPPLIER_SHAREHOLDER_REWARD_DISTRIBUTION,
					Coin:             cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 1000),
				},
				{
					RecipientAddress: "pokt1abc",
					OpReason:         SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_DAO_REWARD_DISTRIBUTION,
					Coin:             cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 500),
				},
			},
			expectedDetails: []RewardDistributionDetail{
				{
					RecipientAddress: "pokt1abc",
					OpReason:         SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SUPPLIER_SHAREHOLDER_REWARD_DISTRIBUTION,
					Amount:           "1000upokt",
				},
				{
					RecipientAddress: "pokt1abc",
					OpReason:         SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_DAO_REWARD_DISTRIBUTION,
					Amount:           "500upokt",
				},
			},
		},
		{
			name: "multiple recipients multiple reasons",
			modToAcctTransfers: []ModToAcctTransfer{
				{
					RecipientAddress: "pokt1alice",
					OpReason:         SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SUPPLIER_SHAREHOLDER_REWARD_DISTRIBUTION,
					Coin:             cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 800),
				},
				{
					RecipientAddress: "pokt1bob",
					OpReason:         SettlementOpReason_TLM_GLOBAL_MINT_DAO_REWARD_DISTRIBUTION,
					Coin:             cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 200),
				},
				{
					RecipientAddress: "pokt1alice",
					OpReason:         SettlementOpReason_TLM_GLOBAL_MINT_SUPPLIER_SHAREHOLDER_REWARD_DISTRIBUTION,
					Coin:             cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 100),
				},
			},
			expectedDetails: []RewardDistributionDetail{
				{
					RecipientAddress: "pokt1alice",
					OpReason:         SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SUPPLIER_SHAREHOLDER_REWARD_DISTRIBUTION,
					Amount:           "800upokt",
				},
				{
					RecipientAddress: "pokt1bob",
					OpReason:         SettlementOpReason_TLM_GLOBAL_MINT_DAO_REWARD_DISTRIBUTION,
					Amount:           "200upokt",
				},
				{
					RecipientAddress: "pokt1alice",
					OpReason:         SettlementOpReason_TLM_GLOBAL_MINT_SUPPLIER_SHAREHOLDER_REWARD_DISTRIBUTION,
					Amount:           "100upokt",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ClaimSettlementResult{
				ModToAcctTransfers: tt.modToAcctTransfers,
			}

			details := result.GetRewardDistributionDetailed()
			require.Equal(t, len(tt.expectedDetails), len(details))
			for i, expected := range tt.expectedDetails {
				require.Equal(t, expected.RecipientAddress, details[i].RecipientAddress, "index %d", i)
				require.Equal(t, expected.OpReason, details[i].OpReason, "index %d", i)
				require.Equal(t, expected.Amount, details[i].Amount, "index %d", i)
			}
		})
	}
}

// TestNewEventClaimSettled_RewardDistributionDetailed verifies that NewEventClaimSettled
// populates the RewardDistributionDetailed field from the settlement result's transfers.
func TestNewEventClaimSettled_RewardDistributionDetailed(t *testing.T) {
	result := &ClaimSettlementResult{
		Claim: prooftypes.Claim{
			SessionHeader: &sessiontypes.SessionHeader{
				SessionId:            "test-session",
				ServiceId:            "test-svc",
				ApplicationAddress:   "pokt1app",
				SessionEndBlockHeight: 100,
			},
			SupplierOperatorAddress: "pokt1supplier-op",
		},
		ModToAcctTransfers: []ModToAcctTransfer{
			{
				RecipientAddress: "pokt1supplier",
				OpReason:         SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SUPPLIER_SHAREHOLDER_REWARD_DISTRIBUTION,
				Coin:             cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 700),
			},
			{
				RecipientAddress: "pokt1dao",
				OpReason:         SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_DAO_REWARD_DISTRIBUTION,
				Coin:             cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 300),
			},
		},
	}

	claimeduPOKT := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 1000)
	event := NewEventClaimSettled(10, 5, 50, 0, &claimeduPOKT, result)

	// Verify the legacy field is still populated (merges by address).
	require.Len(t, event.RewardDistribution, 2)
	require.Equal(t, "700upokt", event.RewardDistribution["pokt1supplier"])
	require.Equal(t, "300upokt", event.RewardDistribution["pokt1dao"])

	// Verify the new detailed field preserves OpReason per entry.
	require.Len(t, event.RewardDistributionDetailed, 2)

	require.Equal(t, "pokt1supplier", event.RewardDistributionDetailed[0].RecipientAddress)
	require.Equal(t, SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SUPPLIER_SHAREHOLDER_REWARD_DISTRIBUTION, event.RewardDistributionDetailed[0].OpReason)
	require.Equal(t, "700upokt", event.RewardDistributionDetailed[0].Amount)

	require.Equal(t, "pokt1dao", event.RewardDistributionDetailed[1].RecipientAddress)
	require.Equal(t, SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_DAO_REWARD_DISTRIBUTION, event.RewardDistributionDetailed[1].OpReason)
	require.Equal(t, "300upokt", event.RewardDistributionDetailed[1].Amount)
}
