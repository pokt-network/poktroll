package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/pocket"
	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
)

func TestMsgUpdateParam_UpdateProofRequestProbabilityOnly(t *testing.T) {
	expectedProofRequestProbability := 0.1

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
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	// Query the updated params from the keeper
	updatedParams := k.GetParams(ctx)
	require.NotEqual(t, defaultParams.ProofRequestProbability, updatedParams.ProofRequestProbability)
	require.Equal(t, expectedProofRequestProbability, updatedParams.ProofRequestProbability)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, &updatedParams, string(prooftypes.KeyProofRequestProbability))
}

func TestMsgUpdateParam_UpdateProofRequirementThresholdOnly(t *testing.T) {
	var expectedProofRequirementThreshold = cosmostypes.NewCoin(pocket.DenomuPOKT, math.NewInt(100))

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
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	// Query the updated params from the keeper
	updatedParams := k.GetParams(ctx)
	require.NotEqual(t, defaultParams.ProofRequirementThreshold, updatedParams.ProofRequirementThreshold)
	require.Equal(t, &expectedProofRequirementThreshold, updatedParams.ProofRequirementThreshold)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, &updatedParams, string(prooftypes.KeyProofRequirementThreshold))
}

func TestMsgUpdateParam_UpdateProofMissingPenaltyOnly(t *testing.T) {
	expectedProofMissingPenalty := cosmostypes.NewCoin(pocket.DenomuPOKT, math.NewInt(500))

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
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	// Query the updated params from the keeper
	updatedParams := k.GetParams(ctx)
	require.NotEqual(t, defaultParams.ProofMissingPenalty, updatedParams.ProofMissingPenalty)
	require.Equal(t, &expectedProofMissingPenalty, updatedParams.ProofMissingPenalty)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, &updatedParams, string(prooftypes.KeyProofMissingPenalty))
}

func TestMsgUpdateParam_UpdateProofSubmissionFeeOnly(t *testing.T) {
	expectedProofSubmissionFee := cosmostypes.NewCoin(pocket.DenomuPOKT, math.NewInt(1000001))

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
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	// Query the updated params from the keeper
	updatedParams := k.GetParams(ctx)
	require.NotEqual(t, defaultParams.ProofSubmissionFee, updatedParams.ProofSubmissionFee)
	require.Equal(t, &expectedProofSubmissionFee, updatedParams.ProofSubmissionFee)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, &updatedParams, string(prooftypes.KeyProofSubmissionFee))
}
