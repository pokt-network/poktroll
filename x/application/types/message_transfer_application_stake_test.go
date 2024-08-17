package types

import (
	"testing"

	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/stretchr/testify/require"
)

func TestMsgTransferApplicationStake_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgTransferApplicationStake
		err  error
	}{
		{
			name: "invalid application address",
			msg: MsgTransferApplicationStake{
				Address:     "invalid_address",
				Beneficiary: sample.AccAddress(),
			},
			err: ErrAppInvalidAddress,
		},
		{
			name: "invalid beneficiary address",
			msg: MsgTransferApplicationStake{
				Address:     sample.AccAddress(),
				Beneficiary: "invalid_address",
			},
			err: ErrAppInvalidAddress,
		},
		{
			name: "valid application and beneficiary address",
			msg: MsgTransferApplicationStake{
				Address:     sample.AccAddress(),
				Beneficiary: sample.AccAddress(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				require.Contains(t, err.Error(), tt.msg.Address)
				return
			}
			require.NoError(t, err)
		})
	}
}
