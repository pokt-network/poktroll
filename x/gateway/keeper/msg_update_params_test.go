package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
)

func TestMsgUpdateParams(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	params := gatewaytypes.DefaultParams()
	require.NoError(t, k.SetParams(ctx, params))

	// default params
	tests := []struct {
		desc           string
		input          *gatewaytypes.MsgUpdateParams
		shouldError    bool
		expectedErrMsg string
	}{
		{
			desc: "invalid: authority address invalid",
			input: &gatewaytypes.MsgUpdateParams{
				Authority: "invalid",
				Params:    params,
			},
			shouldError:    true,
			expectedErrMsg: "invalid authority",
		},
		{
			desc: "invalid: send empty params",
			input: &gatewaytypes.MsgUpdateParams{
				Authority: k.GetAuthority(),
				Params:    gatewaytypes.Params{},
			},
			shouldError: true,
		},
		{
			desc: "valid: send minimal params",
			input: &gatewaytypes.MsgUpdateParams{
				Authority: k.GetAuthority(),
				Params: gatewaytypes.Params{
					MinStake: &gatewaytypes.DefaultMinStake,
				},
			},
			shouldError: false,
		},
		{
			desc: "valid: send default params",
			input: &gatewaytypes.MsgUpdateParams{
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
