package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/x/application/types"
)

func TestMsgUpdateParams(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	params := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, params))

	// default params
	tests := []struct {
		desc  string
		input *types.MsgUpdateParams

		shouldError    bool
		expectedErrMsg string
	}{
		{
			desc: "invalid: authority address invalid",
			input: &types.MsgUpdateParams{
				Authority: "invalid",
				Params:    params,
			},
			shouldError:    true,
			expectedErrMsg: "invalid authority",
		},
		{
			desc: "invalid: send empty params",
			input: &types.MsgUpdateParams{
				Authority: k.GetAuthority(),
				Params:    types.Params{},
			},

			shouldError:    true,
			expectedErrMsg: "max_delegated_gateways must be greater than 0",
		},
		{
			desc: "valid: send default params",
			input: &types.MsgUpdateParams{
				Authority: k.GetAuthority(),
				Params:    params,
			},
			shouldError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			_, err := ms.UpdateParams(ctx, test.input)

			if test.shouldError {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.expectedErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
