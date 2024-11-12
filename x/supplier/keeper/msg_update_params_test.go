package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

func TestMsgUpdateParams(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	params := suppliertypes.DefaultParams()
	require.NoError(t, k.SetParams(ctx, params))

	// default params
	tests := []struct {
		desc           string
		params         *suppliertypes.MsgUpdateParams
		shouldError    bool
		expectedErrMsg string
	}{
		{
			desc: "invalid: authority address invalid",
			params: &suppliertypes.MsgUpdateParams{
				Authority: "invalid",
				Params:    params,
			},
			shouldError:    true,
			expectedErrMsg: "invalid authority",
		},
		{
			desc: "invalid: send empty params",
			params: &suppliertypes.MsgUpdateParams{
				Authority: k.GetAuthority(),
				Params:    suppliertypes.Params{},
			},
			shouldError: true,
		},
		{
			desc: "valid: send minimal params",
			params: &suppliertypes.MsgUpdateParams{
				Authority: k.GetAuthority(),
				Params: suppliertypes.Params{
					MinStake: &suppliertypes.DefaultMinStake,
				},
			},
			shouldError: false,
		},
		{
			desc: "valid: send default params",
			params: &suppliertypes.MsgUpdateParams{
				Authority: k.GetAuthority(),
				Params:    params,
			},
			shouldError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			_, err := ms.UpdateParams(ctx, test.params)

			if test.shouldError {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.expectedErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
