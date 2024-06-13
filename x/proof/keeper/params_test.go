package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
)

func TestGetParams(t *testing.T) {
	k, ctx := keepertest.ProofKeeper(t)
	params := prooftypes.DefaultParams()

	require.NoError(t, k.SetParams(ctx, params))
	require.EqualValues(t, params, k.GetParams(ctx))
}
func TestParams_ValidateMinRelayDifficulty(t *testing.T) {
	tests := []struct {
		desc                   string
		minRelayDifficultyBits any
		expectedErr            error
	}{
		{
			desc:                   "invalid type",
			minRelayDifficultyBits: int64(-1),
			expectedErr:            prooftypes.ErrProofParamInvalid.Wrapf("invalid parameter type: int64"),
		},
		{
			desc:                   "valid MinRelayDifficultyBits",
			minRelayDifficultyBits: uint64(4),
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := prooftypes.ValidateMinRelayDifficultyBits(tt.minRelayDifficultyBits)
			if tt.expectedErr != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParams_ValidateProofRequestProbability(t *testing.T) {
	tests := []struct {
		desc                    string
		proofRequestProbability any
		expectedErr             error
	}{
		{
			desc:                    "invalid type",
			proofRequestProbability: "invalid",
			expectedErr:             prooftypes.ErrProofParamInvalid.Wrapf("invalid parameter type: %T", "invalid"),
		},
		{
			desc:                    "ProofRequestProbability less than zero",
			proofRequestProbability: float32(-0.25),
			expectedErr:             prooftypes.ErrProofParamInvalid.Wrapf("invalid ProofRequestProbability: (%v)", float32(-0.25)),
		},
		{
			desc:                    "ProofRequestProbability greater than one",
			proofRequestProbability: float32(1.1),
			expectedErr:             prooftypes.ErrProofParamInvalid.Wrapf("invalid ProofRequestProbability: (%v)", float32(1.1)),
		},
		{
			desc:                    "valid ProofRequestProbability",
			proofRequestProbability: float32(0.25),
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := prooftypes.ValidateProofRequestProbability(tt.proofRequestProbability)
			if tt.expectedErr != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParams_ValidateProofRequirementThreshold(t *testing.T) {
	tests := []struct {
		desc                      string
		proofRequirementThreshold any
		expectedErr               error
	}{
		{
			desc:                      "invalid type",
			proofRequirementThreshold: int64(-1),
			expectedErr:               prooftypes.ErrProofParamInvalid.Wrapf("invalid parameter type: int64"),
		},
		{
			desc:                      "valid ProofRequirementThreshold",
			proofRequirementThreshold: uint64(20),
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := prooftypes.ValidateProofRequirementThreshold(tt.proofRequirementThreshold)
			if tt.expectedErr != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
