package cmd

import (
	"github.com/cosmos/cosmos-sdk/client"
	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/cmd/flags"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

func RecoverMorseAccountCmd() *cobra.Command {
	recoverCmd := &cobra.Command{
		Use:   "recover-account [morse-src-address] [shannon-dest-address-or-key-name]",
		Args:  cobra.ExactArgs(2),
		Short: "Recover a Morse account which is BOTH unclaimable AND on the recoverable accounts allowlist",
		Long: `Recover a Morse account that is BOTH unclaimable AND on the recoverable accounts allowlist.

The morse account recovery process is authority-gated and can be invoked via:
  - Authz (authorization)
  - Governance proposal

This CLI command uses authz, so YOU MUST have an onchain authorization for the following message: pocket.migration.MsgRecoverMorseAccount.

The authorization grantee (the --from account) MUST be the message signer.

To check existing authz authorizations, run:
	pocketd query authz -h
`,
		Example: `Examples:

# Recover the dao module account by Shannon destination key name
pocketd tx migration recover-account dao pnf --from=pnf --network=beta

# Recover the dao module account by Shannon destination address
pocketd tx migration recover-account dao pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw --from=pnf

# Recover the dao module account by key name on MainNet with OS keyring backend
pocketd tx migration recover-account dao pnf --from=pnf --network=main --keyring-backend=os
`,
		RunE: runRecover,
	}

	// Add standard Cosmos SDK transaction flags
	cosmosflags.AddTxFlagsToCmd(recoverCmd)

	// Add common pocket specific flags
	recoverCmd.Flags().String(flags.FlagLogLevel, flags.DefaultLogLevel, flags.FlagLogLevelUsage)
	recoverCmd.Flags().String(flags.FlagLogOutput, flags.DefaultLogOutput, flags.FlagLogOutputUsage)

	return recoverCmd
}

func runRecover(cmd *cobra.Command, args []string) error {
	morseSrcAddress := args[0]
	shannonDestAddressOrKeyName := args[1]

	clientCtx, err := client.GetClientTxContext(cmd)
	if err != nil {
		return err
	}

	// Attempt to parse the first argument as an address first (no key name should be an address).
	shannonDestAddress, err := cosmostypes.AccAddressFromBech32(shannonDestAddressOrKeyName)
	if err != nil {
		// Attempt to retrieve the address from the keyring.
		// If the key name is not found, an error is returned.
		var record *keyring.Record
		record, err = clientCtx.Keyring.Key(shannonDestAddressOrKeyName)
		if err != nil {
			return err
		}

		shannonDestAddress, err = record.GetAddress()
		if err != nil {
			return err
		}
	}

	// Create the MsgRecoverMorseAccount message.
	msgRecoverMorseAccount := migrationtypes.NewMsgRecoverMorseAccount(
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		shannonDestAddress.String(),
		morseSrcAddress,
	)

	// Check that the message is valid (i.e. addresses are valid).
	if err = msgRecoverMorseAccount.ValidateBasic(); err != nil {
		return err
	}

	// Package the MsgRecoverMorseAccount message into a MsgAuthzExec message.
	//
	// MsgRecoverMorseAccount is an authority-gated message.
	// By default, the governance module address is the configured onchain authority.
	// In order to facilitate authorization of externally owned accounts (e.g. the foundation),
	// the authz module is used.
	//
	// DEV_NOTE: This exec message requires a corresponding authz authorization to
	// be present onchain.
	//
	// See: https://docs.cosmos.network/v0.50/build/modules/authz#authorization-and-grant.
	msgAuthzExec := authz.NewMsgExec(
		clientCtx.FromAddress,
		[]cosmostypes.Msg{msgRecoverMorseAccount},
	)

	return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), &msgAuthzExec)
}
