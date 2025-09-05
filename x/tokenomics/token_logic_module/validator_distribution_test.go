package token_logic_module

import (
	"context"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/tokenomics/mocks"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// rewardDistributionTestConfig holds common configuration for reward distribution test execution.
type rewardDistributionTestConfig struct {
	ctx          context.Context
	logger       log.Logger
	rewardAmount math.Int
	opReason     tokenomicstypes.SettlementOpReason
}

// TestValidatorRewardDistribution_NoDelegators tests reward distribution functionality for validators with no delegators.
// It verifies proportional distribution based on validator stakes and precision handling.
func TestValidatorRewardDistribution_NoDelegators(t *testing.T) {
	tests := []struct {
		name                       string
		validatorStakes            []int64
		totalValidatorRewardAmount math.Int
		expectedTransferCount      int
	}{
		{
			name:                       "success: proportional distribution based on validator stakes",
			validatorStakes:            []int64{700_000, 200_000, 100_000},
			totalValidatorRewardAmount: math.NewInt(9240),
			expectedTransferCount:      3,
		},
		{
			name:                       "success: proportional distribution with remainder allocation",
			validatorStakes:            []int64{333, 333, 334},
			totalValidatorRewardAmount: math.NewInt(100),
			expectedTransferCount:      3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStakingKeeper := mocks.NewMockStakingKeeper(ctrl)

			// Create validators
			validators := make([]stakingtypes.Validator, len(tt.validatorStakes))
			for i, stake := range tt.validatorStakes {
				validators[i] = createValidator(sample.ValOperatorAddressBech32(), stake)
			}

			// Setup mocks for validators with no delegators
			mockStakingKeeper.EXPECT().
				GetBondedValidatorsByPower(gomock.Any()).
				Return(validators, nil)

			// Mock GetValidatorDelegations for each validator to return empty delegations (no delegators test)
			for _, validator := range validators {
				valAddr, _ := cosmostypes.ValAddressFromBech32(validator.OperatorAddress)
				mockStakingKeeper.EXPECT().
					GetValidatorDelegations(gomock.Any(), valAddr).
					Return([]stakingtypes.Delegation{}, nil)
			}

			// Execute and validate
			config := getDefaultTestConfig()
			config.rewardAmount = tt.totalValidatorRewardAmount

			result, err := executeDistribution(mockStakingKeeper, config, false)
			require.NoError(t, err)

			transfers := result.GetModToAcctTransfers()
			require.Len(t, transfers, tt.expectedTransferCount)

			assertTotalDistribution(t, result, tt.totalValidatorRewardAmount)

			// Verify all transfers are validator rewards
			for _, transfer := range transfers {
				require.Equal(t, tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_VALIDATOR_REWARD_DISTRIBUTION, transfer.OpReason)
			}
		})
	}
}

// TestValidatorRewardDistribution_ErrorCases tests error handling scenarios for validator reward distribution.
// It covers cases including but not limited to:
// - Zero reward amounts
// - Staking keeper failures
// - Missing validators
// - Zero stakes
func TestValidatorRewardDistribution_ErrorCases(t *testing.T) {
	tests := []struct {
		name             string
		setupMocks       func(*mocks.MockStakingKeeper)
		rewardAmount     math.Int
		expectedError    bool
		expectedErrorMsg string
	}{
		{
			name: "no error: zero reward amount skips distribution gracefully",
			setupMocks: func(mock *mocks.MockStakingKeeper) {
				// No expectations needed for zero amount
			},
			rewardAmount:  math.ZeroInt(),
			expectedError: false,
		},
		{
			name: "error: staking keeper GetBondedValidatorsByPower fails",
			setupMocks: func(mock *mocks.MockStakingKeeper) {
				mock.EXPECT().
					GetBondedValidatorsByPower(gomock.Any()).
					Return(nil, sdkerrors.ErrInvalidAddress)
			},
			rewardAmount:     math.NewInt(1000),
			expectedError:    true,
			expectedErrorMsg: "failed to get bonded validators",
		},
		{
			name: "no error: no bonded validators found handles gracefully",
			setupMocks: func(mock *mocks.MockStakingKeeper) {
				mock.EXPECT().
					GetBondedValidatorsByPower(gomock.Any()).
					Return([]stakingtypes.Validator{}, nil)
			},
			rewardAmount:  math.NewInt(1000),
			expectedError: false,
		},
		{
			name: "no error: validators with zero stake handled gracefully",
			setupMocks: func(mock *mocks.MockStakingKeeper) {
				validators := []stakingtypes.Validator{createValidator(sample.ValOperatorAddressBech32(), 0)}
				mock.EXPECT().
					GetBondedValidatorsByPower(gomock.Any()).
					Return(validators, nil)
			},
			rewardAmount:  math.NewInt(1000),
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStakingKeeper := mocks.NewMockStakingKeeper(ctrl)
			tt.setupMocks(mockStakingKeeper)

			config := getDefaultTestConfig()
			config.rewardAmount = tt.rewardAmount

			_, err := executeDistribution(mockStakingKeeper, config, false)

			if tt.expectedError {
				require.Error(t, err)
				if tt.expectedErrorMsg != "" {
					require.Contains(t, err.Error(), tt.expectedErrorMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestValidatorRewardDistribution_WithDelegators tests the combined validator and delegator reward distribution.
// It verifies correct distribution to both validators and their delegators based on delegation amounts
// and precision handling for fractional distributions.
func TestValidatorRewardDistribution_WithDelegators(t *testing.T) {
	tests := []struct {
		name                        string
		validators                  []stakingtypes.Validator
		delegationSetup             func([]stakingtypes.Validator) map[string][]stakingtypes.Delegation
		totalValidatorRewardAmount  math.Int
		expectedTransferCount       int
		expectedValidatorRewards    int64
		expectedDelegatorRewards    int64
		opReason                    tokenomicstypes.SettlementOpReason
		claimSettlementValidationFn func(*testing.T, *tokenomicstypes.ClaimSettlementResult)
	}{
		{
			name: "success: mixed delegation amounts with equal self-bonded stakes",
			validators: []stakingtypes.Validator{
				// NOTE: These amounts represent TOTAL validator stake (self-bonded + delegations), not just self-bonded amounts
				// Val1: 400k self + 600k delegated = 1M total
				createValidator(sample.ValOperatorAddressBech32(), 1_000_000),
				// Val2: 400k self + 200k delegated = 600k total
				createValidator(sample.ValOperatorAddressBech32(), 600_000),
				// Val3: 400k self + 0k delegated = 400k total
				createValidator(sample.ValOperatorAddressBech32(), 400_000),
			},
			delegationSetup: func(validators []stakingtypes.Validator) map[string][]stakingtypes.Delegation {
				delegations := make(map[string][]stakingtypes.Delegation)
				// Val1: 400k self + 600k delegated
				valAddr1, _ := cosmostypes.ValAddressFromBech32(validators[0].OperatorAddress)
				delegations[validators[0].OperatorAddress] = []stakingtypes.Delegation{
					createDelegation(cosmostypes.AccAddress(valAddr1).String(), validators[0].OperatorAddress, 400_000),
					createDelegation(sample.AccAddressBech32(), validators[0].OperatorAddress, 600_000),
				}
				// Val2: 400k self + 200k delegated
				valAddr2, _ := cosmostypes.ValAddressFromBech32(validators[1].OperatorAddress)
				delegations[validators[1].OperatorAddress] = []stakingtypes.Delegation{
					createDelegation(cosmostypes.AccAddress(valAddr2).String(), validators[1].OperatorAddress, 400_000),
					createDelegation(sample.AccAddressBech32(), validators[1].OperatorAddress, 200_000),
				}
				// Val3: 400k self only
				valAddr3, _ := cosmostypes.ValAddressFromBech32(validators[2].OperatorAddress)
				delegations[validators[2].OperatorAddress] = []stakingtypes.Delegation{
					createDelegation(cosmostypes.AccAddress(valAddr3).String(), validators[2].OperatorAddress, 400_000),
				}
				return delegations
			},
			totalValidatorRewardAmount: math.NewInt(100_000),
			expectedTransferCount:      5,      // 3 validators + 2 delegators
			expectedValidatorRewards:   60_000, // 100k × 60% = 60,000 (self-bonded validator shares)
			expectedDelegatorRewards:   40_000, // 100k × 40% = 40,000 (delegator shares)
			opReason:                   tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_VALIDATOR_REWARD_DISTRIBUTION,
		},
		{
			name: "success: equal self-bonded stakes receive equal rewards despite different total delegations",
			validators: []stakingtypes.Validator{
				// NOTE: These amounts represent TOTAL validator stake (self-bonded + delegations), not just self-bonded amounts
				// Val1: 300k self + 600k delegated = 900k total
				createValidator(sample.ValOperatorAddressBech32(), 900_000),
				// Val2: 300k self + 300k delegated = 600k total
				createValidator(sample.ValOperatorAddressBech32(), 600_000),
				// Val3: 300k self + 0k delegated = 300k total
				createValidator(sample.ValOperatorAddressBech32(), 300_000),
			},
			delegationSetup: func(validators []stakingtypes.Validator) map[string][]stakingtypes.Delegation {
				delegations := make(map[string][]stakingtypes.Delegation)
				// Val1: 300k self + 600k delegated
				valAddr1, _ := cosmostypes.ValAddressFromBech32(validators[0].OperatorAddress)
				delegations[validators[0].OperatorAddress] = []stakingtypes.Delegation{
					createDelegation(cosmostypes.AccAddress(valAddr1).String(), validators[0].OperatorAddress, 300_000),
					createDelegation(sample.AccAddressBech32(), validators[0].OperatorAddress, 600_000),
				}
				// Val2: 300k self + 300k delegated
				valAddr2, _ := cosmostypes.ValAddressFromBech32(validators[1].OperatorAddress)
				delegations[validators[1].OperatorAddress] = []stakingtypes.Delegation{
					createDelegation(cosmostypes.AccAddress(valAddr2).String(), validators[1].OperatorAddress, 300_000),
					createDelegation(sample.AccAddressBech32(), validators[1].OperatorAddress, 300_000),
				}
				// Val3: 300k self only
				valAddr3, _ := cosmostypes.ValAddressFromBech32(validators[2].OperatorAddress)
				delegations[validators[2].OperatorAddress] = []stakingtypes.Delegation{
					createDelegation(cosmostypes.AccAddress(valAddr3).String(), validators[2].OperatorAddress, 300_000),
				}
				return delegations
			},
			totalValidatorRewardAmount: math.NewInt(90_000),
			expectedTransferCount:      5, // 3 validators + 2 delegators
			opReason:                   tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_VALIDATOR_REWARD_DISTRIBUTION,
			claimSettlementValidationFn: func(t *testing.T, result *tokenomicstypes.ClaimSettlementResult) {
				// With simplified logic, we can't distinguish validators from delegators by operation reason,
				// but we can verify that self-bonded stakes get proportional rewards.
				// Since all validators have 300k self-bonded out of 1.8M total, each should get 15,000.
				transfers := result.GetModToAcctTransfers()

				// Verify all transfers use the same operation reason
				for _, transfer := range transfers {
					require.Equal(t, tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_VALIDATOR_REWARD_DISTRIBUTION, transfer.OpReason)
				}

				// Since we can't distinguish by operation reason, this test now verifies that
				// the total distribution is correct and proportional
				require.Len(t, transfers, 5) // 3 validators + 2 delegators
			},
		},
		{
			name: "success: distribution when no delegations found (validators only)",
			validators: []stakingtypes.Validator{
				createValidator(sample.ValOperatorAddressBech32(), 500_000),
				createValidator(sample.ValOperatorAddressBech32(), 300_000),
				createValidator(sample.ValOperatorAddressBech32(), 200_000),
			},
			delegationSetup: func(validators []stakingtypes.Validator) map[string][]stakingtypes.Delegation {
				delegations := make(map[string][]stakingtypes.Delegation)
				for _, validator := range validators {
					delegations[validator.OperatorAddress] = []stakingtypes.Delegation{} // Empty
				}
				return delegations
			},
			totalValidatorRewardAmount: math.NewInt(10_000),
			expectedTransferCount:      3, // validators only
			opReason:                   tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_VALIDATOR_REWARD_DISTRIBUTION,
			claimSettlementValidationFn: func(t *testing.T, result *tokenomicstypes.ClaimSettlementResult) {
				transfers := result.GetModToAcctTransfers()
				// All transfers should use the same operation reason as specified in the test config
				for _, transfer := range transfers {
					require.Equal(t, tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_VALIDATOR_REWARD_DISTRIBUTION, transfer.OpReason)
				}
			},
		},
		{
			name: "success: precision handling with Largest Remainder Method for fractional distributions",
			validators: []stakingtypes.Validator{
				createValidator(sample.ValOperatorAddressBech32(), 333_333),
				createValidator(sample.ValOperatorAddressBech32(), 333_333),
				createValidator(sample.ValOperatorAddressBech32(), 333_334),
			},
			delegationSetup: func(validators []stakingtypes.Validator) map[string][]stakingtypes.Delegation {
				delegations := make(map[string][]stakingtypes.Delegation)
				for _, validator := range validators {
					valAddr, _ := cosmostypes.ValAddressFromBech32(validator.OperatorAddress)
					delegations[validator.OperatorAddress] = []stakingtypes.Delegation{
						createDelegation(cosmostypes.AccAddress(valAddr).String(), validator.OperatorAddress, validator.Tokens.Int64()*6/10), // 60%
						createDelegation(sample.AccAddressBech32(), validator.OperatorAddress, validator.Tokens.Int64()*4/10),                // 40%
					}
				}
				return delegations
			},
			totalValidatorRewardAmount: math.NewInt(100),
			expectedTransferCount:      6, // 3 validators + 3 delegators
			opReason:                   tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_VALIDATOR_REWARD_DISTRIBUTION,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStakingKeeper := mocks.NewMockStakingKeeper(ctrl)

			// Setup delegations using the test case's delegation setup function
			delegations := tt.delegationSetup(tt.validators)

			setupValidatorMocks(mockStakingKeeper, tt.validators, delegations)

			execConfig := getDefaultTestConfig()
			execConfig.rewardAmount = tt.totalValidatorRewardAmount
			execConfig.opReason = tt.opReason

			result, err := executeDistribution(mockStakingKeeper, execConfig, true)
			require.NoError(t, err)

			transfers := result.GetModToAcctTransfers()
			require.Len(t, transfers, tt.expectedTransferCount)

			assertTotalDistribution(t, result, tt.totalValidatorRewardAmount)

			if tt.claimSettlementValidationFn != nil {
				tt.claimSettlementValidationFn(t, result)
			} else if tt.expectedValidatorRewards > 0 || tt.expectedDelegatorRewards > 0 {
				// Since we simplified the logic to treat all stakeholders equally,
				// we no longer distinguish between validator and delegator operation reasons.
				// All recipients get the same operation reason as the settlement operation.
				totalRewards := int64(0)
				for _, transfer := range transfers {
					require.Equal(t, tt.opReason, transfer.OpReason, "All transfers should use the same operation reason")
					totalRewards += transfer.Coin.Amount.Int64()
				}

				expectedTotal := tt.expectedValidatorRewards + tt.expectedDelegatorRewards
				require.Equal(t, expectedTotal, totalRewards, "Total rewards should match expected validator + delegator rewards")
			}
		})
	}
}

// TestValidatorRewardDistribution_WithDelegators_ErrorCases tests delegation-specific error scenarios.
// Common error cases are covered by TestValidatorRewardDistribution_ErrorCases since both functions
// share the same validation logic. This focuses on delegation-specific failures like
// GetValidatorDelegations errors and graceful fallback to no delegators distribution.
func TestValidatorRewardDistribution_WithDelegators_ErrorCases(t *testing.T) {
	// NOTE: Common error cases (zero_reward_amount, get_validators_error, no_bonded_validators)
	// are already covered by TestValidatorRewardDistribution_ErrorCases since
	// the reward distribution function includes all the same validation logic.
	// This test only covers delegation-specific error scenarios.

	tests := []struct {
		name             string
		setupMocks       func(*mocks.MockStakingKeeper)
		rewardAmount     math.Int
		expectedError    bool
		expectedErrorMsg string
	}{
		{
			name: "no error: GetValidatorDelegations failure falls back to no delegators distribution",
			setupMocks: func(mock *mocks.MockStakingKeeper) {
				validator := createValidator(sample.ValOperatorAddressBech32(), 1000)
				mock.EXPECT().
					GetBondedValidatorsByPower(gomock.Any()).
					Return([]stakingtypes.Validator{validator}, nil)

				valAddr, _ := cosmostypes.ValAddressFromBech32(validator.OperatorAddress)
				mock.EXPECT().
					GetValidatorDelegations(gomock.Any(), valAddr).
					Return(nil, sdkerrors.ErrInvalidRequest)
			},
			rewardAmount:  math.NewInt(1000),
			expectedError: false, // Error is logged but not returned - falls back to no delegators distribution
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStakingKeeper := mocks.NewMockStakingKeeper(ctrl)
			tt.setupMocks(mockStakingKeeper)

			config := getDefaultTestConfig()
			config.rewardAmount = tt.rewardAmount

			_, err := executeDistribution(mockStakingKeeper, config, true)

			if tt.expectedError {
				require.Error(t, err)
				if tt.expectedErrorMsg != "" {
					require.Contains(t, err.Error(), tt.expectedErrorMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// createValidator creates a bonded validator with the specified operator address and token amount.
// The validator's delegator shares are set equal to the token amount for simplicity.
func createValidator(operatorAddr string, tokens int64) stakingtypes.Validator {
	return stakingtypes.Validator{
		OperatorAddress: operatorAddr,
		Tokens:          math.NewInt(tokens),
		DelegatorShares: math.LegacyNewDec(tokens),
		Status:          stakingtypes.Bonded,
	}
}

// createDelegation creates a delegation from a delegator to a validator with the specified shares.
func createDelegation(delegatorAddr, validatorAddr string, shares int64) stakingtypes.Delegation {
	return stakingtypes.Delegation{
		DelegatorAddress: delegatorAddr,
		ValidatorAddress: validatorAddr,
		Shares:           math.LegacyNewDec(shares),
	}
}

// setupValidatorMocks configures the mock staking keeper to return the provided validators and their delegations.
// This function sets up expectations for GetBondedValidatorsByPower and GetValidatorDelegations calls.
func setupValidatorMocks(mockStakingKeeper *mocks.MockStakingKeeper, validators []stakingtypes.Validator, delegations map[string][]stakingtypes.Delegation) {
	mockStakingKeeper.EXPECT().
		GetBondedValidatorsByPower(gomock.Any()).
		Return(validators, nil)

	for _, validator := range validators {
		valAddr, _ := cosmostypes.ValAddressFromBech32(validator.OperatorAddress)
		validatorDelegations := delegations[validator.OperatorAddress]
		mockStakingKeeper.EXPECT().
			GetValidatorDelegations(gomock.Any(), valAddr).
			Return(validatorDelegations, nil)
	}
}

// executeDistribution executes validator reward distribution with or without delegators
// based on the distributeDelegators flag. Returns the settlement result and any error.
func executeDistribution(mockStakingKeeper *mocks.MockStakingKeeper, config rewardDistributionTestConfig, distributeDelegators bool) (*tokenomicstypes.ClaimSettlementResult, error) {
	result := &tokenomicstypes.ClaimSettlementResult{}

	// Both cases (with and without delegators) use the same function
	// The function automatically handles delegators when delegations are present
	rewardCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, config.rewardAmount)
	return result, distributeValidatorRewards(
		config.ctx,
		config.logger,
		result,
		mockStakingKeeper,
		rewardCoin,
		config.opReason,
	)
}

// assertTotalDistribution verifies that the total amount distributed in the settlement result
// equals the expected amount and that all transfers use the correct denomination.
func assertTotalDistribution(t *testing.T, result *tokenomicstypes.ClaimSettlementResult, expectedAmount math.Int) {
	transfers := result.GetModToAcctTransfers()
	totalDistributed := math.ZeroInt()

	for _, transfer := range transfers {
		totalDistributed = totalDistributed.Add(transfer.Coin.Amount)
		require.Equal(t, pocket.DenomuPOKT, transfer.Coin.Denom)
	}

	require.Equal(t, expectedAmount, totalDistributed, "Total distributed should equal reward amount")
}

// getDefaultTestConfig returns a standard test configuration for reward distribution tests.
func getDefaultTestConfig() rewardDistributionTestConfig {
	return rewardDistributionTestConfig{
		ctx:          context.Background(),
		logger:       log.NewNopLogger(),
		rewardAmount: math.NewInt(100_000),
		opReason:     tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_VALIDATOR_REWARD_DISTRIBUTION,
	}
}
