package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/tokenomics/keeper"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

func TestUpdateParams_Validity(t *testing.T) {
	Keeper, ctx := testkeeper.TokenomicsKeeper(t)
	srv := keeper.NewMsgServerImpl(*Keeper)

	params := types.DefaultParams()
	Keeper.SetParams(ctx, params)

	tests := []struct {
		desc string

		req *types.MsgUpdateParams

		expectErr     bool
		expectedPanic bool
		expErrMsg     string
	}{
		{
			desc: "invalid authority address",

			req: &types.MsgUpdateParams{
				Authority: "invalid",
				Params: types.Params{
					ComputeUnitsToTokensMultiplier: 1,
				},
			},

			expectErr:     true,
			expectedPanic: false,
			expErrMsg:     "invalid authority",
		},
		{
			desc: "incorrect authority address",

			req: &types.MsgUpdateParams{
				Authority: sample.AccAddress(),
				Params: types.Params{
					ComputeUnitsToTokensMultiplier: 1,
				},
			},

			expectErr:     true,
			expectedPanic: false,
			expErrMsg:     "the provided authority address does not match the on-chain governance address",
		},
		{
			desc: "invalid ComputeUnitsToTokensMultiplier",

			req: &types.MsgUpdateParams{
				Authority: Keeper.GetAuthority(),

				Params: types.Params{
					ComputeUnitsToTokensMultiplier: 0,
				},
			},

			expectErr:     true,
			expectedPanic: true,
			expErrMsg:     "invalid compute to tokens multiplier",
		},
		{
			desc: "successful param update",

			req: &types.MsgUpdateParams{
				Authority: Keeper.GetAuthority(),

				Params: types.Params{
					ComputeUnitsToTokensMultiplier: 1000000,
				},
			},

			expectedPanic: false,
			expectErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if tt.expectedPanic {
				defer func() {
					if r := recover(); r != nil {
						_, err := srv.UpdateParams(ctx, tt.req)
						require.Error(t, err)
					}
				}()
				return
			}
			_, err := srv.UpdateParams(ctx, tt.req)
			if tt.expectErr {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.expErrMsg)
			} else {
				require.Nil(t, err)
			}
		})
	}
}

func TestUpdateParams_ComputeUnitsToTokensMultiplier(t *testing.T) {
	Keeper, ctx := testkeeper.TokenomicsKeeper(t)
	srv := keeper.NewMsgServerImpl(*Keeper)

	// Set the default params
	Keeper.SetParams(ctx, types.DefaultParams())

	// Verify the default value for ComputeUnitsToTokensMultiplier
	getParamsReq := &types.QueryParamsRequest{}
	getParamsRes, err := Keeper.Params(ctx, getParamsReq)
	require.Nil(t, err)
	require.Equal(t, uint64(42), getParamsRes.Params.GetComputeUnitsToTokensMultiplier())

	// Update the value for ComputeUnitsToTokensMultiplier
	updateParamsReq := &types.MsgUpdateParams{
		Authority: Keeper.GetAuthority(),
		Params: types.Params{
			ComputeUnitsToTokensMultiplier: 69,
		},
	}
	_, err = srv.UpdateParams(ctx, updateParamsReq)
	require.Nil(t, err)

	// Verify that ComputeUnitsToTokensMultiplier was updated
	getParamsRes, err = Keeper.Params(ctx, getParamsReq)
	require.Nil(t, err)
	require.Equal(t, uint64(69), getParamsRes.Params.GetComputeUnitsToTokensMultiplier())
}
