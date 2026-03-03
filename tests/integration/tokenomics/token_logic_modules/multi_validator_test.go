package token_logic_modules

import (
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/pocket"
	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	tlm "github.com/pokt-network/poktroll/x/tokenomics/token_logic_module"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

const (
	// Standard test values for validator reward distribution tests
	testClaimsCount = 1000 // Number of claims to create for testing reward distribution
	// testExpectedValidatorRewards = testClaimsCount × claimSettlementAmount × validatorAllocationPercentage
	// = 1000 × 1100 × 0.10 = 110,000 uPOKT
	testExpectedValidatorRewards = 110_000
)

// TestValidatorRewardDistribution tests validator and delegator reward distribution
// across multiple scenarios using table-driven tests to reduce code duplication.
func (s *tokenLogicModuleTestSuite) TestValidatorRewardDistribution() {
	testCases := []struct {
		name                     string
		validatorConfigs         []testkeeper.ValidatorDelegationConfig
		numClaims                int
		expectedValidatorRewards []int64
		expectedTotalRewards     int64
		validationFunc           func(t *testing.T, validatorRewards, delegatorRewards []int64, expectedTotal int64)
	}{
		{
			name: "No validator delegators: Stakes that divide cleanly",
			validatorConfigs: []testkeeper.ValidatorDelegationConfig{
				{SelfBondedStake: 500_000, ExternalDelegators: []int64{}},
				{SelfBondedStake: 400_000, ExternalDelegators: []int64{}},
				{SelfBondedStake: 200_000, ExternalDelegators: []int64{}},
			},
			numClaims:                testClaimsCount,
			expectedValidatorRewards: []int64{50_000, 40_000, 20_000}, // testExpectedValidatorRewards × stakes ratio [5:4:2]
			expectedTotalRewards:     testExpectedValidatorRewards,
			validationFunc: func(t *testing.T, validatorRewards, delegatorRewards []int64, expectedTotal int64) {
				// Validator stakes: [500k, 400k, 200k] = 1.1M total
				// With 110,000 total rewards:
				// - Val 1: 500k/1.1M = 45.45% → 50,000 uPOKT
				// - Val 2: 400k/1.1M = 36.36% → 40,000 uPOKT
				// - Val 3: 200k/1.1M = 18.18% → 20,000 uPOKT
				// Clean division with no remainder due to ratio 5:4:2

				require.Len(t, validatorRewards, 3, "Should have 3 validators")
				require.Empty(t, delegatorRewards, "Should have no delegator rewards")
				require.ElementsMatch(t, []int64{50_000, 40_000, 20_000}, validatorRewards,
					"Validator rewards should match expected proportional distribution")
			},
		},
		{
			name: "No validator delegators: Single validator gets all rewards",
			validatorConfigs: []testkeeper.ValidatorDelegationConfig{
				{SelfBondedStake: 1_000_000, ExternalDelegators: []int64{}},
			},
			numClaims:                testClaimsCount,
			expectedValidatorRewards: []int64{110_000}, // testExpectedValidatorRewards (all to single validator)
			expectedTotalRewards:     testExpectedValidatorRewards,
			validationFunc: func(t *testing.T, validatorRewards, delegatorRewards []int64, expectedTotal int64) {
				// Single validator with 1M stake gets 100% of rewards
				// Total rewards: 110,000 uPOKT (all to single validator)

				require.Len(t, validatorRewards, 1, "Should have 1 validator")
				require.Empty(t, delegatorRewards, "Should have no delegator rewards")
				require.Equal(t, []int64{110_000}, validatorRewards, "Single validator should get all rewards")
			},
		},
		{
			name: "No validator delegators: Equal stakes receive equal rewards",
			validatorConfigs: []testkeeper.ValidatorDelegationConfig{
				{SelfBondedStake: 200_000, ExternalDelegators: []int64{}},
				{SelfBondedStake: 200_000, ExternalDelegators: []int64{}},
				{SelfBondedStake: 200_000, ExternalDelegators: []int64{}},
				{SelfBondedStake: 200_000, ExternalDelegators: []int64{}},
				{SelfBondedStake: 200_000, ExternalDelegators: []int64{}},
			},
			numClaims:                testClaimsCount,
			expectedValidatorRewards: []int64{22_000, 22_000, 22_000, 22_000, 22_000}, // 110,000 ÷ 5 validators = 22,000 each
			expectedTotalRewards:     testExpectedValidatorRewards,                    // 1000 claims × 1100 × 10% = 110,000
			validationFunc: func(t *testing.T, validatorRewards, delegatorRewards []int64, expectedTotal int64) {
				// 5 validators with equal stakes (200k each) = 1M total
				// Each gets exactly 1/5 of 110,000 = 22,000 uPOKT
				// Perfect division with no remainder

				require.Len(t, validatorRewards, 5, "Should have 5 validators")
				require.Empty(t, delegatorRewards, "Should have no delegator rewards")
				for _, reward := range validatorRewards {
					require.Equal(t, int64(22_000), reward, "All validators should receive equal rewards")
				}
			},
		},
		{
			name: "With validator delegators: Mixed delegation amounts",
			validatorConfigs: []testkeeper.ValidatorDelegationConfig{
				{SelfBondedStake: 250_000, ExternalDelegators: []int64{250_000}}, // Equal self-bonded stakes with delegation
				{SelfBondedStake: 250_000, ExternalDelegators: []int64{250_000}}, // Different delegation amounts (clean divisible)
				{SelfBondedStake: 250_000, ExternalDelegators: []int64{}},
			},
			numClaims:            160,
			expectedTotalRewards: 17_600, // 160 claims × 1100 × 10% = 17,600
			validationFunc: func(t *testing.T, validatorRewards, delegatorRewards []int64, expectedTotal int64) {
				// Stake distribution (perfectly clean divisible numbers):
				// - Validator 1: 250k self + 250k delegated = 500k total (2/5 = 40% of 1.25M)
				// - Validator 2: 250k self + 250k delegated = 500k total (2/5 = 40% of 1.25M)
				// - Validator 3: 250k self + 0k delegated = 250k total (1/5 = 20% of 1.25M)
				// Total: 1,250k tokens
				//
				// Expected individual stakeholder rewards (17,600 total):
				// - 3 validator self-stakes (250k each): 17,600 × (250k/1.25M) = 3,520 each = 10,560 total
				// - 2 delegator stakes (250k each): 17,600 × (250k/1.25M) = 3,520 each = 7,040 total
				// Total rewards: 5 recipients each getting 3,520 tokens

				// Verify we have correct number of validators and delegators
				require.Len(t, validatorRewards, 3, "Should have 3 validators")
				require.Len(t, delegatorRewards, 2, "Should have 2 delegators")

				// Verify the expected reward distribution
				expectedValidatorRewards := []int64{3_520, 3_520, 3_520}
				expectedDelegatorRewards := []int64{3_520, 3_520}
				require.ElementsMatch(t, expectedValidatorRewards, validatorRewards,
					"Validator rewards should match expected distribution")
				require.ElementsMatch(t, expectedDelegatorRewards, delegatorRewards,
					"Delegator rewards should match expected distribution")

				// Verify total matches expected
				totalRewards := s.sumRewards(validatorRewards) + s.sumRewards(delegatorRewards)
				require.Equal(t, expectedTotal, totalRewards,
					"Total distributed should equal expected total")
			},
		},
		{
			name: "No validator delegators: Fractional stakes with batched distribution",
			validatorConfigs: []testkeeper.ValidatorDelegationConfig{
				{SelfBondedStake: 333_333, ExternalDelegators: []int64{}},
				{SelfBondedStake: 333_333, ExternalDelegators: []int64{}},
				{SelfBondedStake: 333_334, ExternalDelegators: []int64{}},
			},
			numClaims:            1000,
			expectedTotalRewards: 110_000, // 1000 claims × 1100 × 10% = 110,000
			validationFunc: func(t *testing.T, validatorRewards, delegatorRewards []int64, expectedTotal int64) {
				// Validator stakes: [333_333, 333_333, 333_334] = 1M total
				// These create fractional shares that can't divide evenly:
				// - Val 1: 333,333/1M = 33.3333% → ideal 36,666.63 uPOKT
				// - Val 2: 333,333/1M = 33.3333% → ideal 36,666.63 uPOKT
				// - Val 3: 333,334/1M = 33.3334% → ideal 36,666.74 uPOKT
				//
				// With batched validator reward distribution (#1758), rewards are
				// accumulated per OpReason and distributed in 2 calls (RBEM + GlobalMint),
				// each with 55,000 uPOKT. Each LRM call has ≤1 uPOKT rounding per
				// stakeholder, so max per-stakeholder error is ≤2 uPOKT.

				require.Len(t, validatorRewards, 3, "Should have 3 validators")
				require.Empty(t, delegatorRewards, "Should have no delegator rewards")

				// Total conservation must be exact.
				total := s.sumRewards(validatorRewards)
				require.Equal(t, expectedTotal, total, "total rewards should be conserved exactly")

				// Each reward within 2 uPOKT of ideal (1 per OpReason LRM call).
				idealPerValidator := float64(expectedTotal) / 3.0 // ~36,666.67
				for i, reward := range validatorRewards {
					require.InDelta(t, idealPerValidator, float64(reward), 2.0,
						"validator %d: reward %d should be within 2 uPOKT of ideal %.2f", i, reward, idealPerValidator)
				}
			},
		},
		{
			name: "With validator delegators: Fractional stakes with batched distribution",
			validatorConfigs: []testkeeper.ValidatorDelegationConfig{
				{SelfBondedStake: 333_333, ExternalDelegators: []int64{166_667}}, // Equal self-bonded stakes (fractional)
				{SelfBondedStake: 333_333, ExternalDelegators: []int64{333_333}}, // Unequal delegations creating more fractional complexity
				{SelfBondedStake: 333_334, ExternalDelegators: []int64{500_000}},
			},
			numClaims:            1000,
			expectedTotalRewards: 110_000, // 1000 claims × 1100 × 10% = 110,000
			validationFunc: func(t *testing.T, validatorRewards, delegatorRewards []int64, expectedTotal int64) {
				// This test validates that batched distribution (#1758) achieves
				// near-perfect precision in delegation scenarios.
				//
				// Total stakes:
				// - Validator 1: 333,333 self + 166,667 delegated = 500,000 total
				// - Validator 2: 333,333 self + 333,333 delegated = 666,666 total
				// - Validator 3: 333,334 self + 500,000 delegated = 833,334 total
				// Total: 2,000,000 tokens
				//
				// With 2 batched calls (RBEM + GlobalMint), each stakeholder has at
				// most ≤2 uPOKT LRM rounding error (1 per OpReason call).

				require.Len(t, validatorRewards, 3, "Should have 3 validators")
				require.Len(t, delegatorRewards, 3, "Should have 3 delegators")

				// Total conservation must be exact.
				totalActual := s.sumRewards(validatorRewards) + s.sumRewards(delegatorRewards)
				require.Equal(t, expectedTotal, totalActual,
					"Total distributed should equal expected total")

				// Ideal validator self-bonded rewards: all have ~equal self-bonded stake
				// (~333,333), so each should get approximately the same amount.
				// Ideal ≈ 110,000 × (333,333/2,000,000) ≈ 18,333.3 per validator.
				for i, reward := range validatorRewards {
					require.InDelta(t, 18_333.3, float64(reward), 2.0,
						"validator %d: self-bonded reward %d should be within 2 uPOKT of ideal", i, reward)
				}

				// Ideal delegator rewards are proportional to delegation amounts:
				// - Del 1: 110,000 × (166,667/2,000,000) ≈ 9,166.7
				// - Del 2: 110,000 × (333,333/2,000,000) ≈ 18,333.3
				// - Del 3: 110,000 × (500,000/2,000,000) = 27,500.0
				idealDelegatorRewards := []float64{9_166.7, 18_333.3, 27_500.0}
				for i, idealReward := range idealDelegatorRewards {
					// Find closest actual reward (order may differ)
					found := false
					for _, actual := range delegatorRewards {
						diff := float64(actual) - idealReward
						if diff < 0 {
							diff = -diff
						}
						if diff <= 2.0 {
							found = true
							break
						}
					}
					require.True(t, found,
						"delegator %d: no actual reward within 2 uPOKT of ideal %.1f, actuals: %v",
						i, idealReward, delegatorRewards)
				}
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			// Setup keepers with appropriate validator/delegation configuration
			s.setupValidatorTest(t, tc.validatorConfigs)

			// Create claims and settle
			s.createClaims(&s.keepers, tc.numClaims)
			settledResults, expiredResults := s.settleClaims(t)
			require.NotEmpty(t, settledResults)
			require.Empty(t, expiredResults)

			// Extract rewards
			validatorRewards, delegatorRewards := s.extractRewards(settledResults, s.hasExternalDelegators(tc.validatorConfigs))

			// Validate total rewards if expected
			if tc.expectedTotalRewards > 0 {
				totalRewards := s.sumRewards(validatorRewards) + s.sumRewards(delegatorRewards)
				require.Equal(t, tc.expectedTotalRewards, totalRewards,
					"Total rewards should equal expected")
			}

			// Run test-specific validation
			tc.validationFunc(t, validatorRewards, delegatorRewards, tc.expectedTotalRewards)

			// Ensure no pending claims remain
			s.assertNoPendingClaims(t)
		})
	}
}

// setupValidatorTest initializes keepers for validator reward testing.
// Supports validators both with and without delegators.
func (s *tokenLogicModuleTestSuite) setupValidatorTest(t *testing.T, validatorConfigs []testkeeper.ValidatorDelegationConfig) {
	t.Helper()

	// Common setup options
	setupOpts := []testkeeper.TokenomicsModuleKeepersOptFn{
		testkeeper.WithService(*s.service),
		testkeeper.WithApplication(*s.app),
		testkeeper.WithSupplier(*s.supplier),
		testkeeper.WithBlockProposer(
			cosmostypes.ConsAddress(s.proposerConsAddr),
			cosmostypes.ValAddress(s.proposerValOperatorAddr),
		),
		testkeeper.WithModuleParams(map[string]cosmostypes.Msg{
			prooftypes.ModuleName:      s.getProofParams(),
			sharedtypes.ModuleName:     s.getSharedParams(),
			tokenomicstypes.ModuleName: s.getTokenomicsParamsWithCleanValidatorMath(), // Use 10% validator allocation
		}),
		testkeeper.WithDefaultModuleBalances(),
		testkeeper.WithValidatorDelegationConfigs(validatorConfigs),
	}

	s.setupKeepers(t, setupOpts...)
}

// hasExternalDelegators checks if any validator config has external delegators
func (s *tokenLogicModuleTestSuite) hasExternalDelegators(validatorConfigs []testkeeper.ValidatorDelegationConfig) bool {
	for _, config := range validatorConfigs {
		if len(config.ExternalDelegators) > 0 {
			return true
		}
	}
	return false
}

// extractRewards extracts validator and/or delegator reward amounts from settlement results.
// Returns separate slices for validator and delegator rewards.
func (s *tokenLogicModuleTestSuite) extractRewards(settledResults tlm.ClaimSettlementResults, includeDelegators bool) (validatorRewards []int64, delegatorRewards []int64) {
	// Maps to aggregate rewards by recipient address
	validatorRewardMap := make(map[string]int64)
	delegatorRewardMap := make(map[string]int64)

	for _, result := range settledResults {
		for _, transfer := range result.ModToAcctTransfers {
			// Verify the transfer is from tokenomics module and is valid
			if transfer.SenderModule != tokenomicstypes.ModuleName ||
				transfer.Coin.Denom != pocket.DenomuPOKT ||
				!transfer.Coin.Amount.IsPositive() {
				continue
			}

			// Check transfer type and aggregate appropriately
			switch transfer.OpReason {
			case tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_VALIDATOR_REWARD_DISTRIBUTION,
				tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_VALIDATOR_REWARD_DISTRIBUTION:
				validatorRewardMap[transfer.RecipientAddress] += transfer.Coin.Amount.Int64()

			case tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_DELEGATOR_REWARD_DISTRIBUTION,
				tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_DELEGATOR_REWARD_DISTRIBUTION:
				if includeDelegators {
					delegatorRewardMap[transfer.RecipientAddress] += transfer.Coin.Amount.Int64()
				}
			}
		}
	}

	// Convert maps to slices of reward amounts
	for _, reward := range validatorRewardMap {
		validatorRewards = append(validatorRewards, reward)
	}
	for _, reward := range delegatorRewardMap {
		delegatorRewards = append(delegatorRewards, reward)
	}

	return validatorRewards, delegatorRewards
}

// sumRewards calculates the total of all rewards in a slice.
func (s *tokenLogicModuleTestSuite) sumRewards(rewards []int64) int64 {
	total := int64(0)
	for _, reward := range rewards {
		total += reward
	}
	return total
}
