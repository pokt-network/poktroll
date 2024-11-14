package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

func TestMsgUpdateParams(t *testing.T) {
	tokenomicsKeeper, srv, ctx := setupMsgServer(t)
	require.NoError(t, tokenomicsKeeper.SetParams(ctx, tokenomicstypes.DefaultParams()))

	tests := []struct {
		desc string

		req *tokenomicstypes.MsgUpdateParams

		shouldError    bool
		expectedErrMsg string
	}{
		{
			desc: "invalid authority address",

			req: &tokenomicstypes.MsgUpdateParams{
				Authority: "invalid",
				Params:    tokenomicstypes.DefaultParams(),
			},

			shouldError:    true,
			expectedErrMsg: "invalid authority",
		},
		{
			desc: "incorrect authority address",

			req: &tokenomicstypes.MsgUpdateParams{
				Authority: sample.AccAddress(),
				Params:    tokenomicstypes.DefaultParams(),
			},

			shouldError:    true,
			expectedErrMsg: "the provided authority address does not match the on-chain governance address",
		},
		{
			desc: "successful param update",

			req: &tokenomicstypes.MsgUpdateParams{
				Authority: tokenomicsKeeper.GetAuthority(),
				Params: tokenomicstypes.Params{
					MintAllocationDao:         0.1,
					MintAllocationProposer:    0.1,
					MintAllocationSupplier:    0.1,
					MintAllocationSourceOwner: 0.1,
					MintAllocationApplication: 0.6,
				},
			},

			shouldError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			_, err := srv.UpdateParams(ctx, test.req)
			if test.shouldError {
				require.Error(t, err)
				require.ErrorContains(t, err, test.expectedErrMsg)
			} else {
				require.Nil(t, err)
			}
		})
	}
}
