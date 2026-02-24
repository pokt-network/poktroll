package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/x/shared/keeper"
	"github.com/pokt-network/poktroll/x/shared/types"
)

func TestMsgUpdateParams(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	params := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, params))
	wctx := sdk.UnwrapSDKContext(ctx)

	// default params
	testCases := []struct {
		name      string
		input     *types.MsgUpdateParams
		expErr    bool
		expErrMsg string
	}{
		{
			name: "invalid: authority address invalid",
			input: &types.MsgUpdateParams{
				Authority: "invalid",
				Params:    params,
			},
			expErr:    true,
			expErrMsg: "invalid authority",
		},
		{
			name: "invalid: send empty params",
			input: &types.MsgUpdateParams{
				Authority: k.GetAuthority(),
				Params:    types.Params{},
			},
			expErr:    true,
			expErrMsg: "invalid NumBlocksPerSession: (0): the provided param is invalid",
		},
		{
			name: "valid: send default params",
			input: &types.MsgUpdateParams{
				Authority: k.GetAuthority(),
				Params:    params,
			},
			expErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ms.UpdateParams(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestMsgUpdateParams_RecordsParamsHistory verifies that the bulk UpdateParams
// (governance/MsgUpdateParams) records params history, matching the behavior of
// the singular UpdateParam and the session module's UpdateParams.
func TestMsgUpdateParams_RecordsParamsHistory(t *testing.T) {
	// Use a fresh keeper to avoid interference from other tests' state.
	k, ctx := keepertest.SharedKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	defaultParams := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, defaultParams))

	// Set block height so session boundary calculations work.
	// Default NumBlocksPerSession=10, so at height 5 the current session
	// ends at height 10 and the next session starts at height 11.
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(5)

	// History should be empty before any UpdateParams call.
	history := k.GetAllParamsHistory(sdkCtx)
	require.Empty(t, history, "params history should be empty before UpdateParams")

	// Call UpdateParams (bulk) with modified params.
	updatedParams := defaultParams
	updatedParams.ClaimWindowOpenOffsetBlocks = defaultParams.ClaimWindowOpenOffsetBlocks + 1
	_, err := ms.UpdateParams(sdkCtx, &types.MsgUpdateParams{
		Authority: k.GetAuthority(),
		Params:    updatedParams,
	})
	require.NoError(t, err)

	// Verify history was recorded (initial entry at current height + new entry
	// at next session start).
	history = k.GetAllParamsHistory(sdkCtx)
	require.GreaterOrEqual(t, len(history), 2,
		"UpdateParams should record both initial and new params in history")

	// The new params should be effective at the start of the next session (height 11).
	expectedEffectiveHeight := int64(11)
	lastEntry := history[len(history)-1]
	require.Equal(t, expectedEffectiveHeight, lastEntry.EffectiveHeight,
		"new params should be effective at the start of the next session")
	require.Equal(t, updatedParams.ClaimWindowOpenOffsetBlocks,
		lastEntry.Params.ClaimWindowOpenOffsetBlocks,
		"recorded params should match the updated values")

	// Verify GetParamsAtHeight returns the new params at the effective height.
	paramsAtEffective := k.GetParamsAtHeight(sdkCtx, expectedEffectiveHeight)
	require.Equal(t, updatedParams.ClaimWindowOpenOffsetBlocks,
		paramsAtEffective.ClaimWindowOpenOffsetBlocks,
		"GetParamsAtHeight should return updated params at the effective height")

	// Verify GetParamsAtHeight returns the old params before the effective height.
	paramsBeforeEffective := k.GetParamsAtHeight(sdkCtx, expectedEffectiveHeight-1)
	require.Equal(t, defaultParams.ClaimWindowOpenOffsetBlocks,
		paramsBeforeEffective.ClaimWindowOpenOffsetBlocks,
		"GetParamsAtHeight should return old params before the effective height")
}
