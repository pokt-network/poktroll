package cmd

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"

	cmtjson "github.com/cometbft/cometbft/libs/json"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/cmd/logger"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/testutil/testmigration"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

// Note: This test should not be included in CI due to its size (90GB).
// Users should manually run wget to download the data and verify it on their computer.
// Reference: https://github.com/pokt-network/poktroll/pull/1039#discussion_r1947036729

func init() {
	logger.Logger = polyzero.NewLogger(polyzero.WithLevel(polyzero.DebugLevel))
	numAccountsPerDebugLog = 1
}

func TestCollectMorseAccounts(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "morse-state-output.json")
	inputFile, err := os.CreateTemp(tmpDir, "morse-state-input.json")
	require.NoError(t, err)

	// Generate and write the MorseStateExport input JSON file.
	morseStateExport, morseAccountState, err := testmigration.NewMorseStateExportAndAccountState(
		10, testmigration.RoundRobinAllMorseAccountActorTypes)
	require.NoError(t, err)

	morseStateExportBz, err := cmtjson.MarshalIndent(morseStateExport, "", "  ")
	require.NoError(t, err)

	_, err = inputFile.Write(morseStateExportBz)
	require.NoError(t, err)

	err = inputFile.Close()
	require.NoError(t, err)

	// Call the function under test.
	_, err = collectMorseAccounts(inputFile.Name(), outputPath)
	require.NoError(t, err)

	outputJSON, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	var (
		actualMsgImportMorseClaimableAccounts *migrationtypes.MsgImportMorseClaimableAccounts
	)

	expectedMsgImportMorseClaimableAccounts, err := migrationtypes.NewMsgImportMorseClaimableAccounts(
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		*morseAccountState,
	)
	require.NoError(t, err)

	err = cmtjson.Unmarshal(outputJSON, &actualMsgImportMorseClaimableAccounts)
	require.NoError(t, err)

	require.NoError(t, err)
	require.Equal(t, expectedMsgImportMorseClaimableAccounts, actualMsgImportMorseClaimableAccounts)
}

func TestCollectMorseAccounts_InvalidAndModuleAddresses(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "morse-state-output.json")
	inputFile, err := os.CreateTemp(tmpDir, "morse-state-input.json")
	require.NoError(t, err)

	// Generate MorseExportStates and MorseAccountStates with invalid and module addresses.
	invalidAddrMorseStateExport, expectedInvalidAddrMorseAccountState := testmigration.GenerateInvalidAddressMorseStateExportAndAccountState(t)
	moduleAddrMorseStateExport, moduleAddrMorseAccountState := testmigration.GenerateModuleAddressMorseStateExportAndAccountState(t, migrationtypes.MorseModuleAccountNames)

	// DEV_NOTE: This is a shallow copy, solely for improving readability via naming.
	mergedMorseStateExport := invalidAddrMorseStateExport
	expectedMergedMorseAccountState := expectedInvalidAddrMorseAccountState

	// Merge MorseStateExport accounts
	mergedMorseStateExport.AppState.Auth.Accounts = append(
		mergedMorseStateExport.AppState.Auth.Accounts,
		moduleAddrMorseStateExport.AppState.Auth.Accounts...,
	)

	// Merge MorseAccountState accounts
	expectedMergedMorseAccountState.Accounts = append(
		expectedMergedMorseAccountState.Accounts,
		moduleAddrMorseAccountState.Accounts...,
	)

	morseStateExportBz, err := cmtjson.MarshalIndent(mergedMorseStateExport, "", "  ")
	require.NoError(t, err)

	_, err = inputFile.Write(morseStateExportBz)
	require.NoError(t, err)

	err = inputFile.Close()
	require.NoError(t, err)

	// Call the function under test.
	_, err = collectMorseAccounts(inputFile.Name(), outputPath)
	require.NoError(t, err)

	outputJSON, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	var (
		actualMsgImportMorseClaimableAccounts *migrationtypes.MsgImportMorseClaimableAccounts
	)

	expectedMsgImportMorseClaimableAccounts, err := migrationtypes.NewMsgImportMorseClaimableAccounts(
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		*expectedMergedMorseAccountState,
	)
	require.NoError(t, err)

	err = cmtjson.Unmarshal(outputJSON, &actualMsgImportMorseClaimableAccounts)
	require.NoError(t, err)

	require.NoError(t, err)
	require.Equal(t, expectedMsgImportMorseClaimableAccounts, actualMsgImportMorseClaimableAccounts)
}

// TestNewTestMorseStateExport exercises the NewTestMorseStateExport testutil function.
// It generates MorseStateExport instances with an increasing number of accounts, then verifies:
//   - The correct number of accounts in each export
//   - The total balances in each export
//   - The total stakes in each export (via transformAndIncludeMorseState)
func TestNewTestMorseStateExport(t *testing.T) {
	for numAccounts := 1; numAccounts <= 10; numAccounts++ {
		t.Run(fmt.Sprintf("num_accounts=%d", numAccounts), func(t *testing.T) {
			morseStateExport := new(migrationtypes.MorseStateExport)
			stateExportBz, _, err := testmigration.NewMorseStateExportAndAccountStateBytes(
				numAccounts, testmigration.RoundRobinAllMorseAccountActorTypes)
			require.NoError(t, err)

			err = cmtjson.Unmarshal(stateExportBz, morseStateExport)
			require.NoError(t, err)

			exportAccounts := morseStateExport.AppState.Auth.Accounts
			require.Equal(t, numAccounts, len(exportAccounts))

			morseWorkspace := newMorseImportWorkspace()
			err = transformAndIncludeMorseState(morseStateExport, morseWorkspace)
			require.NoError(t, err)

			// Construct account number expectations based on equal distribution of unstaked, app, and supplier accounts.
			expectedNumSuppliers := numAccounts / 3
			expectedNumApps := numAccounts / 3
			expectedActorType := testmigration.RoundRobinAllMorseAccountActorTypes(uint64(numAccounts - 1))
			if expectedActorType == testmigration.MorseApplicationActor {
				expectedNumApps++
			}
			t.Logf("numAccounts: %d; expectedNumApps: %d; expectedNumSuppliers: %d", numAccounts, expectedNumApps, expectedNumSuppliers)

			// Assert the number of accounts and staked actors matches expectations.
			require.Equal(t, uint64(numAccounts), morseWorkspace.getNumAccounts())
			require.Equal(t, uint64(expectedNumApps), morseWorkspace.numApplications)
			require.Equal(t, uint64(expectedNumSuppliers), morseWorkspace.numSuppliers)

			// Compute expected totals for unstaked balance, application stake, and supplier stake, for all MorseClaimableAccounts.
			var expectedShannonTotalUnstakedBalance,
				expectedShannonTotalAppStake,
				expectedShannonTotalSupplierStake int64

			for i := 0; i < numAccounts; i++ {
				expectedShannonTotalUnstakedBalance += testmigration.GenMorseUnstakedBalanceAmount(uint64(i))

				morseAccountType := testmigration.RoundRobinAllMorseAccountActorTypes(uint64(i))
				switch morseAccountType {
				case testmigration.MorseUnstakedActor:
					// No-op.
				case testmigration.MorseApplicationActor:
					expectedShannonTotalAppStake += testmigration.GenMorseApplicationStakeAmount(uint64(i))
				case testmigration.MorseSupplierActor:
					expectedShannonTotalSupplierStake += testmigration.GenMorseSupplierStakeAmount(uint64(i))
				default:
					t.Fatalf("unknown morse account stake state: %q", morseAccountType)
				}
			}

			require.Equal(t, expectedShannonTotalUnstakedBalance, morseWorkspace.accumulatedTotalBalance.Int64())
			require.Equal(t, expectedShannonTotalAppStake, morseWorkspace.accumulatedTotalAppStake.Int64())
			require.Equal(t, expectedShannonTotalSupplierStake, morseWorkspace.accumulatedTotalSupplierStake.Int64())
		})
	}
}

func BenchmarkTransformMorseState(b *testing.B) {
	for i := 0; i < 5; i++ {
		numAccounts := int(math.Pow10(i + 1))
		morseStateExport := new(migrationtypes.MorseStateExport)
		morseStateExportBz, _, err := testmigration.NewMorseStateExportAndAccountStateBytes(
			numAccounts, testmigration.RoundRobinAllMorseAccountActorTypes)
		require.NoError(b, err)

		err = cmtjson.Unmarshal(morseStateExportBz, morseStateExport)
		require.NoError(b, err)

		b.Run(fmt.Sprintf("num_accounts=%d", numAccounts), func(b *testing.B) {
			morseWorkspace := newMorseImportWorkspace()

			// Call the function under test.
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				err = transformAndIncludeMorseState(morseStateExport, morseWorkspace)
				require.NoError(b, err)
			}
		})
	}
}
