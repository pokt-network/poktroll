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
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestMsgUpdateParam_UpdateProofRequestProbabilityOnly(t *testing.T) {
	var expectedProofRequestProbability float64 = 0.1

	// Set the parameters to their default values
	k, msgSrv, ctx := setupMsgServer(t)
	defaultParams := prooftypes.DefaultParams()
	require.NoError(t, k.SetInitialParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedProofRequestProbability, defaultParams.ProofRequestProbability)

	// Update the proof request probability
	updateParamMsg := &prooftypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      prooftypes.ParamProofRequestProbability,
		AsType:    &prooftypes.MsgUpdateParam_AsFloat{AsFloat: expectedProofRequestProbability},
	}
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)

	// Assert that the onchain proof request probability is not updated yet.
	params := k.GetParams(ctx)
	require.NotEqual(t, expectedProofRequestProbability, params.ProofRequestProbability)

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	sharedParams := sharedtypes.DefaultParams()
	nextSessionStartHeight := currentHeight + int64(sharedParams.NumBlocksPerSession)
	sdkCtx = sdkCtx.WithBlockHeight(nextSessionStartHeight)

	_, err = k.BeginBlockerActivateProofParams(sdkCtx)
	require.NoError(t, err)

	params = k.GetParams(ctx)
	require.NotEqual(t, defaultParams.ProofRequestProbability, params.ProofRequestProbability)
	require.Equal(t, expectedProofRequestProbability, params.ProofRequestProbability)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, &params, string(prooftypes.KeyProofRequestProbability))
}

func TestMsgUpdateParam_UpdateProofRequirementThresholdOnly(t *testing.T) {
	var expectedProofRequirementThreshold = sdk.NewCoin(volatile.DenomuPOKT, math.NewInt(100))

	// Set the parameters to their default values
	k, msgSrv, ctx := setupMsgServer(t)
	defaultParams := prooftypes.DefaultParams()
	require.NoError(t, k.SetInitialParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedProofRequirementThreshold, defaultParams.ProofRequirementThreshold)

	// Update the proof requirement threshold
	updateParamMsg := &prooftypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      prooftypes.ParamProofRequirementThreshold,
		AsType:    &prooftypes.MsgUpdateParam_AsCoin{AsCoin: &expectedProofRequirementThreshold},
	}
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)

	// Assert that the onchain proof requirement threshold is not updated yet.
	params := k.GetParams(ctx)
	require.NotEqual(t, expectedProofRequirementThreshold, params.ProofRequirementThreshold)

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	sharedParams := sharedtypes.DefaultParams()
	nextSessionStartHeight := currentHeight + int64(sharedParams.NumBlocksPerSession)
	sdkCtx = sdkCtx.WithBlockHeight(nextSessionStartHeight)

	_, err = k.BeginBlockerActivateProofParams(sdkCtx)
	require.NoError(t, err)

	params = k.GetParams(ctx)
	require.NotEqual(t, defaultParams.ProofRequirementThreshold, params.ProofRequirementThreshold)
	require.Equal(t, &expectedProofRequirementThreshold, params.ProofRequirementThreshold)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, &params, string(prooftypes.KeyProofRequirementThreshold))
}

func TestMsgUpdateParam_UpdateProofMissingPenaltyOnly(t *testing.T) {
	expectedProofMissingPenalty := cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(500))

	// Set the parameters to their default values
	k, msgSrv, ctx := setupMsgServer(t)
	defaultParams := prooftypes.DefaultParams()
	require.NoError(t, k.SetInitialParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedProofMissingPenalty, defaultParams.ProofMissingPenalty)

	// Update the proof request probability
	updateParamMsg := &prooftypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      prooftypes.ParamProofMissingPenalty,
		AsType:    &prooftypes.MsgUpdateParam_AsCoin{AsCoin: &expectedProofMissingPenalty},
	}
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)

	// Assert that the onchain proof missing penalty is not updated yet.
	params := k.GetParams(ctx)
	require.NotEqual(t, expectedProofMissingPenalty, params.ProofMissingPenalty)

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	sharedParams := sharedtypes.DefaultParams()
	nextSessionStartHeight := currentHeight + int64(sharedParams.NumBlocksPerSession)
	sdkCtx = sdkCtx.WithBlockHeight(nextSessionStartHeight)

	_, err = k.BeginBlockerActivateProofParams(sdkCtx)
	require.NoError(t, err)

	params = k.GetParams(ctx)
	require.NotEqual(t, defaultParams.ProofMissingPenalty, params.ProofMissingPenalty)
	require.Equal(t, &expectedProofMissingPenalty, params.ProofMissingPenalty)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, &params, string(prooftypes.KeyProofMissingPenalty))
}

func TestMsgUpdateParam_UpdateProofSubmissionFeeOnly(t *testing.T) {
	expectedProofSubmissionFee := cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(1000001))

	// Set the parameters to their default values
	k, msgSrv, ctx := setupMsgServer(t)
	defaultParams := prooftypes.DefaultParams()
	require.NoError(t, k.SetInitialParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedProofSubmissionFee, defaultParams.ProofSubmissionFee)

	// Update the proof submission fee
	updateParamMsg := &prooftypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      prooftypes.ParamProofSubmissionFee,
		AsType:    &prooftypes.MsgUpdateParam_AsCoin{AsCoin: &expectedProofSubmissionFee},
	}
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)

	// Assert that the onchain proof submission fee is not updated yet.
	params := k.GetParams(ctx)
	require.NotEqual(t, expectedProofSubmissionFee, params.ProofSubmissionFee)

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	sharedParams := sharedtypes.DefaultParams()
	nextSessionStartHeight := currentHeight + int64(sharedParams.NumBlocksPerSession)
	sdkCtx = sdkCtx.WithBlockHeight(nextSessionStartHeight)

	_, err = k.BeginBlockerActivateProofParams(sdkCtx)
	require.NoError(t, err)

	params = k.GetParams(ctx)
	require.NotEqual(t, defaultParams.ProofSubmissionFee, params.ProofSubmissionFee)
	require.Equal(t, &expectedProofSubmissionFee, params.ProofSubmissionFee)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, &params, string(prooftypes.KeyProofSubmissionFee))
}
