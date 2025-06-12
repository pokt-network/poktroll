package cmd

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/cometbft/cometbft/crypto/ed25519"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/cmd/flags"
	"github.com/pokt-network/poktroll/cmd/logger"
	"github.com/pokt-network/poktroll/x/migration/types"
)

var (
	flagDestination string
)

const (
	flagDestinationName = "destination"
	flagDestinationDesc = "shannon destination address for the morse account migration"
)

// TODO_MAINNET_MIGRATION: Update the docs in https://dev.poktroll.com/operate/morse_migration/claiming
// ClaimMorseAccountBulkCmd returns the cobra command for bulk claiming Morse accounts and mapping them to new Shannon accounts.
func ClaimMorseAccountBulkCmd() *cobra.Command {
	claimAcctBulkCmd := &cobra.Command{
		Use:  "claim-accounts --input-file [morse_hex_keys_file] --output-file [morse_shannon_map_output_file]",
		Args: cobra.ExactArgs(0),
		Example: `

1. Safe example (does not export shannon private keys in plaintext)

$ pocketd tx migration claim-accounts \
  --input-file ./bulk-accounts.json \
  --output-file ./bulk-accounts-output.json \
  --from <SIGNING-ACCOUNT> \
  --home ./localnet/pocketd \
  --keyring-backend test

2. Unsafe example (exports shannon private keys in plaintext):

$ pocketd tx migration claim-accounts \
  --input-file ./bulk-accounts.json \
  --output-file ./bulk-accounts-output.json \
  --from <SIGNING-ACCOUNT> \
  --home ./localnet/pocketd \
  --keyring-backend test \
  --unsafe \
  --unarmored-json

3. Set the same Destination for many accounts and avoid creating new wallets for each account (like operators claiming remaining pocket on operation wallets):

$ pocketd tx migration claim-accounts \
  --input-file ./bulk-accounts.json \
  --output-file ./bulk-accounts-output.json \
  --from <SIGNING-ACCOUNT> \
  --home ./localnet/pocketd \
  --keyring-backend test \
  --unsafe \
  --unarmored-json \
  --destination <SHANNON-ADDRESS> # MAKE SURE YOU HAVE THE PRIVATE KEYS FOR THIS ADDRESS
`,
		Short: "Claim many Morse accounts as unstaked accounts (i.e. non-actor, balance only account)",
		Long: `Claim many Morse accounts as unstaked accounts (i.e. non-actor, balance only account).



This automates the batch transition and mapping of accounts from Morse to Shannon, streamlining the migration process.

This command:
- Accepts a JSON file with Morse private keys (hex format).
- For each Morse account, a new Shannon account will be created.
- After running, you'll get an output JSON mapping each Morse account to its new Shannon account, with both addresses and private keys (hex).


Example input file format:

[
	"<MORSE_PRIVATE_KEY_1>",
	"<MORSE_PRIVATE_KEY_2>",
	...
	"<MORSE_PRIVATE_KEY_N>"
]

Example output file format:

{
  "mappings": [
    {
      "morse": {
        "address": "<MORSE_ADDRESS>",
		"private_key": ".... only if --unsafe --unarmored-json"
      },
      "shannon": {
        "address": "<SHANNON_ADDRESS>",
		"private_key": ".... only if --unsafe --unarmored-json"
      },
      "migration_msg": {
        "shannon_signing_address": "<SHANNON_SIGNING_ADDRESS>",
        "shannon_dest_address": "SHANNON_ADDRESS",
        "morse_public_key": "....",
        "morse_signature": "...."
      },
      "error": ""
    }
  ],
  "error": "",
  "tx_hash": "<TX_HASH>",
  "tx_code": <TX_CODE>
}

For more information, see: https://dev.poktroll.com/operate/morse_migration/claiming`,
		RunE: runBulkClaimAccount,
	}

	// Prepare the input file path flag.
	claimAcctBulkCmd.Flags().StringVar(
		&flagInputFilePath,
		flags.FlagInputFile,
		"",
		"Path to the JSON file containing Morse private keys (hex-encoded) for all accounts to be migrated in bulk.")

	// Prepare the output file path flag.
	claimAcctBulkCmd.Flags().StringVar(
		&flagOutputFilePath,
		flags.FlagOutputFile,
		"",
		"Path to a JSON file where the mapping of Morse accounts to their newly generated Shannon accounts (addresses and private keys in hex) will be written.")

	// Prepare the unsafe flag.
	claimAcctBulkCmd.Flags().BoolVar(
		&flagUnsafe,
		"unsafe",
		false,
		"Enable unsafe operations. This flag must be switched on along with all unsafe operation-specific options.")

	// Prepare the unarmored JSON flag.
	claimAcctBulkCmd.Flags().BoolVar(
		&flagUnarmoredJSON,
		"unarmored-json",
		false,
		"Export unarmored hex privkey. Requires --unsafe.")

	// Shannon destination address for all funds
	claimAcctBulkCmd.Flags().StringVar(&flagDestination, flagDestinationName, "", flagDestinationDesc)

	// Flag for dry run mode.
	claimAcctBulkCmd.Flags().BoolVar(&flagDryRunClaim, FlagDryRunClaim, false, FlagDryRunClaimDesc)

	// Adds standard Cosmos SDK CLI tx flags.
	cosmosflags.AddTxFlagsToCmd(claimAcctBulkCmd)

	return claimAcctBulkCmd
}

// marshalAccountInfo serializes account info to JSON.
// - Includes private key if both --unsafe and --unarmored-json are set.
// - Otherwise, only address is included.
func marshalAccountInfo(address string, privateKey []byte, keyringName string) ([]byte, error) {
	// Prepare the unsafe struct in the case both --unsafe and --unarmored-json are set.
	unsafeStruct := struct {
		Address     string `json:"address"`
		PrivateKey  string `json:"private_key"`
		KeyringName string `json:"keyring,omitempty"`
	}{
		Address:     address,
		PrivateKey:  hex.EncodeToString(privateKey),
		KeyringName: keyringName,
	}

	// Prepare the safe struct in the case only --unarmored-json is set.
	safeStruct := struct {
		Address     string `json:"address"`
		KeyringName string `json:"keyring,omitempty"`
	}{
		Address:     address,
		KeyringName: keyringName,
	}

	// Marshal the struct based on the flags.
	if flagUnsafe && flagUnarmoredJSON {
		return json.Marshal(&unsafeStruct)
	}
	return json.Marshal(&safeStruct)
}

// MorseShannonMapping maps a Morse account to its corresponding Shannon account and migration message.
type MorseShannonMapping struct {
	// MorseAccount contains Morse account info.
	MorseAccount MorseAccountInfo `json:"morse"`

	// ShannonAccount contains Shannon account info. Could be nil if Morse validation fails.
	ShannonAccount *ShannonAccountInfo `json:"shannon"`

	// MigrationMsg is the migration message for debugging purposes.
	MigrationMsg sdk.Msg `json:"migration_msg"`

	// Error contains an error message if something goes wrong. Always set for result clarity.
	Error string `json:"error"`
}

// MigrationBatchResult holds the results of a migration batch, including all mappings and transaction info.
type MigrationBatchResult struct {
	// Mappings is a list of Morse to Shannon account mappings.
	Mappings []*MorseShannonMapping `json:"mappings"`

	// Error contains an error message for the batch, if any.
	Error string `json:"error"`

	// TxHash is the transaction hash for the migration batch.
	// Single tx with multiple messages for simplified migration.
	// See: https://github.com/pokt-network/poktroll/issues/1267
	TxHash string `json:"tx_hash"`

	// TxCode is the transaction code for the migration batch.
	TxCode uint32 `json:"tx_code"`
}

// runBulkClaimAccount executes the bulk claim operation for Morse accounts, mapping them to Shannon accounts and broadcasting the migration transaction.
func runBulkClaimAccount(cmd *cobra.Command, _ []string) error {
	logger.Logger.Info().Msg("Initializing claim accounts process...")
	ctx := cmd.Context()

	// Ensure that unsafe and unarmored-json flags are used together.
	if (flagUnarmoredJSON && !flagUnsafe) || flagUnsafe && !flagUnarmoredJSON {
		return fmt.Errorf("unsafe and unarmored-json flags must be used together")
	}

	// Construct a tx client.
	txClient, err := flags.GetTxClientFromFlags(ctx, cmd)
	if err != nil {
		return err
	}
	logger.Logger.Info().Msg("Cosmos tx client successfully configured")

	// Derive a cosmos-sdk client context from the cobra command.
	clientCtx, err := cosmosclient.GetClientTxContext(cmd)
	if err != nil {
		return err
	}
	logger.Logger.Info().Msg("Cosmos client successfully configured")

	// Read Morse private keys from file.
	morseAccounts, err := readMorseAccountsPrivateKeysFile(ctx, clientCtx)
	if err != nil {
		return err
	}
	logger.Logger.Info().Msg("Successfully loaded morse accounts from file")

	// Prepare the migration batch result.
	migrationBatchResult := MigrationBatchResult{
		Mappings: make([]*MorseShannonMapping, 0),
		Error:    "",
		TxHash:   "",
	}

	// Prepare the claimMessages slice.
	claimMessages := make([]sdk.Msg, 0)

	// Clean up keyring if tx results in error.
	defer func() {
		if migrationBatchResult.Error == "" {
			return
		}
		for _, morseToShannonMapping := range migrationBatchResult.Mappings {
			shannonAddr := morseToShannonMapping.ShannonAccount.Address.String()
			keyringErr := clientCtx.Keyring.Delete(shannonAddr)
			if keyringErr != nil {
				logger.Logger.Error().Err(keyringErr).
					Msgf(
						"failed to delete private key for shannon address '%s' associated with morse account '%s'. Please check the logs and try again",
						shannonAddr,
						hex.EncodeToString(morseToShannonMapping.MorseAccount.Address),
					)
			}
		}
	}()

	// Write output JSON file to --output-file path.
	defer func() {
		outputJSON, _ := json.MarshalIndent(migrationBatchResult, "", "  ")
		logger.Logger.Info().Msgf("Writing migration output JSON to %s", flagOutputFilePath)
		writeErr := os.WriteFile(flagOutputFilePath, outputJSON, 0644)
		if writeErr != nil {
			logger.Logger.Error().Err(writeErr).
				Msgf("output file content printed due to error writing file to %s", flagOutputFilePath)
			println(outputJSON)
		}
	}()

	logger.Logger.Info().
		Str("input_file", flagInputFilePath).
		Str("output_file", flagOutputFilePath).
		Msg("Starting the claim process for each Morse account...")
	// Loop through all EXISTING Morse accounts and map them to NEW Shannon accounts.
	for _, morseAccount := range morseAccounts {
		mapping, mappingErr := mappingAccounts(clientCtx, morseAccount)
		if mappingErr != nil {
			// On error, break loop and return.
			// Partial results will be written to output-file due to deferred write above.
			return mappingErr
		}

		// Append both successful and failed mappings to the batch result.
		migrationBatchResult.Mappings = append(migrationBatchResult.Mappings, mapping)

		// If there was an error, log it and continue to the next Morse account.
		if mapping.Error != "" {
			logger.Logger.Error().Err(mappingErr).
				Msgf(
					"Morse account: %s will not be migrated due to the following error: %s",
					mapping.MorseAccount.Address.String(),
					mapping.Error,
				)
			continue
		}

		// Log the successful mapping will be claimed
		logger.Logger.Info().Msgf(
			"Mapping morse account=%s to shannon account=%s",
			mapping.MorseAccount.Address.String(),
			mapping.ShannonAccount.Address.String(),
		)
		claimMessages = append(claimMessages, mapping.MigrationMsg)
	}

	if flagDryRunClaim {
		logger.Logger.Info().
			Str("path", flagOutputFilePath).
			Msg("tx IS NOT being broadcasted because: '--dry-run-claim=true'.")
		return nil
	}

	// Sign and broadcast the claim Morse account message.
	logger.Logger.Info().Int("messages", len(claimMessages)).Msg("Sign and broadcast transaction")
	tx, eitherErr := txClient.SignAndBroadcast(ctx, claimMessages...)
	broadcastErr, broadcastErrCh := eitherErr.SyncOrAsyncError()

	// Handle a successful tx broadcast.
	if tx != nil {
		migrationBatchResult.TxHash = tx.TxHash
		migrationBatchResult.TxCode = tx.Code

		if tx.Code != 0 {
			txError := fmt.Errorf("%s", tx.RawLog)
			migrationBatchResult.Error = tx.RawLog
			return txError
		}
	}

	// Handle broadcast errors.
	if broadcastErr != nil {
		migrationBatchResult.Error = broadcastErr.Error()
		return broadcastErr
	}
	broadcastErr = <-broadcastErrCh
	if broadcastErr != nil {
		migrationBatchResult.Error = broadcastErr.Error()
		return broadcastErr
	}

	if migrationBatchResult.Error != "" {
		logger.Logger.Error().Msgf("error migrating morse accounts: %s. \n Check the output file for more details: %s", migrationBatchResult.Error, flagOutputFilePath)
	} else {
		logger.Logger.Info().
			Int("tx_messages", len(claimMessages)).
			Str("tx_hash", migrationBatchResult.TxHash).
			Msg("Morse accounts migration tx delivered successfully")
	}

	return nil
}

// readMorseAccountsPrivateKeysFile reads ad validates Morse account info. It:
// 1. Reads Morse private keys from the input file.
// 2. Validates their claimable status.
// 3. Returns a list of MorseAccountInfo.
func readMorseAccountsPrivateKeysFile(
	ctx context.Context,
	clientCtx cosmosclient.Context,
) ([]MorseAccountInfo, error) {
	// Prepare the Morse accounts slice that will be returned.
	var morseAccounts []MorseAccountInfo

	// Prepare a new query client.
	queryClient := types.NewQueryClient(clientCtx)

	// Read the input file.
	fileContents, fileReadErr := os.ReadFile(flagInputFilePath)
	if fileReadErr != nil {
		return nil, fileReadErr
	}

	// Unmarshal the Morse private keys from the input file.
	var morsePrivateKeys []string
	if unmarshallErr := json.Unmarshal(fileContents, &morsePrivateKeys); unmarshallErr != nil {
		return nil, unmarshallErr
	}

	// Loop through all Morse private keys and validate their claimable status.
	for _, morsePrivateKeyHex := range morsePrivateKeys {
		morseHexKey, morseHexKeyErr := hex.DecodeString(morsePrivateKeyHex)
		if morseHexKeyErr != nil {
			return nil, morseHexKeyErr
		}
		morsePrivateKey := ed25519.PrivKey(morseHexKey)
		morseAddress := morsePrivateKey.PubKey().Address()
		morseAddressStr := strings.ToUpper(morseAddress.String()) // uppercase for query

		logger.Logger.Info().Str("address", morseAddressStr).Msg("Checking MorseClaimableAccount")
		req := &types.QueryMorseClaimableAccountRequest{Address: morseAddressStr}
		res, queryErr := queryClient.MorseClaimableAccount(ctx, req)
		// if we are not able to validate the existence of the morse account, we return the error and stop the process
		if queryErr != nil {
			return nil, queryErr
		}

		// Check if the Morse account is already claimed.
		if res.MorseClaimableAccount.IsClaimed() {
			// Ignore already-claimed Morse accounts during the bulk supplier migration.
			// This is intentional behaviour assuming a human error of not removing a key from the input-file that has already been claimed.
			logger.Logger.Warn().Msgf("Skipping accounts with morse address (%s) because it has already been claimed: %v", morseAddressStr, res.MorseClaimableAccount)
			continue
		}

		// Morse account is in the snapshot and not yet claimed.
		// Add it to migration list.
		morseAccounts = append(morseAccounts, MorseAccountInfo{
			PrivateKey: morsePrivateKey,
			Address:    morseAddress,
		})
		logger.Logger.Info().Str("address", morseAddressStr).Msg("MorseClaimableAccount found!")
	}

	if len(morseAccounts) == 0 {
		return nil, fmt.Errorf(
			"0/%d claimable Morse accounts found in the snapshot. Check the logs and the input file before trying again",
			len(morsePrivateKeys),
		)
	}

	return morseAccounts, nil
}

// mappingAccounts:
// - Maps a Morse account to a new Shannon account.
// - Creates the migration message.
// - Imports the Shannon private key into the keyring.
func mappingAccounts(
	clientCtx cosmosclient.Context,
	morseAccount MorseAccountInfo,
) (*MorseShannonMapping, error) {
	// Ensure that a signing account is provided.
	shannonSigningAddr := clientCtx.GetFromAddress().String()
	if shannonSigningAddr == "" {
		return nil, fmt.Errorf("missing --from signing account (it needs to be the name of the key in the keyring)")
	}

	// Prepare a new morse to shannon account mapping.
	// The shannon account is hydrated below
	morseShannonMapping := &MorseShannonMapping{
		MorseAccount: morseAccount,
	}

	var shannonPrivateKey *secp256k1.PrivKey
	var shannonAddress sdk.AccAddress
	var destinationReadErr error
	var hasDestination bool

	// assign morse to this one
	if flagDestination != "" {
		shannonAddress, destinationReadErr = sdk.AccAddressFromBech32(flagDestination)
		if destinationReadErr != nil {
			return nil, fmt.Errorf("--destination address is not a valid shannon address %w", destinationReadErr)
		}
		hasDestination = true
		logger.Logger.Info().
			Str("shannon_dest_address", shannonAddress.String()).
			Str("morse_address", hex.EncodeToString(morseAccount.Address)).
			Msg("Assigning --destination value as shannon_dest_address")
		// mock empty private key
		shannonPrivateKey = &secp256k1.PrivKey{}
	} else {
		// otherwise create a new shannon key
		// Create a new Shannon account
		// 1. Generate a secp256k1 private key
		shannonPrivateKey = secp256k1.GenPrivKey()
		// 2. Get the Cosmos shannonAddress (bech32 format) from a public key
		shannonAddress = sdk.AccAddress(shannonPrivateKey.PubKey().Address())
		logger.Logger.Info().
			Str("shannon_dest_address", shannonAddress.String()).
			Str("morse_address", hex.EncodeToString(morseAccount.Address)).
			Msg("Generated new Shannon account")
	}

	// 3. Assign Shannon account to mapping.
	morseShannonMapping.ShannonAccount = &ShannonAccountInfo{
		Address:    shannonAddress,
		PrivateKey: *shannonPrivateKey,
	}

	// Create a new claim Morse account message.
	msgClaimMorseAccount, claimMorseAccountErr := types.NewMsgClaimMorseAccount(
		shannonAddress.String(),
		morseAccount.PrivateKey,
		shannonSigningAddr,
	)
	if claimMorseAccountErr != nil {
		morseShannonMapping.Error = claimMorseAccountErr.Error()
		return morseShannonMapping, nil
	}

	if !hasDestination {
		// we need to store the pk
		// Import Shannon private key into keyring.
		morseShannonMapping.ShannonAccount.KeyringName = morseShannonMapping.ShannonAccount.Address.String()
		logger.Logger.Info().
			Str("name", morseShannonMapping.ShannonAccount.KeyringName).
			Str("morse_node_address", hex.EncodeToString(morseAccount.Address)).
			Str("shannon_operator_address", morseShannonMapping.ShannonAccount.KeyringName).
			Msg("Storing shannon operator address into the keyring")
		keyringErr := clientCtx.Keyring.ImportPrivKeyHex(
			morseShannonMapping.ShannonAccount.KeyringName,
			hex.EncodeToString(morseShannonMapping.ShannonAccount.PrivateKey.Key),
			"secp256k1",
		)
		if keyringErr != nil {
			logger.Logger.Error().Msg("failed to import private key for shannon account, please check the logs and try again")
			return morseShannonMapping, keyringErr
		}
	}

	// For debugging: record migration message attempted.
	morseShannonMapping.MigrationMsg = msgClaimMorseAccount

	return morseShannonMapping, nil
}
