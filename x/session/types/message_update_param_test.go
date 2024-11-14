package types

import (
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
)

func TestMsgUpdateParam_ValidateBasic(t *testing.T) {
	tests := []struct {
		desc        string
		msg         MsgUpdateParam
		expectedErr error
	}{
		{
			desc: "invalid: authority address invalid",
			msg: MsgUpdateParam{
				Authority: "invalid_address",
				Name:      "", // Doesn't matter for this test
				AsType:    &MsgUpdateParam_AsUint64{AsUint64: 0},
			},
			expectedErr: sdkerrors.ErrInvalidAddress,
		}, {
			desc: "invalid: param name incorrect (non-existent)",
			msg: MsgUpdateParam{
				Authority: sample.AccAddress(),
				Name:      "non_existent",
				AsType:    &MsgUpdateParam_AsUint64{AsUint64: DefaultNumSuppliersPerSession},
			},
			expectedErr: ErrSessionParamInvalid,
		}, {
			desc: "valid: correct address, param name, and type",
			msg: MsgUpdateParam{
				Authority: sample.AccAddress(),
				Name:      ParamNumSuppliersPerSession,
				AsType:    &MsgUpdateParam_AsUint64{AsUint64: DefaultNumSuppliersPerSession},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
