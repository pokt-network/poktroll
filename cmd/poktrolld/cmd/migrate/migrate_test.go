package migrate

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cmtjson "github.com/cometbft/cometbft/libs/json"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/testmigration"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

func TestCollectMorseAccounts(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "morse-state-output.json")
	inputFile, err := os.CreateTemp(tmpDir, "morse-state-input.json")
	require.NoError(t, err)

	morseStateExportBz, morseAccountStateBz := testmigration.NewMorseStateExportAndAccountStateBytes(t, 10)
	_, err = inputFile.Write(morseStateExportBz)
	require.NoError(t, err)

	err = inputFile.Close()
	require.NoError(t, err)

	// Call the function under test.
	err = collectMorseAccounts(inputFile.Name(), outputPath)
	require.NoError(t, err)

	outputJSON, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	expectedJSON := string(morseAccountStateBz)
	require.NoError(t, err)

	// Strip all whitespace from the expected JSON.
	expectedJSON = strings.ReplaceAll(expectedJSON, "\n", "")
	expectedJSON = strings.ReplaceAll(expectedJSON, " ", "")

	require.NoError(t, err)
	require.Equal(t, expectedJSON, string(outputJSON))
}

func TestNewTestMorseStateExport(t *testing.T) {
	for i := 1; i < 10; i++ {
		t.Run(fmt.Sprintf("num_accounts=%d", i), func(t *testing.T) {
			morseStateExport := new(migrationtypes.MorseStateExport)
			stateExportBz, _ := testmigration.NewMorseStateExportAndAccountStateBytes(t, i)
			err := cmtjson.Unmarshal(stateExportBz, morseStateExport)
			require.NoError(t, err)

			exportAccounts := morseStateExport.AppState.Auth.Accounts
			require.Equal(t, i, len(exportAccounts))

			expectedShannonBalance := fmt.Sprintf("%d%d%d0%d%d%d", i, i, i, i, i, i)
			morseAccountState := new(migrationtypes.MorseAccountState)
			morseAccountStateBz, err := transformMorseState(morseStateExport)
			require.NoError(t, err)

			err = cmtjson.Unmarshal(morseAccountStateBz, morseAccountState)
			require.NoError(t, err)

			require.Equal(t, expectedShannonBalance, morseAccountState.Accounts[i-1].Coins[0].Amount.String())
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

			// Call the function under test.
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err = transformMorseState(morseStateExport)
				require.NoError(b, err)
			}
		})
	}
}
