package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
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
		desc                      string
		relayDifficultyTargetHash any
		expectedErr               error
	}{
		{
			desc:                      "invalid type",
			relayDifficultyTargetHash: int64(-1),
			expectedErr:               prooftypes.ErrProofParamInvalid.Wrapf("invalid parameter type: int64"),
		},
		{
			desc:                      "valid RelayDifficultyTargetHash",
			relayDifficultyTargetHash: prooftypes.DefaultRelayDifficultyTargetHash,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := prooftypes.ValidateRelayDifficultyTargetHash(tt.relayDifficultyTargetHash)
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

func TestParams_ValidateProofMissingPenalty(t *testing.T) {
	invalidDenomCoin := cosmostypes.NewCoin("invalid_denom", math.NewInt(1))

	tests := []struct {
		desc                string
		proofMissingPenalty any
		expectedErr         error
	}{
		{
			desc:                "invalid type",
			proofMissingPenalty: int64(-1),
			expectedErr:         prooftypes.ErrProofParamInvalid.Wrap("invalid parameter type: int64"),
		},
		{
			desc:                "invalid denomination",
			proofMissingPenalty: &invalidDenomCoin,
			expectedErr:         prooftypes.ErrProofParamInvalid.Wrap("invalid coin denom: invalid_denom"),
		},
		{
			desc:                "missing",
			proofMissingPenalty: nil,
			expectedErr:         prooftypes.ErrProofParamInvalid.Wrap("invalid parameter type: <nil>"),
		},
		{
			desc:                "missing (typed)",
			proofMissingPenalty: (*cosmostypes.Coin)(nil),
			expectedErr:         prooftypes.ErrProofParamInvalid.Wrap("missing proof_missing_penalty"),
		},
		{
			desc:                "valid",
			proofMissingPenalty: &prooftypes.DefaultProofMissingPenalty,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := prooftypes.ValidateProofMissingPenalty(tt.proofMissingPenalty)
			if tt.expectedErr != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParams_ValidateProofSubmissionFee(t *testing.T) {
	invalidDenomCoin := cosmostypes.NewCoin("invalid_denom", math.NewInt(1))

	tests := []struct {
		desc               string
		proofSubmissionFee any
		expectedErr        error
	}{
		{
			desc:               "invalid type",
			proofSubmissionFee: int64(-1),
			expectedErr:        prooftypes.ErrProofParamInvalid.Wrap("invalid parameter type: int64"),
		},
		{
			desc:               "invalid denomination",
			proofSubmissionFee: &invalidDenomCoin,
			expectedErr:        prooftypes.ErrProofParamInvalid.Wrap("invalid coin denom: invalid_denom"),
		},
		{
			desc:               "missing",
			proofSubmissionFee: nil,
			expectedErr:        prooftypes.ErrProofParamInvalid.Wrap("invalid parameter type: <nil>"),
		},
		{
			desc:               "missing (typed)",
			proofSubmissionFee: (*cosmostypes.Coin)(nil),
			expectedErr:        prooftypes.ErrProofParamInvalid.Wrap("missing proof_submission_fee"),
		},
		{
			desc:               "valid",
			proofSubmissionFee: &prooftypes.DefaultProofSubmissionFee,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := prooftypes.ValidateProofSubmissionFee(tt.proofSubmissionFee)
			if tt.expectedErr != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
