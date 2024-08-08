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
				Signer:  ownerAddress,
				Address: "invalid_address",
			},
			expectedErr: ErrSupplierInvalidAddress,
		},
		{
			desc: "missing operator address",
			msg: MsgUnstakeSupplier{
				Signer: ownerAddress,
			},
			expectedErr: ErrSupplierInvalidAddress,
		},
		{
			desc: "invalid owner address",
			msg: MsgUnstakeSupplier{
				Signer:  "invalid_address",
				Address: operatorAddress,
			},
			expectedErr: ErrSupplierInvalidAddress,
		},
		{
			desc: "missing owner address",
			msg: MsgUnstakeSupplier{
				Address: operatorAddress,
			},
			expectedErr: ErrSupplierInvalidAddress,
		},
		{
			desc: "valid message",
			msg: MsgUnstakeSupplier{
				Signer:  ownerAddress,
				Address: operatorAddress,
			},
		},
		{
			desc: "valid message - same operator and owner addresses",
			msg: MsgUnstakeSupplier{
				Signer:  ownerAddress,
				Address: ownerAddress,
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
