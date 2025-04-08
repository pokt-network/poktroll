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
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func ClaimApplicationCmd() *cobra.Command {
	claimAppCmd := &cobra.Command{
		Use:   "claim-application [morse_key_export_path] [shannon_service_id] --from [shannon_dest_key_name]",
		Args:  cobra.ExactArgs(2),
		Short: "Claim an onchain MorseClaimableAccount as a staked application account",
		Long: `Claim an onchain MorseClaimableAccount as a staked application account.

The unstaked balance amount of the onchain MorseClaimableAccount will be minted to the Shannon account specified by the --from flag.
The Shannon account will also be staked as an application with a stake equal to the application stake the MorseClaimableAccount had on Morse.

This will construct, sign, and broadcast a tx containing a MsgClaimMorseApplication message.

For more information, see: https://dev.poktroll.com/operate/morse_migration/claiming`,
		// Example: TODO_MAINNET_CRITICAL(@bryanchriswhite): Add a few examples,
		RunE:    runClaimApplication,
		PreRunE: logger.PreRunESetup,
	}

	// Add a string flag for providing a passphrase to decrypt the Morse keyfile.
	claimAppCmd.Flags().StringVarP(
		&morseKeyfileDecryptPassphrase,
		flags.FlagPassphrase,
		flags.FlagPassphraseShort,
		"",
		flags.FlagPassphraseUsage,
	)

	// Add a bool flag indicating whether to skip the passphrase prompt.
	claimAppCmd.Flags().BoolVar(
		&noPassphrase,
		flags.FlagNoPassphrase,
		false,
		flags.FlagNoPassphraseUsage,
	)

	// This command depends on the conventional cosmos-sdk CLI tx flags.
	cosmosflags.AddTxFlagsToCmd(claimAppCmd)

	return claimAppCmd
}

// runClaimApplication performs the following sequence:
// - Load the Morse private key from the morse_key_export_path argument (arg 0).
// - Load and validate the service ID from the shannon_service_id argument (arg 1).
// - Construct a MsgClaimMorseApplication message from the Morse key and the service ID.
// - Sign and broadcast the MsgClaimMorseApplication message using the Shannon key named by the `--from` flag.
// - Wait until the tx is committed onchain for either a synchronous or asynchronous error.
func runClaimApplication(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Retrieve and validate the morse key based on the first argument provided.
	morseKeyExportPath := args[0]
	morsePrivKey, err := loadMorsePrivateKey(morseKeyExportPath, morseKeyfileDecryptPassphrase)
	if err != nil {
		return err
	}

	// Retrieve and validate the service ID based on the second argument provided.
	serviceID := args[1]
	if !sharedtypes.IsValidServiceId(serviceID) {
		return ErrInvalidUsage.Wrapf("invalid service ID: %q", serviceID)
	}

	// Conventionally derive a cosmos-sdk client context from the cobra command.
	clientCtx, err := cosmosclient.GetClientTxContext(cmd)
	if err != nil {
		return err
	}

	// Construct a MsgClaimMorseApplication message.
	shannonDestAddr := clientCtx.GetFromAddress().String()
	msgClaimMorseApplication, err := types.NewMsgClaimMorseApplication(
		shannonDestAddr,
		morsePrivKey,
		// Construct a new staked application service config with the service ID.
		&sharedtypes.ApplicationServiceConfig{
			ServiceId: serviceID,
		},
	)
	if err != nil {
		return err
	}

	// Serialize, as JSON, and print the MsgClaimMorseApplication for posterity and/or confirmation.
	msgClaimMorseAppJSON, err := json.MarshalIndent(msgClaimMorseApplication, "", "  ")
	if err != nil {
		return err
	}

	fmt.Printf("MsgClaimMorseApplication %s\n", string(msgClaimMorseAppJSON))

	// Last chance for the user to abort.
	skipConfirmation, err := cmd.Flags().GetBool(cosmosflags.FlagSkipConfirmation)
	if err != nil {
		return err
	}

	if !skipConfirmation {
		fmt.Printf("Confirm MsgClaimMorseApplication: y/[n]: ")
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
	_, eitherErr := txClient.SignAndBroadcast(ctx, msgClaimMorseApplication)
	err, errCh := eitherErr.SyncOrAsyncError()
	if err != nil {
		return err
	}

	// Wait for an async error, timeout, or the errCh to close on success.
	return <-errCh
}
