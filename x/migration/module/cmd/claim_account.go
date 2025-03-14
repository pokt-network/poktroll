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

func claimAccountCmd() *cobra.Command {
	claimAcctCmd := &cobra.Command{
		Use:   "claim-account [morse_key_export_path] --from [shannon_dest_key_name]",
		Args:  cobra.ExactArgs(1),
		Short: "Claim an onchain MorseClaimableAccount as an unstaked/non-actor account",
		Long: `Claim an onchain MorseClaimableAccount as an unstaked/non-actor account.
The unstaked balance amount of the onchain MorseClaimableAccount will be minted to the Shannon account specified by the --from flag.

This will construct, sign, and broadcast a tx containing a MsgClaimMorseAccount message.
See: https://dev.poktroll.com/operate/morse_migration/claiming for more information.
`,
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
