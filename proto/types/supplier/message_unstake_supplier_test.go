package supplier

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
)

func TestMsgUnstakeSupplier_ValidateBasic(t *testing.T) {
	tests := []struct {
		desc        string
		msg         MsgUnstakeSupplier
		expectedErr error
	}{
		{
			desc: "invalid address",
			msg: MsgUnstakeSupplier{
				Address: "invalid_address",
			},
			expectedErr: ErrSupplierInvalidAddress,
		}, {
			desc:        "missing address",
			msg:         MsgUnstakeSupplier{},
			expectedErr: ErrSupplierInvalidAddress,
		}, {
			desc: "valid address",
			msg: MsgUnstakeSupplier{
				Address: sample.AccAddress(),
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
