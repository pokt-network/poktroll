package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/cmd/poktrolld/cmd"
	"github.com/pokt-network/poktroll/testutil/sample"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

func init() {
	cmd.InitSDKConfig()
}

func TestMsgUpdateParams_ValidateBasic(t *testing.T) {
	tests := []struct {
		desc        string
		msg         tokenomicstypes.MsgUpdateParams
		expectedErr error
	}{
		{
			desc: "invalid: non-bech32 authority address",
			msg: tokenomicstypes.MsgUpdateParams{
				Authority: "invalid_address",
				Params:    tokenomicstypes.Params{},
			},
			expectedErr: tokenomicstypes.ErrTokenomicsAddressInvalid,
		},
		{
			desc: "invalid: empty params",
			msg: tokenomicstypes.MsgUpdateParams{
				Authority: sample.AccAddress(),
				Params:    tokenomicstypes.Params{},
			},
			expectedErr: tokenomicstypes.ErrTokenomicsParamInvalid,
		},
		{
			desc: "valid: address and default params",
			msg: tokenomicstypes.MsgUpdateParams{
				Authority: sample.AccAddress(),
				Params:    tokenomicstypes.DefaultParams(),
			},
		},
		{
			desc: "invalid: mint allocation params don't sum to 1",
			msg: tokenomicstypes.MsgUpdateParams{
				Authority: sample.AccAddress(),
				Params: tokenomicstypes.Params{
					MintAllocationDao:         0.1,
					MintAllocationProposer:    0.1,
					MintAllocationSupplier:    0.1,
					MintAllocationSourceOwner: 0.1,
					MintAllocationApplication: 0.1,
				},
			},
			expectedErr: tokenomicstypes.ErrTokenomicsParamInvalid,
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
