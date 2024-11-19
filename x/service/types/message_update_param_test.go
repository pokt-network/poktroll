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
		desc        string
		msg         MsgUpdateParam
		expectedErr error
	}{
		{
			name: "invalid address",
			desc: "invalid: authority address invalid",
			msg: MsgUpdateParam{
				Authority: "invalid_address",
				Name:      "", // Doesn't matter for this test
				AsType:    &MsgUpdateParam_AsCoin{AsCoin: nil},
			},
			expectedErr: sdkerrors.ErrInvalidAddress,
		},
		{
			desc: "invalid: param name incorrect (non-existent)",
			msg: MsgUpdateParam{
				Authority: sample.AccAddress(),
				Name:      "non_existent",
				AsType:    &MsgUpdateParam_AsCoin{AsCoin: &MinAddServiceFee},
			},
			expectedErr: ErrServiceParamInvalid,
		},
		{
			name: "valid address",
			desc: "valid: correct address, param name, and type",
			msg: MsgUpdateParam{
				Authority: sample.AccAddress(),
				Name:      ParamAddServiceFee,
				AsType:    &MsgUpdateParam_AsCoin{AsCoin: &MinAddServiceFee},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := test.msg.ValidateBasic()
			if test.expectedErr != nil {
				require.ErrorContains(t, err, test.expectedErr.Error())
				return
			}
			require.NoError(t, err)
		})
	}
}
