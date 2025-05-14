package keeper_test

import (
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

func TestGetParams(t *testing.T) {
	k, ctx, _, _, _ := testkeeper.TokenomicsKeeperWithActorAddrs(t)
	params := types.DefaultParams()

	require.NoError(t, k.SetParams(ctx, params))
	require.EqualValues(t, params, k.GetParams(ctx))
}

func TestParamsQuery(t *testing.T) {
	keeper, ctx, _, _, _ := testkeeper.TokenomicsKeeperWithActorAddrs(t)
	params := types.DefaultParams()
	require.NoError(t, keeper.SetInitialParams(ctx, params))

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(1)

	response, err := keeper.Params(sdkCtx, &types.QueryParamsRequest{})
	require.NoError(t, err)

	expectedParamsRes := &types.QueryParamsResponse{
		Params:             params,
		ActivationHeight:   1,
		DeactivationHeight: 0,
	}
	require.Equal(t, expectedParamsRes, response)
}
