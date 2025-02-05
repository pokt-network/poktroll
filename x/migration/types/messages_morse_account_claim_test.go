package types

import (
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/stretchr/testify/require"
)

func TestMsgCreateMorseAccountClaim_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgCreateMorseAccountClaim
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgCreateMorseAccountClaim{
				ShannonDestAddress: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgCreateMorseAccountClaim{
				ShannonDestAddress: sample.AccAddress(),
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

func TestMsgUpdateMorseAccountClaim_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgUpdateMorseAccountClaim
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgUpdateMorseAccountClaim{
				ShannonDestAddress: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgUpdateMorseAccountClaim{
				ShannonDestAddress: sample.AccAddress(),
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

func TestMsgDeleteMorseAccountClaim_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgDeleteMorseAccountClaim
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgDeleteMorseAccountClaim{
				ShannonDestAddress: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgDeleteMorseAccountClaim{
				ShannonDestAddress: sample.AccAddress(),
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
