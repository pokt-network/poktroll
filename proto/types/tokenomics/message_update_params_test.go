package tokenomics

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
)

func TestMsgUpdateParams_ValidateBasic(t *testing.T) {
	tests := []struct {
		desc        string
		msg         MsgUpdateParams
		expectedErr error
	}{
		{
			desc: "invalid authority address",
			msg: MsgUpdateParams{
				Authority: "invalid_address",
				Params: Params{
					ComputeUnitsToTokensMultiplier: 1,
				},
			},
			expectedErr: ErrTokenomicsAddressInvalid,
		},
		{
			desc: "valid address",
			msg: MsgUpdateParams{
				Authority: sample.AccAddress(),
				Params: Params{
					ComputeUnitsToTokensMultiplier: 1,
				},
			},
		},
		{
			desc: "invalid ComputeUnitsToTokensMultiplier",
			msg: MsgUpdateParams{
				Authority: sample.AccAddress(),
				Params: Params{
					ComputeUnitsToTokensMultiplier: 0,
				},
			},
			expectedErr: ErrTokenomicsParamsInvalid,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := test.msg.ValidateBasic()
			if test.expectedErr != nil {
				require.ErrorIs(t, err, test.expectedErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
