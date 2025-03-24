package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/pocket/testutil/sample"
)

func TestMsgUnstakeSupplier_ValidateBasic(t *testing.T) {
	signerAddress := sample.AccAddress()
	operatorAddress := sample.AccAddress()
	tests := []struct {
		desc        string
		msg         MsgUnstakeSupplier
		expectedErr error
	}{
		{
			desc: "invalid operator address",
			msg: MsgUnstakeSupplier{
				Signer:          signerAddress,
				OperatorAddress: "invalid_address",
			},
			expectedErr: ErrSupplierInvalidAddress,
		},
		{
			desc: "missing operator address",
			msg: MsgUnstakeSupplier{
				Signer: signerAddress,
			},
			expectedErr: ErrSupplierInvalidAddress,
		},
		{
			desc: "invalid signer address",
			msg: MsgUnstakeSupplier{
				Signer:          "invalid_address",
				OperatorAddress: operatorAddress,
			},
			expectedErr: ErrSupplierInvalidAddress,
		},
		{
			desc: "missing signer address",
			msg: MsgUnstakeSupplier{
				OperatorAddress: operatorAddress,
			},
			expectedErr: ErrSupplierInvalidAddress,
		},
		{
			desc: "valid message",
			msg: MsgUnstakeSupplier{
				Signer:          signerAddress,
				OperatorAddress: operatorAddress,
			},
		},
		{
			desc: "valid message - same signer and operator addresses",
			msg: MsgUnstakeSupplier{
				Signer:          operatorAddress,
				OperatorAddress: operatorAddress,
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
