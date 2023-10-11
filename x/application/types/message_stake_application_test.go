package types

import (
	"pocket/testutil/sample"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestMsgStakeApplication_ValidateBasic(t *testing.T) {
	coins := sdk.NewCoin("upokt", sdk.NewInt(100))
	tests := []struct {
		name string
		msg  MsgStakeApplication
		err  error
	}{
		{
			name: "invalid address - no stake",
			msg: MsgStakeApplication{
				Address: "invalid_address",
				// Stake explicitly nil
			},
			err: ErrAppInvalidAddress,
		}, {
			name: "valid address - nil stake",
			msg: MsgStakeApplication{
				Address: sample.AccAddress(),
				// Stake explicitly nil
			},
			err: ErrAppInvalidStake,
		}, {
			name: "valid address - valid stake",
			msg: MsgStakeApplication{
				Address: sample.AccAddress(),
				Stake:   &coins,
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
