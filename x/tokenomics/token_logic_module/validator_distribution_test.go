package token_logic_module

import (
	"context"
	"fmt"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/pokt-network/poktroll/app/pocket"
	testutilevents "github.com/pokt-network/poktroll/testutil/events"
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
				// We can distinguish validators from delegators by operation reason.
				// Since all validators have 300k self-bonded out of 1.8M total, each should get 15,000.
				transfers := result.GetModToAcctTransfers()

				// Count validators and delegators based on operation reason
				validatorCount := 0
				delegatorCount := 0
				for _, transfer := range transfers {
					switch transfer.OpReason {
					case tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_VALIDATOR_REWARD_DISTRIBUTION:
						validatorCount++
					case tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_DELEGATOR_REWARD_DISTRIBUTION:
						delegatorCount++
					default:
						t.Errorf("Unexpected operation reason: %v", transfer.OpReason)
					}
				}

				// Verify we have the expected number of validators and delegators
				require.Equal(t, 3, validatorCount, "Should have 3 validators")
				require.Equal(t, 2, delegatorCount, "Should have 2 delegators")
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
				// Verify that transfers use the correct operation reasons:
				// Validators get the base operation reason, delegators get the delegator-specific reason
				totalValidatorRewards := int64(0)
				totalDelegatorRewards := int64(0)

				var delegatorOpReason tokenomicstypes.SettlementOpReason
				switch tt.opReason {
				case tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_VALIDATOR_REWARD_DISTRIBUTION:
					delegatorOpReason = tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_DELEGATOR_REWARD_DISTRIBUTION
				case tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_VALIDATOR_REWARD_DISTRIBUTION:
					delegatorOpReason = tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_DELEGATOR_REWARD_DISTRIBUTION
				default:
					delegatorOpReason = tt.opReason
				}

				for _, transfer := range transfers {
					switch transfer.OpReason {
					case tt.opReason:
						totalValidatorRewards += transfer.Coin.Amount.Int64()
					case delegatorOpReason:
						totalDelegatorRewards += transfer.Coin.Amount.Int64()
					default:
						t.Errorf("Unexpected operation reason: %v", transfer.OpReason)
					}
				}

				require.Equal(t, tt.expectedValidatorRewards, totalValidatorRewards, "Validator rewards should match expected")
				require.Equal(t, tt.expectedDelegatorRewards, totalDelegatorRewards, "Delegator rewards should match expected")
			}
		})
	}
}

// TestValidatorRewardDistribution_MultiValidatorDelegator tests the scenario where a single
// delegator has delegations to more than one validator simultaneously.
//
// In Cosmos SDK staking, a delegator may stake tokens to multiple validators. Each call to
// GetValidatorDelegations returns delegations for exactly one validator, so a cross-validator
// delegator is processed once per validator's post-commission remainder distribution. The
// per-recipient reward accumulator (addReward) must merge these into a single transfer so the
// delegator is rewarded for their full network-wide stake, not just one validator's slice.
func TestValidatorRewardDistribution_MultiValidatorDelegator(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Delegator D stakes 200k to Val1 and 100k to Val2.
	// Val1 self-bonds 100k; Val2 self-bonds 100k.
	// Total bonded: 100k + 200k + 100k + 100k = 500k
	// D's correct total stake: 200k + 100k = 300k  (60% of total)
	// Val1 self-stake: 100k (20% of total)
	// Val2 self-stake: 100k (20% of total)
	//
	// Total reward: 500k tokens → each token earns 1 reward token for easy verification.
	// Correct distribution:
	//   D:    500k × 300k/500k = 300k
	//   Val1: 500k × 100k/500k = 100k
	//   Val2: 500k × 100k/500k = 100k
	const (
		val1SelfStake   = int64(100_000) // Val1 self-delegation
		val1TotalTokens = int64(300_000) // Val1 total = self + D's delegation to Val1
		val2SelfStake   = int64(100_000) // Val2 self-delegation
		val2TotalTokens = int64(200_000) // Val2 total = self + D's delegation to Val2
		dToVal1         = int64(200_000) // D's delegation to Val1
		dToVal2         = int64(100_000) // D's delegation to Val2
		totalReward     = int64(500_000) // chosen so results are round numbers
	)
	totalBonded := val1TotalTokens + val2TotalTokens // 500k

	mockStakingKeeper := mocks.NewMockStakingKeeper(ctrl)

	val1 := createValidator(sample.ValOperatorAddressBech32(), val1TotalTokens)
	val2 := createValidator(sample.ValOperatorAddressBech32(), val2TotalTokens)

	valAddr1, _ := cosmostypes.ValAddressFromBech32(val1.OperatorAddress)
	valAddr2, _ := cosmostypes.ValAddressFromBech32(val2.OperatorAddress)

	val1AccAddr := cosmostypes.AccAddress(valAddr1).String()
	val2AccAddr := cosmostypes.AccAddress(valAddr2).String()

	// Shared delegator D address
	delegatorDAddr := sample.AccAddressBech32()

	mockStakingKeeper.EXPECT().
		GetBondedValidatorsByPower(gomock.Any()).
		Return([]stakingtypes.Validator{val1, val2}, nil)

	// Val1's delegations: self + D
	mockStakingKeeper.EXPECT().
		GetValidatorDelegations(gomock.Any(), valAddr1).
		Return([]stakingtypes.Delegation{
			createDelegation(val1AccAddr, val1.OperatorAddress, val1SelfStake),
			createDelegation(delegatorDAddr, val1.OperatorAddress, dToVal1),
		}, nil)

	// Val2's delegations: self + D (same D, different validator)
	mockStakingKeeper.EXPECT().
		GetValidatorDelegations(gomock.Any(), valAddr2).
		Return([]stakingtypes.Delegation{
			createDelegation(val2AccAddr, val2.OperatorAddress, val2SelfStake),
			createDelegation(delegatorDAddr, val2.OperatorAddress, dToVal2),
		}, nil)

	config := getDefaultTestConfig()
	config.rewardAmount = math.NewInt(totalReward)
	config.opReason = tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_VALIDATOR_REWARD_DISTRIBUTION

	result, err := executeDistribution(mockStakingKeeper, config, true)
	require.NoError(t, err)

	transfers := result.GetModToAcctTransfers()

	// 3 recipients: Val1 self, Val2 self, and delegator D (appears in both validators but must be merged)
	require.Len(t, transfers, 3, "Expected 3 transfers: 2 validators + 1 merged delegator")

	// Build a map of recipient → amount for easy lookup
	rewardMap := make(map[string]int64, len(transfers))
	for _, transfer := range transfers {
		rewardMap[transfer.RecipientAddress] += transfer.Coin.Amount.Int64()
	}

	// Verify total distribution is correct
	assertTotalDistribution(t, result, math.NewInt(totalReward))

	// D's correct stake fraction = 300k/500k = 60% → reward = 300k
	dReward, dFound := rewardMap[delegatorDAddr]
	require.True(t, dFound, "Delegator D must appear exactly once in transfers")
	require.Equal(t, int64(300_000), dReward,
		"Delegator D should receive 300k (200k from Val1 + 100k from Val2 = 300k stake, 60%% of %d total bonded)",
		totalBonded,
	)

	// Val1 self-stake fraction = 100k/500k = 20% → reward = 100k
	val1Reward, val1Found := rewardMap[val1AccAddr]
	require.True(t, val1Found, "Val1 self must appear in transfers")
	require.Equal(t, int64(100_000), val1Reward, "Val1 should receive 100k (20%% of total)")

	// Val2 self-stake fraction = 100k/500k = 20% → reward = 100k
	val2Reward, val2Found := rewardMap[val2AccAddr]
	require.True(t, val2Found, "Val2 self must appear in transfers")
	require.Equal(t, int64(100_000), val2Reward, "Val2 should receive 100k (20%% of total)")
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
// TestBuildValidatorStakes_DuplicateAccAddressIsSkipped pins the defense-in-depth
// guard at the AccAddress dedupe step of buildValidatorStakes. Two validators
// sharing the same operator address (consensus-impossible in production, since
// the staking module keys by OperatorAddress) must NOT silently overwrite each
// other in the stake map — the second occurrence is skipped and its tokens are
// NOT counted in totalValidatorStake.
func TestBuildValidatorStakes_DuplicateAccAddressIsSkipped(t *testing.T) {
	const duplicateValBondedTokens = int64(1_000_000)
	const uniqueValBondedTokens = int64(500_000)

	duplicateOperator := sample.ValOperatorAddressBech32()
	uniqueOperator := sample.ValOperatorAddressBech32()

	// Same OperatorAddress on both validators → identical AccAddress derivation.
	// Mirrors the "duplicate underlying bytes" hypothetical the audit's H3 flags.
	validators := []stakingtypes.Validator{
		createValidator(duplicateOperator, duplicateValBondedTokens),
		createValidator(duplicateOperator, duplicateValBondedTokens), // exact duplicate
		createValidator(uniqueOperator, uniqueValBondedTokens),
	}

	entries, stakeMap, totalStake := buildValidatorStakes(log.NewNopLogger(), validators)

	// Two unique entries — the duplicate's second occurrence was dropped.
	require.Len(t, entries, 2, "duplicate accAddr must be skipped, not merged or counted twice")
	require.Len(t, stakeMap, 2)

	// Total reflects ONLY the unique-addressed contributions, NOT 2× the duplicate.
	expectedTotal := math.NewInt(duplicateValBondedTokens + uniqueValBondedTokens)
	require.Equal(t, expectedTotal, totalStake,
		"duplicate validator tokens must not double-count toward the Level-1 LRM denominator")
}

func createValidator(operatorAddr string, tokens int64) stakingtypes.Validator {
	return stakingtypes.Validator{
		OperatorAddress: operatorAddr,
		Tokens:          math.NewInt(tokens),
		DelegatorShares: math.LegacyNewDec(tokens),
		Status:          stakingtypes.Bonded,
	}
}

// createValidatorWithCommission creates a bonded validator with the specified operator
// address, token amount, and commission rate (e.g. 0.10 for 10%).
func createValidatorWithCommission(operatorAddr string, tokens int64, commissionRate math.LegacyDec) stakingtypes.Validator {
	validator := createValidator(operatorAddr, tokens)
	validator.Commission = stakingtypes.Commission{
		CommissionRates: stakingtypes.CommissionRates{Rate: commissionRate},
	}
	return validator
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
	return result, DistributeValidatorRewards(
		config.ctx,
		config.logger,
		result,
		mockStakingKeeper,
		rewardCoin,
		config.opReason,
		int64(1), // sessionEndHeight (test value; only surfaced in the emitted event)
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
	// Use an sdk.Context (not context.Background) so EmitTypedEvent works: the
	// per-validator EventValidatorRewardDistribution is emitted via the event manager.
	sdkCtx := cosmostypes.Context{}.
		WithContext(context.Background()).
		WithEventManager(cosmostypes.NewEventManager())

	return rewardDistributionTestConfig{
		ctx:          sdkCtx,
		logger:       log.NewNopLogger(),
		rewardAmount: math.NewInt(100_000),
		opReason:     tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_VALIDATOR_REWARD_DISTRIBUTION,
	}
}

// TestValidatorRewardDistribution_Commission verifies that the validator commission rate
// is carved out of each validator's pool share before the remainder is distributed to
// delegators, mirroring the Cosmos consensus-reward model.
func TestValidatorRewardDistribution_Commission(t *testing.T) {
	tests := []struct {
		name           string
		selfStake      int64
		delegatorStake int64
		commissionRate math.LegacyDec
		rewardAmount   int64
		// Expected reward to the validator operator account (commission + self-delegation slice).
		expectedValidatorReward int64
		// Expected reward to the (single) external delegator.
		expectedDelegatorReward int64
		// Expected EventValidatorRewardDistribution field values.
		expectedCommission   int64
		expectedSelfDelegate int64
		expectedDelegators   int64
	}{
		{
			name:                    "10% commission with delegator",
			selfStake:               100_000,
			delegatorStake:          900_000,
			commissionRate:          math.LegacyNewDecWithPrec(10, 2), // 0.10
			rewardAmount:            100_000,
			expectedValidatorReward: 19_000, // 10k commission + 90k×(100k/1M)=9k self
			expectedDelegatorReward: 81_000, // 90k×(900k/1M)
			expectedCommission:      10_000,
			expectedSelfDelegate:    9_000,
			expectedDelegators:      81_000,
		},
		{
			name:                    "zero commission passes everything through by stake",
			selfStake:               100_000,
			delegatorStake:          900_000,
			commissionRate:          math.LegacyZeroDec(),
			rewardAmount:            100_000,
			expectedValidatorReward: 10_000, // 0 commission + 100k×(100k/1M)=10k self
			expectedDelegatorReward: 90_000, // 100k×(900k/1M)
			expectedCommission:      0,
			expectedSelfDelegate:    10_000,
			expectedDelegators:      90_000,
		},
		{
			name:                    "50% commission",
			selfStake:               200_000,
			delegatorStake:          800_000,
			commissionRate:          math.LegacyNewDecWithPrec(50, 2), // 0.50
			rewardAmount:            100_000,
			expectedValidatorReward: 60_000, // 50k commission + 50k×(200k/1M)=10k self
			expectedDelegatorReward: 40_000, // 50k×(800k/1M)
			expectedCommission:      50_000,
			expectedSelfDelegate:    10_000,
			expectedDelegators:      40_000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStakingKeeper := mocks.NewMockStakingKeeper(ctrl)

			totalStake := tt.selfStake + tt.delegatorStake
			valOperAddr := sample.ValOperatorAddressBech32()
			validator := createValidatorWithCommission(valOperAddr, totalStake, tt.commissionRate)

			valAddr, err := cosmostypes.ValAddressFromBech32(valOperAddr)
			require.NoError(t, err)
			validatorAccAddr := cosmostypes.AccAddress(valAddr).String()
			delegatorAddr := sample.AccAddressBech32()

			delegations := map[string][]stakingtypes.Delegation{
				valOperAddr: {
					createDelegation(validatorAccAddr, valOperAddr, tt.selfStake),
					createDelegation(delegatorAddr, valOperAddr, tt.delegatorStake),
				},
			}
			setupValidatorMocks(mockStakingKeeper, []stakingtypes.Validator{validator}, delegations)

			config := getDefaultTestConfig()
			config.rewardAmount = math.NewInt(tt.rewardAmount)
			config.opReason = tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_VALIDATOR_REWARD_DISTRIBUTION

			result, err := executeDistribution(mockStakingKeeper, config, true)
			require.NoError(t, err)

			// Full reward must be distributed with no dust.
			assertTotalDistribution(t, result, math.NewInt(tt.rewardAmount))

			var validatorReward, delegatorReward int64
			for _, transfer := range result.GetModToAcctTransfers() {
				switch transfer.RecipientAddress {
				case validatorAccAddr:
					validatorReward += transfer.Coin.Amount.Int64()
					require.Equal(t,
						tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_VALIDATOR_REWARD_DISTRIBUTION,
						transfer.OpReason, "validator transfer must use the validator op reason")
				case delegatorAddr:
					delegatorReward += transfer.Coin.Amount.Int64()
					require.Equal(t,
						tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_DELEGATOR_REWARD_DISTRIBUTION,
						transfer.OpReason, "delegator transfer must use the delegator op reason")
				default:
					t.Fatalf("unexpected recipient: %s", transfer.RecipientAddress)
				}
			}

			require.Equal(t, tt.expectedValidatorReward, validatorReward, "validator reward (commission + self) mismatch")
			require.Equal(t, tt.expectedDelegatorReward, delegatorReward, "delegator reward mismatch")

			// Verify the per-validator summary event carries the commission breakdown.
			events := cosmostypes.UnwrapSDKContext(config.ctx).EventManager().Events()
			rewardEvents := testutilevents.FilterEvents[*tokenomicstypes.EventValidatorRewardDistribution](t, events)
			require.Len(t, rewardEvents, 1, "exactly one per-validator reward event expected")

			ev := rewardEvents[0]
			require.Equal(t, valOperAddr, ev.ValidatorOperatorAddress)
			require.Equal(t, validatorAccAddr, ev.ValidatorAccountAddress)
			require.Equal(t, tt.commissionRate.String(), ev.CommissionRate)
			require.Equal(t, fmt.Sprintf("%d", tt.rewardAmount), ev.PoolShareUpokt, "single validator → pool share is the full reward")
			require.Equal(t, fmt.Sprintf("%d", tt.expectedCommission), ev.CommissionUpokt)
			require.Equal(t, fmt.Sprintf("%d", tt.expectedSelfDelegate), ev.SelfDelegationRewardUpokt)
			require.Equal(t, fmt.Sprintf("%d", tt.expectedDelegators), ev.DelegatorsRewardUpokt)
			require.Equal(t, fmt.Sprintf("%d", totalStake), ev.TotalDelegatedStakeUpokt)
			require.Equal(t, uint32(1), ev.NumDelegators, "one external delegator")
		})
	}
}

// TestValidatorRewardDistribution_FullCommission covers the upper-boundary
// 100% commission case as a standalone test (NOT a table entry of
// TestValidatorRewardDistribution_Commission) because the production code
// short-circuits the remainder path before calling GetValidatorDelegations.
// The table test's shared setupValidatorMocks unconditionally expects that
// call, which would fail the gomock controller for this case.
//
// Distribution semantics under 100% commission:
//   - commission = poolShare (all of it)
//   - remainder = poolShare - commission = 0
//   - The !remainder.IsPositive() branch in distributeRewardsToValidatorsAndDelegators
//     emits the per-validator event with selfDelegationReward = 0,
//     delegatorsReward = 0, totalDelegatedStake = 0, numDelegators = 0, and
//     `continue`s WITHOUT consulting the delegations index.
//
// This guards against accidental division-by-zero or 'commission > pool'
// regressions at the upper boundary and pins the event shape indexers will
// see for full-commission validators.
func TestValidatorRewardDistribution_FullCommission(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStakingKeeper := mocks.NewMockStakingKeeper(ctrl)

	const (
		totalStake   int64 = 1_000_000
		rewardAmount int64 = 100_000
	)

	valOperAddr := sample.ValOperatorAddressBech32()
	validator := createValidatorWithCommission(valOperAddr, totalStake, math.LegacyOneDec())

	// Only GetBondedValidatorsByPower is expected — NOT GetValidatorDelegations.
	// The 100% commission short-circuit returns before consulting delegations.
	mockStakingKeeper.EXPECT().
		GetBondedValidatorsByPower(gomock.Any()).
		Return([]stakingtypes.Validator{validator}, nil)

	valAddr, err := cosmostypes.ValAddressFromBech32(valOperAddr)
	require.NoError(t, err)
	validatorAccAddr := cosmostypes.AccAddress(valAddr).String()

	config := getDefaultTestConfig()
	config.rewardAmount = math.NewInt(rewardAmount)
	config.opReason = tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_VALIDATOR_REWARD_DISTRIBUTION

	result, err := executeDistribution(mockStakingKeeper, config, true)
	require.NoError(t, err)

	// All of the reward must reach the validator account; nothing leaks elsewhere.
	assertTotalDistribution(t, result, math.NewInt(rewardAmount))

	transfers := result.GetModToAcctTransfers()
	require.Len(t, transfers, 1,
		"100%% commission with single validator → exactly one transfer (commission → validator account)")
	require.Equal(t, validatorAccAddr, transfers[0].RecipientAddress,
		"sole recipient must be the validator account")
	require.Equal(t, int64(rewardAmount), transfers[0].Coin.Amount.Int64(),
		"validator account receives the full reward amount as commission")
	require.Equal(t,
		tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_VALIDATOR_REWARD_DISTRIBUTION,
		transfers[0].OpReason,
		"transfer must carry the validator op_reason — there is no delegator transfer to bucket under DELEGATOR")

	// Verify the per-validator summary event reflects the full-commission shape.
	events := cosmostypes.UnwrapSDKContext(config.ctx).EventManager().Events()
	rewardEvents := testutilevents.FilterEvents[*tokenomicstypes.EventValidatorRewardDistribution](t, events)
	require.Len(t, rewardEvents, 1, "exactly one per-validator reward event expected")

	ev := rewardEvents[0]
	require.Equal(t, valOperAddr, ev.ValidatorOperatorAddress)
	require.Equal(t, validatorAccAddr, ev.ValidatorAccountAddress)
	require.Equal(t, math.LegacyOneDec().String(), ev.CommissionRate, "commission rate must serialize as the canonical OneDec string")
	require.Equal(t, fmt.Sprintf("%d", rewardAmount), ev.PoolShareUpokt, "single validator → pool share is the full reward")
	require.Equal(t, fmt.Sprintf("%d", rewardAmount), ev.CommissionUpokt, "100%% commission → commission equals pool share")
	require.Equal(t, "0", ev.SelfDelegationRewardUpokt, "no remainder after 100%% commission → self-delegation reward is zero")
	require.Equal(t, "0", ev.DelegatorsRewardUpokt, "no remainder after 100%% commission → delegators reward is zero")
	require.Equal(t, "0", ev.TotalDelegatedStakeUpokt,
		"short-circuit path reports totalDelegatedStake=0 because the delegations index is not consulted")
	require.Equal(t, uint32(0), ev.NumDelegators,
		"short-circuit path reports numDelegators=0 because the delegations index is not consulted")
}

// TestValidatorRewardDistribution_CrossDelegatorWithCommission covers the
// scenario the audit flagged as P2 (cross-delegation bucketing): when the
// SAME pokt account is both validator A's operator account AND a delegator
// on validator B, the bank-batch accumulator buckets A's combined income
// (commission + self-delegation from A's own pool + delegator-side slice
// from B's pool) under the VALIDATOR op_reason. The per-validator
// EventValidatorRewardDistribution still reports the correct breakdown.
//
// This test pins the behavior documented in the
// EventValidatorRewardDistribution proto's CROSS-DELEGATION ACCOUNTING
// NOTE: indexers building "VALIDATOR vs DELEGATOR" totals must sum from
// the per-validator event (which correctly separates commission /
// self-delegation / external-delegator income) rather than from
// EventSettlementBatch alone.
//
// Setup:
//   - Validator A: Tokens=1M, self-delegation 1M, commission 10%.
//   - Validator B: Tokens=1M, self-delegation 800k, A delegates 200k.
//     B's commission 10%.
//   - reward = 200k, evenly split (both have equal stake).
//
// Math:
//   - poolShare_A = poolShare_B = 100k.
//   - A's pool: commission_A=10k, remainder=90k, A-self=1M sole delegator
//     → A gets 90k self → A account += 10k + 90k = 100k from A's pool.
//   - B's pool: commission_B=10k, remainder=90k, delegations [B-self=800k,
//     A=200k]. B-self gets 90k×(800k/1M) = 72k; A gets 90k×(200k/1M) = 18k.
//     B account += 10k + 72k = 82k from B's pool. A account += 18k from
//     B's pool (as delegator).
//   - Total to A: 100k + 18k = 118k. Total to B: 82k. Pool: 200k. No dust.
//
// Bucketing assertion:
//   - The transfer to A's account is tagged VALIDATOR op_reason for the
//     ENTIRE 118k (NOT split into 100k VALIDATOR + 18k DELEGATOR). This is
//     because the transfer accumulator keys on recipient ∈ validatorAccAddresses,
//     not on the source of each contribution. The per-validator event for
//     B still correctly reports delegatorsReward=18k (with A as the sole
//     external delegator), so indexers retain the breakdown.
func TestValidatorRewardDistribution_CrossDelegatorWithCommission(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStakingKeeper := mocks.NewMockStakingKeeper(ctrl)

	const (
		totalValidatorStake int64 = 1_000_000

		// Validator B's delegation split.
		bSelfStake  int64 = 800_000
		aCrossStake int64 = 200_000 // A delegates this on B

		rewardAmount int64 = 200_000

		// Pre-computed expected amounts (see the doc-block math above).
		expectedCommission     int64 = 10_000
		expectedASelfFromAPool int64 = 90_000           // A's full 1M self-delegation, sole on A
		expectedBSelfFromBPool int64 = 72_000           // 90k × (800k/1M)
		expectedAFromBPool     int64 = 18_000           // 90k × (200k/1M)
		expectedATotalReceived int64 = 100_000 + 18_000 // 118k
		expectedBTotalReceived int64 = 82_000           // 10k + 72k
	)
	commissionRate := math.LegacyNewDecWithPrec(10, 2) // 0.10

	// --- Validators -----------------------------------------------------------
	aOperAddr := sample.ValOperatorAddressBech32()
	bOperAddr := sample.ValOperatorAddressBech32()
	validatorA := createValidatorWithCommission(aOperAddr, totalValidatorStake, commissionRate)
	validatorB := createValidatorWithCommission(bOperAddr, totalValidatorStake, commissionRate)

	aValAddr, err := cosmostypes.ValAddressFromBech32(aOperAddr)
	require.NoError(t, err)
	bValAddr, err := cosmostypes.ValAddressFromBech32(bOperAddr)
	require.NoError(t, err)
	aAccAddr := cosmostypes.AccAddress(aValAddr).String()
	bAccAddr := cosmostypes.AccAddress(bValAddr).String()

	// Cross-delegation: A delegates aCrossStake on B in addition to A's full
	// self-delegation on itself.
	delegations := map[string][]stakingtypes.Delegation{
		aOperAddr: {
			createDelegation(aAccAddr, aOperAddr, totalValidatorStake),
		},
		bOperAddr: {
			createDelegation(bAccAddr, bOperAddr, bSelfStake),
			createDelegation(aAccAddr, bOperAddr, aCrossStake),
		},
	}
	setupValidatorMocks(mockStakingKeeper, []stakingtypes.Validator{validatorA, validatorB}, delegations)

	// --- Execute distribution -------------------------------------------------
	config := getDefaultTestConfig()
	config.rewardAmount = math.NewInt(rewardAmount)
	config.opReason = tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_VALIDATOR_REWARD_DISTRIBUTION

	result, err := executeDistribution(mockStakingKeeper, config, true)
	require.NoError(t, err)

	assertTotalDistribution(t, result, math.NewInt(rewardAmount))

	// --- Tally transfers by recipient ----------------------------------------
	transfersByRecipient := make(map[string]int64)
	opReasonByRecipient := make(map[string]tokenomicstypes.SettlementOpReason)
	for _, transfer := range result.GetModToAcctTransfers() {
		transfersByRecipient[transfer.RecipientAddress] += transfer.Coin.Amount.Int64()
		// All transfers to a given recipient should share the same op_reason
		// (the bucketing collapses by recipient address); the loop captures
		// the last-seen and the post-loop assertion verifies consistency.
		opReasonByRecipient[transfer.RecipientAddress] = transfer.OpReason
	}

	require.Equal(t, expectedATotalReceived, transfersByRecipient[aAccAddr],
		"A's account must receive: commission_A + self-delegation_A + delegator-on-B slice = 10k + 90k + 18k = 118k")
	require.Equal(t, expectedBTotalReceived, transfersByRecipient[bAccAddr],
		"B's account must receive: commission_B + self-delegation_B = 10k + 72k = 82k")

	// --- Bucketing assertion (P2 audit finding) -------------------------------
	// Even though 18k of A's 118k came from a DELEGATOR-side source (A's stake
	// on validator B), the transfer is tagged VALIDATOR because A is in
	// validatorAccAddresses. This is the documented behavior — indexers
	// summing VALIDATOR vs DELEGATOR from EventSettlementBatch ALONE would
	// over-count A under VALIDATOR.
	require.Equal(t,
		tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_VALIDATOR_REWARD_DISTRIBUTION,
		opReasonByRecipient[aAccAddr],
		"A's combined transfer (validator + delegator income) MUST bucket under VALIDATOR — documented in EventValidatorRewardDistribution proto comment")
	require.Equal(t,
		tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_VALIDATOR_REWARD_DISTRIBUTION,
		opReasonByRecipient[bAccAddr],
		"B's transfer is purely validator income and is tagged VALIDATOR")

	// No DELEGATOR-tagged transfer for A is expected — the cross-delegation
	// slice is absorbed under VALIDATOR. This is the bucketing audit finding.
	for _, transfer := range result.GetModToAcctTransfers() {
		if transfer.RecipientAddress == aAccAddr {
			require.NotEqual(t,
				tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_DELEGATOR_REWARD_DISTRIBUTION,
				transfer.OpReason,
				"no transfer to A should be tagged DELEGATOR — the cross-delegation slice is absorbed under VALIDATOR")
		}
	}

	// --- Per-validator events preserve the breakdown indexers need ------------
	// The EventValidatorRewardDistribution events are the source of truth for
	// indexers building cross-delegation-aware totals (per the proto note).
	events := cosmostypes.UnwrapSDKContext(config.ctx).EventManager().Events()
	rewardEvents := testutilevents.FilterEvents[*tokenomicstypes.EventValidatorRewardDistribution](t, events)
	require.Len(t, rewardEvents, 2, "exactly one per-validator reward event per validator")

	// Locate the events by validator operator address (emission order is not
	// guaranteed to match construction order).
	var evA, evB *tokenomicstypes.EventValidatorRewardDistribution
	for _, ev := range rewardEvents {
		switch ev.ValidatorOperatorAddress {
		case aOperAddr:
			evA = ev
		case bOperAddr:
			evB = ev
		}
	}
	require.NotNil(t, evA, "expected EventValidatorRewardDistribution for validator A")
	require.NotNil(t, evB, "expected EventValidatorRewardDistribution for validator B")

	// Validator A: sole self-delegator, no external delegators.
	require.Equal(t, fmt.Sprintf("%d", expectedCommission), evA.CommissionUpokt)
	require.Equal(t, fmt.Sprintf("%d", expectedASelfFromAPool), evA.SelfDelegationRewardUpokt)
	require.Equal(t, "0", evA.DelegatorsRewardUpokt, "validator A has no external delegators")
	require.Equal(t, uint32(0), evA.NumDelegators)

	// Validator B: one external delegator (A) with 200k of the 1M stake.
	require.Equal(t, fmt.Sprintf("%d", expectedCommission), evB.CommissionUpokt)
	require.Equal(t, fmt.Sprintf("%d", expectedBSelfFromBPool), evB.SelfDelegationRewardUpokt)
	require.Equal(t, fmt.Sprintf("%d", expectedAFromBPool), evB.DelegatorsRewardUpokt,
		"validator B's external delegator slice — this is the cross-delegation income that gets bucketed under VALIDATOR for A in EventSettlementBatch")
	require.Equal(t, uint32(1), evB.NumDelegators, "A is the sole external delegator on B")
	require.Equal(t, fmt.Sprintf("%d", totalValidatorStake), evB.TotalDelegatedStakeUpokt,
		"B's total delegated stake = 800k self + 200k A = 1M")
}
