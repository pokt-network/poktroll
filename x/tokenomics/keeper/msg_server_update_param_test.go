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

func TestMsgUpdateParam_UpdateMintAllocationDaoOnly(t *testing.T) {
	t.Skip("since the mint allocation percentages must sum to 1, it is not possible to modify only one of them")
}

func TestMsgUpdateParam_UpdateMintAllocationProposerOnly(t *testing.T) {
	t.Skip("since the mint allocation percentages must sum to 1, it is not possible to modify only one of them")
}

func TestMsgUpdateParam_UpdateMintAllocationSupplierOnly(t *testing.T) {
	t.Skip("since the mint allocation percentages must sum to 1, it is not possible to modify only one of them")
}

func TestMsgUpdateParam_UpdateMintAllocationSourceOwnerOnly(t *testing.T) {
	t.Skip("since the mint allocation percentages must sum to 1, it is not possible to modify only one of them")
}

func TestMsgUpdateParam_UpdateMintAllocationApplicationOnly(t *testing.T) {
	t.Skip("since the mint allocation percentages must sum to 1, it is not possible to modify only one of them")
}

func TestMsgUpdateParam_UpdateDaoRewardAddressOnly(t *testing.T) {
	expectedDaoRewardAddress := sample.AccAddress()

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
	res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	// Assert that the response contains the expected dao reward address.
	require.NotEqual(t, defaultParams.DaoRewardAddress, res.Params.DaoRewardAddress)
	require.Equal(t, expectedDaoRewardAddress, res.Params.DaoRewardAddress)

	// Assert that the on-chain dao reward address is updated.
	params := k.GetParams(ctx)
	require.Equal(t, expectedDaoRewardAddress, params.DaoRewardAddress)

	// Ensure the other parameters are unchanged
	testkeeper.AssertDefaultParamsEqualExceptFields(t, &defaultParams, res.Params, string(tokenomicstypes.KeyDaoRewardAddress))
}
