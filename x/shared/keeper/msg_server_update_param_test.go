package keeper_test

import (
	"testing"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/x/shared/keeper"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestMsgUpdateParam_UpdateNumBlocksPerSession(t *testing.T) {
	var expectedNumBlocksPerSession int64 = 8

	k, ctx := keepertest.SharedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)

	// Set the parameters to their default values
	defaultParams := sharedtypes.DefaultParams()
	require.NoError(t, k.SetParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, uint64(expectedNumBlocksPerSession), defaultParams.NumBlocksPerSession)

	// Update the number of blocks per session
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamNumBlocksPerSession,
		AsType:    &sharedtypes.MsgUpdateParam_AsInt64{AsInt64: expectedNumBlocksPerSession},
	}
	res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	require.Equal(t, uint64(expectedNumBlocksPerSession), res.Params.NumBlocksPerSession)

	// Ensure the other parameters are unchanged
	require.Equal(t, defaultParams.GetClaimWindowOpenOffsetBlocks(), res.Params.GetClaimWindowOpenOffsetBlocks())
	require.Equal(t, defaultParams.GetClaimWindowCloseOffsetBlocks(), res.Params.GetClaimWindowCloseOffsetBlocks())
	require.Equal(t, defaultParams.GetProofWindowOpenOffsetBlocks(), res.Params.GetProofWindowOpenOffsetBlocks())
	require.Equal(t, defaultParams.GetProofWindowCloseOffsetBlocks(), res.Params.GetProofWindowCloseOffsetBlocks())
	require.Equal(t, defaultParams.GetGracePeriodEndOffsetBlocks(), res.Params.GetGracePeriodEndOffsetBlocks())
}

func TestMsgUpdateParam_UpdateClaimWindowOpenOffsetBlocks(t *testing.T) {
	var expectedClaimWindowOpenOffestBlocks int64 = 4

	k, ctx := keepertest.SharedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)

	// Set the parameters to their default values
	defaultParams := sharedtypes.DefaultParams()
	require.NoError(t, k.SetParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, uint64(expectedClaimWindowOpenOffestBlocks), defaultParams.ClaimWindowOpenOffsetBlocks)

	// Update the claim window open offset blocks param
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamClaimWindowOpenOffsetBlocks,
		AsType:    &sharedtypes.MsgUpdateParam_AsInt64{AsInt64: expectedClaimWindowOpenOffestBlocks},
	}
	res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	require.Equal(t, uint64(expectedClaimWindowOpenOffestBlocks), res.Params.ClaimWindowOpenOffsetBlocks)

	// Ensure the other parameters are unchanged
	require.Equal(t, defaultParams.GetNumBlocksPerSession(), res.Params.GetNumBlocksPerSession())
	require.Equal(t, defaultParams.GetClaimWindowCloseOffsetBlocks(), res.Params.GetClaimWindowCloseOffsetBlocks())
	require.Equal(t, defaultParams.GetProofWindowOpenOffsetBlocks(), res.Params.GetProofWindowOpenOffsetBlocks())
	require.Equal(t, defaultParams.GetProofWindowCloseOffsetBlocks(), res.Params.GetProofWindowCloseOffsetBlocks())
	require.Equal(t, defaultParams.GetGracePeriodEndOffsetBlocks(), res.Params.GetGracePeriodEndOffsetBlocks())
}

func TestMsgUpdateParam_UpdateClaimWindowCloseOffsetBlocks(t *testing.T) {
	var expectedClaimWindowCloseOffestBlocks int64 = 8

	k, ctx := keepertest.SharedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)

	// Set the parameters to their default values
	defaultParams := sharedtypes.DefaultParams()
	require.NoError(t, k.SetParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, uint64(expectedClaimWindowCloseOffestBlocks), defaultParams.ClaimWindowCloseOffsetBlocks)

	// Update the claim window close offset blocks param
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamClaimWindowCloseOffsetBlocks,
		AsType:    &sharedtypes.MsgUpdateParam_AsInt64{AsInt64: expectedClaimWindowCloseOffestBlocks},
	}
	res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	require.Equal(t, uint64(expectedClaimWindowCloseOffestBlocks), res.Params.ClaimWindowCloseOffsetBlocks)

	// Ensure the other parameters are unchanged
	require.Equal(t, defaultParams.GetNumBlocksPerSession(), res.Params.GetNumBlocksPerSession())
	require.Equal(t, defaultParams.GetClaimWindowOpenOffsetBlocks(), res.Params.GetClaimWindowOpenOffsetBlocks())
	require.Equal(t, defaultParams.GetProofWindowOpenOffsetBlocks(), res.Params.GetProofWindowOpenOffsetBlocks())
	require.Equal(t, defaultParams.GetProofWindowCloseOffsetBlocks(), res.Params.GetProofWindowCloseOffsetBlocks())
	require.Equal(t, defaultParams.GetGracePeriodEndOffsetBlocks(), res.Params.GetGracePeriodEndOffsetBlocks())
}

func TestMsgUpdateParam_UpdateProofWindowOpenOffsetBlocks(t *testing.T) {
	var expectedProofWindowOpenOffestBlocks int64 = 8

	k, ctx := keepertest.SharedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)

	// Set the parameters to their default values
	defaultParams := sharedtypes.DefaultParams()
	require.NoError(t, k.SetParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, uint64(expectedProofWindowOpenOffestBlocks), defaultParams.ProofWindowOpenOffsetBlocks)

	// Update the proof window open offset blocks param
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamProofWindowOpenOffsetBlocks,
		AsType:    &sharedtypes.MsgUpdateParam_AsInt64{AsInt64: expectedProofWindowOpenOffestBlocks},
	}
	res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	require.Equal(t, uint64(expectedProofWindowOpenOffestBlocks), res.Params.ProofWindowOpenOffsetBlocks)

	// Ensure the other parameters are unchanged
	require.Equal(t, defaultParams.GetNumBlocksPerSession(), res.Params.GetNumBlocksPerSession())
	require.Equal(t, defaultParams.GetClaimWindowOpenOffsetBlocks(), res.Params.GetClaimWindowOpenOffsetBlocks())
	require.Equal(t, defaultParams.GetClaimWindowCloseOffsetBlocks(), res.Params.GetClaimWindowCloseOffsetBlocks())
	require.Equal(t, defaultParams.GetProofWindowCloseOffsetBlocks(), res.Params.GetProofWindowCloseOffsetBlocks())
	require.Equal(t, defaultParams.GetGracePeriodEndOffsetBlocks(), res.Params.GetGracePeriodEndOffsetBlocks())
}

func TestMsgUpdateParam_UpdateProofWindowCloseOffsetBlocks(t *testing.T) {
	var expectedProofWindowCloseOffestBlocks int64 = 8

	k, ctx := keepertest.SharedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)

	// Set the parameters to their default values
	defaultParams := sharedtypes.DefaultParams()
	require.NoError(t, k.SetParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, uint64(expectedProofWindowCloseOffestBlocks), defaultParams.ProofWindowCloseOffsetBlocks)

	// Update the proof window close offset blocks param
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamProofWindowCloseOffsetBlocks,
		AsType:    &sharedtypes.MsgUpdateParam_AsInt64{AsInt64: expectedProofWindowCloseOffestBlocks},
	}
	res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	require.Equal(t, uint64(expectedProofWindowCloseOffestBlocks), res.Params.ProofWindowCloseOffsetBlocks)

	// Ensure the other parameters are unchanged
	require.Equal(t, defaultParams.GetNumBlocksPerSession(), res.Params.GetNumBlocksPerSession())
	require.Equal(t, defaultParams.GetClaimWindowOpenOffsetBlocks(), res.Params.GetClaimWindowOpenOffsetBlocks())
	require.Equal(t, defaultParams.GetClaimWindowCloseOffsetBlocks(), res.Params.GetClaimWindowCloseOffsetBlocks())
	require.Equal(t, defaultParams.GetProofWindowOpenOffsetBlocks(), res.Params.GetProofWindowOpenOffsetBlocks())
	require.Equal(t, defaultParams.GetGracePeriodEndOffsetBlocks(), res.Params.GetGracePeriodEndOffsetBlocks())
}

func TestMsgUpdateParam_UpdateGracePeriodEndOffsetBlocks(t *testing.T) {
	var expectedGracePeriodEndOffestBlocks int64 = 8

	k, ctx := keepertest.SharedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)

	// Set the parameters to their default values
	defaultParams := sharedtypes.DefaultParams()
	require.NoError(t, k.SetParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, uint64(expectedGracePeriodEndOffestBlocks), defaultParams.GetGracePeriodEndOffsetBlocks())

	// Update the proof window close offset blocks param
	updateParamMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamGracePeriodEndOffsetBlocks,
		AsType:    &sharedtypes.MsgUpdateParam_AsInt64{AsInt64: expectedGracePeriodEndOffestBlocks},
	}
	res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	require.Equal(t, uint64(expectedGracePeriodEndOffestBlocks), res.Params.GetGracePeriodEndOffsetBlocks())

	// Ensure the other parameters are unchanged
	require.Equal(t, defaultParams.GetNumBlocksPerSession(), res.Params.GetNumBlocksPerSession())
	require.Equal(t, defaultParams.GetClaimWindowOpenOffsetBlocks(), res.Params.GetClaimWindowOpenOffsetBlocks())
	require.Equal(t, defaultParams.GetClaimWindowCloseOffsetBlocks(), res.Params.GetClaimWindowCloseOffsetBlocks())
	require.Equal(t, defaultParams.GetProofWindowOpenOffsetBlocks(), res.Params.GetProofWindowOpenOffsetBlocks())
	require.Equal(t, defaultParams.GetProofWindowCloseOffsetBlocks(), res.Params.GetProofWindowCloseOffsetBlocks())
}
