package keeper_test

import (
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/x/service/types"
)

func TestParamsQuery(t *testing.T) {
	keeper, ctx := keepertest.ServiceKeeper(t)
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
