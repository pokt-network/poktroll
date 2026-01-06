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

// TestMsgUpdateParam_UpdateMintRatioOnly tests updating the MintRatio parameter (PIP-41).
func TestMsgUpdateParam_UpdateMintRatioOnly(t *testing.T) {
	// PIP-41 target: 0.975 (2.5% deflation)
	expectedMintRatio := 0.975

	// Set the parameters to their default values
	k, msgSrv, ctx := setupMsgServer(t)
	defaultParams := tokenomicstypes.DefaultParams()
	require.NoError(t, k.SetParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, expectedMintRatio, defaultParams.MintRatio)

	// Update the mint ratio.
	updateParamMsg := &tokenomicstypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      tokenomicstypes.ParamMintRatio,
		AsType:    &tokenomicstypes.MsgUpdateParam_AsFloat{AsFloat: expectedMintRatio},
	}
	_, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	// Query the updated params from the keeper
	updatedParams := k.GetParams(ctx)
	require.NotEqual(t, defaultParams.MintRatio, updatedParams.MintRatio)
	require.Equal(t, expectedMintRatio, updatedParams.MintRatio)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, &updatedParams, string(tokenomicstypes.KeyMintRatio))
}

// TestMsgUpdateParam_UpdateMintRatioInvalid tests that invalid MintRatio values are rejected.
func TestMsgUpdateParam_UpdateMintRatioInvalid(t *testing.T) {
	tests := []struct {
		desc         string
		mintRatio    float64
		expectErrMsg string
	}{
		{
			desc:         "greater than 1",
			mintRatio:    1.1,
			expectErrMsg: "mint_ratio must be in range (0, 1]",
		},
		{
			desc:         "negative value",
			mintRatio:    -0.5,
			expectErrMsg: "mint_ratio must be in range (0, 1]",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			// Set the parameters to their default values
			k, msgSrv, ctx := setupMsgServer(t)
			defaultParams := tokenomicstypes.DefaultParams()
			require.NoError(t, k.SetParams(ctx, defaultParams))

			// Attempt to update the mint ratio with invalid value
			updateParamMsg := &tokenomicstypes.MsgUpdateParam{
				Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
				Name:      tokenomicstypes.ParamMintRatio,
				AsType:    &tokenomicstypes.MsgUpdateParam_AsFloat{AsFloat: test.mintRatio},
			}
			_, err := msgSrv.UpdateParam(ctx, updateParamMsg)
			require.Error(t, err)
			require.Contains(t, err.Error(), test.expectErrMsg)

			// Ensure the parameter was not updated
			updatedParams := k.GetParams(ctx)
			require.Equal(t, defaultParams.MintRatio, updatedParams.MintRatio)
		})
	}
}
