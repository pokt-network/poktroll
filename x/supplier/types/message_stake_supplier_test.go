package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"pocket/testutil/sample"
)

func TestMsgStakeSupplier_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgStakeSupplier
		err  error
	}{
		{
			name: "invalid address - nil stake",
			msg: MsgStakeSupplier{
				Address: "invalid_address",
				// Stake explicitly nil
			},
			err: ErrSupplierInvalidAddress,
		}, {
			name: "valid address - nil stake",
			msg: MsgStakeSupplier{
				Address: sample.AccAddress(),
				// Stake explicitly nil
			},
			err: ErrSupplierInvalidStake,
		}, {
			name: "valid address - valid stake",
			msg: MsgStakeSupplier{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
			},
		}, {
			name: "valid address - zero stake",
			msg: MsgStakeSupplier{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(0)},
			},
			err: ErrSupplierInvalidStake,
		}, {
			name: "valid address - negative stake",
			msg: MsgStakeSupplier{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(-100)},
			},
			err: ErrSupplierInvalidStake,
		}, {
			name: "valid address - invalid stake denom",
			msg: MsgStakeSupplier{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "invalid", Amount: sdk.NewInt(100)},
			},
			err: ErrSupplierInvalidStake,
		}, {
			name: "valid address - invalid stake missing denom",
			msg: MsgStakeSupplier{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "", Amount: sdk.NewInt(100)},
			},
			err: ErrSupplierInvalidStake,
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
