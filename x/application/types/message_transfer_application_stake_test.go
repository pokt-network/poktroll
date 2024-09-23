package types

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
)

func TestMsgTransferApplicationStake_ValidateBasic(t *testing.T) {
	dupAddr := sample.AccAddress()

	tests := []struct {
		name string
		msg  MsgTransferApplicationStake
		err  error
	}{
		{
			name: "invalid duplicate source address",
			msg: MsgTransferApplicationStake{
				SourceAddress:      dupAddr,
				DestinationAddress: dupAddr,
			},
			err: ErrAppDuplicateAddress,
		},
		{
			name: "invalid bech32 source address",
			msg: MsgTransferApplicationStake{
				SourceAddress:      "invalid_address",
				DestinationAddress: sample.AccAddress(),
			},
			err: ErrAppInvalidAddress,
		},
		{
			name: "invalid bech32 destination address",
			msg: MsgTransferApplicationStake{
				SourceAddress:      sample.AccAddress(),
				DestinationAddress: "invalid_address",
			},
			err: ErrAppInvalidAddress,
		},
		{
			name: "valid source and destination addresses",
			msg: MsgTransferApplicationStake{
				SourceAddress:      sample.AccAddress(),
				DestinationAddress: sample.AccAddress(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				if strings.Contains(err.Error(), tt.msg.GetSourceAddress()) {
					require.Contains(t, err.Error(), tt.msg.GetSourceAddress())
				} else {
					require.Contains(t, err.Error(), tt.msg.GetDestinationAddress())
				}
				return
			}
			require.NoError(t, err)
		})
	}
}
