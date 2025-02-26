package migrate

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"

	cmtjson "github.com/cometbft/cometbft/libs/json"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/testutil/testmigration"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

// TODO_MAINNET(@bryanchriswhite): Add an E2E/integration test using real data.
// Note: This test should not be included in CI due to its size (90GB).
// Users should manually run wget to download the data and verify it on their computer.
// Reference: https://github.com/pokt-network/poktroll/pull/1039#discussion_r1947036729

func init() {
	logger = polyzero.NewLogger(polyzero.WithLevel(polyzero.DebugLevel))
	flagDebugAccountsPerLog = 1
}

func TestCollectMorseAccounts(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "morse-state-output.json")
	inputFile, err := os.CreateTemp(tmpDir, "morse-state-input.json")
	require.NoError(t, err)

	// Generate and write the MorseStateExport input JSON file.
	morseStateExportBz, morseAccountStateBz := testmigration.NewMorseStateExportAndAccountStateBytes(t, 10)
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
		expectedMorseAccountState,
		actualMorseAccountState *migrationtypes.MorseAccountState
	)

	err = cmtjson.Unmarshal(morseAccountStateBz, &expectedMorseAccountState)
	require.NoError(t, err)

	err = cmtjson.Unmarshal(outputJSON, &actualMorseAccountState)
	require.NoError(t, err)

	require.NoError(t, err)
	require.Equal(t, expectedMorseAccountState, actualMorseAccountState)
}

func TestNewTestMorseStateExport(t *testing.T) {
	// DEV_NOTE: Beyond i=3, the naive method for calculating the expected Shannon accumulated actor stakes fails.
	for i := 1; i < 4; i++ {
		t.Run(fmt.Sprintf("num_accounts=%d", i), func(t *testing.T) {
			morseStateExport := new(migrationtypes.MorseStateExport)
			stateExportBz, _ := testmigration.NewMorseStateExportAndAccountStateBytes(t, i)
			err := cmtjson.Unmarshal(stateExportBz, morseStateExport)
			require.NoError(t, err)

			exportAccounts := morseStateExport.AppState.Auth.Accounts
			require.Equal(t, i, len(exportAccounts))

			numTotalAccounts := 1
			for k := i; k > 1; k-- {
				numTotalAccounts += k
			}

			// i=1 -> "100000001", i=2 -> "200000002": creates scaled balance with unique ID
			expectedShannonAccountBalance := fmt.Sprintf("%d00000%d", i, i)

			// n=5 -> "5000050": scales with total accounts plus unique suffix
			expectedShannonTotalAppStake := fmt.Sprintf("%d000%d0", numTotalAccounts, numTotalAccounts)

			// n=5 -> "505000": different scaling pattern using same total accounts
			expectedShannonTotalSupplierStake := fmt.Sprintf("%d0%d00", numTotalAccounts, numTotalAccounts)

			morseWorkspace := newMorseImportWorkspace()
			err = transformMorseState(morseStateExport, morseWorkspace)
			require.NoError(t, err)

			require.Equal(t, uint64(i), morseWorkspace.getNumAccounts())
			require.Equal(t, uint64(i), morseWorkspace.numApplications)
			require.Equal(t, uint64(i), morseWorkspace.numSuppliers)

			morseAccounts := morseWorkspace.accountState.Accounts[i-1]
			require.Equal(t, expectedShannonAccountBalance, morseAccounts.UnstakedBalance.Amount.String())
			require.Equal(t, expectedShannonTotalAppStake, morseWorkspace.accumulatedTotalAppStake.String())
			require.Equal(t, expectedShannonTotalSupplierStake, morseWorkspace.accumulatedTotalSupplierStake.String())
		})
	}
}

func BenchmarkTransformMorseState(b *testing.B) {
	for i := 0; i < 5; i++ {
		numAccounts := int(math.Pow10(i + 1))
		morseStateExport := new(migrationtypes.MorseStateExport)
		morseStateExportBz, _ := testmigration.NewMorseStateExportAndAccountStateBytes(b, numAccounts)
		err := cmtjson.Unmarshal(morseStateExportBz, morseStateExport)
		require.NoError(b, err)

		b.Run(fmt.Sprintf("num_accounts=%d", numAccounts), func(b *testing.B) {
			morseWorkspace := newMorseImportWorkspace()

			// Call the function under test.
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				err = transformMorseState(morseStateExport, morseWorkspace)
				require.NoError(b, err)
			}
		})
	}
}
