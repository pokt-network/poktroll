package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/pocket/cmd/pocketd/cmd"
	"github.com/pokt-network/pocket/testutil/sample"
	tokenomicstypes "github.com/pokt-network/pocket/x/tokenomics/types"
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
					MintAllocationPercentages: tokenomicstypes.MintAllocationPercentages{
						Dao:         0.1,
						Proposer:    0.1,
						Supplier:    0.1,
						SourceOwner: 0.1,
						Application: 0.1,
					},
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
