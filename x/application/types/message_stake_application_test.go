package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"pocket/testutil/sample"
)

func TestMsgStakeApplication_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgStakeApplication
		err  error
	}{
		{
			name: "invalid address - nil stake",
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
				Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
			},
		}, {
			name: "valid address - zero stake",
			msg: MsgStakeApplication{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(0)},
			},
			err: ErrAppInvalidStake,
		}, {
			name: "valid address - negative stake",
			msg: MsgStakeApplication{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(-100)},
			},
			err: ErrAppInvalidStake,
		}, {
			name: "valid address - invalid stake denom",
			msg: MsgStakeApplication{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "invalid", Amount: sdk.NewInt(100)},
			},
			err: ErrAppInvalidStake,
		}, {
			name: "valid address - invalid stake missing denom",
			msg: MsgStakeApplication{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "", Amount: sdk.NewInt(100)},
			},
			err: ErrAppInvalidStake,
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
