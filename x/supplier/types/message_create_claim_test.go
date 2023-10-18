package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	"pocket/testutil/sample"
)

// TODO(@bryanchriswhite): Add unit tests for message validation when adding the business logic.

func TestMsgCreateClaim_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgCreateClaim
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgCreateClaim{
				SupplierAddress: "invalid_address",
			},
			err: ErrSupplierInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgCreateClaim{
				SupplierAddress: sample.AccAddress(),
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
