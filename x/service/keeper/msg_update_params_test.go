package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	servicetypes "github.com/pokt-network/poktroll/x/service/types"
)

func TestMsgUpdateParams(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	params := servicetypes.DefaultParams()
	require.NoError(t, k.SetParams(ctx, params))

	// default params
	tests := []struct {
		desc           string
		input          *servicetypes.MsgUpdateParams
		shouldError    bool
		expectedErrMsg string
	}{
		{
			desc: "invalid: authority address invalid",
			input: &servicetypes.MsgUpdateParams{
				Authority: "invalid",
				Params:    params,
			},
			shouldError:    true,
			expectedErrMsg: "invalid authority",
		},
		{
			desc: "invalid: send empty params",
			input: &servicetypes.MsgUpdateParams{
				Authority: k.GetAuthority(),
				Params:    servicetypes.Params{},
			},
			shouldError:    true,
			expectedErrMsg: "missing add_service_fee",
		},
		{
			desc: "invalid: send default params",
			input: &servicetypes.MsgUpdateParams{
				Authority: k.GetAuthority(),
				Params:    params,
			},
			shouldError:    true,
			expectedErrMsg: "target_num_relays must be greater than 0: got 0",
		},
		{
			desc: "valid: send minimal params",
			input: &servicetypes.MsgUpdateParams{
				Authority: k.GetAuthority(),
				Params: servicetypes.Params{
					AddServiceFee:   &servicetypes.MinAddServiceFee,
					TargetNumRelays: 1,
				},
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
