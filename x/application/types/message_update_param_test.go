package types

import (
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
)

func TestMsgUpdateParam_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgUpdateParam
		err  error
	}{
		{
			name: "invalid: authority address invalid",
			msg: MsgUpdateParam{
				Authority: "invalid_address",
				Name:      "",
				AsType:    &MsgUpdateParam_AsCoin{AsCoin: &DefaultMinStake},
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "invalid: param name incorrect (non-existent)",
			msg: MsgUpdateParam{
				Authority: sample.AccAddress(),
				Name:      "non_existent",
				AsType:    &MsgUpdateParam_AsCoin{AsCoin: nil},
			},
			err: ErrAppParamInvalid,
		}, {
			name: "valid: correct authority, param name, and type",
			msg: MsgUpdateParam{
				Authority: sample.AccAddress(),
				Name:      ParamMinStake,
				AsType:    &MsgUpdateParam_AsCoin{AsCoin: &DefaultMinStake},
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
