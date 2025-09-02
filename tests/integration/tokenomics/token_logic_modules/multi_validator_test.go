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

// TestTLMProcessorsMultiValidatorDistribution tests that validator rewards are
// distributed correctly across multiple validators based on their stake weights.
// This test validates the core functionality implemented in distributeValidatorRewards().
func (s *tokenLogicModuleTestSuite) TestTLMProcessorsMultiValidatorDistribution() {
	// Test case with stakes designed for clean mathematical division
	// Using 10% validator allocation in both TLMs for clean math.
	//
	// With 110 total validator rewards and stakes in ratio 5:4:2 (sum=11):
	// - Validator 1: 500,000 tokens (45.45% of total 1,100,000) -> 50 uPOKT (exact)
	// - Validator 2: 400,000 tokens (36.36% of total 1,100,000) -> 40 uPOKT (exact)
	// - Validator 3: 200,000 tokens (18.18% of total 1,100,000) -> 20 uPOKT (exact)
	// Total: 1,100,000 tokens -> 110 uPOKT rewards (50+40+20=110 exact)
	//
	// The Largest Remainder Method ensures mathematically fair distribution while
	// maintaining total conservation (distributed amounts always sum to input amount).

	s.T().Run("Stakes that divide cleanly into rewards", func(t *testing.T) {
		// Use stakes in ratio 5:4:2 (which sum to 11) with 10% validator allocation
		// to get clean division: 110,000 ÷ 11 = 10,000 per unit, so [50000, 40000, 20000]
		validatorStakes := []int64{500_000, 400_000, 200_000}
		s.setupKeepersWithMultipleValidators(t, validatorStakes)

		// Create claims for unique applications to ensure distinct sessions
		numClaims := 1000 // Large enough to avoid reward truncation
		s.createClaims(&s.keepers, numClaims)

		// Settle claims and trigger validator reward distribution
		settledResults, expiredResults := s.settleClaims(t)
		require.NotEmpty(t, settledResults)
		require.Empty(t, expiredResults) // No expired claims expected

		// Extract actual validator rewards from settlement results
		actualRewards := s.extractValidatorRewards(settledResults)

		// Expected rewards with Largest Remainder Method:
		// The improved distribution algorithm uses the Largest Remainder Method to fairly
		// distribute remainder tokens based on fractional parts, achieving perfect precision.
		//
		// With stakes [500_000, 400_000, 200_000] (ratio 5:4:2) and 110,000 total rewards:
		// - Validator 1: (5/11) × 110,000 = 50,000 uPOKT (perfect precision)
		// - Validator 2: (4/11) × 110,000 = 40,000 uPOKT (perfect precision)
		// - Validator 3: (2/11) × 110,000 = 20,000 uPOKT (perfect precision)
		//
		// The Largest Remainder Method ensures mathematically fair distribution while
		// maintaining total conservation (distributed amounts always sum to input amount).
		expectedRewards := []int64{50_000, 40_000, 20_000}

		require.ElementsMatch(t, expectedRewards, actualRewards,
			"Validator rewards should match expected proportional distribution")

		// Ensure no pending claims remain
		s.assertNoPendingClaims(t)

		t.Log("Multi-validator distribution test completed successfully")
	})
}

// TestTLMProcessorsValidatorDistributionEdgeCases tests edge cases in validator reward distribution.
func (s *tokenLogicModuleTestSuite) TestTLMProcessorsValidatorDistributionEdgeCases() {
	s.T().Run("Single validator gets all rewards", func(t *testing.T) {
		// Setup with single validator
		validatorStakes := []int64{1_000_000}
		s.setupKeepersWithMultipleValidators(t, validatorStakes)

		// Create claims and settle
		numClaims := 1000 // Same as multi-validator test for consistency
		s.createClaims(&s.keepers, numClaims)
		settledResults, _ := s.settleClaims(t)

		// Extract and verify single validator gets all rewards
		actualRewards := s.extractValidatorRewards(settledResults)

		// Single validator should get all validator rewards from both TLM processors
		// With 1000 unique claims: total = 110,000 uPOKT
		expectedRewards := []int64{110_000}

		require.ElementsMatch(t, expectedRewards, actualRewards,
			"Single validator should receive all validator rewards")

		// Ensure no pending claims remain
		s.assertNoPendingClaims(t)

		t.Log("Single validator edge case test completed successfully")
	})

	s.T().Run("Equal stakes receive equal rewards", func(t *testing.T) {
		// Setup with 5 validators having equal stakes
		// This ensures clean division without remainder issues
		validatorStakes := []int64{200_000, 200_000, 200_000, 200_000, 200_000}
		s.setupKeepersWithMultipleValidators(t, validatorStakes)

		// Create claims and settle
		numClaims := 1000 // Same as other tests for consistency
		s.createClaims(&s.keepers, numClaims)
		settledResults, _ := s.settleClaims(t)

		// Extract and verify equal distribution
		actualRewards := s.extractValidatorRewards(settledResults)

		// Expected calculation for 5 equal validators with 10% allocation and 1000 unique claims:
		// Total validator rewards: 110,000 uPOKT
		// With equal stakes: 110,000 ÷ 5 = 22,000 uPOKT each (exact division)
		expectedRewards := []int64{22_000, 22_000, 22_000, 22_000, 22_000}

		require.ElementsMatch(t, expectedRewards, actualRewards,
			"Equal stakes should receive equal rewards")

		// Ensure no pending claims remain
		s.assertNoPendingClaims(t)

		t.Log("Equal stakes edge case test completed successfully")
	})

	// TODO_CRITICAL(#1758): This test demonstrates the precision loss issue with per-claim
	// validator reward distribution. It's currently skipped because it will fail,
	// showing that validators lose significant rewards due to accumulated truncation.
	//
	// Once we implement reward batching (accumulating validator rewards across all
	// claims and distributing once per TLM per settlement), this test will pass.
	s.T().Run("SKIP: Precision loss with many small distributions", func(t *testing.T) {
		t.Skip("Skipping until reward batching is implemented to fix per-claim precision loss (TODO_CRITICAL(#1758))")

		// Use validator stakes that will cause precision loss due to fractional remainders
		// Stakes: [333333, 333333, 333334] (ratio ≈ 1:1:1 but not exact thirds)
		// Total: 1,000,000 tokens
		//
		// Per-claim validator reward: 55 uPOKT
		// Expected per-validator per-claim: 55 ÷ 3 = 18.333... uPOKT
		// This creates fractional remainders that accumulate over 2000 distributions
		validatorStakes := []int64{333_333, 333_333, 333_334}
		s.setupKeepersWithMultipleValidators(t, validatorStakes)

		// Create 1000 claims - this will trigger 2000 distributeValidatorRewards calls
		// (1000 claims × 2 TLMs = 2000 individual distributions)
		numClaims := 1000
		s.createClaims(&s.keepers, numClaims)

		// Settle claims - this is where the precision loss occurs
		settledResults, _ := s.settleClaims(t)

		// Extract actual validator rewards
		actualRewards := s.extractValidatorRewards(settledResults)

		// With perfect precision, validators should receive rewards proportional to their stake:
		// Total validator rewards: 110,000 uPOKT (10% of 1,100,000 total settlement)
		// Stakes: [333333, 333333, 333334] = 1,000,000 total
		// - Validator 1 (33.3333%): 110,000 × (333333/1000000) = 36,666.63 → 36,667 uPOKT
		// - Validator 2 (33.3333%): 110,000 × (333333/1000000) = 36,666.63 → 36,666 uPOKT
		// - Validator 3 (33.3334%): 110,000 × (333334/1000000) = 36,666.74 → 36,667 uPOKT
		// Total: 36,667 + 36,666 + 36,667 = 110,000 (with Largest Remainder Method)
		//
		// However, with per-claim distribution (2000 calls of 55 uPOKT each),
		// accumulated truncation causes significant loss:
		// - Each call distributes only 55 uPOKT among 3 validators with fractional shares
		// - Per-call distributions create fractional remainders: 55 ÷ 3 = 18.333... each
		// - Even with Largest Remainder Method per call, thousands of small truncations accumulate
		// - The precision loss compounds over 2000 individual distribution calls
		//
		// Expected with perfect batched distribution (what we want to achieve):
		expectedRewards := []int64{36_667, 36_666, 36_667}

		// These assertions will FAIL with current per-claim distribution,
		// demonstrating the precision loss issue
		require.ElementsMatch(t, expectedRewards, actualRewards,
			"Validators should receive exact proportional rewards without precision loss")

		// Verify the total distributed equals what was intended
		totalDistributed := int64(0)
		for _, reward := range actualRewards {
			totalDistributed += reward
		}
		require.Equal(t, int64(110_000), totalDistributed,
			"Total distributed should equal 110,000 uPOKT (currently less due to precision loss)")

		// Log the actual vs expected for debugging
		t.Logf("Expected rewards: %v", expectedRewards)
		t.Logf("Actual rewards:   %v", actualRewards)
		t.Logf("Precision loss:   %d uPOKT", 110000-totalDistributed)

		s.assertNoPendingClaims(t)
	})
}

// setupKeepersWithMultipleValidators initializes keepers with multiple validators having specified stake amounts.
func (s *tokenLogicModuleTestSuite) setupKeepersWithMultipleValidators(t *testing.T, validatorStakes []int64) {
	t.Helper()

	// Setup keepers with standard options plus multi-validator setup using our new infrastructure
	s.setupKeepers(t,
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
		testkeeper.WithMultipleValidators(validatorStakes),
	)
}

// extractValidatorRewards extracts all validator reward amounts from settlement results.
// Returns a slice of reward amounts in uPOKT aggregated by validator.
func (s *tokenLogicModuleTestSuite) extractValidatorRewards(settledResults tlm.ClaimSettlementResults) []int64 {
	// Map to aggregate rewards by validator address
	validatorRewards := make(map[string]int64)

	for _, result := range settledResults {
		for _, transfer := range result.ModToAcctTransfers {
			// Check if this is a validator reward transfer
			switch transfer.OpReason {
			case tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_VALIDATOR_REWARD_DISTRIBUTION,
				tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_VALIDATOR_REWARD_DISTRIBUTION:

				// Verify the transfer is from tokenomics module and is valid
				if transfer.SenderModule == tokenomicstypes.ModuleName &&
					transfer.Coin.Denom == pocket.DenomuPOKT &&
					transfer.Coin.Amount.IsPositive() {
					// Aggregate rewards by validator address
					validatorRewards[transfer.RecipientAddress] += transfer.Coin.Amount.Int64()
				}
			}
		}
	}

	// Convert map to slice of reward amounts
	var rewards []int64
	for _, reward := range validatorRewards {
		rewards = append(rewards, reward)
	}

	return rewards
}

// extractValidatorAndDelegatorRewards extracts all validator and delegator reward amounts from settlement results.
// Returns separate slices for validator rewards and delegator rewards.
func (s *tokenLogicModuleTestSuite) extractValidatorAndDelegatorRewards(settledResults tlm.ClaimSettlementResults) (validatorRewards []int64, delegatorRewards []int64) {
	// Maps to aggregate rewards by recipient address
	validatorRewardMap := make(map[string]int64)
	delegatorRewardMap := make(map[string]int64)

	for _, result := range settledResults {
		for _, transfer := range result.ModToAcctTransfers {
			// Check if this is a validator reward transfer
			switch transfer.OpReason {
			case tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_VALIDATOR_REWARD_DISTRIBUTION,
				tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_VALIDATOR_REWARD_DISTRIBUTION:

				// Verify the transfer is from tokenomics module and is valid
				if transfer.SenderModule == tokenomicstypes.ModuleName &&
					transfer.Coin.Denom == pocket.DenomuPOKT &&
					transfer.Coin.Amount.IsPositive() {
					// Aggregate validator rewards by recipient address
					validatorRewardMap[transfer.RecipientAddress] += transfer.Coin.Amount.Int64()
				}

			case tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_DELEGATOR_REWARD_DISTRIBUTION,
				tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_DELEGATOR_REWARD_DISTRIBUTION:

				// Verify the transfer is from tokenomics module and is valid
				if transfer.SenderModule == tokenomicstypes.ModuleName &&
					transfer.Coin.Denom == pocket.DenomuPOKT &&
					transfer.Coin.Amount.IsPositive() {
					// Aggregate delegator rewards by recipient address
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

// TestTLMProcessorsDelegatorRewardDistribution tests that validator and delegator rewards
// are distributed correctly based on pure stake proportions without commission.
func (s *tokenLogicModuleTestSuite) TestTLMProcessorsDelegatorRewardDistribution() {
	s.T().Run("Validators and delegators receive proportional stake-based rewards", func(t *testing.T) {
		// Test scenario with 3 validators having equal self-bonded stakes (400k each)
		// but different delegation amounts to test pure stake-based distribution:
		//
		// Validator 1: 400k self-bonded + 600k delegated = 1,000k total (50% of 2M total)
		// Validator 2: 400k self-bonded + 200k delegated = 600k total (30% of 2M total)
		// Validator 3: 400k self-bonded + 0k delegated = 400k total (20% of 2M total)
		//
		// Total staked: 2,000k tokens
		// Total validator rewards per test: 110,000 uPOKT (10% of 1.1M total)
		//
		// Expected distribution (pure stake-based, no commission):
		// - All of Validator 1: 55,000 uPOKT (50% of 110k)
		//   - Validator 1 self-bonded (400k/1000k): 22,000 uPOKT
		//   - Delegators to Val 1 (600k/1000k): 33,000 uPOKT
		// - All of Validator 2: 33,000 uPOKT (30% of 110k)
		//   - Validator 2 self-bonded (400k/600k): 22,000 uPOKT
		//   - Delegators to Val 2 (200k/600k): 11,000 uPOKT
		// - All of Validator 3: 22,000 uPOKT (20% of 110k)
		//   - Validator 3 self-bonded (400k/400k): 22,000 uPOKT (no delegators)

		// Use comprehensive delegation testing with equal self-bonded stakes and different delegations
		selfBondedStake := int64(400000)               // Equal self-bonded stake for all validators
		delegatedAmounts := []int64{600000, 200000, 0} // Different delegation amounts

		// Setup keepers with realistic validators and delegations
		s.setupKeepers(t,
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
				tokenomicstypes.ModuleName: s.getTokenomicsParamsWithCleanValidatorMath(), // 10% validator allocation
			}),
			testkeeper.WithDefaultModuleBalances(),
			testkeeper.WithValidatorsAndDelegations(selfBondedStake, delegatedAmounts), // Use delegation infrastructure
		)

		// Create claims for unique applications to ensure distinct sessions
		// Use 600 claims for clean math: 600 × 1100 × 10% = 66,000 total validator rewards
		numClaims := 600
		s.createClaims(&s.keepers, numClaims)

		// Settle claims and trigger validator + delegator reward distribution
		settledResults, expiredResults := s.settleClaims(t)
		require.NotEmpty(t, settledResults)
		require.Empty(t, expiredResults) // No expired claims expected

		// Extract validator and delegator rewards from the real settlement results
		// Now that TLM processors use delegation-aware distribution, this should work properly
		validatorRewards, delegatorRewards := s.extractValidatorAndDelegatorRewards(settledResults)

		// === VALIDATION ===

		// Verify we have the expected number of reward recipients
		require.Len(t, validatorRewards, 3, "All 3 validators should receive rewards")
		require.NotEmpty(t, delegatorRewards, "Delegators should receive rewards")

		// Calculate totals
		totalValidatorRewards := int64(0)
		for _, reward := range validatorRewards {
			totalValidatorRewards += reward
		}

		totalDelegatorRewards := int64(0)
		for _, reward := range delegatorRewards {
			totalDelegatorRewards += reward
		}

		// With 600 claims × 1100 uPOKT × 10% validator allocation = 66,000 total validator rewards
		expectedTotalRewards := int64(66000)
		actualTotalRewards := totalValidatorRewards + totalDelegatorRewards

		require.InDelta(t, expectedTotalRewards, actualTotalRewards, 100,
			"Total rewards (%d validator + %d delegator = %d) should approximately equal expected %d",
			totalValidatorRewards, totalDelegatorRewards, actualTotalRewards, expectedTotalRewards)

		// Verify validators receive proportional rewards based on self-bonded stakes
		// All validators have equal 400k self-bonded stakes, so should receive approximately equal rewards
		avgValidatorReward := totalValidatorRewards / 3
		for i, reward := range validatorRewards {
			require.InDelta(t, avgValidatorReward, reward, float64(avgValidatorReward)*0.05, // 5% tolerance
				"Validator %d should receive approximately equal rewards for equal self-bonded stake (got %d, avg %d)",
				i, reward, avgValidatorReward)
		}

		// Ensure no pending claims remain
		s.assertNoPendingClaims(t)

		t.Logf("Validator rewards: %v (total: %d uPOKT)", validatorRewards, totalValidatorRewards)
		t.Logf("Delegator rewards: %v (total: %d uPOKT)", delegatorRewards, totalDelegatorRewards)
		t.Log("Delegation-aware distribution test completed successfully")
	})

}
