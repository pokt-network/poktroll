package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	"pocket/testutil/sample"
)

func TestMsgStakeGateway_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgStakeGateway
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgStakeGateway{
				Address: "invalid_address",
			},
			err: ErrSample,
		}, {
			name: "valid address",
			msg: MsgStakeGateway{
				Address: sample.AccAddress(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}
