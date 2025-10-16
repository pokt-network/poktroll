package types

import (
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/pocket"
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
