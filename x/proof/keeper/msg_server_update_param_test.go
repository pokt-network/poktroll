package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/volatile"
	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
)

func TestMsgUpdateParam_UpdateProofRequestProbabilityOnly(t *testing.T) {
	var expectedProofRequestProbability float64 = 0.1

	// Set the parameters to their default values
	k, msgSrv, ctx := setupMsgServer(t)
	defaultParams := prooftypes.DefaultParams()
	require.NoError(t, k.SetParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedProofRequestProbability, defaultParams.ProofRequestProbability)

	// Update the proof request probability
	updateParamMsg := &prooftypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      prooftypes.ParamProofRequestProbability,
		AsType:    &prooftypes.MsgUpdateParam_AsFloat{AsFloat: expectedProofRequestProbability},
	}
	res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	require.NotEqual(t, defaultParams.ProofRequestProbability, res.Params.ProofRequestProbability)
	require.Equal(t, expectedProofRequestProbability, res.Params.ProofRequestProbability)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, res.Params, string(prooftypes.KeyProofRequestProbability))
}

func TestMsgUpdateParam_UpdateProofRequirementThresholdOnly(t *testing.T) {
	var expectedProofRequirementThreshold = sdk.NewCoin(volatile.DenomuPOKT, math.NewInt(100))

	// Set the parameters to their default values
	k, msgSrv, ctx := setupMsgServer(t)
	defaultParams := prooftypes.DefaultParams()
	require.NoError(t, k.SetParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedProofRequirementThreshold, defaultParams.ProofRequirementThreshold)

	// Update the proof request probability
	updateParamMsg := &prooftypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      prooftypes.ParamProofRequirementThreshold,
		AsType:    &prooftypes.MsgUpdateParam_AsCoin{AsCoin: &expectedProofRequirementThreshold},
	}
	res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	require.NotEqual(t, defaultParams.ProofRequirementThreshold, res.Params.ProofRequirementThreshold)
	require.Equal(t, &expectedProofRequirementThreshold, res.Params.ProofRequirementThreshold)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, res.Params, string(prooftypes.KeyProofRequirementThreshold))
}

func TestMsgUpdateParam_UpdateProofMissingPenaltyOnly(t *testing.T) {
	expectedProofMissingPenalty := cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(500))

	// Set the parameters to their default values
	k, msgSrv, ctx := setupMsgServer(t)
	defaultParams := prooftypes.DefaultParams()
	require.NoError(t, k.SetParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedProofMissingPenalty, defaultParams.ProofMissingPenalty)

	// Update the proof request probability
	updateParamMsg := &prooftypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      prooftypes.ParamProofMissingPenalty,
		AsType:    &prooftypes.MsgUpdateParam_AsCoin{AsCoin: &expectedProofMissingPenalty},
	}
	res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	require.NotEqual(t, defaultParams.ProofMissingPenalty, res.Params.ProofMissingPenalty)
	require.Equal(t, &expectedProofMissingPenalty, res.Params.ProofMissingPenalty)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, res.Params, string(prooftypes.KeyProofMissingPenalty))
}

func TestMsgUpdateParam_UpdateProofSubmissionFeeOnly(t *testing.T) {
	expectedProofSubmissionFee := cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(1000001))

	// Set the parameters to their default values
	k, msgSrv, ctx := setupMsgServer(t)
	defaultParams := prooftypes.DefaultParams()
	require.NoError(t, k.SetParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedProofSubmissionFee, defaultParams.ProofSubmissionFee)

	// Update the proof request probability
	updateParamMsg := &prooftypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      prooftypes.ParamProofSubmissionFee,
		AsType:    &prooftypes.MsgUpdateParam_AsCoin{AsCoin: &expectedProofSubmissionFee},
	}
	res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	require.NotEqual(t, defaultParams.ProofSubmissionFee, res.Params.ProofSubmissionFee)
	require.Equal(t, &expectedProofSubmissionFee, res.Params.ProofSubmissionFee)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, res.Params, string(prooftypes.KeyProofSubmissionFee))
}
