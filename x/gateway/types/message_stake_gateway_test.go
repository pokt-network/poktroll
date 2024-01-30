package types

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
)

func TestMsgStakeGateway_ValidateBasic(t *testing.T) {
	coins := sdk.NewCoin("upokt", math.NewInt(100))
	tests := []struct {
		desc string
		msg  MsgStakeGateway
		err  error
	}{
		{
			desc: "invalid address - no stake",
			msg: MsgStakeGateway{
				Address: "invalid_address",
				// Stake explicitly nil
			},
			err: ErrGatewayInvalidAddress,
		}, {
			desc: "valid address - zero stake",
			msg: MsgStakeGateway{
				Address: sample.AccAddress(),
				Stake:   sdk.Coin{Denom: "upokt", Amount: math.NewInt(0)},
			},
			err: ErrGatewayInvalidStake,
		}, {
			desc: "valid address - negative stake",
			msg: MsgStakeGateway{
				Address: sample.AccAddress(),
				Stake:   sdk.Coin{Denom: "upokt", Amount: math.NewInt(-100)},
			},
			err: ErrGatewayInvalidStake,
		}, {
			desc: "valid address - invalid stake denom",
			msg: MsgStakeGateway{
				Address: sample.AccAddress(),
				Stake:   sdk.Coin{Denom: "invalid", Amount: math.NewInt(100)},
			},
			err: ErrGatewayInvalidStake,
		}, {
			desc: "valid address - invalid stake missing denom",
			msg: MsgStakeGateway{
				Address: sample.AccAddress(),
				Stake:   sdk.Coin{Denom: "", Amount: math.NewInt(100)},
			},
			err: ErrGatewayInvalidStake,
		}, {
			desc: "valid address - valid stake",
			msg: MsgStakeGateway{
				Address: sample.AccAddress(),
				Stake:   coins,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}
