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

func AdminRecoverMorseAccountCmd() *cobra.Command {
	adminRecoverCmd := &cobra.Command{
		Use:   "admin-recover-account [morse-src-address-or-module-name] [shannon-dest-address-or-key-name]",
		Args:  cobra.ExactArgs(2),
		Short: "Admin recovery of Morse account WITHOUT allowlist check (authority only)",
		Long: `Admin recovery of a Morse account WITHOUT checking the allowlist.

⚠️  WARNING: This command bypasses the normal allowlist validation and should ONLY be used
by the authority (PNF) for legitimate recovery requests that have been validated off-chain.

SECURITY:
- Can ONLY be called by the module authority (via authz grant)
- The account MUST exist on-chain (imported via MsgImportMorseClaimableAccounts)
- The account MUST NOT have been claimed/recovered already

SAFETY CHECKS PERFORMED:
✅ Authority validation (only PNF can call)
✅ Account exists check
✅ Not already claimed check
✅ Proper token minting

SAFETY CHECKS SKIPPED:
❌ Allowlist check (IsMorseAddressRecoverable)

USE CASE:
When a legitimate recovery request comes in but the address is not in the allowlist,
and waiting for an upgrade cycle would be too slow. The authority must validate the
request off-chain before executing this command.

This CLI command uses authz, so YOU MUST have an onchain authorization for:
  pocket.migration.MsgAdminRecoverMorseAccount

To check existing authz authorizations, run:
	pocketd query authz grants-by-grantee [your-address]
`,
		Example: `Examples:

# Admin recover by Shannon destination key name
pocketd tx migration admin-recover-account ABC123... pnf --from=pnf --network=main

# Admin recover by Shannon destination address
pocketd tx migration admin-recover-account ABC123... pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw --from=pnf

# Admin recover on MainNet with OS keyring backend
pocketd tx migration admin-recover-account ABC123... pnf --from=pnf --network=main --keyring-backend=os
`,
		RunE: runAdminRecover,
	}

	// Add standard Cosmos SDK transaction flags
	cosmosflags.AddTxFlagsToCmd(adminRecoverCmd)

	// Add common pocket specific flags
	adminRecoverCmd.Flags().String(cosmosflags.FlagLogLevel, flags.DefaultLogLevel, flags.FlagLogLevelUsage)
	adminRecoverCmd.Flags().String(flags.FlagLogOutput, flags.DefaultLogOutput, flags.FlagLogOutputUsage)

	return adminRecoverCmd
}

func runAdminRecover(cmd *cobra.Command, args []string) error {
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

	// Create the MsgAdminRecoverMorseAccount message.
	msgAdminRecoverMorseAccount := migrationtypes.NewMsgAdminRecoverMorseAccount(
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		shannonDestAddress.String(),
		morseSrcAddress,
	)

	// Check that the message is valid (i.e. addresses are valid).
	if err = msgAdminRecoverMorseAccount.ValidateBasic(); err != nil {
		return err
	}

	// Package the MsgAdminRecoverMorseAccount message into a MsgAuthzExec message.
	//
	// MsgAdminRecoverMorseAccount is an authority-gated message.
	// By default, the governance module address is the configured onchain authority.
	// In order to facilitate authorization of externally owned accounts (e.g. PNF),
	// the authz module is used.
	//
	// DEV_NOTE: This exec message requires a corresponding authz authorization to
	// be present onchain for pocket.migration.MsgAdminRecoverMorseAccount.
	//
	// See: https://docs.cosmos.network/v0.50/build/modules/authz#authorization-and-grant.
	msgAuthzExec := authz.NewMsgExec(
		clientCtx.FromAddress,
		[]cosmostypes.Msg{msgAdminRecoverMorseAccount},
	)

	return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), &msgAuthzExec)
}
