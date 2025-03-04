package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/x/migration/types"
)

var (
	morseKeyfileDecryptPassphrase string
	noPassphrase                  bool
)

// TODO_DOCUMENT: E2E flow...
// pocket accounts export <hex_morse_src_address> [key_export_path]
//
//	(prompt for passphrase)
//
// poktrolld migrate claim-account <key_export_path> --from <shannon_key_name> [-p <passphrase> | -n]
//
//	(prompt for passphrase, unless -p | --passphrase is provided)
func claimAccountCmd() *cobra.Command {
	claimAcctCmd := &cobra.Command{
		Use:  "claim-account [morse_key_export_path] --from [shannon_dest_key_name]",
		Args: cobra.ExactArgs(1),
		// TODO_IN_THIS_COMMIT: ...
		// Long: "",
		// Short: "",
		RunE: runClaimAccount,
	}

	claimAcctCmd.Flags().StringVarP(
		&morseKeyfileDecryptPassphrase,
		flagPassphrase,
		flagPassphraseShort,
		"",
		flagPassphraseUsage,
	)
	claimAcctCmd.Flags().BoolVar(
		&noPassphrase,
		flagNoPassphrase,
		false,
		flagNoPassphraseUsage,
	)

	// This command depends on the following conventional tx flags:
	// - FlagFrom | --from
	// - FlagSkipConfirmation | --yes
	flags.AddTxFlagsToCmd(claimAcctCmd)

	return claimAcctCmd
}

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

	shannonDestAddr := clientCtx.GetFromAddress().String()
	msgClaimMorseAccount, err := types.NewMsgClaimMorseAccount(
		shannonDestAddr,
		morsePrivKey.PubKey().Address().String(),
		morsePrivKey,
	)
	if err != nil {
		return err
	}

	// Serialize, as JSON, and print the MsgClaimMorseAccount for posterity and/or confirmation.
	msgClaimMorseAcctJSON, err := json.MarshalIndent(msgClaimMorseAccount, "", "  ")
	if err != nil {
		return err
	}

	fmt.Printf("MsgClaimMorseAccount %s\n", string(msgClaimMorseAcctJSON))

	// Last chance for the user to bail.
	skipConfirmation, err := cmd.Flags().GetBool(flags.FlagSkipConfirmation)
	if err != nil {
		return err
	}

	if !skipConfirmation {
		fmt.Printf("Confirm MsgClaimMorseAccount: y/[n]: ")
		stdinReader := bufio.NewReader(os.Stdin)

		// This call to ReadLine() will block until the user sends a new line to stdin.
		inputLine, _, readErr := stdinReader.ReadLine()
		if readErr != nil {
			return err
		}

		fmt.Println()

		// Bail unless some affirmative confirmation is given.
		switch string(inputLine) {
		case "Yes", "yes", "Y", "y":
		default:
			return nil
		}
	}

	// Construct a tx client.
	txClient, err := getTxClient(ctx, cmd)
	if err != nil {
		return err
	}

	// Sign and broadcast the claim Morse account message.
	eitherErr := txClient.SignAndBroadcast(ctx, msgClaimMorseAccount)
	err, errCh := eitherErr.SyncOrAsyncError()
	if err != nil {
		return err
	}

	// Wait for an async error, timeout, or the errCh to close on success.
	return <-errCh
}
