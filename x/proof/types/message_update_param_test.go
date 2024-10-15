package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
)

func TestMsgUpdateParam_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgUpdateParam

		expectedErr error
	}{
		{
			name: "invalid: authority address invalid",
			msg: MsgUpdateParam{
				Authority: "invalid_address",
				Name:      "", // Doesn't matter for this test
				AsType:    &MsgUpdateParam_AsFloat{AsFloat: 0},
			},

			expectedErr: ErrProofInvalidAddress,
		}, {
			name: "invalid: param name incorrect (non-existent)",
			msg: MsgUpdateParam{
				Authority: sample.AccAddress(),
				Name:      "non_existent",
				AsType:    &MsgUpdateParam_AsFloat{AsFloat: 0},
			},

			expectedErr: ErrProofParamNameInvalid,
		}, {
			name: "invalid: incorrect param type",
			msg: MsgUpdateParam{
				Authority: sample.AccAddress(),
				Name:      ParamProofMissingPenalty,
				AsType:    &MsgUpdateParam_AsFloat{AsFloat: 0},
			},
			expectedErr: ErrProofParamInvalid,
		}, {
			name: "valid: correct authority, param name, and type",
			msg: MsgUpdateParam{
				Authority: sample.AccAddress(),
				Name:      ParamProofMissingPenalty,
				AsType:    &MsgUpdateParam_AsCoin{AsCoin: &DefaultProofMissingPenalty},
			},

			expectedErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.expectedErr != nil {
				require.ErrorContains(t, err, tt.expectedErr.Error())
				return
			}
			require.NoError(t, err)
		})
	}
}
