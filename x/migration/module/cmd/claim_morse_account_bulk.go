package cmd

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	cometcrypto "github.com/cometbft/cometbft/crypto"
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
	inputFilePath  string
	outputFilePath string
	unsafe         bool
	unarmoredJSON  bool
)

// ClaimMorseAccountBulkCmd returns the cobra command for bulk claiming Morse accounts and mapping them to new Shannon accounts.
func ClaimMorseAccountBulkCmd() *cobra.Command {
	claimAcctBulkCmd := &cobra.Command{
		Use:  "claim-accounts --input-file [morse_hex_keys_file] --output-file [morse_shannon_map_output_file]",
		Args: cobra.ExactArgs(0),
		Example: `Safe example (does not export shannon private keys in plaintext)
$ pocketd tx migration claim-accounts \
  --input-file ./bulk-accounts.json \
  --output-file ./bulk-accounts-output.json \
  --from <SIGNING-ACCOUNT> \
  --home ./localnet/pocketd \
  --keyring-backend test

Unsafe example (exports shannon private keys in plaintext):
$ pocketd tx migration claim-accounts \
  --input-file ./bulk-accounts.json \
  --output-file ./bulk-accounts-output.json \
  --from <SIGNING-ACCOUNT> \
  --home ./localnet/pocketd \
  --keyring-backend test \
  --unsafe \
  --unarmored-json`,
		Short: "Bulk claim unstaked balances from multiple Morse accounts to new Shannon accounts using JSON input/output files.",
		Long: `Easily claim unstaked balances from multiple Morse accounts in bulk.

This automates the batch transition and mapping of accounts from Morse to Shannon, streamlining the migration process.

In particular, this command:
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
		RunE:    runBulkClaimAccount,
		PreRunE: logger.PreRunESetup,
	}

	// Prepare the input file path flag.
	claimAcctBulkCmd.Flags().StringVarP(
		&inputFilePath,
		flags.FlagInputFile,
		flags.FlagInputFileShort,
		"",
		"path to the JSON file containing Morse private keys (hex-encoded) for all accounts to be migrated in bulk",
	)

	// Prepare the output file path flag.
	claimAcctBulkCmd.Flags().StringVarP(
		&outputFilePath,
		flags.FlagOutputFile,
		flags.FlagOutputFileShort,
		"",
		"path to the JSON file where the mapping of Morse accounts to their newly generated Shannon accounts (addresses and private keys in hex) will be written",
	)

	// Prepare the unsafe flag.
	claimAcctBulkCmd.Flags().BoolVarP(
		&unsafe,
		"unsafe",
		"",
		false,
		"unsafe operation, do not auto-load the shannon account private keys into the keyring",
	)

	// Prepare the unarmored JSON flag.
	claimAcctBulkCmd.Flags().BoolVarP(
		&unarmoredJSON,
		"unarmored-json",
		"",
		false,
		"unarmored JSON output file, this is useful for the migration of operators into a shannon keyring for later use",
	)

	// Adds standard Cosmos SDK CLI tx flags.
	cosmosflags.AddTxFlagsToCmd(claimAcctBulkCmd)

	return claimAcctBulkCmd
}

// marshalAccountInfo serializes account info to JSON.
// - Includes private key if both --unsafe and --unarmored-json are set.
// - Otherwise, only address is included.
func marshalAccountInfo(address string, privateKey []byte) ([]byte, error) {
	// Prepare the unsafe struct in the case both --unsafe and --unarmored-json are set.
	unsafeStruct := struct {
		Address    string `json:"address"`
		PrivateKey string `json:"private_key"`
	}{
		Address:    address,
		PrivateKey: hex.EncodeToString(privateKey),
	}

	// Prepare the safe struct in the case only --unarmored-json is set.
	safeStruct := struct {
		Address string `json:"address"`
	}{
		Address: address,
	}

	// Marshal the struct based on the flags.
	if unsafe && unarmoredJSON {
		return json.Marshal(&unsafeStruct)
	}
	return json.Marshal(&safeStruct)
}

// MorseAccountInfo holds Morse account data.
// - Address and private key are in hex format.
type MorseAccountInfo struct {
	// Address is the Morse account address in hex format.
	Address cometcrypto.Address `json:"address"`

	// PrivateKey is the Morse account private key in ed25519 format.
	PrivateKey ed25519.PrivKey `json:"private_key"`
}

// MarshalJSON customizes MorseAccountInfo JSON output.
// - Includes private key if unsafe/unarmored flags are set.
func (m MorseAccountInfo) MarshalJSON() ([]byte, error) {
	addressStr := hex.EncodeToString(m.Address)
	return marshalAccountInfo(addressStr, m.PrivateKey)
}

// ShannonAccountInfo holds Shannon account data.
// - Address and private key are in bech32 format.
type ShannonAccountInfo struct {
	// Address is the Shannon account address in bech32 format.
	Address sdk.AccAddress `json:"address"`

	// PrivateKey is the Shannon account private key in secp256k1 format.
	PrivateKey secp256k1.PrivKey `json:"private_key"`
}

// MarshalJSON customizes ShannonAccountInfo JSON output.
// - Includes private key if unsafe/unarmored flags are set.
func (s ShannonAccountInfo) MarshalJSON() ([]byte, error) {
	addressStr := s.Address.String()
	return marshalAccountInfo(addressStr, s.PrivateKey.Bytes())
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
	ctx := cmd.Context()

	// Ensure that unsafe and unarmored-json flags are used together.
	if (unarmoredJSON && !unsafe) || unsafe && !unarmoredJSON {
		return fmt.Errorf("unsafe and unarmored-json flags must be used together")
	}

	// Construct a tx client.
	txClient, err := flags.GetTxClientFromFlags(ctx, cmd)
	if err != nil {
		return err
	}

	// Conventionally derive a cosmos-sdk client context from the cobra command.
	clientCtx, err := cosmosclient.GetClientTxContext(cmd)
	if err != nil {
		return err
	}

	// Read Morse private keys from file.
	morseAccounts, err := readMorsePrivateKeysFile(ctx, clientCtx)
	if err != nil {
		return err
	}

	logger.Logger.Info().
		Str("input_file", inputFilePath).
		Str("output_file", outputFilePath).
		Msgf("About to start running MsgClaimMorseAccount for %d Morse accounts", len(morseAccounts))

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
		logger.Logger.Info().Msgf("writing migration output JSON to %s", outputFilePath)
		writeErr := os.WriteFile(outputFilePath, outputJSON, 0644)
		if writeErr != nil {
			logger.Logger.Error().Err(writeErr).
				Msgf("output file content printed due to error writing file to %s", outputFilePath)
			println(outputJSON)
		}
	}()

	// Loop through all EXISTING Morse accounts and map them to NEW Shannon accounts.
	for _, morseAccount := range morseAccounts {
		mapping, mappingErr := mappingAccounts(cmd, morseAccount)
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
					"morse account: %s will not be migrated due to the following error: %s",
					mapping.MorseAccount.Address.String(),
					mapping.Error,
				)
			continue
		}

		// Log the successful mapping will be claimed
		logger.Logger.Info().Msgf(
			"mapping morse account=%s to shannon account=%s",
			mapping.MorseAccount.Address.String(),
			mapping.ShannonAccount.Address.String(),
		)
		claimMessages = append(claimMessages, mapping.MigrationMsg)
	}

	// Sign and broadcast the claim Morse account message.
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
		logger.Logger.Error().Msgf("error migrating morse accounts: %s. \n Check the output file for more details: %s", migrationBatchResult.Error, outputFilePath)
	} else {
		logger.Logger.Info().
			Int("tx_messages", len(claimMessages)).
			Str("tx_hash", migrationBatchResult.TxHash).
			Msg("morse accounts migration tx delivered successfully")
	}

	return nil
}

// readMorsePrivateKeysFile reads Morse private keys from the input file, validates their claimable status, and returns a list of MorseAccountInfo.
func readMorsePrivateKeysFile(
	ctx context.Context,
	clientCtx cosmosclient.Context,
) ([]MorseAccountInfo, error) {
	// Prepare a new morse private keys slice.
	var morsePrivateKeys []string
	var morseAccounts []MorseAccountInfo

	// Prepare a new query client.
	queryClient := types.NewQueryClient(clientCtx)

	// Read the input file.
	fileContents, err := os.ReadFile(inputFilePath)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(fileContents, &morsePrivateKeys); err != nil {
		return nil, err
	}

	for _, morsePrivateKey := range morsePrivateKeys {
		morseHexKey, err := hex.DecodeString(morsePrivateKey)
		if err != nil {
			return nil, err
		}
		morsePrivateKey := ed25519.PrivKey(morseHexKey)
		morseAddress := morsePrivateKey.PubKey().Address()
		morseAddressStr := strings.ToUpper(morseAddress.String()) // uppercase for query

		req := &types.QueryMorseClaimableAccountRequest{Address: morseAddressStr}
		res, queryErr := queryClient.MorseClaimableAccount(ctx, req)
		// if we are not able to validate the existence of the morse account, we return the error and stop the process
		if queryErr != nil {
			return nil, queryErr
		}

		// exists at snapshot but could or not be claimed yet
		if res.MorseClaimableAccount.IsClaimed() {
			// this morse account was already claimed
			// Ignore already-claimed Morse accounts in migration.
			logger.Logger.Warn().Msgf("morse account %s already claimed: %v", morseAddressStr, res.MorseClaimableAccount)
			continue
		}

		// Morse account is in the snapshot and not yet claimed.
		// Add it to migration list.
		morseAccounts = append(morseAccounts, MorseAccountInfo{
			PrivateKey: morsePrivateKey,
			Address:    morseAddress,
		})
	}

	if len(morseAccounts) == 0 {
		return nil, fmt.Errorf("no claimable morse accounts found in the snapshot. Check the logs and the input file before trying again.")
	}

	return morseAccounts, nil
}

// mappingAccounts:
// - Maps a Morse account to a new Shannon account.
// - Creates the migration message.
// - Imports the Shannon private key into the keyring.
func mappingAccounts(cmd *cobra.Command, morseAccount MorseAccountInfo) (*MorseShannonMapping, error) {
	// Conventionally derive a cosmos-sdk client context from the cobra command.
	clientCtx, err := cosmosclient.GetClientTxContext(cmd)
	if err != nil {
		return nil, err
	}

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

	// Create a new Shannon account
	// 1. Generate a secp256k1 private key
	shannonPrivateKey := secp256k1.GenPrivKey()
	// 2. Get the Cosmos shannonAddress (bech32 format) from a public key
	shannonAddress := sdk.AccAddress(shannonPrivateKey.PubKey().Address())
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

	// Import Shannon private key into keyring.
	name := morseShannonMapping.ShannonAccount.Address.String()
	keyringErr := clientCtx.Keyring.ImportPrivKeyHex(
		name,
		hex.EncodeToString(morseShannonMapping.ShannonAccount.PrivateKey.Key),
		"secp256k1",
	)
	if keyringErr != nil {
		logger.Logger.Error().Msg("failed to import private key for shannon account, please check the logs and try again")
		return morseShannonMapping, keyringErr
	}

	// For debugging: record migration message attempted.
	morseShannonMapping.MigrationMsg = msgClaimMorseAccount

	return morseShannonMapping, nil
}
