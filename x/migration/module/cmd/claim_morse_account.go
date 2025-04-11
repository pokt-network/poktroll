package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/cmd/flags"
	"github.com/pokt-network/poktroll/cmd/logger"
	"github.com/pokt-network/poktroll/x/migration/types"
)

var (
	morseKeyfileDecryptPassphrase string
	noPassphrase                  bool
)

func ClaimAccountCmd() *cobra.Command {
	claimAcctCmd := &cobra.Command{
		Use:   "claim-account [morse_key_export_path] --from [shannon_dest_key_name]",
		Args:  cobra.ExactArgs(1),
		Short: "Claim an onchain MorseClaimableAccount as an unstaked/non-actor account",
		Long: `Claim an onchain MorseClaimableAccount as an unstaked/non-actor account.

The unstaked balance amount of the onchain MorseClaimableAccount will be minted to the Shannon account specified by the --from flag.
This will construct, sign, and broadcast a tx containing a MsgClaimMorseAccount message.

For more information, see: https://dev.poktroll.com/operate/morse_migration/claiming`,
		// Example: TODO_MAINNET_CRITICAL(@bryanchriswhite): Add a few examples,
		RunE:    runClaimAccount,
		PreRunE: logger.PreRunESetup,
	}

	// Add a string flag for providing a passphrase to decrypt the Morse keyfile.
	claimAcctCmd.Flags().StringVarP(
		&morseKeyfileDecryptPassphrase,
		flags.FlagPassphrase,
		flags.FlagPassphraseShort,
		"",
		flags.FlagPassphraseUsage,
	)

	// Add a bool flag indicating whether to skip the passphrase prompt.
	claimAcctCmd.Flags().BoolVar(
		&noPassphrase,
		flags.FlagNoPassphrase,
		false,
		flags.FlagNoPassphraseUsage,
	)

	// This command depends on the conventional cosmos-sdk CLI tx flags.
	cosmosflags.AddTxFlagsToCmd(claimAcctCmd)

	return claimAcctCmd
}

// runClaimAccount performs the following sequence:
// - Load the Morse private key from the morse_key_export_path argument.
// - Construct a MsgClaimMorseAccount message from the Morse key.
// - Sign and broadcast the MsgClaimMorseAccount message using the Shannon key named by the `--from` flag.
// - Wait until the tx is committed onchain for either a synchronous or asynchronous error.
func runClaimAccount(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	morseKeyExportPath := args[0]
	morsePrivKey, err := loadMorsePrivateKey(morseKeyExportPath, morseKeyfileDecryptPassphrase)
	if err != nil {
		return err
	}

	// Conventionally derive a cosmos-sdk client context from the cobra command.
	clientCtx, err := cosmosclient.GetClientTxContext(cmd)
	if err != nil {
		return err
	}

	shannonSigningAddr := clientCtx.GetFromAddress().String()

	// Construct a MsgClaimMorseAccount message.
	shannonDestAddr := clientCtx.GetFromAddress().String()
	msgClaimMorseAccount, err := types.NewMsgClaimMorseAccount(
		shannonDestAddr,
		morsePrivKey.PubKey().Address().String(),
		morsePrivKey,
		shannonSigningAddr,
	)
	if err != nil {
		return err
	}

	// Serialize, as JSON, and print the MsgClaimMorseAccount for posterity and/or confirmation.
	msgClaimMorseAcctJSON, err := json.MarshalIndent(msgClaimMorseAccount, "", "  ")
	if err != nil {
		return err
	}

	logger.Logger.Info().Msgf("MsgClaimMorseAccount %s\n", string(msgClaimMorseAcctJSON))

	// Last chance for the user to abort.
	skipConfirmation, err := cmd.Flags().GetBool(cosmosflags.FlagSkipConfirmation)
	if err != nil {
		return err
	}

	// If the user has not set the --skip-confirmation flag, prompt for confirmation.
	if !skipConfirmation {
		// DEV_NOTE: Intentionally using fmt instead of logger.Logger to receive user input on the same line.
		fmt.Printf("Confirm MsgClaimMorseAccount: y/[n]: ")
		stdinReader := bufio.NewReader(os.Stdin)

		// This call to ReadLine() will block until the user sends a new line to stdin.
		inputLine, _, readErr := stdinReader.ReadLine()
		if readErr != nil {
			return err
		}

		// Terminate the confirmation prompt output line.
		fmt.Println()

		// Abort unless some affirmative confirmation is given.
		switch string(inputLine) {
		case "Yes", "yes", "Y", "y":
		default:
			return nil
		}
	}

	// Construct a tx client.
	txClient, err := flags.GetTxClient(ctx, cmd)
	if err != nil {
		return err
	}

	// Sign and broadcast the claim Morse account message.
	_, eitherErr := txClient.SignAndBroadcast(ctx, msgClaimMorseAccount)
	err, errCh := eitherErr.SyncOrAsyncError()
	if err != nil {
		return err
	}

	// Wait for an async error, timeout, or the errCh to close on success.
	return <-errCh
}
