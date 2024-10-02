package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/x/proof/types"
)

func TestMsgUpdateParams(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	params := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, params))

	// default params
	tests := []struct {
		desc           string
		params         *types.MsgUpdateParams
		shouldError    bool
		expectedErrMsg string
	}{
		{
			desc: "invalid: authority address invalid",
			params: &types.MsgUpdateParams{
				Authority: "invalid",
				Params:    params,
			},
			shouldError:    true,
			expectedErrMsg: "invalid authority",
		},
		{
			desc: "invalid: send empty params",
			params: &types.MsgUpdateParams{
				Authority: k.GetAuthority(),
				Params:    types.Params{},
			},
			shouldError: true,
		},
		{
			desc: "valid: send minimal params",
			params: &types.MsgUpdateParams{
				Authority: k.GetAuthority(),
				Params: types.Params{
					ProofRequirementThreshold: &types.DefaultProofRequirementThreshold,
					ProofMissingPenalty:       &types.DefaultProofMissingPenalty,
					ProofSubmissionFee:        &types.MinProofSubmissionFee,
					RelayDifficultyTargetHash: types.DefaultRelayDifficultyTargetHash,
				},
			},
			shouldError: false,
		},
		{
			desc: "valid: send default params",
			params: &types.MsgUpdateParams{
				Authority: k.GetAuthority(),
				Params:    params,
			},
			shouldError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			_, err := ms.UpdateParams(ctx, test.params)

			if test.shouldError {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.expectedErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
