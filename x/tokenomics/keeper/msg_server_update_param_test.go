package keeper_test

import (
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
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
	require.NoError(t, k.SetInitialParams(ctx, defaultParams))

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

	// Assert that the onchain mint allocation percentages is not updated yet.
	params := k.GetParams(ctx)
	require.NotEqual(t, expectedMintAllocationPercentages, params.MintAllocationPercentages)

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	sharedParams := sharedtypes.DefaultParams()
	nextSessionStartHeight := currentHeight + int64(sharedParams.NumBlocksPerSession)
	sdkCtx = sdkCtx.WithBlockHeight(nextSessionStartHeight)

	_, err = k.BeginBlockerActivateTokenomicsParams(sdkCtx)
	require.NoError(t, err)

	// Assert that the onchain mint allocation percentages is updated.
	params = k.GetParams(ctx)
	require.NotEqual(t, defaultParams.MintAllocationPercentages, params.MintAllocationPercentages)
	require.Equal(t, expectedMintAllocationPercentages, params.MintAllocationPercentages)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, &params, string(tokenomicstypes.KeyMintAllocationPercentages))
}

func TestMsgUpdateParam_UpdateDaoRewardAddressOnly(t *testing.T) {
	expectedDaoRewardAddress := sample.AccAddress()

	// Set the parameters to their default values
	k, msgSrv, ctx := setupMsgServer(t)
	defaultParams := tokenomicstypes.DefaultParams()
	require.NoError(t, k.SetInitialParams(ctx, defaultParams))

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

	// Assert that the onchain dao reward address is not updated yet.
	params := k.GetParams(ctx)
	require.NotEqual(t, expectedDaoRewardAddress, params.DaoRewardAddress)

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	sharedParams := sharedtypes.DefaultParams()
	nextSessionStartHeight := currentHeight + int64(sharedParams.NumBlocksPerSession)
	sdkCtx = sdkCtx.WithBlockHeight(nextSessionStartHeight)

	_, err = k.BeginBlockerActivateTokenomicsParams(sdkCtx)
	require.NoError(t, err)

	// Assert that the onchain dao reward address is updated.
	params = k.GetParams(ctx)
	// Assert that the response contains the expected dao reward address.
	require.NotEqual(t, defaultParams.DaoRewardAddress, params.DaoRewardAddress)
	require.Equal(t, expectedDaoRewardAddress, params.DaoRewardAddress)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, &params, string(tokenomicstypes.KeyDaoRewardAddress))
}

func TestMsgUpdateParam_UpdateGlobalInflationPerClaimOnly(t *testing.T) {
	expectedGlobalInflationPerClaim := 0.666

	// Set the parameters to their default values
	k, msgSrv, ctx := setupMsgServer(t)
	defaultParams := tokenomicstypes.DefaultParams()
	require.NoError(t, k.SetInitialParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedGlobalInflationPerClaim, defaultParams.GlobalInflationPerClaim)

	// Update the global inflation per claim.
	updateParamMsg := &tokenomicstypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      tokenomicstypes.ParamGlobalInflationPerClaim,
		AsType:    &tokenomicstypes.MsgUpdateParam_AsFloat{AsFloat: expectedGlobalInflationPerClaim},
	}
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	// Assert that the onchain global inflation per claim is not yet updated.
	params := k.GetParams(ctx)
	require.NotEqual(t, expectedGlobalInflationPerClaim, params.GlobalInflationPerClaim)

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	sharedParams := sharedtypes.DefaultParams()
	nextSessionStartHeight := currentHeight + int64(sharedParams.NumBlocksPerSession)
	sdkCtx = sdkCtx.WithBlockHeight(nextSessionStartHeight)

	_, err = k.BeginBlockerActivateTokenomicsParams(sdkCtx)
	require.NoError(t, err)

	// Assert that the onchain global inflation per claim is updated.
	params = k.GetParams(ctx)
	require.NotEqual(t, defaultParams.GlobalInflationPerClaim, params.GlobalInflationPerClaim)
	require.Equal(t, expectedGlobalInflationPerClaim, params.GlobalInflationPerClaim)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, &params, string(tokenomicstypes.KeyGlobalInflationPerClaim))
}
