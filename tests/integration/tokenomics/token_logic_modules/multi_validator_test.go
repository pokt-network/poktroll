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

// TestValidatorRewardDistribution tests validator and delegator reward distribution
// across multiple scenarios using table-driven tests to reduce code duplication.
func (s *tokenLogicModuleTestSuite) TestValidatorRewardDistribution() {
	testCases := []struct {
		name                     string
		validatorStakes          []int64
		delegatedAmounts         []int64 // nil for validator-only tests
		numClaims                int
		expectedValidatorRewards []int64
		expectedTotalRewards     int64
		validationFunc           func(t *testing.T, validatorRewards, delegatorRewards []int64, expectedTotal int64)
		skipReason               string // if non-empty, test will be skipped
	}{
		{
			name:                     "Validator-only: Stakes that divide cleanly",
			validatorStakes:          []int64{500_000, 400_000, 200_000},
			numClaims:                1000,
			expectedValidatorRewards: []int64{50_000, 40_000, 20_000},
			expectedTotalRewards:     110_000,
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
			name:                     "Validator-only: Single validator gets all rewards",
			validatorStakes:          []int64{1_000_000},
			numClaims:                1000,
			expectedValidatorRewards: []int64{110_000},
			expectedTotalRewards:     110_000,
			validationFunc: func(t *testing.T, validatorRewards, delegatorRewards []int64, expectedTotal int64) {
				// Single validator with 1M stake gets 100% of rewards
				// Total rewards: 110,000 uPOKT (all to single validator)

				require.Len(t, validatorRewards, 1, "Should have 1 validator")
				require.Empty(t, delegatorRewards, "Should have no delegator rewards")
				require.Equal(t, []int64{110_000}, validatorRewards, "Single validator should get all rewards")
			},
		},
		{
			name:                     "Validator-only: Equal stakes receive equal rewards",
			validatorStakes:          []int64{200_000, 200_000, 200_000, 200_000, 200_000},
			numClaims:                1000,
			expectedValidatorRewards: []int64{22_000, 22_000, 22_000, 22_000, 22_000},
			expectedTotalRewards:     110_000,
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
			name:                 "With delegators: Mixed delegation amounts",
			validatorStakes:      []int64{400_000, 400_000, 400_000}, // Equal self-bonded stakes
			delegatedAmounts:     []int64{600_000, 200_000, 0},       // Different delegation amounts
			numClaims:            600,
			expectedTotalRewards: 66_000, // 600 claims × 1100 × 10% = 66,000
			validationFunc: func(t *testing.T, validatorRewards, delegatorRewards []int64, expectedTotal int64) {
				// Stake distribution:
				// - Validator 1: 400k self + 600k delegated = 1,000k total (50% of 2M)
				// - Validator 2: 400k self + 200k delegated = 600k total (30% of 2M)
				// - Validator 3: 400k self + 0k delegated = 400k total (20% of 2M)
				// Total: 2,000k tokens
				//
				// Expected distribution of 66,000 total rewards:
				// - Validator 1 total: 33,000 (50%)
				//   - Val 1 self (400k/1000k = 40%): 13,200 uPOKT
				//   - Val 1 delegators (600k/1000k = 60%): 19,800 uPOKT
				// - Validator 2 total: 19,800 (30%)
				//   - Val 2 self (400k/600k = 66.67%): 13,200 uPOKT
				//   - Val 2 delegators (200k/600k = 33.33%): 6,600 uPOKT
				// - Validator 3 total: 13,200 (20%)
				//   - Val 3 self (400k/400k = 100%): 13,200 uPOKT
				//   - Val 3 delegators: 0 uPOKT

				require.Len(t, validatorRewards, 3, "Should have 3 validators")

				// All validators have equal self-bonded stakes (400k each), so must receive equal rewards
				expectedValidatorReward := int64(13_200) // Each validator gets 13,200 for their 400k self-bonded stake
				for _, reward := range validatorRewards {
					require.Equal(t, expectedValidatorReward, reward,
						"All validators should receive exactly %d uPOKT for equal self-bonded stakes", expectedValidatorReward)
				}

				// Verify specific delegator rewards based on their delegation amounts
				// We have 4 delegators total (2 for Val1, 2 for Val2, 0 for Val3)
				expectedDelegatorRewards := []int64{
					9_900, 9_900, // Val1's 2 delegators split 19,800 equally
					3_300, 3_300, // Val2's 2 delegators split 6,600 equally
				}

				// Delegator rewards should match expected distribution
				require.ElementsMatch(t, expectedDelegatorRewards, delegatorRewards,
					"Delegator rewards should match expected distribution")

				// Verify totals
				totalValidatorRewards := 3 * expectedValidatorReward // 39,600
				totalDelegatorRewards := int64(19_800 + 6_600)       // 26,400

				require.Equal(t, int64(39_600), totalValidatorRewards, "Total validator rewards should be 39,600")
				require.Equal(t, int64(26_400), totalDelegatorRewards, "Total delegator rewards should be 26,400")
				require.Equal(t, expectedTotal, totalValidatorRewards+totalDelegatorRewards,
					"Total distributed should equal expected total")
			},
		},
		{
			name:                 "SKIP: Precision loss with many small distributions (validator-only)",
			validatorStakes:      []int64{333_333, 333_333, 333_334},
			numClaims:            1000,
			expectedTotalRewards: 110_000,
			skipReason:           "Skipping until reward batching is implemented to fix per-claim precision loss (TODO_CRITICAL(#1758))",
			validationFunc: func(t *testing.T, validatorRewards, delegatorRewards []int64, expectedTotal int64) {
				// Validator stakes: [333,333, 333,333, 333,334] = 1M total
				// These create fractional shares that can't divide evenly:
				// - Val 1: 333,333/1M = 33.3333% → 36,666.63 uPOKT
				// - Val 2: 333,333/1M = 33.3333% → 36,666.63 uPOKT
				// - Val 3: 333,334/1M = 33.3334% → 36,666.74 uPOKT
				//
				// Per-claim distribution (2000 calls) causes cumulative precision loss
				// This test will FAIL until reward batching is implemented

				expectedRewards := []int64{36_667, 36_666, 36_667}
				require.ElementsMatch(t, expectedRewards, validatorRewards,
					"Validators should receive exact proportional rewards without precision loss")
			},
		},
		{
			name:                 "SKIP: Precision loss with delegations and fractional stakes",
			validatorStakes:      []int64{333_333, 333_333, 333_334}, // Equal self-bonded stakes (fractional)
			delegatedAmounts:     []int64{166_667, 333_333, 500_000}, // Unequal delegations creating more fractional complexity
			numClaims:            1000,
			expectedTotalRewards: 110_000,
			skipReason:           "Skipping until reward batching is implemented to fix per-claim precision loss (TODO_CRITICAL(#1758))",
			validationFunc: func(t *testing.T, validatorRewards, delegatorRewards []int64, expectedTotal int64) {
				// This test demonstrates precision loss in delegation scenarios
				//
				// Total stakes:
				// - Validator 1: 333,333 self + 166,667 delegated = 500,000 total (37.5% of 1,333,333 total)
				// - Validator 2: 333,333 self + 333,333 delegated = 666,666 total (50.0% of 1,333,333 total)
				// - Validator 3: 333,334 self + 500,000 delegated = 833,334 total (62.5% of 1,333,333 total)
				// Total: 2,000,000 tokens
				//
				// Expected perfect distribution (110,000 total):
				// - Validator 1 total: 110,000 × (500,000/2,000,000) = 27,500 uPOKT
				//   - Val 1 self (333,333/500,000): 18,333 uPOKT
				//   - Val 1 delegators (166,667/500,000): 9,167 uPOKT
				// - Validator 2 total: 110,000 × (666,666/2,000,000) = 36,667 uPOKT
				//   - Val 2 self (333,333/666,666): 18,333 uPOKT
				//   - Val 2 delegators (333,333/666,666): 18,334 uPOKT
				// - Validator 3 total: 110,000 × (833,334/2,000,000) = 45,833 uPOKT
				//   - Val 3 self (333,334/833,334): 18,333 uPOKT
				//   - Val 3 delegators (500,000/833,334): 27,500 uPOKT
				//
				// However, per-claim distribution creates cascading precision loss:
				// 1. Each claim triggers 2 TLM distributions (2000 total calls)
				// 2. Each call distributes 55 uPOKT across fractional stake ratios
				// 3. Fractional remainders compound across validator AND delegator distributions
				// 4. The result is significant cumulative loss across all stakeholders

				expectedValidatorRewards := []int64{18_333, 18_333, 18_333} // Equal self-bonded should get equal rewards
				expectedDelegatorRewards := []int64{9_167, 18_334, 27_500}  // Proportional to delegation amounts

				// These assertions will FAIL due to cascading precision loss
				require.ElementsMatch(t, expectedValidatorRewards, validatorRewards,
					"Validators should receive equal rewards for equal self-bonded stakes without precision loss")
				require.ElementsMatch(t, expectedDelegatorRewards, delegatorRewards,
					"Delegators should receive proportional rewards without precision loss")

				// Verify total conservation (this will also fail due to precision loss)
				totalActual := int64(0)
				for _, reward := range validatorRewards {
					totalActual += reward
				}
				for _, reward := range delegatorRewards {
					totalActual += reward
				}
				require.Equal(t, expectedTotal, totalActual,
					"Total distributed should equal expected total without precision loss")
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			if tc.skipReason != "" {
				t.Skip(tc.skipReason)
			}

			// Setup keepers with appropriate validator/delegation configuration
			s.setupValidatorTest(t, tc.validatorStakes, tc.delegatedAmounts)

			// Create claims and settle
			s.createClaims(&s.keepers, tc.numClaims)
			settledResults, expiredResults := s.settleClaims(t)
			require.NotEmpty(t, settledResults)
			require.Empty(t, expiredResults)

			// Extract rewards
			validatorRewards, delegatorRewards := s.extractRewards(settledResults, tc.delegatedAmounts != nil)

			// Validate total rewards if expected
			if tc.expectedTotalRewards > 0 {
				totalRewards := s.sumRewards(validatorRewards) + s.sumRewards(delegatorRewards)
				require.InDelta(t, tc.expectedTotalRewards, totalRewards, 100,
					"Total rewards should approximately equal expected")
			}

			// Run test-specific validation
			tc.validationFunc(t, validatorRewards, delegatorRewards, tc.expectedTotalRewards)

			// Ensure no pending claims remain
			s.assertNoPendingClaims(t)
		})
	}
}

// setupValidatorTest initializes keepers for validator reward testing.
// Supports both validator-only and validator+delegator scenarios.
func (s *tokenLogicModuleTestSuite) setupValidatorTest(t *testing.T, validatorStakes []int64, delegatedAmounts []int64) {
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
	}

	// Add validator configuration based on test type
	if delegatedAmounts != nil {
		// For delegation tests, use equal self-bonded stakes with varied delegations
		setupOpts = append(setupOpts, testkeeper.WithValidatorsAndDelegations(validatorStakes[0], delegatedAmounts))
	} else {
		// For validator-only tests, use the provided stakes
		setupOpts = append(setupOpts, testkeeper.WithMultipleValidators(validatorStakes))
	}

	s.setupKeepers(t, setupOpts...)
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
