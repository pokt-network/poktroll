package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/volatile"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
)

func TestGetParams(t *testing.T) {
	k, ctx := keepertest.ProofKeeper(t)
	params := prooftypes.DefaultParams()

	require.NoError(t, k.SetParams(ctx, params))
	require.EqualValues(t, params, k.GetParams(ctx))
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
			proofRequestProbability: float64(-0.25),
			expectedErr:             prooftypes.ErrProofParamInvalid.Wrapf("invalid ProofRequestProbability: (%v)", float64(-0.25)),
		},
		{
			desc:                    "ProofRequestProbability greater than one",
			proofRequestProbability: float64(1.1),
			expectedErr:             prooftypes.ErrProofParamInvalid.Wrapf("invalid ProofRequestProbability: (%v)", float64(1.1)),
		},
		{
			desc:                    "valid ProofRequestProbability",
			proofRequestProbability: float64(0.25),
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
			proofRequirementThreshold: &cosmostypes.Coin{Denom: volatile.DenomuPOKT, Amount: math.NewInt(20)},
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
			expectedErr:         prooftypes.ErrProofParamInvalid.Wrap("invalid proof_missing_penalty denom: invalid_denom"),
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

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := prooftypes.ValidateProofMissingPenalty(test.proofMissingPenalty)
			if test.expectedErr != nil {
				require.Error(t, err)
				require.EqualError(t, err, test.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParams_ValidateProofSubmissionFee(t *testing.T) {
	invalidDenomCoin := cosmostypes.NewCoin("invalid_denom", math.NewInt(1))
	belowMinProofSubmissionFee := prooftypes.MinProofSubmissionFee.
		Sub(cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(1)))

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
			expectedErr:        prooftypes.ErrProofParamInvalid.Wrap("invalid proof_submission_fee denom: invalid_denom"),
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
			desc:               "below minimum",
			proofSubmissionFee: &belowMinProofSubmissionFee,
			expectedErr: prooftypes.ErrProofParamInvalid.Wrapf(
				"proof_submission_fee is below minimum value %s: got %s",
				prooftypes.MinProofSubmissionFee,
				belowMinProofSubmissionFee,
			),
		},
		{
			desc:               "valid",
			proofSubmissionFee: &prooftypes.MinProofSubmissionFee,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := prooftypes.ValidateProofSubmissionFee(test.proofSubmissionFee)
			if test.expectedErr != nil {
				require.Error(t, err)
				require.EqualError(t, err, test.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
