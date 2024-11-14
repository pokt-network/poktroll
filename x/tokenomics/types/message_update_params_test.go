package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
)

func TestMsgUpdateParams_ValidateBasic(t *testing.T) {
	tests := []struct {
		desc        string
		msg         MsgUpdateParams
		expectedErr error
	}{
		{
			desc: "invalid: non-bech32 authority address",
			msg: MsgUpdateParams{
				Authority: "invalid_address",
				Params:    Params{},
			},
			expectedErr: ErrTokenomicsAddressInvalid,
		},
		{
			desc: "invalid: empty params",
			msg: MsgUpdateParams{
				Authority: sample.AccAddress(),
				Params:    Params{},
			},
			expectedErr: ErrTokenomicsParamInvalid,
		},
		{
			desc: "valid: address and default params",
			msg: MsgUpdateParams{
				Authority: sample.AccAddress(),
				Params:    DefaultParams(),
			},
		},
		{
			desc: "invalid: mint allocation params don't sum to 1",
			msg: MsgUpdateParams{
				Authority: sample.AccAddress(),
				Params: Params{
					MintAllocationDao:         0.1,
					MintAllocationProposer:    0.1,
					MintAllocationSupplier:    0.1,
					MintAllocationSourceOwner: 0.1,
					MintAllocationApplication: 0.1,
				},
			},
			expectedErr: ErrTokenomicsParamInvalid,
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
