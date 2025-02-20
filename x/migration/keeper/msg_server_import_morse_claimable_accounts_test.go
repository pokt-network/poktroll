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

func TestMsgServer_ImportMorseClaimableAccounts_Success(t *testing.T) {
	k, ctx := keepertest.MigrationKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	numAccounts := 10
	_, accountState := testmigration.NewMorseStateExportAndAccountState(t, numAccounts)

	// Assert that the MorseAccountState is not set initially.
	morseClaimableAccounts := k.GetAllMorseClaimableAccounts(ctx)
	require.Equal(t, 0, len(morseClaimableAccounts))

	// Create the on-chain MorseAccountState.
	msgImportMorseClaimableAccounts, err := migrationtypes.NewMsgImportMorseClaimableAccounts(
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		*accountState,
	)
	require.NoError(t, err)

	res, err := srv.ImportMorseClaimableAccounts(ctx, msgImportMorseClaimableAccounts)
	require.NoError(t, err)

	// Assert that the response matches expectations.
	expectedUploadMsg := &migrationtypes.MsgImportMorseClaimableAccounts{
		Authority:         authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		MorseAccountState: *accountState,
	}
	expectedStateHash, err := expectedUploadMsg.MorseAccountState.GetHash()
	require.NoError(t, err)
	require.NotEmpty(t, expectedStateHash)
	require.Len(t, expectedStateHash, 32)

	expectedRes := &migrationtypes.MsgImportMorseClaimableAccountsResponse{
		StateHash:   expectedStateHash,
		NumAccounts: uint64(numAccounts),
	}
	require.Equal(t, expectedRes, res)

	// Assert that the MorseAccountState was created and matches expectations.
	morseClaimableAccounts = k.GetAllMorseClaimableAccounts(ctx)
	require.Equal(t, len(morseClaimableAccounts), numAccounts)
	require.NoError(t, err)

	// Assert that the EventCreateMorseAccountState event was emitted.
	evts := ctx.EventManager().Events()
	filteredEvts := events.FilterEvents[*migrationtypes.EventImportMorseClaimableAccounts](t, evts)
	require.Equal(t, 1, len(filteredEvts))

	expectedEvent := &migrationtypes.EventImportMorseClaimableAccounts{
		CreatedAtHeight:       ctx.BlockHeight(),
		MorseAccountStateHash: expectedStateHash,
		NumAccounts:           uint64(numAccounts),
	}
	require.Equal(t, expectedEvent, filteredEvts[0])
}

func TestMsgServer_ImportMorseClaimableAccounts_ErrorAlreadySet(t *testing.T) {
	k, ctx := keepertest.MigrationKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	// Assert that the MorseAccountState is not set initially.
	morseClaimableAccounts := k.GetAllMorseClaimableAccounts(ctx)
	require.Equal(t, 0, len(morseClaimableAccounts))

	numAccounts := 10
	_, accountState := testmigration.NewMorseStateExportAndAccountState(t, numAccounts)
	k.ImportFromMorseAccountState(ctx, accountState)

	// Assert that the MorseAccountState have been set.
	morseClaimableAccounts = k.GetAllMorseClaimableAccounts(ctx)
	require.Equal(t, 10, len(morseClaimableAccounts))

	// Assert that the MorseAccountState can ONLY be set once.
	msgImportMorseClaimableAccounts, err := migrationtypes.NewMsgImportMorseClaimableAccounts(
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		*accountState,
	)
	require.NoError(t, err)

	_, err = srv.ImportMorseClaimableAccounts(ctx, msgImportMorseClaimableAccounts)
	stat := status.Convert(err)
	require.Equal(t, codes.FailedPrecondition, stat.Code())
	require.ErrorContains(t, err, "Morse claimable accounts already imported")
}

func TestMsgServer_ImportMorseClaimableAccounts_ErrorInvalidAuthority(t *testing.T) {
	k, ctx := keepertest.MigrationKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	numAccounts := 10
	_, accountState := testmigration.NewMorseStateExportAndAccountState(t, numAccounts)

	msgImportMorseClaimableAccounts, err := migrationtypes.NewMsgImportMorseClaimableAccounts(
		authtypes.NewModuleAddress("invalid_authority").String(),
		*accountState,
	)
	require.NoError(t, err)

	_, err = srv.ImportMorseClaimableAccounts(ctx, msgImportMorseClaimableAccounts)
	stat := status.Convert(err)
	require.Equal(t, codes.PermissionDenied, stat.Code())
	require.ErrorContains(t, err, "invalid authority address")
}
