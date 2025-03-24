package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sessiontypes "github.com/pokt-network/pocket/x/session/types"
)

func TestMsgUpdateParams(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	params := sessiontypes.DefaultParams()
	require.NoError(t, k.SetParams(ctx, params))

	// default params
	tests := []struct {
		desc           string
		params         *sessiontypes.MsgUpdateParams
		shouldError    bool
		expectedErrMsg string
	}{
		{
			desc: "invalid: authority address invalid",
			params: &sessiontypes.MsgUpdateParams{
				Authority: "invalid",
				Params:    params,
			},
			shouldError:    true,
			expectedErrMsg: "invalid authority",
		},
		{
			desc: "invalid: send empty params",
			params: &sessiontypes.MsgUpdateParams{
				Authority: k.GetAuthority(),
				Params:    sessiontypes.Params{},
			},
			shouldError: true,
		},
		{
			desc: "valid: send minimal params",
			params: &sessiontypes.MsgUpdateParams{
				Authority: k.GetAuthority(),
				Params: sessiontypes.Params{
					NumSuppliersPerSession: 42,
				},
			},
			shouldError: false,
		},
		{
			desc: "valid: send default params",
			params: &sessiontypes.MsgUpdateParams{
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
