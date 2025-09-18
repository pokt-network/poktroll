package keeper_test

import (
	"testing"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

func TestMsgUpdateParam_UpdateMintAllocationPercentagesOnly(t *testing.T) {
	expectedMintAllocationPercentages := tokenomicstypes.MintAllocationPercentages{
		Dao:         0.1,
		Proposer:    0.2,
		Supplier:    0.3,
		SourceOwner: 0.4,
		Application: 0.0,
	}

	// Set the parameters to their default values
	k, msgSrv, ctx := setupMsgServer(t)
	defaultParams := tokenomicstypes.DefaultParams()
	require.NoError(t, k.SetParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedMintAllocationPercentages, defaultParams.MintAllocationPercentages)

	// Update the mint allocation percentages.
	updateParamMsg := &tokenomicstypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      tokenomicstypes.ParamMintAllocationPercentages,
		AsType:    &tokenomicstypes.MsgUpdateParam_AsMintAllocationPercentages{AsMintAllocationPercentages: &expectedMintAllocationPercentages},
	}
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	// Query the updated params from the keeper
	updatedParams := k.GetParams(ctx)
	require.NotEqual(t, defaultParams.MintAllocationPercentages, updatedParams.MintAllocationPercentages)
	require.Equal(t, expectedMintAllocationPercentages, updatedParams.MintAllocationPercentages)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, &updatedParams, string(tokenomicstypes.KeyMintAllocationPercentages))
}

func TestMsgUpdateParam_UpdateDaoRewardAddressOnly(t *testing.T) {
	expectedDaoRewardAddress := sample.AccAddressBech32()

	// Set the parameters to their default values
	k, msgSrv, ctx := setupMsgServer(t)
	defaultParams := tokenomicstypes.DefaultParams()
	require.NoError(t, k.SetParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedDaoRewardAddress, defaultParams.DaoRewardAddress)

	// Update the dao reward address.
	updateParamMsg := &tokenomicstypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      tokenomicstypes.ParamDaoRewardAddress,
		AsType:    &tokenomicstypes.MsgUpdateParam_AsString{AsString: expectedDaoRewardAddress},
	}
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	// Query the updated params from the keeper
	updatedParams := k.GetParams(ctx)
	require.NotEqual(t, defaultParams.DaoRewardAddress, updatedParams.DaoRewardAddress)
	require.Equal(t, expectedDaoRewardAddress, updatedParams.DaoRewardAddress)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, &updatedParams, string(tokenomicstypes.KeyDaoRewardAddress))
}

func TestMsgUpdateParam_UpdateGlobalInflationPerClaimOnly(t *testing.T) {
	expectedGlobalInflationPerClaim := 0.666

	// Set the parameters to their default values
	k, msgSrv, ctx := setupMsgServer(t)
	defaultParams := tokenomicstypes.DefaultParams()
	require.NoError(t, k.SetParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedGlobalInflationPerClaim, defaultParams.GlobalInflationPerClaim)

	// Update the dao reward address.
	updateParamMsg := &tokenomicstypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      tokenomicstypes.ParamGlobalInflationPerClaim,
		AsType:    &tokenomicstypes.MsgUpdateParam_AsFloat{AsFloat: expectedGlobalInflationPerClaim},
	}
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	// Query the updated params from the keeper
	updatedParams := k.GetParams(ctx)
	require.NotEqual(t, defaultParams.GlobalInflationPerClaim, updatedParams.GlobalInflationPerClaim)
	require.Equal(t, expectedGlobalInflationPerClaim, updatedParams.GlobalInflationPerClaim)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, &updatedParams, string(tokenomicstypes.KeyGlobalInflationPerClaim))
}
