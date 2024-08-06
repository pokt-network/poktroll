package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
)

func TestMsgUnstakeSupplier_ValidateBasic(t *testing.T) {
	ownerAddress := sample.AccAddress()
	operatorAddress := sample.AccAddress()
	tests := []struct {
		desc        string
		msg         MsgUnstakeSupplier
		expectedErr error
	}{
		{
			desc: "invalid operator address",
			msg: MsgUnstakeSupplier{
				OwnerAddress:    ownerAddress,
				OperatorAddress: "invalid_address",
			},
			expectedErr: ErrSupplierInvalidAddress,
		},
		{
			desc: "missing operator address",
			msg: MsgUnstakeSupplier{
				OwnerAddress: ownerAddress,
			},
			expectedErr: ErrSupplierInvalidAddress,
		},
		{
			desc: "invalid owner address",
			msg: MsgUnstakeSupplier{
				OwnerAddress:    "invalid_address",
				OperatorAddress: operatorAddress,
			},
			expectedErr: ErrSupplierInvalidAddress,
		},
		{
			desc: "missing owner address",
			msg: MsgUnstakeSupplier{
				OperatorAddress: operatorAddress,
			},
			expectedErr: ErrSupplierInvalidAddress,
		},
		{
			desc: "valid message",
			msg: MsgUnstakeSupplier{
				OwnerAddress:    ownerAddress,
				OperatorAddress: operatorAddress,
			},
		},
		{
			desc: "valid message - same operator and owner addresses",
			msg: MsgUnstakeSupplier{
				OwnerAddress:    ownerAddress,
				OperatorAddress: ownerAddress,
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
