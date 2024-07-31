package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/tokenomics/keeper"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

func TestMsgUpdateParams(t *testing.T) {
	tokenomicsKeeper, srv, ctx := setupMsgServer(t)
	require.NoError(t, tokenomicsKeeper.SetParams(ctx, types.DefaultParams()))

	tests := []struct {
		desc string

		req *types.MsgUpdateParams

		shouldError    bool
		expectedErrMsg string
	}{
		{
			desc: "invalid authority address",

			req: &types.MsgUpdateParams{
				Authority: "invalid",
				Params: types.Params{
					ComputeUnitsToTokensMultiplier: 1,
				},
			},

			shouldError:    true,
			expectedErrMsg: "invalid authority",
		},
		{
			desc: "incorrect authority address",

			req: &types.MsgUpdateParams{
				Authority: sample.AccAddress(),
				Params: types.Params{
					ComputeUnitsToTokensMultiplier: 1,
				},
			},

			shouldError:    true,
			expectedErrMsg: "the provided authority address does not match the on-chain governance address",
		},
		{
			desc: "invalid ComputeUnitsToTokensMultiplier",

			req: &types.MsgUpdateParams{
				Authority: tokenomicsKeeper.GetAuthority(),

				Params: types.Params{
					ComputeUnitsToTokensMultiplier: 0,
				},
			},

			shouldError:    true,
			expectedErrMsg: "invalid ComputeUnitsToTokensMultiplier",
		},
		{
			desc: "successful param update",

			req: &types.MsgUpdateParams{
				Authority: tokenomicsKeeper.GetAuthority(),

				Params: types.Params{
					ComputeUnitsToTokensMultiplier: 1000000,
				},
			},

			shouldError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			_, err := srv.UpdateParams(ctx, test.req)
			if test.shouldError {
				require.Error(t, err)
				require.ErrorContains(t, err, test.expectedErrMsg)
			} else {
				require.Nil(t, err)
			}
		})
	}
}

func TestUpdateParams_ComputeUnitsToTokensMultiplier(t *testing.T) {
	tokenomicsKeeper, ctx, _, _, _ := testkeeper.TokenomicsKeeperWithActorAddrs(t)
	srv := keeper.NewMsgServerImpl(tokenomicsKeeper)

	// Set the default params
	tokenomicsKeeper.SetParams(ctx, types.DefaultParams())

	getParamsReq := &types.QueryParamsRequest{}

	// Verify the default value for ComputeUnitsToTokensMultiplier
	getParamsRes, err := tokenomicsKeeper.Params(ctx, getParamsReq)
	require.NoError(t, err)
	require.Equal(t,
		types.DefaultComputeUnitsToTokensMultiplier,
		getParamsRes.Params.GetComputeUnitsToTokensMultiplier(),
	)

	// Update the value for ComputeUnitsToTokensMultiplier
	updateParamsReq := &types.MsgUpdateParams{
		Authority: tokenomicsKeeper.GetAuthority(),
		Params: types.Params{
			ComputeUnitsToTokensMultiplier: 69,
		},
	}
	_, err = srv.UpdateParams(ctx, updateParamsReq)
	require.NoError(t, err)

	// Verify that ComputeUnitsToTokensMultiplier was updated
	getParamsRes, err = tokenomicsKeeper.Params(ctx, getParamsReq)
	require.NoError(t, err)
	require.Equal(t, uint64(69), getParamsRes.Params.GetComputeUnitsToTokensMultiplier())
}
