package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// TestParams_ValidateOverservicingBonusMultiplier verifies the multiplier param accepts
// any uint64 up to MaxOverservicingBonusMultiplier (0 => coerced to 1 at settlement,
// 1 = legacy no-op, n = n*floor cap), and rejects both non-uint64 types and values above
// the bound (economically identical to the bound, so almost certainly a typo).
func TestParams_ValidateOverservicingBonusMultiplier(t *testing.T) {
	for _, m := range []uint64{0, 1, 2, 20, tokenomicstypes.MaxOverservicingBonusMultiplier} {
		require.NoError(t, tokenomicstypes.ValidateOverservicingBonusMultiplier(m),
			"m=%d should be valid", m)
	}

	for _, m := range []uint64{
		tokenomicstypes.MaxOverservicingBonusMultiplier + 1,
		1_000_000,
		^uint64(0), // max uint64: the fat-finger / clobbered-write case
	} {
		require.Error(t, tokenomicstypes.ValidateOverservicingBonusMultiplier(m),
			"m=%d exceeds MaxOverservicingBonusMultiplier and must be rejected", m)
	}

	require.Error(t, tokenomicstypes.ValidateOverservicingBonusMultiplier(int64(1)),
		"non-uint64 type must be rejected")
	require.Error(t, tokenomicstypes.ValidateOverservicingBonusMultiplier("1"),
		"non-uint64 type must be rejected")
}

// TestParams_AntiCollusionInvariant verifies CheckAntiCollusionInvariant reports on
// mint_ratio * mint_equals_burn_claim_distribution.supplier < 1, the round-trip factor
// that keeps app+supplier self-dealing a losing trade once the head-split cap is demoted
// to a floor.
//
// The check is REPORTING-ONLY: ValidateBasic MUST accept a violating param set so that a
// DAO-governed policy value can never halt a chain from an upgrade handler or block a
// genesis. See Params.CheckAntiCollusionInvariant.
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
			// i.e. self-dealing with zero loss. Since mint_ratio <= 1 and the shares sum to
			// 1, the product can never EXCEED 1 — self-dealing is never profitable, only
			// break-even, which is why this is reported rather than rejected.
			name:        "product exactly 1 is reported (break-even self-dealing)",
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

			// The invariant is reporting-only: a violating param set MUST still validate,
			// otherwise it could halt a chain from an upgrade handler or block a genesis.
			require.NoError(t, params.ValidateBasic(),
				"ValidateBasic must never reject on the anti-collusion invariant")

			err := params.CheckAntiCollusionInvariant()
			if tc.expectValid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), "anti-collusion invariant")
			}
		})
	}
}

// TestParams_MintRatioUpperBoundGuardsAntiCollusion is a TRIPWIRE.
//
// The anti-collusion invariant ships as a warning rather than a rejection only because
// mint_ratio is capped at 1: combined with the distribution shares summing to 1, that makes
// mint_ratio * supplier <= 1 for every legal param set, so app+supplier self-dealing is at
// worst break-even.
//
// The network has already run net-inflationary, neutral (mint == burn) & deflationary
// (PIP-41, mint_ratio 0.975) emission regimes, so a swing back is precedent. Note WHICH
// lever is safe: global_inflation_per_claim cannot break the bound because the application
// is charged settlement * (1 + I) and therefore funds its own inflation. Raising mint_ratio
// above 1 DOES break it — the extra mint is not charged to anyone — and makes collusion
// profitable (e.g. 1.3 * 0.79 = 1.027).
//
// So if this test ever has to change, CheckAntiCollusionInvariant MUST be re-escalated to a
// hard rejection on the MsgUpdateParam(s) path (and ONLY there — never in ValidateBasic,
// which genesis validation & upgrade handlers run inside consensus).
func TestParams_MintRatioUpperBoundGuardsAntiCollusion(t *testing.T) {
	require.NoError(t, tokenomicstypes.ValidateMintRatio(1.0),
		"mint_ratio == 1 (the neutral mint == burn regime) must remain valid")

	for _, mintRatio := range []float64{1.0000001, 1.3, 2.0} {
		require.Errorf(t, tokenomicstypes.ValidateMintRatio(mintRatio),
			"mint_ratio %f > 1 must be rejected; the anti-collusion invariant is only a warning because this bound holds", mintRatio)
	}

	// Demonstrate the consequence the bound is protecting against: a mainnet-shaped
	// distribution paired with a net-inflationary mint_ratio is a PROFITABLE round trip,
	// not merely break-even.
	params := tokenomicstypes.DefaultParams()
	params.MintRatio = 1.3
	params.MintEqualsBurnClaimDistribution = tokenomicstypes.MintEqualsBurnClaimDistribution{
		Supplier: 0.79,
		Dao:      0.21,
	}
	require.ErrorContains(t, params.CheckAntiCollusionInvariant(), "anti-collusion invariant violated",
		"an above-1 mint_ratio with an ordinary supplier share makes self-dealing profitable")
}

// TestParams_DefaultOverservicingBonusMultiplier documents that the shipped default is the
// no-op multiplier (1), so the settlement-budget-redistribution consensus change is inert
// until governance opts in.
func TestParams_DefaultOverservicingBonusMultiplier(t *testing.T) {
	require.Equal(t, uint64(1), tokenomicstypes.DefaultOverservicingBonusMultiplier)
	require.Equal(t, uint64(1), tokenomicstypes.DefaultParams().OverservicingBonusMultiplier)
}
