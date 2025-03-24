package types

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/pocket/testutil/sample"
)

func TestMsgStakeGateway_ValidateBasic(t *testing.T) {
	coins := sdk.NewCoin("upokt", math.NewInt(100))
	tests := []struct {
		desc        string
		msg         MsgStakeGateway
		expectedErr error
	}{
		{
			desc: "invalid address - no stake",
			msg: MsgStakeGateway{
				Address: "invalid_address",
				// Stake explicitly nil
			},
			expectedErr: ErrGatewayInvalidAddress,
		}, {
			desc: "valid address - nil stake",
			msg: MsgStakeGateway{
				Address: sample.AccAddress(),
				// Stake explicitly nil
			},
			expectedErr: ErrGatewayInvalidStake,
		}, {
			desc: "valid address - zero stake",
			msg: MsgStakeGateway{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(0)},
			},
			expectedErr: ErrGatewayInvalidStake,
		}, {
			desc: "valid address - negative stake",
			msg: MsgStakeGateway{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "upokt", Amount: math.NewInt(-100)},
			},
			expectedErr: ErrGatewayInvalidStake,
		}, {
			desc: "valid address - invalid stake denom",
			msg: MsgStakeGateway{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "invalid", Amount: math.NewInt(100)},
			},
			expectedErr: ErrGatewayInvalidStake,
		}, {
			desc: "valid address - invalid stake missing denom",
			msg: MsgStakeGateway{
				Address: sample.AccAddress(),
				Stake:   &sdk.Coin{Denom: "", Amount: math.NewInt(100)},
			},
			expectedErr: ErrGatewayInvalidStake,
		}, {
			desc: "valid address - valid stake",
			msg: MsgStakeGateway{
				Address: sample.AccAddress(),
				Stake:   &coins,
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
