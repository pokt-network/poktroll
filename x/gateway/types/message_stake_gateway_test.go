package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
)

func TestMsgStakeGateway_ValidateBasic(t *testing.T) {
	coins := sdk.NewCoin("upokt", sdk.NewInt(100))
	tests := []struct {
		name string
		msg  MsgStakeGateway
		err  error
	}{
		{
			name: "invalid address - no stake",
			msg: MsgStakeGateway{
				Address: "invalid_address",
				// Stake explicitly nil
			},
			err: ErrGatewayInvalidAddress,
		}, {
			name: "valid address - nil stake",
			msg: MsgStakeGateway{
				Address: sample.AccAddress(),
				// Stake explicitly nil
			},
			err: ErrGatewayInvalidStake,
		}, {
			name: "valid address - zero stake",
			msg: MsgStakeGateway{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(0)},
			},
			err: ErrGatewayInvalidStake,
		}, {
			name: "valid address - negative stake",
			msg: MsgStakeGateway{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(-100)},
			},
			err: ErrGatewayInvalidStake,
		}, {
			name: "valid address - invalid stake denom",
			msg: MsgStakeGateway{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "invalid", Amount: sdk.NewInt(100)},
			},
			err: ErrGatewayInvalidStake,
		}, {
			name: "valid address - invalid stake missing denom",
			msg: MsgStakeGateway{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "", Amount: sdk.NewInt(100)},
			},
			err: ErrGatewayInvalidStake,
		}, {
			name: "valid address - valid stake",
			msg: MsgStakeGateway{
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
