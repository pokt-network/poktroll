package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/pocket/testutil/sample"
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
				AsType:    &MsgUpdateParam_AsUint64{AsUint64: 1},
			},
			expectedErr: ErrSharedInvalidAddress,
		}, {
			desc: "invalid: param name incorrect (non-existent)",
			msg: MsgUpdateParam{
				Authority: sample.AccAddress(),
				Name:      "WRONG_num_blocks_per_session",
				AsType:    &MsgUpdateParam_AsUint64{AsUint64: 1},
			},
			expectedErr: ErrSharedParamNameInvalid,
		}, {
			desc: "invalid: incorrect param type",
			msg: MsgUpdateParam{
				Authority: sample.AccAddress(),
				Name:      ParamNumBlocksPerSession,
				AsType:    &MsgUpdateParam_AsString{AsString: "invalid"},
			},
			expectedErr: ErrSharedParamInvalid,
		}, {
			desc: "valid: correct authority, param name, and type",
			msg: MsgUpdateParam{
				Authority: sample.AccAddress(),
				Name:      ParamNumBlocksPerSession,
				AsType:    &MsgUpdateParam_AsUint64{AsUint64: 1},
			},
		}, {
			desc: "invalid ComputeUnitsToTokensMultiplier",
			msg: MsgUpdateParam{
				Authority: sample.AccAddress(),
				Name:      ParamComputeUnitsToTokensMultiplier,
				AsType:    &MsgUpdateParam_AsUint64{AsUint64: 0},
			},
			expectedErr: ErrSharedParamInvalid,
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
