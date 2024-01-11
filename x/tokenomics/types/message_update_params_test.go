package types

import (
	"testing"

	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/stretchr/testify/require"
)

func TestMsgUpdateParams_ValidateBasic(t *testing.T) {
	tests := []struct {
		desc string

		msg MsgUpdateParams

		expectedErr error
	}{
		{
			desc: "invalid authority address",
			msg: MsgUpdateParams{
				Authority: "invalid_address",
			},
			expectedErr: ErrTokenomicsAuthorityInvalidAddress,
		}, {
			desc: "valid address",
			msg: MsgUpdateParams{
				Authority: sample.AccAddress(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
