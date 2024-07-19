package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/proto/types/proof"
	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
)

func TestMsgUpdateParam_UpdateMinRelayDifficultyBitsOnly(t *testing.T) {
	var expectedMinRelayDifficultyBits uint64 = 8

	// Set the parameters to their default values
	k, msgSrv, ctx := setupMsgServer(t)
	defaultParams := proof.DefaultParams()
	require.NoError(t, k.SetParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedMinRelayDifficultyBits, defaultParams.MinRelayDifficultyBits)

	// Update the min relay difficulty bits
	updateParamMsg := &proof.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      proof.ParamMinRelayDifficultyBits,
		AsType:    &proof.MsgUpdateParam_AsInt64{AsInt64: int64(expectedMinRelayDifficultyBits)},
	}
	res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	require.NotEqual(t, defaultParams.MinRelayDifficultyBits, res.Params.MinRelayDifficultyBits)
	require.Equal(t, expectedMinRelayDifficultyBits, res.Params.MinRelayDifficultyBits)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, res.Params, "MinRelayDifficultyBits")
}

func TestMsgUpdateParam_UpdateProofRequestProbabilityOnly(t *testing.T) {
	var expectedProofRequestProbability float32 = 0.1

	// Set the parameters to their default values
	k, msgSrv, ctx := setupMsgServer(t)
	defaultParams := proof.DefaultParams()
	require.NoError(t, k.SetParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedProofRequestProbability, defaultParams.ProofRequestProbability)

	// Update the proof request probability
	updateParamMsg := &proof.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      proof.ParamProofRequestProbability,
		AsType:    &proof.MsgUpdateParam_AsFloat{AsFloat: expectedProofRequestProbability},
	}
	res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	require.NotEqual(t, defaultParams.ProofRequestProbability, res.Params.ProofRequestProbability)
	require.Equal(t, expectedProofRequestProbability, res.Params.ProofRequestProbability)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, res.Params, "ProofRequestProbability")
}

func TestMsgUpdateParam_UpdateProofRequirementThresholdOnly(t *testing.T) {
	var expectedProofRequirementThreshold uint64 = 100

	// Set the parameters to their default values
	k, msgSrv, ctx := setupMsgServer(t)
	defaultParams := proof.DefaultParams()
	require.NoError(t, k.SetParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedProofRequirementThreshold, defaultParams.ProofRequirementThreshold)

	// Update the proof request probability
	updateParamMsg := &proof.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      proof.ParamProofRequirementThreshold,
		AsType:    &proof.MsgUpdateParam_AsInt64{AsInt64: int64(expectedProofRequirementThreshold)},
	}
	res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	require.NotEqual(t, defaultParams.ProofRequirementThreshold, res.Params.ProofRequirementThreshold)
	require.Equal(t, expectedProofRequirementThreshold, res.Params.ProofRequirementThreshold)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, res.Params, "ProofRequirementThreshold")
}

func TestMsgUpdateParam_UpdateProofMissingPenaltyOnly(t *testing.T) {
	expectedProofMissingPenalty := cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(500))

	// Set the parameters to their default values
	k, msgSrv, ctx := setupMsgServer(t)
	defaultParams := proof.DefaultParams()
	require.NoError(t, k.SetParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedProofMissingPenalty, defaultParams.ProofMissingPenalty)

	// Update the proof request probability
	updateParamMsg := &proof.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      proof.ParamProofMissingPenalty,
		AsType:    &proof.MsgUpdateParam_AsCoin{AsCoin: &expectedProofMissingPenalty},
	}
	res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	require.NotEqual(t, defaultParams.ProofMissingPenalty, res.Params.ProofMissingPenalty)
	require.Equal(t, &expectedProofMissingPenalty, res.Params.ProofMissingPenalty)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, res.Params, "ProofMissingPenalty")
}
