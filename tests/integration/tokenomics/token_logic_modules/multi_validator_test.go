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
	// Using 10% validator allocation in both TLMs for clean math:
	// With 110 total validator rewards and stakes in ratio 5:4:2 (sum=11):
	// Validator 1: 500,000 tokens (45.45% of total 1,100,000) -> 50 uPOKT (exact)
	// Validator 2: 400,000 tokens (36.36% of total 1,100,000) -> 40 uPOKT (exact)
	// Validator 3: 200,000 tokens (18.18% of total 1,100,000) -> 20 uPOKT (exact)
	// Total: 1,100,000 tokens -> 110 uPOKT rewards (50+40+20=110 exact)

	s.T().Run("Stakes that divide cleanly into rewards", func(t *testing.T) {
		// Use stakes in ratio 5:4:2 (which sum to 11) with 10% validator allocation
		// to get clean division: 110,000 ÷ 11 = 10,000 per unit, so [50000, 40000, 20000]
		validatorStakes := []int64{500000, 400000, 200000}
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
		// With stakes [500000, 400000, 200000] (ratio 5:4:2) and 110,000 total rewards:
		// - Validator 1: (5/11) × 110,000 = 50,000 uPOKT (perfect precision)
		// - Validator 2: (4/11) × 110,000 = 40,000 uPOKT (perfect precision)
		// - Validator 3: (2/11) × 110,000 = 20,000 uPOKT (perfect precision)
		//
		// The Largest Remainder Method ensures mathematically fair distribution while
		// maintaining total conservation (distributed amounts always sum to input amount).
		expectedRewards := []int64{50000, 40000, 20000}

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
		validatorStakes := []int64{1000000}
		s.setupKeepersWithMultipleValidators(t, validatorStakes)

		// Create claims and settle
		numClaims := 1000 // Same as multi-validator test for consistency
		s.createClaims(&s.keepers, numClaims)
		settledResults, _ := s.settleClaims(t)

		// Extract and verify single validator gets all rewards
		actualRewards := s.extractValidatorRewards(settledResults)

		// Single validator should get all validator rewards from both TLM processors
		// With 1000 unique claims: total = 110,000 uPOKT
		expectedRewards := []int64{110000}

		require.ElementsMatch(t, expectedRewards, actualRewards,
			"Single validator should receive all validator rewards")

		// Ensure no pending claims remain
		s.assertNoPendingClaims(t)

		t.Log("Single validator edge case test completed successfully")
	})

	s.T().Run("Equal stakes receive equal rewards", func(t *testing.T) {
		// Setup with 5 validators having equal stakes
		// This ensures clean division without remainder issues
		validatorStakes := []int64{200000, 200000, 200000, 200000, 200000}
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
		expectedRewards := []int64{22000, 22000, 22000, 22000, 22000}

		require.ElementsMatch(t, expectedRewards, actualRewards,
			"Equal stakes should receive equal rewards")

		// Ensure no pending claims remain
		s.assertNoPendingClaims(t)

		t.Log("Equal stakes edge case test completed successfully")
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
		testkeeper.WithMultipleValidators(validatorStakes), // Use our new multi-validator option
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
