package token_logic_module

import (
	"sort"
	"testing"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/tokenomics/mocks"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// TestValidatorRewardDistribution_DeterministicOrdering tests that reward distribution
// is deterministic when validators have equal stakes. It verifies that:
// 1. Addresses are sorted alphabetically when stakes are equal
// 2. The same rewards are distributed in the same order every time
// 3. The Largest Remainder Method works deterministically with equal fractions
func TestValidatorRewardDistribution_DeterministicOrdering(t *testing.T) {
	// Test Case 1: Equal stakes - should result in alphabetical ordering
	t.Run("equal_stakes_alphabetical_ordering", func(t *testing.T) {
		const numValidators = 3
		const numRuns = 5

		// Generate validator addresses once
		valOperAddresses := make([]string, numValidators)
		expectedAccAddresses := make([]string, numValidators)

		for i := 0; i < numValidators; i++ {
			valOperAddresses[i] = sample.ValOperatorAddressBech32()
			// Convert to account address for expected results
			valAddr, err := cosmostypes.ValAddressFromBech32(valOperAddresses[i])
			require.NoError(t, err)
			expectedAccAddresses[i] = cosmostypes.AccAddress(valAddr).String()
		}

		// Sort expected addresses alphabetically (this is what the implementation should do)
		sort.Strings(expectedAccAddresses)

		// Run multiple times to ensure determinism
		var previousTransfers []tokenomicstypes.ModToAcctTransfer

		for run := 0; run < numRuns; run++ {
			ctrl := gomock.NewController(t)
			mockStakingKeeper := mocks.NewMockStakingKeeper(ctrl)

			// Create validators with equal stakes (1000 each)
			validators := make([]stakingtypes.Validator, numValidators)
			for i := 0; i < numValidators; i++ {
				validators[i] = createValidator(valOperAddresses[i], 1000)
			}

			// Shuffle validators to test ordering is deterministic
			if run > 0 {
				validators[0], validators[len(validators)-1] = validators[len(validators)-1], validators[0]
			}

			// Setup mocks
			mockStakingKeeper.EXPECT().
				GetBondedValidatorsByPower(gomock.Any()).
				Return(validators, nil)

			for _, validator := range validators {
				valAddr, _ := cosmostypes.ValAddressFromBech32(validator.OperatorAddress)
				mockStakingKeeper.EXPECT().
					GetValidatorDelegations(gomock.Any(), valAddr).
					Return([]stakingtypes.Delegation{}, nil)
			}

			// Execute distribution
			config := getDefaultTestConfig()
			config.rewardAmount = math.NewInt(100)

			result, err := executeDistribution(mockStakingKeeper, config, false)
			require.NoError(t, err)

			transfers := result.GetModToAcctTransfers()
			require.Len(t, transfers, numValidators)

			// Verify addresses are in alphabetical order
			for i, transfer := range transfers {
				require.Equal(t, expectedAccAddresses[i], transfer.RecipientAddress,
					"Run %d: Transfer %d should go to %s, but went to %s",
					run, i, expectedAccAddresses[i], transfer.RecipientAddress)
			}

			// Verify determinism across runs
			if run > 0 {
				require.Equal(t, len(previousTransfers), len(transfers))
				for i := range transfers {
					require.Equal(t, previousTransfers[i].RecipientAddress, transfers[i].RecipientAddress,
						"Run %d: Transfer %d recipient differs", run, i)
					require.Equal(t, previousTransfers[i].Coin.Amount, transfers[i].Coin.Amount,
						"Run %d: Transfer %d amount differs", run, i)
				}
			}

			previousTransfers = transfers
			ctrl.Finish()
		}
	})

	// Test Case 2: Mixed stakes - should sort by stake then alphabetically
	t.Run("mixed_stakes_deterministic_ordering", func(t *testing.T) {
		// Create 4 validators with different stakes
		valOperAddresses := make([]string, 4)
		accAddresses := make([]string, 4)
		stakes := []int64{2000, 1000, 1000, 3000}

		for i := 0; i < 4; i++ {
			valOperAddresses[i] = sample.ValOperatorAddressBech32()
			valAddr, err := cosmostypes.ValAddressFromBech32(valOperAddresses[i])
			require.NoError(t, err)
			accAddresses[i] = cosmostypes.AccAddress(valAddr).String()
		}

		// Create expected order: sort by stake desc, then by address for equal stakes
		type addrStake struct {
			addr  string
			stake int64
		}

		addrStakes := make([]addrStake, 4)
		for i := 0; i < 4; i++ {
			addrStakes[i] = addrStake{accAddresses[i], stakes[i]}
		}

		// Sort by stake desc, then by address
		sort.Slice(addrStakes, func(i, j int) bool {
			if addrStakes[i].stake == addrStakes[j].stake {
				return addrStakes[i].addr < addrStakes[j].addr
			}
			return addrStakes[i].stake > addrStakes[j].stake
		})

		expectedOrder := make([]string, 4)
		for i, as := range addrStakes {
			expectedOrder[i] = as.addr
		}

		// Test execution
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStakingKeeper := mocks.NewMockStakingKeeper(ctrl)

		validators := make([]stakingtypes.Validator, 4)
		for i := 0; i < 4; i++ {
			validators[i] = createValidator(valOperAddresses[i], stakes[i])
		}

		mockStakingKeeper.EXPECT().
			GetBondedValidatorsByPower(gomock.Any()).
			Return(validators, nil)

		for _, validator := range validators {
			valAddr, _ := cosmostypes.ValAddressFromBech32(validator.OperatorAddress)
			mockStakingKeeper.EXPECT().
				GetValidatorDelegations(gomock.Any(), valAddr).
				Return([]stakingtypes.Delegation{}, nil)
		}

		config := getDefaultTestConfig()
		config.rewardAmount = math.NewInt(1000)

		result, err := executeDistribution(mockStakingKeeper, config, false)
		require.NoError(t, err)

		transfers := result.GetModToAcctTransfers()
		require.Len(t, transfers, 4)

		// Verify order matches expected
		for i, transfer := range transfers {
			require.Equal(t, expectedOrder[i], transfer.RecipientAddress,
				"Transfer %d should go to %s, but went to %s",
				i, expectedOrder[i], transfer.RecipientAddress)
		}
	})
}

// TestValidatorRewardDistribution_LRM_EqualFractions tests the Largest Remainder Method
// with equal fractional remainders to ensure deterministic distribution.
func TestValidatorRewardDistribution_LRM_EqualFractions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStakingKeeper := mocks.NewMockStakingKeeper(ctrl)

	// Create 3 validators with equal stakes that will produce equal fractional remainders
	valOperAddresses := make([]string, 3)
	accAddresses := make([]string, 3)

	for i := 0; i < 3; i++ {
		valOperAddresses[i] = sample.ValOperatorAddressBech32()
		valAddr, err := cosmostypes.ValAddressFromBech32(valOperAddresses[i])
		require.NoError(t, err)
		accAddresses[i] = cosmostypes.AccAddress(valAddr).String()
	}

	// Sort to know which one should get the extra token
	sort.Strings(accAddresses)

	validators := make([]stakingtypes.Validator, 3)
	for i := 0; i < 3; i++ {
		// Equal stakes of 333 each (total 999)
		validators[i] = createValidator(valOperAddresses[i], 333)
	}

	mockStakingKeeper.EXPECT().
		GetBondedValidatorsByPower(gomock.Any()).
		Return(validators, nil)

	for _, validator := range validators {
		valAddr, _ := cosmostypes.ValAddressFromBech32(validator.OperatorAddress)
		mockStakingKeeper.EXPECT().
			GetValidatorDelegations(gomock.Any(), valAddr).
			Return([]stakingtypes.Delegation{}, nil)
	}

	config := getDefaultTestConfig()
	config.rewardAmount = math.NewInt(100) // 100 tokens to distribute among 3 equal stakes

	result, err := executeDistribution(mockStakingKeeper, config, false)
	require.NoError(t, err)

	transfers := result.GetModToAcctTransfers()
	require.Len(t, transfers, 3)

	// With 100 tokens and 3 equal stakes:
	// Base: 33 each (99 total)
	// Remainder: 1 token
	// The remainder should go to the first address alphabetically

	// Count how many got 34 vs 33
	got34Count := 0
	got33Count := 0
	var totalDistributed int64

	for _, transfer := range transfers {
		amount := transfer.Coin.Amount.Int64()
		totalDistributed += amount

		switch amount {
		case 34:
			got34Count++
			// The one with 34 should be the first alphabetically
			require.Equal(t, accAddresses[0], transfer.RecipientAddress,
				"The extra token should go to the first address alphabetically")
		case 33:
			got33Count++
		default:
			t.Fatalf("Unexpected amount: %d", amount)
		}
	}

	require.Equal(t, 1, got34Count, "Exactly one validator should receive 34 tokens")
	require.Equal(t, 2, got33Count, "Exactly two validators should receive 33 tokens")
	require.Equal(t, int64(100), totalDistributed, "Total distributed should equal total reward")
}

// TestValidatorRewardDistribution_MultipleRunsDeterminism tests that running the
// distribution multiple times with the same inputs produces identical results.
func TestValidatorRewardDistribution_MultipleRunsDeterminism(t *testing.T) {
	const numRuns = 10
	const numValidators = 5

	// Generate fixed addresses and stakes
	valOperAddresses := make([]string, numValidators)
	stakes := []int64{1500, 1500, 1000, 1000, 1000} // Mix of equal and different stakes

	for i := 0; i < numValidators; i++ {
		valOperAddresses[i] = sample.ValOperatorAddressBech32()
	}

	var firstRunTransfers []tokenomicstypes.ModToAcctTransfer

	for run := 0; run < numRuns; run++ {
		ctrl := gomock.NewController(t)
		mockStakingKeeper := mocks.NewMockStakingKeeper(ctrl)

		// Create validators
		validators := make([]stakingtypes.Validator, numValidators)
		for i := 0; i < numValidators; i++ {
			validators[i] = createValidator(valOperAddresses[i], stakes[i])
		}

		// Setup mocks
		mockStakingKeeper.EXPECT().
			GetBondedValidatorsByPower(gomock.Any()).
			Return(validators, nil)

		for _, validator := range validators {
			valAddr, _ := cosmostypes.ValAddressFromBech32(validator.OperatorAddress)
			mockStakingKeeper.EXPECT().
				GetValidatorDelegations(gomock.Any(), valAddr).
				Return([]stakingtypes.Delegation{}, nil)
		}

		// Execute distribution
		config := getDefaultTestConfig()
		config.rewardAmount = math.NewInt(1234) // Use an amount that will create remainders

		result, err := executeDistribution(mockStakingKeeper, config, false)
		require.NoError(t, err)

		transfers := result.GetModToAcctTransfers()

		if run == 0 {
			firstRunTransfers = transfers
		} else {
			// Verify identical results
			require.Equal(t, len(firstRunTransfers), len(transfers),
				"Run %d: Different number of transfers", run)

			// Build maps for comparison
			firstRunMap := make(map[string]math.Int)
			currentRunMap := make(map[string]math.Int)

			for _, transfer := range firstRunTransfers {
				firstRunMap[transfer.RecipientAddress] = transfer.Coin.Amount
			}

			for _, transfer := range transfers {
				currentRunMap[transfer.RecipientAddress] = transfer.Coin.Amount
			}

			// Compare maps
			for addr, amount := range firstRunMap {
				currentAmount, exists := currentRunMap[addr]
				require.True(t, exists, "Run %d: Address %s missing", run, addr)
				require.True(t, amount.Equal(currentAmount),
					"Run %d: Amount differs for %s: first=%s, current=%s",
					run, addr, amount, currentAmount)
			}
		}

		ctrl.Finish()
	}
}

// helper functions are already defined in validator_distribution_test.go
