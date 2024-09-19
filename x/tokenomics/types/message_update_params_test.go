package types

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
				Params:    Params{},
			},
			expectedErr: ErrTokenomicsAddressInvalid,
		},
		{
			desc: "valid address",
			msg: MsgUpdateParams{
				Authority: sample.AccAddress(),
				Params:    Params{},
			},
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
