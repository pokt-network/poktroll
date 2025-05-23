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
	"github.com/pokt-network/poktroll/x/migration/types"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/cmd/flags"
	"github.com/pokt-network/poktroll/cmd/logger"
)

var (
	inputFilePath  string
	outputFilePath string
	unsafe         bool
	unarmoredJSON  bool
)

// marshalAccountInfo serializes account information into JSON, conditionally including the private key based on flags.
func marshalAccountInfo(address string, privateKey []byte) ([]byte, error) {
	if unsafe && unarmoredJSON {
		return json.Marshal(&struct {
			Address    string `json:"address"`
			PrivateKey string `json:"private_key"`
		}{
			Address:    address,
			PrivateKey: hex.EncodeToString(privateKey),
		})
	}
	return json.Marshal(&struct {
		Address string `json:"address"`
	}{
		Address: address,
	})
}

// MorseAccountInfo is a struct for holding the Morse account information.
// The address and private key are in hex format.
type MorseAccountInfo struct {
	Address    cometcrypto.Address `json:"address"`
	PrivateKey ed25519.PrivKey     `json:"private_key"`
}

// MarshalJSON customizes the JSON representation of MorseAccountInfo, including optional unsafe and unarmored formats.
func (m MorseAccountInfo) MarshalJSON() ([]byte, error) {
	addressStr := hex.EncodeToString(m.Address)
	return marshalAccountInfo(addressStr, m.PrivateKey)
}

// ShannonAccountInfo is a struct for holding the Shannon account information.
// The address and private key are in hex format.
type ShannonAccountInfo struct {
	Address    sdk.AccAddress    `json:"address"`
	PrivateKey secp256k1.PrivKey `json:"private_key"`
}

// MarshalJSON customizes the JSON representation of ShannonAccountInfo, including optional unsafe and unarmored formats.
func (s ShannonAccountInfo) MarshalJSON() ([]byte, error) {
	addressStr := s.Address.String()
	return marshalAccountInfo(addressStr, s.PrivateKey.Bytes())
}

type MorseShannonMapping struct {
	MorseAccount   MorseAccountInfo    `json:"morse"`
	ShannonAccount *ShannonAccountInfo `json:"shannon"`       // could be nil of error at morse validation
	MigrationMsg   sdk.Msg             `json:"migration_msg"` // debug purpose
	Error          string              `json:"error"`         // just if something goes wrong, to be sure, we always provide a result
}

type MigrationBatchResult struct {
	Mappings []*MorseShannonMapping `json:"mappings"`
	Error    string                 `json:"error"`
	// a single tx with multiple messages to simplify the process due to: https://github.com/pokt-network/poktroll/issues/1267
	TxHash string `json:"tx_hash"`
	TxCode uint32 `json:"tx_code"`
}

func BulkClaimAccountCmd() *cobra.Command {
	bulkClaimAcctCmd := &cobra.Command{
		Use:   "claim-accounts --input-file [morse_hex_keys_file] --output-file [morse_shannon_map_output_file]",
		Args:  cobra.ExactArgs(0),
		Short: "Bulk claim unstaked balances from multiple MorseClaimableAccounts, mapping each to a newly generated Shannon account, with input and output as JSON files.",
		Long: `This command performs bulk claims of unstaked balances from multiple onchain MorseClaimableAccounts provided in a JSON input file containing Morse private keys (in hex format).

For each Morse account in the input, a new Shannon account is generated. After processing, the results are written to an output JSON file mapping each Morse account to its corresponding Shannon account, including addresses and private keys in hex format.
The input file has the following structure:

[
	"<MORSE_PRIVATE_KEY_1>",
	"<MORSE_PRIVATE_KEY_2>",
	...
	"<MORSE_PRIVATE_KEY_N>"
]

The output file has the following structure:

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

This automates the batch transition and mapping of accounts from Morse to Shannon, streamlining the migration process.

For more information, see: https://dev.poktroll.com/operate/morse_migration/claiming`,
		RunE:    runBulkClaimAccount,
		PreRunE: logger.PreRunESetup,
	}

	bulkClaimAcctCmd.Flags().StringVarP(
		&inputFilePath,
		flags.FlagInputFile,
		flags.FlagInputFileShort,
		"",
		flags.FlagInputFileUsage,
	)

	bulkClaimAcctCmd.Flags().StringVarP(
		&outputFilePath,
		flags.FlagOutputFile,
		flags.FlagOutputFileShort,
		"",
		flags.FlagOutputFileUsage,
	)

	bulkClaimAcctCmd.Flags().BoolVarP(
		&unsafe,
		"unsafe",
		"",
		false,
		"unsafe operation, do not auto-load the shannon account private keys into the keyring",
	)

	bulkClaimAcctCmd.Flags().BoolVarP(
		&unarmoredJSON,
		"unarmored-json",
		"",
		false,
		"unarmored JSON output file, this is useful for the migration of operators into a shannon keyring for later use",
	)

	// This command depends on the conventional cosmos-sdk CLI tx flags.
	cosmosflags.AddTxFlagsToCmd(bulkClaimAcctCmd)

	return bulkClaimAcctCmd
}

func runBulkClaimAccount(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

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

	morseAccounts, err := readMorsePrivateKeysFile(ctx, clientCtx)

	if err != nil {
		return err
	}

	logger.Logger.Info().
		Str("input_file", inputFilePath).
		Str("output_file", outputFilePath).
		Int("morse_accounts", len(morseAccounts)).
		Msg("running MsgClaimMorseAccount in bulk")

	bulkResult := MigrationBatchResult{
		Mappings: make([]*MorseShannonMapping, 0),
		Error:    "",
		TxHash:   "",
	}

	messages := make([]sdk.Msg, 0)

	defer func() {
		// clean up the keyring if tx result in an error
		if bulkResult.Error == "" {
			return
		}

		for _, mapping := range bulkResult.Mappings {
			name := mapping.ShannonAccount.Address.String()
			keyringErr := clientCtx.Keyring.Delete(name)
			if keyringErr != nil {
				logger.Logger.Error().
					Msgf(
						"failed to delete private key for shannon account %s related to morse account %s, please check the logs and try again",
						name,
						hex.EncodeToString(mapping.MorseAccount.Address),
					)
			}
		}
	}()

	defer func() {
		// Write the output JSON file to --output-file [morse_shannon_map_output_file] argument.
		outputJSON, _ := json.MarshalIndent(bulkResult, "", "  ")
		logger.Logger.Info().Msgf("writing migration output JSON at=%s", outputFilePath)
		writeErr := os.WriteFile(outputFilePath, outputJSON, 0644)
		if writeErr != nil {
			logger.Logger.Warn().
				Err(writeErr).
				Msgf("output file content printed due to error writing file to: %s", outputFilePath)
			println(outputJSON)
		}
	}()

	for _, morseAccount := range morseAccounts {
		mapping, mappingErr := mappingAccounts(cmd, morseAccount)

		if mappingErr != nil {
			// this error means break the loop and return the error
			logger.Logger.Error().Msgf("error migrating morse account: %s", mappingErr.Error())
			// we break and what was done to this point will be at the output-file due to the deferring a few lines
			// above, so we don't need to do anything else here, return the error
			return err
		}

		bulkResult.Mappings = append(bulkResult.Mappings, mapping)

		if mapping.Error != "" {
			logger.Logger.Error().
				Msgf(
					"morse account: %s will not be migrated due to the following error: %s",
					mapping.MorseAccount.Address.String(),
					mapping.Error,
				)
			continue
		}

		logger.Logger.Info().Msgf(
			"mapping morse account=%s to shannon account=%s",
			mapping.MorseAccount.Address.String(),
			mapping.ShannonAccount.Address.String(),
		)
		messages = append(messages, mapping.MigrationMsg)
	}

	// Sign and broadcast the claim Morse account message.
	tx, eitherErr := txClient.SignAndBroadcast(ctx, messages...)
	broadcastErr, broadcastErrCh := eitherErr.SyncOrAsyncError()

	if tx != nil {
		bulkResult.TxHash = tx.TxHash
		bulkResult.TxCode = tx.Code

		if tx.Code != 0 {
			txError := fmt.Errorf("%s", tx.RawLog)
			bulkResult.Error = tx.RawLog
			return txError
		}
	}

	if broadcastErr != nil {
		bulkResult.Error = broadcastErr.Error()
		return broadcastErr
	}

	broadcastErr = <-broadcastErrCh
	if broadcastErr != nil {
		bulkResult.Error = broadcastErr.Error()
		return broadcastErr
	}

	if bulkResult.Error != "" {
		logger.Logger.Error().Msgf("error migrating morse accounts: %s - check the output file to know more about it.", bulkResult.Error)
	} else {
		logger.Logger.Info().
			Int("tx_messages", len(messages)).
			Str("tx_hash", bulkResult.TxHash).
			Msg("morse accounts migration tx delivered successfully")
	}

	return nil
}

func readMorsePrivateKeysFile(ctx context.Context, clientCtx cosmosclient.Context) ([]MorseAccountInfo, error) {
	queryClient := types.NewQueryClient(clientCtx)

	var morsePrivateKeys []string

	fileContents, err := os.ReadFile(inputFilePath)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(fileContents, &morsePrivateKeys); err != nil {
		return nil, err
	}

	morseAccounts := make([]MorseAccountInfo, 0)

	for _, morsePrivateKey := range morsePrivateKeys {
		hexKey, hexErr := hex.DecodeString(morsePrivateKey)
		if hexErr != nil {
			return nil, hexErr
		}
		key := ed25519.PrivKey(hexKey)
		address := key.PubKey().Address()
		// to upper just in case, but usually it is already uppercase by sdk; but the query type is looking specific
		strAddress := strings.ToUpper(address.String())

		req := &types.QueryMorseClaimableAccountRequest{Address: strAddress}

		res, queryErr := queryClient.MorseClaimableAccount(ctx, req)
		// if we are not able to validate the existence of the morse account, we return the error and stop the process
		if queryErr != nil {
			return nil, queryErr
		}

		// exists at snapshot but could or not be claimed yet
		if res.MorseClaimableAccount.IsClaimed() {
			// this morse account was already claimed
			// we ignore it in the migration process
			logger.Logger.Warn().Msgf("morse account %s already claimed: %v", strAddress, res.MorseClaimableAccount)
			continue
		}

		// at this point, a morse account is part of the snapshot and is not claimed yet
		morseAccounts = append(morseAccounts, MorseAccountInfo{
			PrivateKey: key,
			Address:    address,
		})
	}

	if len(morseAccounts) == 0 {
		return nil, fmt.Errorf("there is not enough morse accounts to migrate, please check the logs and the input file and try again")
	}

	return morseAccounts, nil
}

func mappingAccounts(cmd *cobra.Command, morseAccount MorseAccountInfo) (*MorseShannonMapping, error) {
	// Conventionally derive a cosmos-sdk client context from the cobra command.
	clientCtx, err := cosmosclient.GetClientTxContext(cmd)
	if err != nil {
		return nil, err
	}

	msMapping := &MorseShannonMapping{
		MorseAccount: morseAccount,
	}

	// Generate a secp256k1 private key
	shannonPrivateKey := secp256k1.GenPrivKey()

	// Get the Cosmos shannonAddress (hex format) from a public key
	shannonAddress := sdk.AccAddress(shannonPrivateKey.PubKey().Address())

	// assign shannon account to the mapping
	msMapping.ShannonAccount = &ShannonAccountInfo{
		Address:    shannonAddress,
		PrivateKey: *shannonPrivateKey,
	}

	shannonSigningAddr := clientCtx.GetFromAddress().String()

	if shannonSigningAddr == "" {
		return nil, fmt.Errorf("missing --from value (it need to be the name of the key in the keyring)")
	}

	msgClaimMorseAccount, claimMorseAccountErr := types.NewMsgClaimMorseAccount(
		shannonAddress.String(),
		morseAccount.PrivateKey,
		shannonSigningAddr,
	)

	if claimMorseAccountErr != nil {
		msMapping.Error = claimMorseAccountErr.Error()
		return msMapping, nil
	}

	// import the shannon private key into the keyring
	name := msMapping.ShannonAccount.Address.String()
	keyringErr := clientCtx.Keyring.ImportPrivKeyHex(
		name,
		hex.EncodeToString(msMapping.ShannonAccount.PrivateKey.Key),
		"secp256k1",
	)
	if keyringErr != nil {
		logger.Logger.Error().Msg("failed to import private key for shannon account, please check the logs and try again")
		return msMapping, keyringErr
	}

	// debug purpose in case some migration goes wrong, the user knows what was trying to deliver
	msMapping.MigrationMsg = msgClaimMorseAccount

	return msMapping, nil
}
