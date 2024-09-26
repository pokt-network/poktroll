package types

import (
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
)

func TestMsgUpdateParam_ValidateBasic(t *testing.T) {
	tests := []struct {
		name        string
		msg         MsgUpdateParam
		expectedErr error
	}{
		{
			name: "invalid: authority address invalid",
			msg: MsgUpdateParam{
				Authority: "invalid_address",
				Name:      "", // Doesn't matter for this test
				AsType:    &MsgUpdateParam_AsCoin{AsCoin: &DefaultMinStake},
			},
			expectedErr: sdkerrors.ErrInvalidAddress,
		}, {
			name: "invalid: param name incorrect (non-existent)",
			msg: MsgUpdateParam{
				Authority: sample.AccAddress(),
				Name:      "WRONG_relay_difficulty_target_hash",
				AsType:    &MsgUpdateParam_AsCoin{AsCoin: nil},
			},

			expectedErr: ErrGatewayParamInvalid,
		}, {
			name: "invalid: value cannot be nil",
			msg: MsgUpdateParam{
				Authority: sample.AccAddress(),
				Name:      ParamMinStake,
				AsType:    &MsgUpdateParam_AsCoin{AsCoin: nil},
			},
			expectedErr: ErrGatewayParamInvalid,
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
			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
