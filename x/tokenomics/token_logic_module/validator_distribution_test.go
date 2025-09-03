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

// testExecutionConfig holds common configuration for reward distribution test execution.
type testExecutionConfig struct {
	ctx          context.Context
	logger       log.Logger
	rewardAmount math.Int
	opReason     tokenomicstypes.SettlementOpReason
}

// TestDistributeValidatorRewards tests the validator-only reward distribution functionality.
// It verifies proportional distribution based on validator stakes and precision handling
// with the Largest Remainder Method for edge cases.
func TestDistributeValidatorRewards(t *testing.T) {
	tests := []struct {
		name            string
		validatorStakes []int64
		rewardAmount    math.Int
		expectedCount   int
	}{
		{
			name:            "success: proportional distribution based on validator stakes",
			validatorStakes: []int64{700_000, 200_000, 100_000},
			rewardAmount:    math.NewInt(9240),
			expectedCount:   3,
		},
		{
			name:            "success: precision handling with Largest Remainder Method",
			validatorStakes: []int64{333, 333, 334},
			rewardAmount:    math.NewInt(100),
			expectedCount:   3,
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

			// Setup mocks for validator-only distribution
			mockStakingKeeper.EXPECT().
				GetBondedValidatorsByPower(gomock.Any()).
				Return(validators, nil)

			// Execute and validate
			config := getDefaultTestConfig()
			config.rewardAmount = tt.rewardAmount

			result, err := executeDistribution(mockStakingKeeper, config, false)
			require.NoError(t, err)

			transfers := result.GetModToAcctTransfers()
			require.Len(t, transfers, tt.expectedCount)

			assertTotalDistribution(t, result, tt.rewardAmount)

			// Verify all transfers are validator rewards
			for _, transfer := range transfers {
				require.Equal(t, tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_VALIDATOR_REWARD_DISTRIBUTION, transfer.OpReason)
			}
		})
	}
}

// TestDistributeValidatorRewards_ErrorCases tests error handling scenarios for validator reward distribution.
// It covers cases like zero reward amounts, staking keeper failures, missing validators, and zero stakes.
func TestDistributeValidatorRewards_ErrorCases(t *testing.T) {
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

// TestDistributeValidatorAndDelegatorRewards tests the combined validator and delegator reward distribution.
// It verifies correct distribution to both validators and their delegators based on delegation amounts,
// backward compatibility fallbacks, and precision handling for fractional distributions.
func TestDistributeValidatorAndDelegatorRewards(t *testing.T) {
	tests := []struct {
		name                     string
		validators               []stakingtypes.Validator
		delegations              map[string][]stakingtypes.Delegation
		rewardAmount             math.Int
		expectedTransferCount    int
		expectedValidatorRewards int64
		expectedDelegatorRewards int64
		opReason                 tokenomicstypes.SettlementOpReason
		validation               func(*testing.T, *tokenomicstypes.ClaimSettlementResult)
	}{
		{
			name: "success: mixed delegation amounts with equal self-bonded stakes",
			validators: []stakingtypes.Validator{
				createValidator(sample.ValOperatorAddressBech32(), 1_000_000),
				createValidator(sample.ValOperatorAddressBech32(), 600_000),
				createValidator(sample.ValOperatorAddressBech32(), 400_000),
			},
			rewardAmount:             math.NewInt(100_000),
			expectedTransferCount:    5, // 3 validators + 2 delegators
			expectedValidatorRewards: 60_000,
			expectedDelegatorRewards: 40_000,
			opReason:                 tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_VALIDATOR_REWARD_DISTRIBUTION,
		},
		{
			name: "success: equal self-bonded stakes receive equal rewards despite different total delegations",
			validators: []stakingtypes.Validator{
				createValidator(sample.ValOperatorAddressBech32(), 900_000),
				createValidator(sample.ValOperatorAddressBech32(), 600_000),
				createValidator(sample.ValOperatorAddressBech32(), 300_000),
			},
			rewardAmount:          math.NewInt(90_000),
			expectedTransferCount: 5, // 3 validators + 2 delegators
			opReason:              tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_VALIDATOR_REWARD_DISTRIBUTION,
			validation: func(t *testing.T, result *tokenomicstypes.ClaimSettlementResult) {
				// Verify equal validator rewards despite different total delegations
				transfers := result.GetModToAcctTransfers()
				validatorRewardMap := make(map[string]int64)
				for _, transfer := range transfers {
					if transfer.OpReason == tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_VALIDATOR_REWARD_DISTRIBUTION {
						validatorRewardMap[transfer.RecipientAddress] = transfer.Coin.Amount.Int64()
					}
				}
				// Each validator should get 15,000 (300k self-bonded out of 1.8M total)
				for _, reward := range validatorRewardMap {
					require.Equal(t, int64(15_000), reward)
				}
			},
		},
		{
			name: "success: backward compatibility when no delegations found (falls back to validator-only)",
			validators: []stakingtypes.Validator{
				createValidator(sample.ValOperatorAddressBech32(), 500_000),
				createValidator(sample.ValOperatorAddressBech32(), 300_000),
				createValidator(sample.ValOperatorAddressBech32(), 200_000),
			},
			rewardAmount:          math.NewInt(10_000),
			expectedTransferCount: 3, // validators only
			opReason:              tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_VALIDATOR_REWARD_DISTRIBUTION,
			validation: func(t *testing.T, result *tokenomicstypes.ClaimSettlementResult) {
				transfers := result.GetModToAcctTransfers()
				// All transfers should be validator rewards
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
			rewardAmount:          math.NewInt(100),
			expectedTransferCount: 6, // 3 validators + 3 delegators
			opReason:              tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_VALIDATOR_REWARD_DISTRIBUTION,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStakingKeeper := mocks.NewMockStakingKeeper(ctrl)

			// Create delegations inline based on test case
			delegations := make(map[string][]stakingtypes.Delegation)
			for i, validator := range tt.validators {
				valAddr, _ := cosmostypes.ValAddressFromBech32(validator.OperatorAddress)

				switch tt.name {
				case "success: mixed delegation amounts with equal self-bonded stakes":
					switch i {
					case 0: // Val1: 400k self + 600k delegated
						delegations[validator.OperatorAddress] = []stakingtypes.Delegation{
							createDelegation(cosmostypes.AccAddress(valAddr).String(), validator.OperatorAddress, 400_000),
							createDelegation(sample.AccAddressBech32(), validator.OperatorAddress, 600_000),
						}
					case 1: // Val2: 400k self + 200k delegated
						delegations[validator.OperatorAddress] = []stakingtypes.Delegation{
							createDelegation(cosmostypes.AccAddress(valAddr).String(), validator.OperatorAddress, 400_000),
							createDelegation(sample.AccAddressBech32(), validator.OperatorAddress, 200_000),
						}
					default: // Val3: 400k self only
						delegations[validator.OperatorAddress] = []stakingtypes.Delegation{
							createDelegation(cosmostypes.AccAddress(valAddr).String(), validator.OperatorAddress, 400_000),
						}
					}
				case "success: equal self-bonded stakes receive equal rewards despite different total delegations":
					switch i {
					case 0: // 300k self + 600k delegated
						delegations[validator.OperatorAddress] = []stakingtypes.Delegation{
							createDelegation(cosmostypes.AccAddress(valAddr).String(), validator.OperatorAddress, 300_000),
							createDelegation(sample.AccAddressBech32(), validator.OperatorAddress, 600_000),
						}
					case 1: // 300k self + 300k delegated
						delegations[validator.OperatorAddress] = []stakingtypes.Delegation{
							createDelegation(cosmostypes.AccAddress(valAddr).String(), validator.OperatorAddress, 300_000),
							createDelegation(sample.AccAddressBech32(), validator.OperatorAddress, 300_000),
						}
					default: // 300k self only
						delegations[validator.OperatorAddress] = []stakingtypes.Delegation{
							createDelegation(cosmostypes.AccAddress(valAddr).String(), validator.OperatorAddress, 300_000),
						}
					}
				case "success: backward compatibility when no delegations found (falls back to validator-only)":
					delegations[validator.OperatorAddress] = []stakingtypes.Delegation{} // Empty
				case "success: precision handling with Largest Remainder Method for fractional distributions":
					delegations[validator.OperatorAddress] = []stakingtypes.Delegation{
						createDelegation(cosmostypes.AccAddress(valAddr).String(), validator.OperatorAddress, validator.Tokens.Int64()*6/10), // 60%
						createDelegation(sample.AccAddressBech32(), validator.OperatorAddress, validator.Tokens.Int64()*4/10),                // 40%
					}
				}
			}

			setupValidatorMocks(mockStakingKeeper, tt.validators, delegations)

			execConfig := getDefaultTestConfig()
			execConfig.rewardAmount = tt.rewardAmount
			execConfig.opReason = tt.opReason

			result, err := executeDistribution(mockStakingKeeper, execConfig, true)
			require.NoError(t, err)

			transfers := result.GetModToAcctTransfers()
			require.Len(t, transfers, tt.expectedTransferCount)

			assertTotalDistribution(t, result, tt.rewardAmount)

			if tt.validation != nil {
				tt.validation(t, result)
			} else if tt.expectedValidatorRewards > 0 || tt.expectedDelegatorRewards > 0 {
				// Count validator vs delegator rewards
				validatorRewards := int64(0)
				delegatorRewards := int64(0)
				for _, transfer := range transfers {
					switch transfer.OpReason {
					case tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_VALIDATOR_REWARD_DISTRIBUTION,
						tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_VALIDATOR_REWARD_DISTRIBUTION:
						validatorRewards += transfer.Coin.Amount.Int64()
					case tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_DELEGATOR_REWARD_DISTRIBUTION,
						tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_DELEGATOR_REWARD_DISTRIBUTION:
						delegatorRewards += transfer.Coin.Amount.Int64()
					}
				}
				if tt.expectedValidatorRewards > 0 {
					require.Equal(t, tt.expectedValidatorRewards, validatorRewards)
				}
				if tt.expectedDelegatorRewards > 0 {
					require.Equal(t, tt.expectedDelegatorRewards, delegatorRewards)
				}
			}
		})
	}
}

// TestDistributeValidatorAndDelegatorRewards_ErrorCases tests delegation-specific error scenarios.
// Common error cases are covered by TestDistributeValidatorRewards_ErrorCases since both functions
// share the same validation logic. This focuses on delegation-specific failures like
// GetValidatorDelegations errors and graceful fallback to validator-only distribution.
func TestDistributeValidatorAndDelegatorRewards_ErrorCases(t *testing.T) {
	// NOTE: Common error cases (zero_reward_amount, get_validators_error, no_bonded_validators)
	// are already covered by TestDistributeValidatorRewards_ErrorCases since
	// distributeValidatorAndDelegatorRewards includes all the same validation logic.
	// This test only covers delegation-specific error scenarios.

	tests := []struct {
		name             string
		setupMocks       func(*mocks.MockStakingKeeper)
		rewardAmount     math.Int
		expectedError    bool
		expectedErrorMsg string
	}{
		{
			name: "no error: GetValidatorDelegations failure falls back to validator-only distribution",
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
			expectedError: false, // Error is logged but not returned - falls back to validator-only distribution
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

// executeDistribution executes either validator-only or validator+delegator reward distribution
// based on the distributeDelegators flag. Returns the settlement result and any error.
func executeDistribution(mockStakingKeeper *mocks.MockStakingKeeper, config testExecutionConfig, distributeDelegators bool) (*tokenomicstypes.ClaimSettlementResult, error) {
	result := &tokenomicstypes.ClaimSettlementResult{}

	if distributeDelegators {
		return result, distributeValidatorAndDelegatorRewards(
			config.ctx,
			config.logger,
			result,
			mockStakingKeeper,
			config.rewardAmount,
			config.opReason,
		)
	}

	return result, distributeValidatorRewards(
		config.ctx,
		config.logger,
		result,
		mockStakingKeeper,
		config.rewardAmount,
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
func getDefaultTestConfig() testExecutionConfig {
	return testExecutionConfig{
		ctx:          context.Background(),
		logger:       log.NewNopLogger(),
		rewardAmount: math.NewInt(100_000),
		opReason:     tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_VALIDATOR_REWARD_DISTRIBUTION,
	}
}
