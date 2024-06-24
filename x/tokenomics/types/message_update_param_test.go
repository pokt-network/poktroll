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
				AsType:    &MsgUpdateParam_AsInt64{AsInt64: 1},
			},

			expectedErr: ErrTokenomicsAddressInvalid,
		}, {
			name: "invalid: param name incorrect (non-existent)",
			msg: MsgUpdateParam{
				Authority: sample.AccAddress(),
				Name:      "invalid",
				AsType:    &MsgUpdateParam_AsInt64{AsInt64: 1},
			},
			expectedErr: ErrTokenomicsParamNameInvalid,
		}, {
			name: "invalid: incorrect param type",
			msg: MsgUpdateParam{
				Authority: sample.AccAddress(),
				Name:      ParamComputeUnitsToTokensMultiplier,
				AsType:    &MsgUpdateParam_AsString{AsString: "invalid"},
			},
			expectedErr: ErrTokenomicsParamInvalid,
		}, {
			name: "valid: correct authority, param name, and type",
			msg: MsgUpdateParam{
				Authority: sample.AccAddress(),
				Name:      ParamComputeUnitsToTokensMultiplier,
				AsType:    &MsgUpdateParam_AsInt64{AsInt64: 1},
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
