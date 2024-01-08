package keeper_test

import (
	"testing"

	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/x/tokenomics/keeper"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
	"github.com/stretchr/testify/require"
)

func TestUpdateParams(t *testing.T) {
	tokenomicsKeeper, ctx := testkeeper.TokenomicsKeeper(t)
	srv := keeper.NewMsgServerImpl(*tokenomicsKeeper)
	// wctx := sdk.WrapSDKContext(sdkCtx)

	params := types.DefaultParams()
	tokenomicsKeeper.SetParams(ctx, params)

	tests := []struct {
		desc string

		req *types.MsgUpdateParams

		expectErr bool
		expErrMsg string
	}{
		{
			desc: "set invalid authority",

			req: &types.MsgUpdateParams{
				Authority: "foo",
			},

			expectErr: true,
			expErrMsg: "invalid authority",
		},
		{
			desc: "set invalid ComputeUnitsToTokensMultiplier",

			req: &types.MsgUpdateParams{
				Authority: tokenomicsKeeper.GetAuthority(),

				Params: types.Params{
					ComputeUnitsToTokensMultiplier: 0,
				},
			},

			expectErr: true,
			expErrMsg: "invalid compute to tokens multiplier",
		},
		{
			desc: "successful update",

			req: &types.MsgUpdateParams{
				Authority: tokenomicsKeeper.GetAuthority(),

				Params: types.Params{
					ComputeUnitsToTokensMultiplier: 1000000,
				},
			},

			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
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
