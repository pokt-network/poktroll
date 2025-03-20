package cmd

import (
	"os"

	cmtjson "github.com/cometbft/cometbft/libs/json"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/cmd/flags"
	"github.com/pokt-network/poktroll/cmd/logger"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

// We're `explicitly omitting default` so the relayer crashes if these aren't specified.
const omittedDefaultFlagValue = "explicitly omitting default"

// TODO_IN_THIS_COMMIT: godoc...
func ImportMorseAccountsCmd() *cobra.Command {
	importMorseAcctsCmd := &cobra.Command{
		Use: "import-morse-accounts [morse-account-state-json-path]",
		// TODO_IN_THIS_COMMIT:
		// Short: ,
		// Long: ,
		Args:    cobra.ExactArgs(1),
		RunE:    runImportMorseAccounts,
		PreRunE: logger.PreRunESetup,
	}

	cosmosflags.AddTxFlagsToCmd(importMorseAcctsCmd)

	// DEV_NOTE: This is required by the TxClient. Despite this being a "tx" command,
	// the TxClient still "queries" for its own TxResult events.
	importMorseAcctsCmd.Flags().String(cosmosflags.FlagGRPC, omittedDefaultFlagValue, "Register the default Cosmos node grpc flag, which is needed to initialize the Cosmos query context with grpc correctly. It can be used to override the `QueryNodeGRPCURL` field in the config file if specified.")
	// TODO_IN_THIS_COMMIT: explain...
	importMorseAcctsCmd.Flags().Bool(cosmosflags.FlagGRPCInsecure, true, "Used to initialize the Cosmos query context with grpc security options. It can be used to override the `QueryNodeGRPCInsecure` field in the config file if specified.")

	return importMorseAcctsCmd
}

func runImportMorseAccounts(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	// Ensure the morse account state file exists.
	accountStatePath := args[0]
	if _, err := os.Stat(accountStatePath); err != nil {
		return err
	}

	// Read and deserialize it.
	morseAccountStateBz, err := os.ReadFile(accountStatePath)
	if err != nil {
		return err
	}

	morseAccountState := new(migrationtypes.MorseAccountState)
	err = cmtjson.Unmarshal(morseAccountStateBz, morseAccountState)
	if err != nil {
		return err
	}

	txClient, err := flags.GetTxClient(ctx, cmd)
	if err != nil {
		return err
	}

	// Construct a MsgImportMorseAccountState message.
	msgImportMorseAccountState, err := migrationtypes.NewMsgImportMorseClaimableAccounts(
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		*morseAccountState,
	)
	if err != nil {
		return err
	}

	// Conventionally derive a cosmos-sdk client context from the cobra command.
	clientCtx, err := cosmosclient.GetClientTxContext(cmd)
	if err != nil {
		return err
	}

	// Package the MsgImportMorseAccountState message into a MsgAuthzExec message.
	msgAuthzExec := authz.NewMsgExec(
		clientCtx.FromAddress,
		[]cosmostypes.Msg{msgImportMorseAccountState},
	)

	// Sign and broadcast the claim Morse account message.
	eitherErr := txClient.SignAndBroadcast(ctx, &msgAuthzExec)
	err, errCh := eitherErr.SyncOrAsyncError()
	if err != nil {
		return err
	}

	// Wait for an async error, timeout, or the errCh to close on success.
	return <-errCh
}
