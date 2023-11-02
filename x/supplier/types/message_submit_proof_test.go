package types

import (
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/stretchr/testify/require"
)

// TODO(@bryanchriswhite): Add unit tests for message validation when adding the business logic.

func TestMsgSubmitProof_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgSubmitProof
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgSubmitProof{
				SupplierAddress: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgSubmitProof{
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
