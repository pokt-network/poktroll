package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
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
				Params:    types.Params{},
			},

			shouldError:    true,
			expectedErrMsg: "invalid authority",
		},
		{
			desc: "incorrect authority address",

			req: &types.MsgUpdateParams{
				Authority: sample.AccAddress(),
				Params:    types.Params{},
			},

			shouldError:    true,
			expectedErrMsg: "the provided authority address does not match the on-chain governance address",
		},
		{
			desc: "successful param update",

			req: &types.MsgUpdateParams{
				Authority: tokenomicsKeeper.GetAuthority(),
				Params:    types.Params{},
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
