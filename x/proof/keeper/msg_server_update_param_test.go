package keeper_test

import (
	"testing"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
)

func TestMsgUpdateParam_UpdateMinRelayDifficultyBitsOnly(t *testing.T) {
	var expectedMinRelayDifficultyBits uint64 = 8

	// Set the parameters to their default values
	k, msgSrv, ctx := setupMsgServer(t)
	defaultParams := prooftypes.DefaultParams()
	require.NoError(t, k.SetParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedMinRelayDifficultyBits, defaultParams.MinRelayDifficultyBits)

	// Update the min relay difficulty bits
	updateParamMsg := &prooftypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      prooftypes.ParamMinRelayDifficultyBits,
		AsType:    &prooftypes.MsgUpdateParam_AsInt64{AsInt64: int64(expectedMinRelayDifficultyBits)},
	}
	res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	require.Equal(t, expectedMinRelayDifficultyBits, res.Params.MinRelayDifficultyBits)
}

func TestMsgUpdateParam_UpdateProofRequestProbabilityOnly(t *testing.T) {
	var expectedProofRequestProbability float32 = 0.1

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

	require.Equal(t, expectedProofRequestProbability, res.Params.ProofRequestProbability)
}

func TestMsgUpdateParam_UpdateProofRequirementThresholdOnly(t *testing.T) {
	var expectedProofRequirementThreshold uint64 = 100

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
		AsType:    &prooftypes.MsgUpdateParam_AsInt64{AsInt64: int64(expectedProofRequirementThreshold)},
	}
	res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	require.Equal(t, expectedProofRequirementThreshold, res.Params.ProofRequirementThreshold)
}
