package flags

import (
	"encoding/binary"
	"fmt"

	"cosmossdk.io/depinject"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	cmd2 "github.com/pokt-network/poktroll/cmd"
	"github.com/pokt-network/poktroll/pkg/client/query"
	"github.com/pokt-network/poktroll/pkg/store"
)

const (
	// OmittedDefaultFlagValue is used whenever a flag is required but no reasonable default value can be provided.
	// In most cases, this forces the user to specify the flag value to avoid unintended behavior.
	OmittedDefaultFlagValue = "intentionally omitting default"

	DefaultLogOutput = "-"

	FlagLogLevel      = "log-level"
	FlagLogLevelUsage = "The logging level (debug|info|warn|error)"

	FlagLogOutput      = "log-output"
	FlagLogOutputUsage = "The logging output (file path); defaults to stdout"

	FlagPassphrase      = "passphrase"
	FlagPassphraseShort = "p"
	FlagPassphraseUsage = "the passphrase used to decrypt the exported Morse key file for signing; the user will be prompted if empty (UNLESS --no-passphrase is used)"

	FlagNoPassphrase      = "no-passphrase"
	FlagNoPassphraseUsage = "attempt to use an empty passphrase to decrypt the exported Morse key file for signing"

	FlagAutoSequence      = "auto-sequence"
	FlagAutoSequenceUsage = "Sets the --offline, --account-number, and --sequence flags based on cached and/or onchain account state for the given --from address"
	DefaultAutoSequence   = false

	FlagOffchainMultiStorePath = "multistore-path"
	// TODO_IN_THIS_COMMIT: update once it's clear whether this is a directory or a file.
	FlagOffchainMultiStorePathUsage = "The path to the offchain multistore root"
	// TODO_IN_THIS_COMMIT: consider how best to align this with the "home dir"...
	DefaultFlagOffchainMultiStorePath = ""
)

// TODO_IN_THIS_COMMIT: godoc..
func CheckAutoSequenceFlag(
	cmd *cobra.Command,
	clientCtx cosmosclient.Context,
	accountSequenceCacheStore store.KeyValueStore,
	registerCleanupFn func(func() error),
) error {
	shouldAutoSequence, err := cmd.PersistentFlags().GetBool(FlagAutoSequence)
	if err != nil {
		return cmd2.ErrAutoSequence.Wrapf("%s", err)
	}

	if shouldAutoSequence {
		// Check if --offline, --account-number, or --sequence flags are already set.
		// This also ensures that each of these flags is registered, which is required.
		offline, err := cmd.Flags().GetBool(flags.FlagOffline)
		if err != nil {
			return cmd2.ErrAutoSequence.Wrapf("%s", err)
		}

		sequenceNumber, err := cmd.Flags().GetUint64(flags.FlagSequence)
		if err != nil {
			return cmd2.ErrAutoSequence.Wrapf("%s", err)
		}

		accountNumber, err := cmd.Flags().GetUint64(flags.FlagAccountNumber)
		if err != nil {
			return cmd2.ErrAutoSequence.Wrapf("%s", err)
		}

		// If --offline or --sequence flags are already set, return an error.
		if offline || sequenceNumber != 0 {
			return cmd2.ErrAutoSequence.Wrap("cannot set --auto-sequence flag when --offline or --sequence flags are already set")
		}

		// Construct an account query client.
		authClient, err := query.NewAccountQuerier(depinject.Supply(clientCtx))
		if err != nil {
			return cmd2.ErrAutoSequence.Wrapf("unable to construct account query client: %s", err)
		}

		// Query for the account with --from address.
		fromAddress, err := cmd.Flags().GetString(flags.FlagFrom)
		if err != nil {
			return err
		}

		account, err := authClient.GetAccount(cmd.Context(), fromAddress)
		if err != nil {
			return cmd2.ErrAutoSequence.Wrapf("unable to get account with address %s: %s", fromAddress, err)
		}

		// Default the sequence number to the onchain query result.
		sequenceNumber = account.GetSequence()

		switch {
		// If the --account-number flag is not set, apply the account number from the query result.
		// DEV_NOTE: 0 is the default value for --account-number flag.
		case accountNumber == 0:
			// Set --account-number flag to the account number.
			accountNumber = account.GetAccountNumber()
		// If the --account-number flag is set, ensure that it matches the account number from the query result.
		case accountNumber != account.GetAccountNumber():
			return fmt.Errorf(
				"account number %d does not match the account number %d of the account with address %s",
				accountNumber, account.GetAccountNumber(), fromAddress,
			)
		// Otherwise, use the account number provided in the --account-number flag.
		default:
		}

		// TODO_IN_THIS_COMMIT: extract...
		accountKey := []byte(account.GetAddress())
		cachedLastSequenceBz := accountSequenceCacheStore.Get(accountKey)
		cachedLastSequence := binary.BigEndian.Uint64(cachedLastSequenceBz)
		// --- END extract ---
		if cachedLastSequenceBz != nil {
			// If the cached sequence number is greater than the onchain sequence number,
			// use the cached sequence number.
			if cachedLastSequence > sequenceNumber {
				sequenceNumber = cachedLastSequence
			}
		}

		// Set --offline flag to true so that the tx client doesn't query for the account state.
		if err := cmd.Flags().Set(flags.FlagOffline, "true"); err != nil {
			return cmd2.ErrAutoSequence.Wrapf("%s", err)
		}

		// Set the --account-number flag.
		if err := cmd.Flags().Set(flags.FlagAccountNumber, fmt.Sprintf("%d", sequenceNumber)); err != nil {
			return cmd2.ErrAutoSequence.Wrapf("%s", err)
		}

		// Set the --sequence flag.
		if err := cmd.Flags().Set(flags.FlagSequence, fmt.Sprintf("%d", sequenceNumber)); err != nil {
			return cmd2.ErrAutoSequence.Wrapf("%s", err)
		}

		// TODO_IN_THIS_COMMIT: comment... update cache...
		updateAccountSequenceCache := func() error {
			// TDOO_IN_THIS_COMMIT: extract...
			lastSequenceBz := make([]byte, 8)
			binary.BigEndian.PutUint64(lastSequenceBz, sequenceNumber)
			// --- END extract ---
			accountSequenceCacheStore.Set(accountKey, lastSequenceBz)
			return nil
		}
		registerCleanupFn(updateAccountSequenceCache)
	}
	return nil
}
