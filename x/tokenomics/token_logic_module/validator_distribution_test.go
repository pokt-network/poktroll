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

func TestDistributeValidatorRewards_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock staking keeper
	mockStakingKeeper := mocks.NewMockStakingKeeper(ctrl)

	// Define stake proportions for test validators
	totalStake := int64(1_000_000)
	val1Proportion := 0.7 // 70%
	val2Proportion := 0.2 // 20%
	val3Proportion := 0.1 // 10%

	// Create test validators with different stakes based on proportions
	validator1 := stakingtypes.Validator{
		OperatorAddress: sample.ValOperatorAddressBech32(),
		Tokens:          math.NewInt(int64(float64(totalStake) * val1Proportion)),
		Status:          stakingtypes.Bonded,
	}
	validator2 := stakingtypes.Validator{
		OperatorAddress: sample.ValOperatorAddressBech32(),
		Tokens:          math.NewInt(int64(float64(totalStake) * val2Proportion)),
		Status:          stakingtypes.Bonded,
	}
	validator3 := stakingtypes.Validator{
		OperatorAddress: sample.ValOperatorAddressBech32(),
		Tokens:          math.NewInt(int64(float64(totalStake) * val3Proportion)),
		Status:          stakingtypes.Bonded,
	}

	validators := []stakingtypes.Validator{validator1, validator2, validator3}

	// Set up mock expectations
	mockStakingKeeper.EXPECT().
		GetBondedValidatorsByPower(gomock.Any()).
		Return(validators, nil).
		Times(1)

	// Create test context and result
	ctx := context.Background()
	logger := log.NewNopLogger()
	result := &tokenomicstypes.ClaimSettlementResult{}
	rewardAmount := math.NewInt(9240) // 9,240 uPOKT to distribute

	// Execute the function
	err := distributeValidatorRewards(
		ctx,
		logger,
		result,
		mockStakingKeeper,
		rewardAmount,
		tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_VALIDATOR_REWARD_DISTRIBUTION,
	)

	// Validate results
	require.NoError(t, err)

	// Check that transfers were created for each validator
	transfers := result.GetModToAcctTransfers()
	require.Len(t, transfers, 3, "Expected 3 validator reward transfers")

	// Validate total distribution equals reward amount
	totalDistributed := math.ZeroInt()
	for _, transfer := range transfers {
		totalDistributed = totalDistributed.Add(transfer.Coin.Amount)
		require.Equal(t, pocket.DenomuPOKT, transfer.Coin.Denom)
		require.Equal(t, tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_VALIDATOR_REWARD_DISTRIBUTION, transfer.OpReason)
	}

	require.Equal(t, rewardAmount, totalDistributed, "Total distributed should equal reward amount")

	// Calculate expected rewards using the same proportions
	expectedVal1Reward := int64(float64(rewardAmount.Int64()) * val1Proportion)
	expectedVal2Reward := int64(float64(rewardAmount.Int64()) * val2Proportion)
	expectedVal3Reward := int64(float64(rewardAmount.Int64()) * val3Proportion)

	// Find transfers by validator address
	var val1Transfer, val2Transfer, val3Transfer *tokenomicstypes.ModToAcctTransfer
	for _, transfer := range transfers {
		switch transfer.RecipientAddress {
		case cosmostypes.AccAddress(cosmostypes.MustValAddressFromBech32(validator1.OperatorAddress)).String():
			val1Transfer = &transfer
		case cosmostypes.AccAddress(cosmostypes.MustValAddressFromBech32(validator2.OperatorAddress)).String():
			val2Transfer = &transfer
		case cosmostypes.AccAddress(cosmostypes.MustValAddressFromBech32(validator3.OperatorAddress)).String():
			val3Transfer = &transfer
		}
	}

	require.NotNil(t, val1Transfer, "Validator 1 should receive reward")
	require.NotNil(t, val2Transfer, "Validator 2 should receive reward")
	require.NotNil(t, val3Transfer, "Validator 3 should receive reward")

	// Validator 1 should get expectedVal1Reward
	require.Equal(t, expectedVal1Reward, val1Transfer.Coin.Amount.Int64())
	// Validator 2 should get expectedVal2Reward
	require.Equal(t, expectedVal2Reward, val2Transfer.Coin.Amount.Int64())
	// Validator 3 should get expectedVal3Reward
	require.Equal(t, expectedVal3Reward, val3Transfer.Coin.Amount.Int64())
}

func TestDistributeValidatorRewards_ErrorCases(t *testing.T) {
	tests := []struct {
		name             string
		setupMocks       func(*mocks.MockStakingKeeper)
		rewardAmount     math.Int
		expectedError    bool
		expectedErrorMsg string
	}{
		{
			name: "zero_reward_amount",
			setupMocks: func(mock *mocks.MockStakingKeeper) {
				// No expectations needed for zero amount
			},
			rewardAmount:  math.ZeroInt(),
			expectedError: false, // Should return nil without error
		},
		{
			name: "staking_keeper_error",
			setupMocks: func(mock *mocks.MockStakingKeeper) {
				mock.EXPECT().
					GetBondedValidatorsByPower(gomock.Any()).
					Return(nil, sdkerrors.ErrInvalidAddress).
					Times(1)
			},
			rewardAmount:     math.NewInt(1000),
			expectedError:    true,
			expectedErrorMsg: "failed to get bonded validators",
		},
		{
			name: "no_bonded_validators",
			setupMocks: func(mock *mocks.MockStakingKeeper) {
				mock.EXPECT().
					GetBondedValidatorsByPower(gomock.Any()).
					Return([]stakingtypes.Validator{}, nil).
					Times(1)
			},
			rewardAmount:  math.NewInt(1000),
			expectedError: false, // Should handle gracefully
		},
		{
			name: "validators_with_zero_stake",
			setupMocks: func(mock *mocks.MockStakingKeeper) {
				validators := []stakingtypes.Validator{
					{
						OperatorAddress: sample.ValOperatorAddressBech32(),
						Tokens:          math.ZeroInt(),
						Status:          stakingtypes.Bonded,
					},
				}
				mock.EXPECT().
					GetBondedValidatorsByPower(gomock.Any()).
					Return(validators, nil).
					Times(1)
			},
			rewardAmount:  math.NewInt(1000),
			expectedError: false, // Should handle gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStakingKeeper := mocks.NewMockStakingKeeper(ctrl)
			tt.setupMocks(mockStakingKeeper)

			ctx := context.Background()
			logger := log.NewNopLogger()
			result := &tokenomicstypes.ClaimSettlementResult{}

			err := distributeValidatorRewards(
				ctx,
				logger,
				result,
				mockStakingKeeper,
				tt.rewardAmount,
				tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_VALIDATOR_REWARD_DISTRIBUTION,
			)

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

func TestDistributeValidatorRewards_PrecisionHandling(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStakingKeeper := mocks.NewMockStakingKeeper(ctrl)

	// Create validators with stakes that don't divide evenly
	validators := []stakingtypes.Validator{
		{
			OperatorAddress: sample.ValOperatorAddressBech32(),
			Tokens:          math.NewInt(333), // ~33.3% of total
			Status:          stakingtypes.Bonded,
		},
		{
			OperatorAddress: sample.ValOperatorAddressBech32(),
			Tokens:          math.NewInt(333), // ~33.3% of total
			Status:          stakingtypes.Bonded,
		},
		{
			OperatorAddress: sample.ValOperatorAddressBech32(),
			Tokens:          math.NewInt(334), // ~33.4% of total
			Status:          stakingtypes.Bonded,
		},
	}

	mockStakingKeeper.EXPECT().
		GetBondedValidatorsByPower(gomock.Any()).
		Return(validators, nil).
		Times(1)

	ctx := context.Background()
	logger := log.NewNopLogger()
	result := &tokenomicstypes.ClaimSettlementResult{}
	rewardAmount := math.NewInt(100) // Small amount that won't divide evenly

	err := distributeValidatorRewards(
		ctx,
		logger,
		result,
		mockStakingKeeper,
		rewardAmount,
		tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_VALIDATOR_REWARD_DISTRIBUTION,
	)

	require.NoError(t, err)

	transfers := result.GetModToAcctTransfers()
	require.Len(t, transfers, 3)

	// Validate total distribution equals reward amount (handles remainder correctly)
	totalDistributed := math.ZeroInt()
	for _, transfer := range transfers {
		totalDistributed = totalDistributed.Add(transfer.Coin.Amount)
	}
	require.Equal(t, rewardAmount, totalDistributed, "Total must equal reward amount despite rounding")
}
