package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/x/session/types"
)

func TestParamsHistory(t *testing.T) {
	k, ctx := keepertest.SessionKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// 1. Initial state: history should be empty, GetParamsAtHeight returns current params
	params0 := k.GetParams(ctx)
	require.Equal(t, params0, k.GetParamsAtHeight(ctx, 10))

	// 2. Record new params at height 10.
	// DefaultNumBlocksPerSession = 10 in x/shared/types/params.go.
	// Height 10 is in session 1 (blocks 1-10).
	// Session 1 end height is 10. Next session start height is 11.
	newParams := types.Params{NumSuppliersPerSession: 10}
	sdkCtx = sdkCtx.WithBlockHeight(10)
	ctx = sdkCtx

	err := k.RecordParamsHistory(ctx, newParams)
	require.NoError(t, err)

	// 3. Verify history
	history := k.GetAllParamsHistory(ctx)
	require.Equal(t, 2, len(history))
	require.Equal(t, int64(1), history[0].EffectiveHeight)
	require.Equal(t, params0, *history[0].Params)
	require.Equal(t, int64(11), history[1].EffectiveHeight)
	require.Equal(t, newParams, *history[1].Params)

	// 4. Verify GetParamsAtHeight
	require.Equal(t, params0, k.GetParamsAtHeight(ctx, 1))
	require.Equal(t, params0, k.GetParamsAtHeight(ctx, 10))
	require.Equal(t, newParams, k.GetParamsAtHeight(ctx, 11))
	require.Equal(t, newParams, k.GetParamsAtHeight(ctx, 100))
}
