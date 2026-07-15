package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// TestParams_ValidateOverservicingBonusMultiplier verifies the new multiplier param accepts
// any uint64 (0 = unlimited, 1 = legacy no-op, n = n*floor cap) and rejects non-uint64 types.
func TestParams_ValidateOverservicingBonusMultiplier(t *testing.T) {
	for _, m := range []uint64{0, 1, 2, 20, 1_000_000} {
		require.NoError(t, tokenomicstypes.ValidateOverservicingBonusMultiplier(m),
			"m=%d should be valid", m)
	}

	require.Error(t, tokenomicstypes.ValidateOverservicingBonusMultiplier(int64(1)),
		"non-uint64 type must be rejected")
	require.Error(t, tokenomicstypes.ValidateOverservicingBonusMultiplier("1"),
		"non-uint64 type must be rejected")
}

// TestParams_AntiCollusionInvariant verifies ValidateBasic enforces
// mint_ratio * mint_equals_burn_claim_distribution.supplier < 1, the invariant that keeps
// app+supplier self-dealing a losing trade once the head-split cap is demoted to a floor.
func TestParams_AntiCollusionInvariant(t *testing.T) {
	tests := []struct {
		name        string
		mintRatio   float64
		supplier    float64
		expectValid bool
	}{
		{
			name:        "default params satisfy the invariant",
			mintRatio:   tokenomicstypes.DefaultMintRatio, // 1.0
			supplier:    0.7,                              // DefaultMintEqualsBurnClaimDistribution.Supplier
			expectValid: true,
		},
		{
			name:        "mainnet-like: 0.975 x 0.79 = 0.770 is safe",
			mintRatio:   0.975,
			supplier:    0.79,
			expectValid: true,
		},
		{
			// The only input combination that violates the invariant while still passing the
			// per-field validations (mint_ratio in (0,1], distribution summing to 1 with
			// non-negative shares): a 100% mint paired with a 100%-to-supplier distribution,
			// i.e. self-dealing with zero loss.
			name:        "product exactly 1 is rejected (break-even self-dealing)",
			mintRatio:   1.0,
			supplier:    1.0,
			expectValid: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			params := tokenomicstypes.DefaultParams()
			params.MintRatio = tc.mintRatio
			// Keep the distribution summing to 1 so only the invariant (not the sum check) is exercised.
			params.MintEqualsBurnClaimDistribution = tokenomicstypes.MintEqualsBurnClaimDistribution{
				Supplier: tc.supplier,
				Dao:      1 - tc.supplier,
			}

			err := params.ValidateBasic()
			if tc.expectValid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), "anti-collusion invariant")
			}
		})
	}
}

// TestParams_DefaultOverservicingBonusMultiplier documents that the shipped default is the
// no-op multiplier (1), so the settlement-budget-redistribution consensus change is inert
// until governance opts in.
func TestParams_DefaultOverservicingBonusMultiplier(t *testing.T) {
	require.Equal(t, uint64(1), tokenomicstypes.DefaultOverservicingBonusMultiplier)
	require.Equal(t, uint64(1), tokenomicstypes.DefaultParams().OverservicingBonusMultiplier)
}
