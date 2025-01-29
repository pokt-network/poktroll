package keeper_test

import (
	"testing"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/testutil/events"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/testmigration"
	"github.com/pokt-network/poktroll/x/migration/keeper"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

func TestMorseAccountStateMsgServerCreate_Success(t *testing.T) {
	k, ctx := keepertest.MigrationKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	numAccounts := 10
	_, accountState := testmigration.NewMorseStateExportAndAccountState(t, numAccounts)

	// Assert that the MorseAccountState is not set initially.
	_, isFound := k.GetMorseAccountState(ctx)
	require.False(t, isFound)

	res, err := srv.CreateMorseAccountState(ctx, &migrationtypes.MsgCreateMorseAccountState{
		Authority:         authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		MorseAccountState: *accountState,
	})
	require.NoError(t, err)

	expectedUploadMsg := &migrationtypes.MsgCreateMorseAccountState{
		Authority:         authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		MorseAccountState: *accountState,
	}
	expectedStateHash, err := expectedUploadMsg.MorseAccountState.GetHash()
	require.NoError(t, err)
	require.NotEmpty(t, expectedStateHash)
	require.Len(t, expectedStateHash, 32)

	expectedRes := &migrationtypes.MsgCreateMorseAccountStateResponse{
		StateHash:   expectedStateHash,
		NumAccounts: uint64(numAccounts),
	}
	require.Equal(t, expectedRes, res)

	MorseAccountState, isFound := k.GetMorseAccountState(ctx)
	require.True(t, isFound)
	require.NoError(t, err)

	actualStateHash, err := MorseAccountState.GetHash()
	require.NoError(t, err)
	require.Equal(t, expectedRes.StateHash, actualStateHash)
	require.Equal(t, int(expectedRes.NumAccounts), len(MorseAccountState.GetAccounts()))

	evts := ctx.EventManager().Events()
	filteredEvts := events.FilterEvents[*migrationtypes.EventCreateMorseAccountState](t, evts)
	require.Equal(t, 1, len(filteredEvts))

	expectedEvent := &migrationtypes.EventCreateMorseAccountState{
		Height:    ctx.BlockHeight(),
		StateHash: expectedStateHash,
	}
	require.Equal(t, expectedEvent, filteredEvts[0])
}

func TestMorseAccountStateMsgServerCreate_Error(t *testing.T) {
	k, ctx := keepertest.MigrationKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	numAccounts := 10
	_, accountState := testmigration.NewMorseStateExportAndAccountState(t, numAccounts)
	k.SetMorseAccountState(ctx, *accountState)

	// Assert that the MorseAccountState is set initially.
	_, isFound := k.GetMorseAccountState(ctx)
	require.True(t, isFound)

	// Assert that the MorseAccountState can ONLY be set once.
	_, err := srv.CreateMorseAccountState(ctx, &migrationtypes.MsgCreateMorseAccountState{
		Authority:         authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		MorseAccountState: *accountState,
	})
	t.Log(err)

	stat := status.Convert(err)
	require.Equal(t, codes.FailedPrecondition, stat.Code())
	require.ErrorContains(t, err, "already set")
}
