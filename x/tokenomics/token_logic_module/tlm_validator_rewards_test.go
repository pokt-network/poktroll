package token_logic_module_test

import (
	"context"
	"testing"

	cosmosmath "cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/testutil/sample"
	tokenomicskeeper "github.com/pokt-network/poktroll/x/tokenomics/keeper"
	"github.com/pokt-network/poktroll/x/tokenomics/token_logic_module"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

func TestTLMValidatorRewardDistribution(t *testing.T) {
	keeper, ctx, _, _, _ := tokenomicskeeper.TokenomicsKeeperWithActorAddrs(t)

	ctrl := gomock.NewController(t)
	mockStakingKeeper := keeper.GetStakingKeeper().(*mocks.MockStakingKeeper)
	mockDistributionKeeper := keeper.GetDistributionKeeper().(*mocks.MockDistributionKeeper)

	// Set up validator for distribution test
	consAddr := sample.ConsAddress()
	validator := stakingtypes.Validator{
		OperatorAddress: sample.ValOperatorAddress().String(),
	}

	// Mock expectations
	mockStakingKeeper.EXPECT().
		GetValidatorByConsAddr(gomock.Any(), consAddr).
		Return(validator, nil).
		Times(1)

	mockDistributionKeeper.EXPECT().
		AllocateTokensToValidator(gomock.Any(), &validator, gomock.Any()).
		DoAndReturn(func(ctx context.Context, val *stakingtypes.Validator, tokens cosmostypes.DecCoins) error {
			// Verify the correct amount is being distributed
			expectedAmount := cosmosmath.NewInt(100000) // 100k uPOKT for this test
			require.Equal(t, pocket.DenomuPOKT, tokens[0].Denom)
			require.True(t, tokens[0].Amount.TruncateInt().Equal(expectedAmount))
			return nil
		}).
		Times(1)

	// Set proposer in context
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx).WithProposer(consAddr)

	// Create TLM context with proposer reward
	proposerReward := cosmostypes.NewCoin(pocket.DenomuPOKT, cosmosmath.NewInt(100000))
	tlmCtx := &token_logic_module.TLMContext{
		StakingKeeper:      mockStakingKeeper,
		DistributionKeeper: mockDistributionKeeper,
		Result: &tokenomicstypes.TokenomicsTLMResult{
			ModToModTransfers: []tokenomicstypes.ModToModTransfer{},
		},
	}

	// Create a minimal TLM to test distribution
	tlm := &testValidatorRewardTLM{
		ctx:    sdkCtx,
		tlmCtx: tlmCtx,
		reward: proposerReward,
	}

	// Execute distribution logic
	err := tlm.Process([]client.Block{})
	require.NoError(t, err)

	// Verify module-to-module transfer was recorded
	require.Len(t, tlmCtx.Result.ModToModTransfers, 1)
	transfer := tlmCtx.Result.ModToModTransfers[0]
	require.Equal(t, tokenomicstypes.ModuleName, transfer.SenderModule)
	require.Equal(t, distributiontypes.ModuleName, transfer.RecipientModule)
	require.Equal(t, proposerReward, transfer.Coin)
}

// testValidatorRewardTLM is a minimal TLM implementation for testing validator reward distribution
type testValidatorRewardTLM struct {
	ctx    context.Context
	tlmCtx *token_logic_module.TLMContext
	reward cosmostypes.Coin
}

func (t *testValidatorRewardTLM) Process([]client.Block) error {
	consAddr := cosmostypes.UnwrapSDKContext(t.ctx).BlockHeader().ProposerAddress

	// Get validator from consensus address
	validator, err := t.tlmCtx.StakingKeeper.GetValidatorByConsAddr(t.ctx, consAddr)
	if err != nil {
		return err
	}

	// Transfer from tokenomics module to distribution module
	t.tlmCtx.Result.AppendModToModTransfer(tokenomicstypes.ModToModTransfer{
		OpReason:        tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_PROPOSER_REWARD_DISTRIBUTION,
		SenderModule:    tokenomicstypes.ModuleName,
		RecipientModule: distributiontypes.ModuleName,
		Coin:            t.reward,
	})

	// Allocate tokens to validator for distribution to delegators
	rewardDecCoin := cosmostypes.NewDecCoinsFromCoins(t.reward)
	return t.tlmCtx.DistributionKeeper.AllocateTokensToValidator(t.ctx, &validator, rewardDecCoin)
}