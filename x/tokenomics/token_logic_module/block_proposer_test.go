package token_logic_module

import (
	"context"
	"testing"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/tokenomics/mocks"
)

// func init() {
// 	cmd.InitSDKConfig()
// }

func Test_getBlockProposerOperatorAddress(t *testing.T) {
	// Prepare a validator consensus address
	consAddr := sample.ConsAddress()

	// Prepare a validator operator address
	testValOperatorAddrString := sample.ValOperatorAddress()
	testValOperatorAddrBz, err := cosmostypes.ValAddressFromBech32(testValOperatorAddrString)
	require.NoError(t, err)

	tests := []struct {
		name                  string
		setupContext          func() context.Context
		setupStakingKeeper    func(*gomock.Controller) (*mocks.MockStakingKeeper, string)
		expectedError         bool
		expectedErrorContains string
	}{
		{
			name: "success - staking keeper returns validator",
			setupContext: func() context.Context {
				ctx := cosmostypes.Context{}.WithBlockHeader(cmtproto.Header{
					ProposerAddress: consAddr,
				})
				return ctx
			},
			setupStakingKeeper: func(ctrl *gomock.Controller) (*mocks.MockStakingKeeper, string) {
				mockStakingKeeper := mocks.NewMockStakingKeeper(ctrl)

				validator := stakingtypes.Validator{
					OperatorAddress: testValOperatorAddrBz.String(),
				}

				mockStakingKeeper.EXPECT().
					GetValidatorByConsAddr(gomock.Any(), consAddr).
					Return(validator, nil).
					Times(1)

				// The function returns an account address (cosmos prefix) converted from the validator operator address
				expectedAccAddr := cosmostypes.AccAddress(testValOperatorAddrBz).String()
				return mockStakingKeeper, expectedAccAddr
			},
			expectedError: false,
		},
		{
			name: "error - staking keeper returns error",
			setupContext: func() context.Context {
				ctx := cosmostypes.Context{}.WithBlockHeader(cmtproto.Header{
					ProposerAddress: consAddr,
				})
				return ctx
			},
			setupStakingKeeper: func(ctrl *gomock.Controller) (*mocks.MockStakingKeeper, string) {
				mockStakingKeeper := mocks.NewMockStakingKeeper(ctrl)

				mockStakingKeeper.EXPECT().
					GetValidatorByConsAddr(gomock.Any(), consAddr).
					Return(stakingtypes.Validator{}, sdkerrors.ErrInvalidAddress).
					Times(1)

				return mockStakingKeeper, ""
			},
			expectedError:         true,
			expectedErrorContains: "invalid address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Setup context and staking keeper
			ctx := tt.setupContext()
			stakingKeeper, expectedOperatorAddress := tt.setupStakingKeeper(ctrl)

			// Execute the function
			resultOperatorAddress, err := getBlockProposerOperatorAddress(ctx, stakingKeeper)

			// Validate the result
			if tt.expectedError {
				require.Error(t, err)
				if tt.expectedErrorContains != "" {
					require.Contains(t, err.Error(), tt.expectedErrorContains)
				}
				require.Empty(t, resultOperatorAddress)
			} else {
				require.NoError(t, err)
				require.Equal(t, expectedOperatorAddress, resultOperatorAddress)
			}
		})
	}
}
