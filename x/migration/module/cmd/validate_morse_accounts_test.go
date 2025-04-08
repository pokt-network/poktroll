package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cmtjson "github.com/cometbft/cometbft/libs/json"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/cmd/logger"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testmigration"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

func TestRunValidateMorseAccounts_Success(t *testing.T) {
	tempDir := t.TempDir()
	tempFilePath := filepath.Join(tempDir, "morse_accounts.json")

	// Generate a MsgImportMorseClaimableAccounts to validate.
	_, morseAccountState, err := testmigration.NewMorseStateExportAndAccountState(3, testmigration.RoundRobinAllMorseAccountActorTypes)
	require.NoError(t, err)

	msgImportClaimableAccounts, err := migrationtypes.NewMsgImportMorseClaimableAccounts("", *morseAccountState)
	require.NoError(t, err)

	// Serialize and write the MsgImportMorseClaimableAccounts JSON to a temporary file.
	msgImportMorseClaimableAccountsJSONBz, err := cmtjson.Marshal(msgImportClaimableAccounts)
	require.NoError(t, err)

	err = os.WriteFile(tempFilePath, msgImportMorseClaimableAccountsJSONBz, 0644)
	require.NoError(t, err)

	// Configure the CLI logger to write to a buffer for assertions.
	logBuffer := new(bytes.Buffer)
	logger.Logger = polyzero.NewLogger(
		polyzero.WithLevel(polyzero.DebugLevel),
		polyzero.WithSetupFn(logger.NewSetupConsoleWriter(logBuffer)),
	)

	t.Run("lowercase hex MorseSrcAddress", func(t *testing.T) {
		// Collect the morse_src_address arguments as lowercase hex.
		morseSrcAddresses := make([]string, len(morseAccountState.Accounts))
		for i, morseAccount := range morseAccountState.Accounts {
			morseSrcAddresses[i] = strings.ToLower(morseAccount.GetMorseSrcAddress())
		}

		// Run the validate-morse-accounts command run function.
		args := append([]string{tempFilePath}, morseSrcAddresses...)
		err = runValidateMorseAccounts(&cobra.Command{}, args)
		require.NoError(t, err)

		// Print the log for manual inspection.
		t.Log(logBuffer.String())

		// Assert that the MorseAccountStateHash was validated as valid.
		require.NotContains(t, logBuffer.String(), "invalid morse account state hash")
		require.Contains(t, logBuffer.String(), "Morse account state hash matches")

		// Assert that the morse accounts given were observed and printed.
		for _, morseSrcAddress := range morseSrcAddresses {
			require.Contains(t, logBuffer.String(), fmt.Sprintf(
				"Found MorseClaimableAccount %s",
				strings.ToUpper(morseSrcAddress),
			))
		}

		require.NotContains(t, logBuffer.String(), "Morse address not found")
	})

	t.Run("uppercase hex MorseSrcAddress", func(t *testing.T) {
		// Collect the morse_src_address arguments as uppercase hex.
		morseSrcAddresses := make([]string, len(morseAccountState.Accounts))
		for i, morseAccount := range morseAccountState.Accounts {
			morseSrcAddresses[i] = strings.ToUpper(morseAccount.GetMorseSrcAddress())
		}

		args := append([]string{tempFilePath}, morseSrcAddresses...)
		err = runValidateMorseAccounts(&cobra.Command{}, args)
		require.NoError(t, err)

		// Print the log for manual inspection.
		t.Log(logBuffer.String())

		// Assert that the MorseAccountStateHash was validated as valid.
		require.NotContains(t, logBuffer.String(), "invalid morse account state hash")
		require.Contains(t, logBuffer.String(), "Morse account state hash matches")

		// Assert that the morse accounts given were observed and printed.
		for _, morseSrcAddress := range morseSrcAddresses {
			require.Contains(t, logBuffer.String(), fmt.Sprintf(
				"Found MorseClaimableAccount %s",
				strings.ToUpper(morseSrcAddress),
			))
		}

		require.NotContains(t, logBuffer.String(), "Morse address not found")
	})
}

func TestRunValidateMorseAccounts_InvalidMorseAccountStateHash(t *testing.T) {
	tempDir := t.TempDir()
	tempFilePath := filepath.Join(tempDir, "morse_accounts.json")

	// Configure the CLI logger to write to a buffer for assertions.
	logBuffer := new(bytes.Buffer)
	logger.Logger = polyzero.NewLogger(
		polyzero.WithLevel(polyzero.DebugLevel),
		polyzero.WithSetupFn(logger.NewSetupConsoleWriter(logBuffer)),
	)

	// Generate a MsgImportMorseClaimableAccounts to validate.
	_, morseAccountState, err := testmigration.NewMorseStateExportAndAccountState(3, testmigration.RoundRobinAllMorseAccountActorTypes)
	require.NoError(t, err)

	// Construct a MsgImportMorseClaimableAccounts with an invalid account state hash.
	msgImportClaimableAccounts, err := migrationtypes.NewMsgImportMorseClaimableAccounts("", *morseAccountState)
	require.NoError(t, err)

	msgImportClaimableAccounts.MorseAccountStateHash = []byte("invalid_hash")

	// Serialize and write the MsgImportMorseClaimableAccounts JSON to a temporary file.
	msgImportMorseClaimableAccountsJSONBz, err := cmtjson.Marshal(msgImportClaimableAccounts)
	require.NoError(t, err)

	err = os.WriteFile(tempFilePath, msgImportMorseClaimableAccountsJSONBz, 0644)
	require.NoError(t, err)

	// Run the validate-morse-accounts command run function.
	morseSrcAddresses := make([]string, len(morseAccountState.Accounts))
	for i, morseAccount := range morseAccountState.Accounts {
		morseSrcAddresses[i] = morseAccount.GetMorseSrcAddress()
	}

	args := append([]string{tempFilePath}, morseSrcAddresses...)
	err = runValidateMorseAccounts(&cobra.Command{}, args)
	require.NoError(t, err)

	// Print the log for manual inspection.
	t.Log(logBuffer.String())

	// Assert that the MorseAccountStateHash was validated as valid.
	computedMorseAccountStateHash, err := morseAccountState.GetHash()
	require.NoError(t, err)
	require.Contains(t, logBuffer.String(), "Invalid morse account state hash")
	require.Contains(t, logBuffer.String(), fmt.Sprintf("Given (expected): %X", msgImportClaimableAccounts.GetMorseAccountStateHash()))
	require.Contains(t, logBuffer.String(), fmt.Sprintf("Computed (actual): %X", computedMorseAccountStateHash))
	require.NotContains(t, logBuffer.String(), "Morse account state hash matches")

	// Assert that the morse accounts given were observed and printed.
	for _, morseSrcAddress := range morseSrcAddresses {
		require.Contains(t, logBuffer.String(), morseSrcAddress)
	}

	require.NotContains(t, logBuffer.String(), "Morse address not found")
}

func TestRunValidateMorseAccounts_MorseAddressesNotFound(t *testing.T) {
	tempDir := t.TempDir()
	tempFilePath := filepath.Join(tempDir, "morse_accounts.json")

	// Configure the CLI logger to write to a buffer for assertions.
	logBuffer := new(bytes.Buffer)
	logger.Logger = polyzero.NewLogger(
		polyzero.WithLevel(polyzero.DebugLevel),
		polyzero.WithSetupFn(logger.NewSetupConsoleWriter(logBuffer)),
	)

	// Generate a MsgImportMorseClaimableAccounts to validate.
	_, morseAccountState, err := testmigration.NewMorseStateExportAndAccountState(10, testmigration.RoundRobinAllMorseAccountActorTypes)
	require.NoError(t, err)

	// Construct a valid MsgImportMorseClaimableAccounts.
	msgImportClaimableAccounts, err := migrationtypes.NewMsgImportMorseClaimableAccounts("", *morseAccountState)
	require.NoError(t, err)

	// Serialize and write the MsgImportMorseClaimableAccounts JSON to a temporary file.
	msgImportMorseClaimableAccountsJSONBz, err := cmtjson.Marshal(msgImportClaimableAccounts)
	require.NoError(t, err)

	err = os.WriteFile(tempFilePath, msgImportMorseClaimableAccountsJSONBz, 0644)
	require.NoError(t, err)

	// Generate 3 random Morse addresses that we can expect to be missing.
	// (i.e. not present in the morse account state).
	expectedMissingMorseSrcAddresses := []string{
		strings.ToUpper(sample.MorseAddressHex()),
		strings.ToUpper(sample.MorseAddressHex()),
		strings.ToUpper(sample.MorseAddressHex()),
	}

	// Initialize a slice for the [morse_src_address, ...] CLI arguments.
	morseSrcAddressArgs := make([]string, 6)

	// Copy the expected missing Morse addresses.
	copy(morseSrcAddressArgs, expectedMissingMorseSrcAddresses)

	// Add some expected found Morse addresses.
	for i, morseAccount := range morseAccountState.Accounts[:3] {
		morseSrcAddressArgs[i+len(expectedMissingMorseSrcAddresses)] = morseAccount.GetMorseSrcAddress()
	}

	// Run the validate-morse-accounts command run function.
	args := append([]string{tempFilePath}, morseSrcAddressArgs...)
	err = runValidateMorseAccounts(&cobra.Command{}, args)
	require.NoError(t, err)

	// Print the log for manual inspection.
	t.Log(logBuffer.String())

	require.NotContains(t, logBuffer.String(), "Invalid morse account state hash")

	// Ensure that the expected missing morse addresses were observed and printed.
	for _, morseAccount := range morseAccountState.Accounts[:3] {
		require.Contains(t, logBuffer.String(), fmt.Sprintf("Found MorseClaimableAccount %s", strings.ToUpper(morseAccount.GetMorseSrcAddress())))
		require.NotContains(t, logBuffer.String(), fmt.Sprintf("- %s", strings.ToUpper(morseAccount.GetMorseSrcAddress())))
	}

	// Ensure that the expected missing morse addresses were observed and printed.
	for _, expectedMissingMorseSrcAddress := range expectedMissingMorseSrcAddresses {
		require.Contains(t, logBuffer.String(), fmt.Sprintf("- %s", strings.ToUpper(expectedMissingMorseSrcAddress)))
		require.NotContains(t, logBuffer.String(), fmt.Sprintf("Found MorseClaimableAccount %s", strings.ToUpper(expectedMissingMorseSrcAddress)))
	}
}
