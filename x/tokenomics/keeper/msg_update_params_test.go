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

	validParams := tokenomicstypes.DefaultParams()
	validParams.DaoRewardAddress = sample.AccAddress()

	tests := []struct {
		desc string

		req *tokenomicstypes.MsgUpdateParams

		shouldError    bool
		expectedErrMsg string
	}{
		{
			desc: "invalid: malformed authority address",

			req: &tokenomicstypes.MsgUpdateParams{
				Authority: "invalid",
				Params:    tokenomicstypes.DefaultParams(),
			},

			shouldError:    true,
			expectedErrMsg: "invalid authority",
		},
		{
			desc: "invalid: incorrect authority address",

			req: &tokenomicstypes.MsgUpdateParams{
				Authority: sample.AccAddress(),
				Params:    validParams,
			},

			shouldError:    true,
			expectedErrMsg: "the provided authority address does not match the onchain governance address",
		},
		{
			desc: "invalid: dao reward address missing",

			req: &tokenomicstypes.MsgUpdateParams{
				Authority: tokenomicsKeeper.GetAuthority(),
				Params: tokenomicstypes.Params{
					// DaoRewardAddress MUST NOT be empty string
					// when MintAllocationDao is greater than 0.
					DaoRewardAddress: "",

					// MintAllocationXXX params MUST sum to 1.
					MintAllocationPercentages: tokenomicstypes.MintAllocationPercentages{
						Dao:         0,
						Proposer:    0.1,
						Supplier:    0.1,
						SourceOwner: 0.1,
						Application: 0.7,
					},
				},
			},

			shouldError:    true,
			expectedErrMsg: "empty address string is not allowed",
		},
		{
			desc: "invalid: negative global inflation per claim",

			req: &tokenomicstypes.MsgUpdateParams{
				Authority: tokenomicsKeeper.GetAuthority(),
				Params: tokenomicstypes.Params{
					// GlobalInflationPerClaim MUST be positive.
					GlobalInflationPerClaim: -0.1,

					// DaoRewardAddress MUST NOT be empty string
					// when MintAllocationDao is greater than 0.
					DaoRewardAddress: sample.AccAddress(),

					// MintAllocationXXX params MUST sum to 1.
					MintAllocationPercentages: tokenomicstypes.MintAllocationPercentages{
						Dao:         0,
						Proposer:    0.1,
						Supplier:    0.1,
						SourceOwner: 0.1,
						Application: 0.7,
					},
				},
			},

			shouldError:    true,
			expectedErrMsg: "GlobalInflationPerClaim must be greater than or equal to 0:",
		},
		{
			desc: "valid: successful param update",

			req: &tokenomicstypes.MsgUpdateParams{
				Authority: tokenomicsKeeper.GetAuthority(),
				Params: tokenomicstypes.Params{
					MintAllocationPercentages: tokenomicstypes.MintAllocationPercentages{
						Dao:         0.1,
						Proposer:    0.1,
						Supplier:    0.1,
						SourceOwner: 0.1,
						Application: 0.6,
					},
					DaoRewardAddress: sample.AccAddress(),
				},
			},

			shouldError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			updateRes, err := srv.UpdateParams(ctx, test.req)
			if test.shouldError {
				require.Error(t, err)
				require.ErrorContains(t, err, test.expectedErrMsg)
			} else {
				require.Equal(t, &test.req.Params, updateRes.GetParams())
				require.Nil(t, err)
			}
		})
	}
}
