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
		validatorStakes          []int64
		delegatedAmounts         []int64 // nil when validators have no delegators
		numClaims                int
		expectedValidatorRewards []int64
		expectedTotalRewards     int64
		validationFunc           func(t *testing.T, validatorRewards, delegatorRewards []int64, expectedTotal int64)
		// TODO_TECHDEBT(#1758): Remove once skipped tests are passing.
		skipReason string // if non-empty, test will be skipped
	}{
		{
			name:                     "No vlidator delegators: Stakes that divide cleanly",
			validatorStakes:          []int64{500_000, 400_000, 200_000},
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
			name:                     "No validator delegators: Single validator gets all rewards",
			validatorStakes:          []int64{1_000_000},
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
			name:                     "No validator delegators: Equal stakes receive equal rewards",
			validatorStakes:          []int64{200_000, 200_000, 200_000, 200_000, 200_000},
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
			name:                 "With validator delegators: Mixed delegation amounts",
			validatorStakes:      []int64{250_000, 250_000, 250_000}, // Equal self-bonded stakes
			delegatedAmounts:     []int64{250_000, 250_000, 0},       // Different delegation amounts (clean divisible)
			numClaims:            160,
			expectedTotalRewards: 17_600, // 160 claims × 1100 × 10% = 17,600
			validationFunc: func(t *testing.T, validatorRewards, delegatorRewards []int64, expectedTotal int64) {
				// With simplified logic, all stakeholders (validators + delegators) get the same operation reason,
				// so extractRewards will put all rewards into validatorRewards array.
				//
				// Stake distribution (perfectly clean divisible numbers):
				// - Validator 1: 250k self + 250k delegated = 500k total (2/5 = 40% of 1.25M)
				// - Validator 2: 250k self + 250k delegated = 500k total (2/5 = 40% of 1.25M)
				// - Validator 3: 250k self + 0k delegated = 250k total (1/5 = 20% of 1.25M)
				// Total: 1,250k tokens
				//
				// Expected individual stakeholder rewards (17,600 total):
				// - 3 validator self-stakes (250k each): 17,600 × (250k/1.25M) = 3,520 each = 10,560 total
				// - 4 delegator stakes (125k each): 17,600 × (125k/1.25M) = 1,760 each = 7,040 total
				// Total rewards: 7 recipients getting [3520, 1760, 1760, 3520, 1760, 1760, 3520]

				// Since all rewards go to validatorRewards array (same operation reason), check total count
				require.Len(t, validatorRewards, 7, "Should have 7 total stakeholders (3 validators + 4 delegators)")
				require.Empty(t, delegatorRewards, "Should have no separate delegator rewards (same operation reason)")

				// Verify the expected reward distribution
				expectedRewards := []int64{3_520, 1_760, 1_760, 3_520, 1_760, 1_760, 3_520}
				require.ElementsMatch(t, expectedRewards, validatorRewards,
					"All stakeholder rewards should match expected distribution")

				// Verify total matches expected
				totalRewards := s.sumRewards(validatorRewards)
				require.Equal(t, expectedTotal, totalRewards,
					"Total distributed should equal expected total")
			},
		},
		{
			name:                 "SKIP: Precision loss with many small distributions (no delegators)",
			validatorStakes:      []int64{333_333, 333_333, 333_334},
			numClaims:            1000,
			expectedTotalRewards: 110_000, // 1000 claims × 1100 × 10% = 110,000
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
			expectedTotalRewards: 110_000, // 1000 claims × 1100 × 10% = 110,000
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
				//
				// - Validator 2 total: 110,000 × (666,666/2,000,000) = 36,667 uPOKT
				//   - Val 2 self (333,333/666,666): 18,333 uPOKT
				//   - Val 2 delegators (333,333/666,666): 18,334 uPOKT
				//
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
		// For validators with no delegators, use the provided stakes
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
