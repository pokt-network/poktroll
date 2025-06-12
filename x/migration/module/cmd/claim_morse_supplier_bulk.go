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
	cosmossdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/pokt-network/poktroll/cmd/flags"
	"github.com/pokt-network/poktroll/cmd/logger"
	"github.com/pokt-network/poktroll/x/migration/types"
	"github.com/pokt-network/poktroll/x/supplier/config"
)

var (
	flagSupplierStakeTemplateFile string
	flagMorsePrivateKeysFile      string
	flagSetOperatorShare          uint64
	flagNewKeyPrefix              string
	flagUseIndexNames             bool
)

const (
	flagSupplierStakeTemplateFileName = "stake-template-file"
	flagSupplierStakeTemplateFileDesc = "Path to a stake template file detailing services, reward shares, etc. The owner, operator, and stake amount are automatically sourced from ClaimableAccounts."

	flagMorsePrivateKeysFileName = "morse-private-keys-file"
	flagMorsePrivateKeysFileDesc = "Path to a file containing Morse private keys (hex-encoded) for all accounts to be migrated in bulk."

	flagSetOperatorShareName = "set-operator-share"
	flagSetOperatorShareDesc = "Sets the operator’s share in the revenue distribution. Useful to automatically allocate rewards to the operator, avoiding the need for repeated manual funding."

	flagNewKeyPrefixName = "name-prefix-pattern"
	flagNewKeyPrefixDesc = "Prefix to use for new keys. Helps operators to migrate and set a prefix for the accounts"

	flagUseIndexNamesName = "use-index-names"
	flagUseIndexNamesDesc = "Use index names for the new keys. Helps operators to migrate and set a prefix for the accounts"
)

// ClaimSupplierBulkCmd returns the cobra command for bulk-claiming Morse nodes as Shannon Suppliers.
func ClaimSupplierBulkCmd() *cobra.Command {
	claimSuppliersCmd := &cobra.Command{
		Use:  "claim-suppliers --stake-template-file=[stake_template_file] --morse-private-keys-file=[morse_private_keys_file] --output-file=[morse_to_shannon_mapping_file] --from=[shannon_dest_key_name]",
		Args: cobra.ExactArgs(0),
		Example: `The following example shows how to claim many Morse nodes as Shannon Suppliers on LocalNet:

$ pocketd tx migration claim-suppliers \
	  --from=signer \
	  --morse-private-keys-file=./morse_private_keys.json \
	  --stake-template-file=./supplier-stake-template.yaml \
	  --output-file=./supplier_output.json \
	  --home=~/.pocket --keyring-backend=file \
	  --gas=auto --gas-prices=0.001upokt --gas-adjustment=1.1
`,
		Short: "Claim many onchain MorseClaimableAccount as staked Supplier accounts.",
		Long: `
Claim many onchain MorseClaimableAccounts as staked Supplier accounts.

For custodial Morse Nodes, the Morse node is claimed as a custodial Shannon Supplier.

For non-custodial Morse Nodes, the Morse node is claimed as a non-custodial Shannon Supplier.
- Precondition: The output_address MUST be claimed on Shannon via MsgClaimMorseAccount first.

What this command does:
1. Read all Morse node private keys from the input-file provided
2. Generates new private keys for every Shannon Supplier.
3. Prepare Supplier stake configurations for each Supplier from the template file.
4. Submit a Claim transactions migrate every Morse Node to a new Shannon supplier using the configs generated above.
5. Outputs the migration results to a JSON file.

Additional options:
1. Supports a '--dry-run-claim' mode that simulates the transaction but doesn't broadcast it onchain.
2. Supports '--add-operator-share=[INT]' which adds operator rev share automatically. Useful to ensure the operator wallet always has funds to work.
3. Supports '--name-prefix-pattern=[prefix]' which adds a prefix to the name used to store the key on the keyring. Results will looks like: '[PREFIX]-[ADDRESS]'
4. Supports '--use-index-names' which will replace the ADDRESS as suffix to use the index of the node in the list. Results will looks like: '[PREFIX]-[INDEX]'
5. Enables exporting unarmored (plain) JSON output with sensitive keys.


Example input Morse private keys file:

[
	"<MORSE_PRIVATE_KEY_1>",
	"<MORSE_PRIVATE_KEY_2>",
	...
	"<MORSE_PRIVATE_KEY_N>"
]

Example input staking template file:

'''yaml
# Supplier Claim Example (with placeholders and clear comments)
# owner_address: ${SHANNON_SUPPLIER_OWNER_ADDRESS}        # intentionally commented out, taken from MorseClaimableAccount
# operator_address: ${SHANNON_SUPPLIER_OPERATOR_ADDRESS}  # intentionally commented out, taken from MorseClaimableAccount
# stake_amount: ${SHANNON_SUPPLIER_STAKE_AMOUNT}          # intentionally commented out, taken from MorseClaimableAccount

# Default (example) revenue share: delegator gets 75%, owner gets 25%
# Using --add-operator-share=1 it will add automatically 1% for operator (supplier) address and reduce that 1 from
# owner rev share.
default_rev_share_percent:
  ${DELEGATOR_REWARDS_ADDRESS}: 75
# ${SUPPLIER_OWNER_ADDRESS}: intentionally commented out, taken from MorseClaimableAccount

services:
  # Example service #1
  - service_id: "<REDACTED_SERVICE_ID_1>"
	endpoints:
	  - publicly_exposed_url: https://service1.example.supplier.url
		rpc_type: JSON_RPC
	rev_share_percent:
	  ${DELEGATOR_REWARDS_ADDRESS}: 90
# 	  ${SUPPLIER_OWNER_ADDRESS}: intentionally commented out, taken from MorseClaimableAccount

  # Example service #2
  - service_id: "<REDACTED_SERVICE_ID_2>"
	endpoints:
	  - publicly_exposed_url: https://service2.example.supplier.url
		rpc_type: JSON_RPC
# 	rev_share_percent commented out; defaults to default_rev_share_percent above
'''

Example output Morse to Shannon Supplier Migration Result:

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
        "shannon_owner_address": "SHANNON_CLAIMED_ADDRESS_OF_MORSE_OUTPUT_ADDRESS",
        "shannon_operator_address": "SHANNON_GENERATED_ADDRESS",
        "morse_node_address": "MORSE_NODE_ADDRESS_OF_PRIVATE_KEY_READ_FROM_FILE",
        "morse_public_key": "MORSE_NODE_PUBLIC_KEY_OF_PRIVATE_KEY_READ_FROM_FILE",
        "morse_signature": "....",
		"signer_is_output_address": false,
		"services": [...STAKE_SERVICES_FROM_YAML_TEMPLATE_FILE...]
      },
      "error": ""
    }
  ],
  "error": "",
  "tx_hash": "<TX_HASH>",
  "tx_code": <TX_CODE>
}


More info: https://dev.poktroll.com/operate/morse_migration/claiming`,
		RunE:    runClaimSuppliers,
		PreRunE: logger.PreRunESetup,
	}

	// Flag for input Morse private keys file.
	claimSuppliersCmd.Flags().StringVar(&flagMorsePrivateKeysFile, flagMorsePrivateKeysFileName, "", flagMorsePrivateKeysFileDesc)
	// Flag for input Morse private keys file.
	claimSuppliersCmd.Flags().StringVar(&flagSupplierStakeTemplateFile, flagSupplierStakeTemplateFileName, "", flagSupplierStakeTemplateFileDesc)
	// Flag for output Morse to Shannon mapping file.
	claimSuppliersCmd.Flags().StringVar(&flagOutputFilePath, flags.FlagOutputFile, "", "Path to a file where the migration result will be written.")
	// Flag for dry run mode.
	claimSuppliersCmd.Flags().BoolVar(&flagDryRunClaim, FlagDryRunClaim, false, FlagDryRunClaimDesc)
	// Flags to export private keys in the output file.
	claimSuppliersCmd.Flags().BoolVar(&flagUnsafe, FlagUnsafe, false, FlagUnsafeDesc)
	claimSuppliersCmd.Flags().BoolVar(&flagUnarmoredJSON, FlagUnarmoredJSON, false, FlagUnarmoredJSONDesc)

	// Flags to customize the new Shannon account.
	claimSuppliersCmd.Flags().StringVar(&flagNewKeyPrefix, flagNewKeyPrefixName, "", flagNewKeyPrefixDesc)
	claimSuppliersCmd.Flags().Uint64Var(&flagSetOperatorShare, flagSetOperatorShareName, 0, flagSetOperatorShareDesc)
	claimSuppliersCmd.Flags().BoolVar(&flagUseIndexNames, flagUseIndexNamesName, false, flagUseIndexNamesDesc)

	// Required flags.
	_ = claimSuppliersCmd.MarkFlagRequired(flagMorsePrivateKeysFileName)
	_ = claimSuppliersCmd.MarkFlagRequired(flagSupplierStakeTemplateFileName)
	_ = claimSuppliersCmd.MarkFlagRequired(flagOutputFilePath)
	_ = claimSuppliersCmd.MarkFlagRequired(cosmosflags.FlagFrom)

	// This command depends on the conventional cosmos-sdk CLI tx flags.
	cosmosflags.AddTxFlagsToCmd(claimSuppliersCmd)

	return claimSuppliersCmd
}

// runClaimSuppliers executes the logic for the bulk supplier claim command.
// - Loads Morse private keys
// - Loads Supplier staking YAML template.
// - Generates new Shannon accounts.
// - Prepares and submits claim transactions.
// - Handles output and error reporting.
func runClaimSuppliers(cmd *cobra.Command, _ []string) error {
	logger.Logger.Info().Msg("Starting the bulk claim-suppliers process...")
	ctx := cmd.Context()

	// Derive a cosmos-sdk client context from the cobra command.
	clientCtx, err := cosmosclient.GetClientTxContext(cmd)
	if err != nil {
		return err
	}
	logger.Logger.Info().Msg("Cosmos tx client successfully configured")

	// Retrieve the Shannon signing address (used to sign the claim transaction).
	shannonSigningAddr := clientCtx.GetFromAddress().String()
	if shannonSigningAddr == "" {
		return fmt.Errorf("no shannon signing address provided via the following flag: --from")
	}
	logger.Logger.Info().Str("shannon_signing_address", shannonSigningAddr).Msg("Validated Shannon signing address")

	// Load the supplier stake config template from the YAML file.
	// Fail fast if the file is missing or invalid.
	templateSupplierStakeConfig, templateError := loadTemplateSupplierStakeConfigYAML(flagSupplierStakeTemplateFile)
	if templateError != nil {
		return templateError
	}
	logger.Logger.Info().Msg("Successfully loaded supplier stake configuration template")

	// Prepare the migration batch result object.
	migrationBatchResult := MigrationBatchResult{
		Mappings: make([]*MorseShannonMapping, 0),
		Error:    "",
		TxHash:   "",
	}

	// Clean up keyring if any error occurs during the bulk claim.
	defer func() {
		if migrationBatchResult.Error == "" {
			return
		}
		// Delete Shannon private keys from keyring for failed migrations.
		for _, morseToShannonMapping := range migrationBatchResult.Mappings {
			shannonAddress := morseToShannonMapping.ShannonAccount.Address.String()
			morseAddress := hex.EncodeToString(morseToShannonMapping.MorseAccount.Address)
			keyringErr := clientCtx.Keyring.Delete(morseToShannonMapping.ShannonAccount.KeyringName)
			if keyringErr != nil {
				logger.Logger.Error().Err(keyringErr).
					Str("name", morseToShannonMapping.ShannonAccount.KeyringName).
					Str("shannon", shannonAddress).
					Str("morse", morseAddress).
					Msg("failed to delete key from the keyring after a handled error")
			}
		}
	}()

	// Write the migration results to an output JSON file at the end of execution.
	defer func() {
		logger.Logger.Info().Msgf("writing migration output JSON to %s", flagOutputFilePath)
		outputJSON, _ := json.MarshalIndent(migrationBatchResult, "", "  ")
		writeErr := os.WriteFile(flagOutputFilePath, outputJSON, 0644)
		if writeErr != nil {
			logger.Logger.Error().Err(writeErr).
				Msgf("Printed migration output JSON due to error writing file to %s", flagOutputFilePath)
			println(outputJSON)
		}
	}()

	// Read Morse private keys from file and ensure its not empty.
	morseNodeAccounts, nodeMorseKeysErr := getMorseAccountsFromFile(flagMorsePrivateKeysFile)
	if nodeMorseKeysErr != nil {
		return nodeMorseKeysErr
	}
	if len(morseNodeAccounts) == 0 {
		return fmt.Errorf(
			"zero Morse nodes found in the file %s. Check the logs and the input file before trying again",
			flagInputFilePath,
		)
	}
	logger.Logger.Info().Int("morse_accounts_count", len(morseNodeAccounts)).Msg("Loaded Morse private keys successfully")

	// Prepare the slice of claim messages for the bulk claim transaction.
	claimMessages := make([]cosmossdk.Msg, 0)

	// ownerAddressToMClaimableAccountMap maps Morse output addresses to MorseClaimableAccounts.
	// - Holds any claimable account associated with a MorseOutputAddress.
	// - For custodial: the claimable account has the same address.
	// - For non-custodial: the claimable account has a different address.
	ownerAddressToMClaimableAccountMap := map[string]*types.MorseClaimableAccount{}
	logger.Logger.Info().Msg("Starting the claim process for each Morse node...")
	// Iterate over each Morse node private key to process migration.
	for idx, morseNodeAccount := range morseNodeAccounts {
		morseNodeAddress := hex.EncodeToString(morseNodeAccount.Address)
		logger.Logger.Info().Str("morse_node_address", morseNodeAddress).Msgf("Processing Morse node #%d", idx+1)

		// Ensure the Morse output address is not empty (i.e., is a node)
		claimableMorseNode, isNode, morseNodeError := queryMorseClaimableAccount(ctx, clientCtx, morseNodeAddress)
		if morseNodeError != nil {
			return morseNodeError
		}
		if !isNode {
			return fmt.Errorf("morse node address '%s' is not a node", morseNodeAddress)
		}
		morseOutputAddress := claimableMorseNode.MorseOutputAddress

		// the right way to know if a node is isCustodial or not
		isCustodial := strings.EqualFold(morseOutputAddress, morseNodeAddress)
		if isCustodial {
			// Populate the ownerAddressMap
			ownerAddressToMClaimableAccountMap[morseOutputAddress] = claimableMorseNode
		} else {
			logger.Logger.Info().
				Str("morse_output_address", claimableMorseNode.MorseOutputAddress).
				Msg("Checking MorseOutputAddress exists as MorseClaimableAccount and is already migrated.")
			// Non-custodial: load and cache MorseClaimableAccount for output address
			claimableMorseAccount, outputAddressIsNode, morseAccountError := queryMorseClaimableAccount(ctx, clientCtx, morseOutputAddress)
			if morseAccountError != nil {
				return morseAccountError
			}
			if outputAddressIsNode {
				// TODO_TECHDEBT(@olshansky): Re-evaluate if/how this can happen and tackle it separately.
				return fmt.Errorf("the bulk claim tool does not have support for non-custodial nodes when the morse output address '%s' is a node", morseOutputAddress)
			}

			if !claimableMorseAccount.IsClaimed() {
				return fmt.Errorf("morse output address '%s' is not claimed", morseOutputAddress)
			}
			// Populate the ownerAddressMap
			ownerAddressToMClaimableAccountMap[morseOutputAddress] = claimableMorseAccount
		}

		if ownerAddressToMClaimableAccountMap[morseOutputAddress] == nil {
			return fmt.Errorf("failed to load MorseClaimableAccount for owner address '%s'", morseOutputAddress)
		}

		// Create a new Shannon account for the migration:
		// - Generate a secp256k1 private key
		// - Derive Cosmos (bech32) address from public key
		// - Register operator address in keyring and mapping
		shannonPrivateKey := secp256k1.GenPrivKey()
		shannonAddress := cosmossdk.AccAddress(shannonPrivateKey.PubKey().Address())
		shannonOperatorAddress := shannonAddress.String()
		morseShannonMapping := MorseShannonMapping{
			MorseAccount: MorseAccountInfo{
				Address:    morseNodeAccount.Address,
				PrivateKey: morseNodeAccount.PrivateKey,
			},
			ShannonAccount: &ShannonAccountInfo{
				Address:    shannonAddress,
				PrivateKey: *shannonPrivateKey,
			},
		}
		logger.Logger.Info().
			Str("shannon_operator_address", shannonAddress.String()).
			Str("morse_node_address", hex.EncodeToString(morseNodeAccount.Address)).
			Msg("Generated new Shannon account")

		// Determine the owner address for the supplier stake config.
		var shannonOwnerAddress string
		if isCustodial {
			// Custodial: use the same address for owner and operator
			shannonOwnerAddress = morseShannonMapping.ShannonAccount.Address.String()
		} else {
			// Non-custodial: use the MorseSrcAddress as the owner address
			shannonOwnerAddress = ownerAddressToMClaimableAccountMap[morseOutputAddress].ShannonDestAddress
		}

		// Build the supplier stake config for this migration.
		supplierStakeConfig, supplierStakeConfigErr := buildSupplierStakeConfig(
			shannonOwnerAddress,
			shannonOperatorAddress,
			templateSupplierStakeConfig,
		)
		if supplierStakeConfigErr != nil {
			return supplierStakeConfigErr
		}

		// Construct a MsgClaimMorseSupplier message for this migration.
		msgClaimMorseSupplier, claimSupplierMsgErr := types.NewMsgClaimMorseSupplier(
			supplierStakeConfig.OwnerAddress,
			supplierStakeConfig.OperatorAddress,
			morseNodeAddress,
			morseNodeAccount.PrivateKey, // morse operator private key
			supplierStakeConfig.Services,
			shannonSigningAddr,
		)
		if claimSupplierMsgErr != nil {
			return claimSupplierMsgErr
		}

		// Import the generated Shannon private key into the keyring.
		keyName := shannonOperatorAddress
		if flagNewKeyPrefix != "" {
			// The default suffix is the address
			suffix := shannonOperatorAddress
			// Override the default suffix if the index flag is provided
			if flagUseIndexNames {
				// avoid it the first item been 0
				suffix = fmt.Sprintf("%d", idx+1)
			}
			// The final keyring name is: 'prefix-idx+1' or 'prefix-address'
			keyName = fmt.Sprintf("%s-%s", flagNewKeyPrefix, suffix)
		}

		morseShannonMapping.ShannonAccount.KeyringName = keyName
		logger.Logger.Info().
			Str("name", keyName).
			Str("morse_node_address", morseNodeAddress).
			Str("shannon_operator_address", shannonOperatorAddress).
			Msg("Storing shannon operator address into the keyring")
		keyringErr := clientCtx.Keyring.ImportPrivKeyHex(
			keyName,
			hex.EncodeToString(morseShannonMapping.ShannonAccount.PrivateKey.Key),
			"secp256k1",
		)
		if keyringErr != nil {
			logger.Logger.Error().Msg("failed to import private key for shannon account, please check the logs and try again")
			return keyringErr
		}

		// Record the migration message attempted (for debugging).
		morseShannonMapping.MigrationMsg = msgClaimMorseSupplier
		// Add the claim message to the transaction batch.
		claimMessages = append(claimMessages, msgClaimMorseSupplier)
		// Add the mapping to the migration result.
		migrationBatchResult.Mappings = append(migrationBatchResult.Mappings, &morseShannonMapping)
	}

	if flagDryRunClaim {
		logger.Logger.Info().
			Str("path", flagOutputFilePath).
			Msg("tx IS NOT being broadcasted because: '--dry-run-claim=true'.")
		return nil
	}

	// Construct a tx client.
	logger.Logger.Info().Msg("Preparing transaction client")
	txClient, err := flags.GetTxClientFromFlags(ctx, cmd)
	if err != nil {
		return err
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
		logger.Logger.Error().Msgf("error migrating morse suppliers: %s. \n Check the output file for more details: %s", migrationBatchResult.Error, flagOutputFilePath)
	} else {
		logger.Logger.Info().
			Int("tx_messages", len(claimMessages)).
			Str("tx_hash", migrationBatchResult.TxHash).
			Msg("Morse suppliers migration tx delivered successfully")
	}

	return nil
}

// getMorseAccountsFromFile reads a file containing Morse private keys and parses them into MorseAccountInfo objects.
//
// - morseNodesFile: path to input JSON file with hex-encoded Morse private keys.
// - Returns: slice of MorseAccountInfo with parsed private keys and addresses.
// - Returns error if file is missing, malformed, or contains invalid data.
func getMorseAccountsFromFile(morseNodesFile string) ([]MorseAccountInfo, error) {
	var morsePrivateKeys []string
	var morseAccounts []MorseAccountInfo

	// Read the input file contents.
	fileContents, fileContentErr := os.ReadFile(morseNodesFile)
	if fileContentErr != nil {
		return nil, fileContentErr
	}
	if err := json.Unmarshal(fileContents, &morsePrivateKeys); err != nil {
		return nil, err
	}

	if len(morsePrivateKeys) == 0 {
		return nil, fmt.Errorf("Zero morse private keys found in %s. Check the logs and the input file before trying again.", morseNodesFile)
	}

	// Parse each hex-encoded Morse private key and derive its address.
	for _, morsePrivateKeyStr := range morsePrivateKeys {
		morseHexKey, hexKeyErr := hex.DecodeString(morsePrivateKeyStr)
		if hexKeyErr != nil {
			return nil, hexKeyErr
		}
		morsePrivateKey := ed25519.PrivKey(morseHexKey)
		morseAddress := morsePrivateKey.PubKey().Address()

		// Add Morse account info to migration list.
		morseAccounts = append(morseAccounts, MorseAccountInfo{
			PrivateKey: morsePrivateKey,
			Address:    morseAddress,
		})
	}

	return morseAccounts, nil
}

// loadTemplateSupplierStakeConfigYAML loads, parses, and validates the supplier stake
// config from a YAML file for the bulk claiming process.
//
// - Stake amount is omitted (set by protocol logic).
// - Owner/operator addresses should not be set in the template.
// - Only service configs and revenue share settings should be included.
func loadTemplateSupplierStakeConfigYAML(configYAMLPath string) (*config.YAMLStakeConfig, error) {
	// Read the YAML file from the provided path.
	yamlStakeConfigBz, err := os.ReadFile(configYAMLPath)
	if err != nil {
		return nil, err
	}

	// Unmarshal the YAML into a config.YAMLStakeConfig struct.
	var yamlStakeConfig config.YAMLStakeConfig
	if err = yaml.Unmarshal(yamlStakeConfigBz, &yamlStakeConfig); err != nil {
		return nil, err
	}

	if len(yamlStakeConfig.Services) == 0 {
		return nil, fmt.Errorf("no services provided in the template")
	}

	if yamlStakeConfig.OwnerAddress != "" {
		return nil, fmt.Errorf("owner address should not be set in the template")
	}

	if yamlStakeConfig.OperatorAddress != "" {
		return nil, fmt.Errorf("operator address should not be set in the template")
	}

	if yamlStakeConfig.StakeAmount != "" {
		return nil, fmt.Errorf("stake amount should not be set in the template")
	}

	return &yamlStakeConfig, nil
}

// - Returns the claimable morse account
// - Returns a boolean indicating if the account is a node or an account
//
// queryMorseClaimableAccount retrieves a MorseClaimableAccount from Shannon's
// onchain state and checks if its a node (staked) or an account (unstaked).
//
// - morseAddress: hex-encoded Morse address to query.
// - Returns: MorseClaimableAccount, isNode flag, or error if not found or invalid.
func queryMorseClaimableAccount(
	ctx context.Context,
	clientCtx cosmosclient.Context,
	morseAddress string,
) (*types.MorseClaimableAccount, bool, error) {
	// Ensure the Morse address is hex-encoded.
	if _, err := hex.DecodeString(morseAddress); err != nil {
		return nil, false, fmt.Errorf("expected morse operating address to be hex-encoded, got: %q", morseAddress)
	}

	// Prepare a new query client for MorseClaimableAccount.
	queryClient := types.NewQueryClient(clientCtx)

	// Query the MorseClaimableAccount from Shannon's onchain state.
	req := &types.QueryMorseClaimableAccountRequest{Address: morseAddress}
	res, queryErr := queryClient.MorseClaimableAccount(ctx, req)
	if queryErr != nil {
		// Stop and return error if the Morse account cannot be validated.
		return nil, false, queryErr
	}

	morseOutputAddress := res.MorseClaimableAccount.MorseOutputAddress

	isNode := morseOutputAddress != ""

	return &res.MorseClaimableAccount, isNode, nil
}

// buildSupplierStakeConfig creates and validates a new Shannon SupplierStakeConfig.
//
// - owner: The owner address (must not be empty).
// - operatorAddress: The operator address (if empty, defaults to the owner address).
// - templateSupplierStakeConfig: The YAMLStakeConfig template to start from.
// - Ensures revenue share sums to 100% (adds owner if needed).
// - Returns: validated SupplierStakeConfig or error.
func buildSupplierStakeConfig(
	ownerAddress string,
	operatorAddress string,
	templateSupplierStakeConfig *config.YAMLStakeConfig,
) (*config.SupplierStakeConfig, error) {
	logger.Logger.Info().
		Str("owner", ownerAddress).
		Str("operator", operatorAddress).
		Msg("Building stake config")

	if ownerAddress == "" {
		return nil, fmt.Errorf("owner address must be non-empty")
	}
	if operatorAddress == "" {
		return nil, fmt.Errorf("operator address must be non-empty")
	}

	// clone the provided template.
	// DefaultRevSharePercent is a map which is a reference if assign to the new one
	// Services are pointers so clone them too
	yamlStakeConfig := &config.YAMLStakeConfig{
		OwnerAddress:    ownerAddress,
		OperatorAddress: operatorAddress,
		StakeAmount:     "", // intentionally empty
		Services:        make([]*config.YAMLStakeService, 0),
		DefaultRevSharePercent: func() map[string]uint64 {
			m := make(map[string]uint64, len(templateSupplierStakeConfig.DefaultRevSharePercent))
			for k, v := range templateSupplierStakeConfig.DefaultRevSharePercent {
				m[k] = v
			}
			return m
		}(),
	}

	for _, service := range templateSupplierStakeConfig.Services {
		// clone to avoid modify the template on multiple iterations
		yamlStakeConfig.Services = append(yamlStakeConfig.Services, &config.YAMLStakeService{
			ServiceId:       service.ServiceId,
			RevSharePercent: service.RevSharePercent,
			Endpoints:       service.Endpoints,
		})
	}

	// Validate the owner and operator addresses.
	if err := yamlStakeConfig.ValidateAndNormalizeAddresses(logger.Logger); err != nil {
		return nil, err
	}

	// Validate the default revenue share map.
	defaultRevShareMap, err := yamlStakeConfig.ValidateAndNormalizeDefaultRevShare()
	if err != nil {
		return nil, err
	}

	// ===== Start TECHDEBT codeblock =====
	// TODO_TECHDEBT(#1372): Re-evaluate this codeblock.
	// See the discussion here: https://github.com/pokt-network/poktroll/pull/1386#discussion_r2115168813

	if len(yamlStakeConfig.DefaultRevSharePercent) != 0 {
		if yamlStakeConfig.DefaultRevSharePercent, err = updateRevShareMapToFullAllocation(ownerAddress, operatorAddress, defaultRevShareMap); err != nil {
			return nil, err
		}
	}

	for _, service := range yamlStakeConfig.Services {
		if len(service.RevSharePercent) == 0 {
			// set the same rev share per service as default which already includes the owner and enforce 100%
			service.RevSharePercent = yamlStakeConfig.DefaultRevSharePercent
			continue
		}
		service.RevSharePercent, err = updateRevShareMapToFullAllocation(ownerAddress, operatorAddress, service.RevSharePercent)
		if err != nil {
			return nil, err
		}
	}
	// ===== End TECHDEBT codeblock =====

	// Validate and parse the service configs.
	supplierServiceConfigs, err := yamlStakeConfig.ValidateAndParseServiceConfigs(defaultRevShareMap)
	if err != nil {
		return nil, err
	}

	return &config.SupplierStakeConfig{
		OwnerAddress:    yamlStakeConfig.OwnerAddress,
		OperatorAddress: yamlStakeConfig.OperatorAddress,
		Services:        supplierServiceConfigs,
		// StakeAmount: (intentionally omitted),
		// The Supplier stake amount is determined by the sum of both of the following:
		// 1. Any existing Shannon supplier stake
		// 2. The supplier stake amount of the associated MorseClaimableAccount.
	}, nil
}

// updateRevShareMapToFullAllocation ensures that the sum of revenue shares in revShareMap equals 100%.
// - Adds or adjusts the owner's share if necessary.
// - If the owner isn't present, it's added to balance the total to 100%.
// - Returns: updated map or error.
func updateRevShareMapToFullAllocation(owner, operator string, revShareMap map[string]uint64) (map[string]uint64, error) {
	totalShare := uint64(0)
	ownerFound := false

	for address, share := range revShareMap {
		totalShare += share
		revShareMap[address] = share
		if strings.EqualFold(address, owner) {
			ownerFound = true
		}
	}

	// Allows the operator to automatically be added to the revshare map to get funds on every reward.
	// This covers the operator's tx fees and increases their rewards.
	if flagSetOperatorShare > 0 {
		if totalShare > 100 {
			return nil, fmt.Errorf(
				"invalid revenue share configuration: total revenue share exceeds 100%% (current: %d%%, attempted to add operator share: %d%%)",
				totalShare, flagSetOperatorShare,
			)
		}
		totalShare += flagSetOperatorShare
		revShareMap[operator] = flagSetOperatorShare
	}

	// if the owner is not part of the list, add it with the difference.
	// this avoids the need for the user to add it.
	if !ownerFound && totalShare < 100 && totalShare > 0 {
		revShareMap[owner] = 100 - totalShare
	}

	return revShareMap, nil
}
