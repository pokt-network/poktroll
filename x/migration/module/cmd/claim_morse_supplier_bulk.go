package cmd

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/cometbft/cometbft/crypto/ed25519"
	cosmossdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/poktroll/cmd/flags"

	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/pokt-network/poktroll/cmd/logger"
	"github.com/pokt-network/poktroll/x/migration/types"
	"github.com/pokt-network/poktroll/x/supplier/config"
)

var (
	nodesFile         string
	stakeTemplateFile string
	simulation        bool // I try to use --dry-run, but it collides with a cosmos-sdk flag
)

// this hold any claimable account that is under MorseOutputAddress
// on custodial same node, on non-custodial another account.
var ownerAddressMap = map[string]*types.MorseClaimableAccount{}

func ClaimSupplierBulkCmd() *cobra.Command {
	claimSuppliersCmd := &cobra.Command{
		Use:   "claim-suppliers --from [shannon_dest_key_name]",
		Args:  cobra.ExactArgs(0),
		Short: "Claim many onchain MorseClaimableAccount as a staked supplier accounts.",
		Long: `Claim multiple on-chain MorseClaimableAccounts as staked supplier accounts.
Pre-Requisites:
- If Morse node is non-custodial, the output address needs to be claimed on Shannon with MsgClaimMorseAccount.

What it does:
- Reads Morse node keys and stake template from files.
- Migrates/claims each Node account to a new Shannon supplier account (custodial or non-custodial).
- Generates and stores keys for each supplier.
- Supports simulation mode—doesn't broadcast the transaction if enabled.
- Outputs migration results to a JSON file.
- Optional: Can export unarmored (plain) JSON output with sensitive keys, if flagged as unsafe.

Flags:
--nodes - Operator keys json list
--stake-template-file - Path to a stake template file detailing services, reward shares, etc. The owner, operator, and stake amount are automatically sourced from ClaimableAccounts.
--simulate - (Default: false) If true, the transaction will be simulated but not broadcasted.
--output-file - (Default: migration_output.json) Path to a file where the migration result will be written.
--unsafe - (Default: false) allow the usage of --unarmored-json flag
--unarmored-json - (Default: false) allow JSON marshaling of the migration result to include morse/shannon private keys on it.

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

// I need to generate the shannon private keys for every operator - why? because olshansky fuck on me about me previous cli cmd :'(

YAML template:
owner_address: [DEDUCTED SHANNON OWNER ADDRESS] - no need to set
operator_address: [DEDUCTED SHANNON NODE ADDRESS] - no need to set
stake_amount: [DEDUCTED SHANNON NODE STAKE AMOUNT] - no need to set
default_rev_share_percent:
  <DELEGATOR_REWARDS_SHANNON_ADDRESS>: 75
  [DEDUCTED SHANNON OWNER ADDRESS]: [DEDUCTED OWNER SHARE - NO NEED TO INCLUDE] -> 100 - $DELEGATOR_REWARDS_SHANNON_ADDRESS = 25
services:
  - service_id: "anvil"
    endpoints:
      - publicly_exposed_url: https://rm1.somewhere.com
        rpc_type: JSON_RPC
	rev_share_percent:
      <DELEGATOR_REWARDS_SHANNON_ADDRESS>: 90
	  [DEDUCTED SHANNON OWNER ADDRESS]: [DEDUCTED OWNER SHARE - NO NEED TO INCLUDE] -> 100 - $DELEGATOR_REWARDS_SHANNON_ADDRESS = 10
  - service_id: "eth"
    endpoints:
      - publicly_exposed_url: https://rm1.somewhere.com
        rpc_type: JSON_RPC
	rev_share_percent: [FILLED WITH VALUES AT default_rev_share_percent] because is empty.

More info: https://dev.poktroll.com/operate/morse_migration/claiming`,

		RunE:    runClaimSuppliers,
		PreRunE: logger.PreRunESetup,
	}

	claimSuppliersCmd.Flags().BoolVarP(
		&simulation,
		"simulate",
		"",
		false,
		"If true, the transaction will be simulated but not broadcasted.",
	)

	claimSuppliersCmd.Flags().StringVarP(
		&nodesFile,
		"nodes-file",
		"",
		"",
		"Path to a file listing Morse node keys",
	)

	claimSuppliersCmd.Flags().StringVarP(
		&stakeTemplateFile,
		"stake-template-file",
		"",
		"",
		"Path to a stake template file detailing services, reward shares, etc. The owner, operator, and stake amount are automatically sourced from ClaimableAccounts.",
	)

	claimSuppliersCmd.Flags().StringVarP(
		&outputFilePath, // declared at claim_morse_account_bulk.go - Should it be somewhere else?
		flags.FlagOutputFile,
		flags.FlagOutputFileShort,
		"",
		"Path to a file where the migration result will be written.",
	)

	// Prepare the unsafe flag.
	claimSuppliersCmd.Flags().BoolVarP(
		&unsafe,
		"unsafe",
		"",
		false,
		"unsafe operation, do not auto-load the shannon account private keys into the keyring",
	)

	// Prepare the unarmored JSON flag.
	claimSuppliersCmd.Flags().BoolVarP(
		&unarmoredJSON,
		"unarmored-json",
		"",
		false,
		"unarmored JSON output file, this is useful for the migration of operators into a shannon keyring for later use",
	)

	// This command depends on the conventional cosmos-sdk CLI tx flags.
	cosmosflags.AddTxFlagsToCmd(claimSuppliersCmd)

	return claimSuppliersCmd
}

// runClaimSuppliers runs the claim suppliers command.
func runClaimSuppliers(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	// Conventionally derive a cosmos-sdk client context from the cobra command.
	logger.Logger.Info().Msg("Configuring cosmos client")
	clientCtx, err := cosmosclient.GetClientTxContext(cmd)
	if err != nil {
		return err
	}

	shannonSigningAddr := clientCtx.GetFromAddress().String()
	if shannonSigningAddr == "" {
		return fmt.Errorf("no shannon signing address provided using --from")
	}

	// Load the supplier stake config template from the YAML file.
	// lets fail faster if something here is wrong (missing file for example)
	logger.Logger.Info().Msgf("Loading stake template file: %s", stakeTemplateFile)
	templateSupplierStakeConfig, templateError := loadTemplateSupplierStakeConfigYAML(stakeTemplateFile)
	if templateError != nil {
		return templateError
	}

	// Prepare the migration batch result.
	migrationBatchResult := MigrationBatchResult{
		Mappings: make([]*MorseShannonMapping, 0),
		Error:    "",
		TxHash:   "",
	}

	// Clean up keyring if tx results in an error.
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

	// Write output JSON file-to --output-file path.
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

	nodeMorseKeys, nodeMorseKeysErr := readMorseNodesPrivateKeysFile(nodesFile)
	if nodeMorseKeysErr != nil {
		return nodeMorseKeysErr
	}

	if len(nodeMorseKeys) == 0 {
		return fmt.Errorf("no morse nodes found in the file. Check the logs and the input file before trying again")
	}

	// Prepare the claimMessages slice.
	claimMessages := make([]cosmossdk.Msg, 0)

	for i := range nodeMorseKeys {
		custodial := false
		morseNode := nodeMorseKeys[i]
		// this will ensure the morse output address is not empty, which means is a node
		claimableMorseNode, morseNodeError := loadMorseClaimableAccount(
			ctx, clientCtx,
			hex.EncodeToString(morseNode.Address),
			true,
		)
		if morseNodeError != nil {
			return morseNodeError
		}

		if ownerAddressMap[claimableMorseNode.MorseOutputAddress] == nil {
			// non-custodial
			if claimableMorseNode.MorseOutputAddress != claimableMorseNode.MorseSrcAddress {
				// let's load it
				// this will ensure morse output address is already claimed on shannon
				claimableMorseAccount, morseAccountError := loadMorseClaimableAccount(
					ctx, clientCtx,
					claimableMorseNode.MorseOutputAddress,
					false,
				)
				if morseAccountError != nil {
					return morseAccountError
				}
				// add this to the map for a later query avoiding more network calls
				ownerAddressMap[claimableMorseNode.MorseOutputAddress] = claimableMorseAccount
			} else {
				// custodial
				custodial = true
				ownerAddressMap[claimableMorseNode.MorseOutputAddress] = claimableMorseNode
			}
		}

		// Create a new Shannon account
		// 1. Generate a secp256k1 private key
		shannonPrivateKey := secp256k1.GenPrivKey()
		// 2. Get the Cosmos shannonAddress (bech32 format) from a public key
		shannonAddress := cosmossdk.AccAddress(shannonPrivateKey.PubKey().Address())
		// 3. Get Shannon address to register on keyring and build stake configuration
		shannonOperatorAddress := shannonAddress.String()
		// 3. Store the relationship for the operator address between morse and shannon accounts
		morseShannonMapping := MorseShannonMapping{
			MorseAccount: MorseAccountInfo{
				Address:    morseNode.Address,
				PrivateKey: morseNode.PrivateKey,
			},
			ShannonAccount: &ShannonAccountInfo{
				Address:    shannonAddress,
				PrivateKey: *shannonPrivateKey,
			},
		}

		ownerAddress := ownerAddressMap[claimableMorseNode.MorseOutputAddress].ShannonDestAddress

		if custodial {
			ownerAddress = morseShannonMapping.ShannonAccount.Address.String()
		}

		supplierStakeConfig, supplierStakeConfigErr := buildSupplierStakeConfig(
			// this needs to be claimed as a protocol rule
			ownerAddress,
			shannonOperatorAddress,
			templateSupplierStakeConfig,
		)

		if supplierStakeConfigErr != nil {
			return supplierStakeConfigErr
		}

		// Construct a MsgClaimMorseSupplier message.
		msgClaimMorseSupplier, msgErr := types.NewMsgClaimMorseSupplier(
			supplierStakeConfig.OwnerAddress,
			supplierStakeConfig.OperatorAddress,
			claimableMorseNode.MorseSrcAddress,
			morseNode.PrivateKey, // morse operator private key
			supplierStakeConfig.Services,
			shannonSigningAddr,
		)
		if msgErr != nil {
			return msgErr
		}

		// Import a Shannon private key into a keyring.
		keyringErr := clientCtx.Keyring.ImportPrivKeyHex(
			shannonOperatorAddress,
			hex.EncodeToString(morseShannonMapping.ShannonAccount.PrivateKey.Key),
			"secp256k1",
		)
		if keyringErr != nil {
			logger.Logger.Error().Msg("failed to import private key for shannon account, please check the logs and try again")
			return keyringErr
		}

		// For debugging: a record migration message attempted.
		morseShannonMapping.MigrationMsg = msgClaimMorseSupplier
		// append to the list of messages that will be delivery on the transaction
		claimMessages = append(claimMessages, msgClaimMorseSupplier)
		// append the relationship to the migration result
		migrationBatchResult.Mappings = append(migrationBatchResult.Mappings, &morseShannonMapping)
	}

	// Construct a tx client.
	txClient, err := flags.GetTxClientFromFlags(ctx, cmd)
	if err != nil {
		return err
	}

	if simulation {
		logger.Logger.Info().Msgf(
			"--simulate=true - tx will be broadcasted, please check %s file for more details.",
			outputFilePath,
		)
		return nil
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
		logger.Logger.Error().Msgf("error migrating morse suppliers: %s. \n Check the output file for more details: %s", migrationBatchResult.Error, outputFilePath)
	} else {
		logger.Logger.Info().
			Int("tx_messages", len(claimMessages)).
			Str("tx_hash", migrationBatchResult.TxHash).
			Msg("morse suppliers migration tx delivered successfully")
	}

	return nil
}

// readMorseNodesPrivateKeysFile reads a file containing Morse private keys and parses them into MorseAccountInfo objects.
// nodesFile is the path to the input file containing Morse private keys in JSON format.
// Returns a slice of MorseAccountInfo containing parsed private keys and their corresponding addresses, or an error.
// Errors are returned if the file does not exist, is improperly formatted, or contains invalid data.
func readMorseNodesPrivateKeysFile(nodesFile string) ([]MorseAccountInfo, error) {
	// Prepare a new morse private keys slice.
	var morsePrivateKeys []string
	var morseAccounts []MorseAccountInfo

	// Read the input file.
	fileContents, fileContentErr := os.ReadFile(nodesFile)
	if fileContentErr != nil {
		return nil, fileContentErr
	}
	if err := json.Unmarshal(fileContents, &morsePrivateKeys); err != nil {
		return nil, err
	}

	if len(morsePrivateKeys) == 0 {
		return nil, fmt.Errorf("no morse private keys found in the file. Check the logs and the input file before trying again")
	}

	for _, morsePrivateKeyStr := range morsePrivateKeys {
		morseHexKey, hexKeyErr := hex.DecodeString(morsePrivateKeyStr)
		if hexKeyErr != nil {
			return nil, hexKeyErr
		}
		morsePrivateKey := ed25519.PrivKey(morseHexKey)
		morseAddress := morsePrivateKey.PubKey().Address()

		// Morse account is in the snapshot and not yet claimed.
		// Add it to a migration list.
		morseAccounts = append(morseAccounts, MorseAccountInfo{
			PrivateKey: morsePrivateKey,
			Address:    morseAddress,
		})
	}

	return morseAccounts, nil
}

// loadTemplateSupplierStakeConfigYAML loads, parses, and validates the supplier stake
// config from configYAMLPath as a template for the bulk claiming process.
//
// The stake amount is not set in the config, as it is determined by the sum of any
// existing supplier stake and the supplier stake amount of the associated
// MorseClaimableAccount.
//
// The owner and operator addresses should not be set in the config.
//
// This should mostly include services configs.
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

	return &yamlStakeConfig, nil
}

// loadMorseClaimableAccount retrieves a MorseClaimableAccount based on the provided address and validates its state.
// It uses the given context and client context to interact with the blockchain.
// If shouldBeNode is true, the function ensures the MorseClaimableAccount is associated with a node account.
func loadMorseClaimableAccount(
	ctx context.Context,
	clientCtx cosmosclient.Context,
	address string,
	shouldBeNode bool,
) (*types.MorseClaimableAccount, error) {
	if _, err := hex.DecodeString(address); err != nil {
		return nil, fmt.Errorf("expected morse operating address to be hex-encoded, got: %q", address)
	}
	// Prepare a new query client.
	queryClient := types.NewQueryClient(clientCtx)

	req := &types.QueryMorseClaimableAccountRequest{Address: address}
	res, queryErr := queryClient.MorseClaimableAccount(ctx, req)
	// if we are not able to validate the existence of the morse account, we return the error and stop the process
	if queryErr != nil {
		return nil, queryErr
	}

	if shouldBeNode && res.MorseClaimableAccount.MorseOutputAddress == "" {
		return nil, fmt.Errorf("morse account %s if not a node account: %v", address, res.MorseClaimableAccount)
	} else if res.MorseClaimableAccount.MorseOutputAddress == "" && !res.MorseClaimableAccount.IsClaimed() {
		// morse account (unstaked) need to be claimed before attempting to claim them as supplier in shannon.
		return nil, fmt.Errorf("morse account %s if not claimed yet: %v", address, res.MorseClaimableAccount)
	}

	return &res.MorseClaimableAccount, nil
}

// buildSupplierStakeConfig creates and validates a SupplierStakeConfig based on given parameters and a YAML template.
// Parameters:
// • owner: The owner address (must not be empty).
// • operator: The operator address (if empty, defaults to the owner address).
// • templateSupplierStakeConfig: The YAMLStakeConfig template to start from.
// • default_rev_share|service.*.rev_share: fulfills share to fit 100% with the owner address because it is required.
// Returns a validated SupplierStakeConfig instance or an error if any validation fails.
func buildSupplierStakeConfig(
	owner string,
	operator string,
	templateSupplierStakeConfig *config.YAMLStakeConfig,
) (*config.SupplierStakeConfig, error) {
	if owner == "" {
		return nil, fmt.Errorf("owner address is required")
	}
	if operator == "" {
		return nil, fmt.Errorf("operator address is required")
	}

	yamlStakeConfig := *templateSupplierStakeConfig // clone to avoid mistakes using the template.
	yamlStakeConfig.OwnerAddress = owner
	yamlStakeConfig.OperatorAddress = operator

	// Validate the owner and operator addresses.
	err := yamlStakeConfig.ValidateAndNormalizeAddresses(logger.Logger)
	if err != nil {
		return nil, err
	}

	// Validate the default revenue share map.
	defaultRevShareMap, err := yamlStakeConfig.ValidateAndNormalizeDefaultRevShare()
	if err != nil {
		return nil, err
	}

	if len(yamlStakeConfig.DefaultRevSharePercent) != 0 {
		yamlStakeConfig.DefaultRevSharePercent, err = enforce100Percent(owner, defaultRevShareMap)
		if err != nil {
			return nil, err
		}
	}

	for i := range yamlStakeConfig.Services {
		service := yamlStakeConfig.Services[i]
		if len(service.RevSharePercent) == 0 {
			// set the same rev share per service as default which already includes the owner and enforce 100%
			service.RevSharePercent = defaultRevShareMap
			continue
		}

		service.RevSharePercent, err = enforce100Percent(owner, service.RevSharePercent)
		if err != nil {
			return nil, err
		}
	}

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
		// The stake amount is determined by the sum of any existing supplier stake
		// and the supplier stake amount of the associated MorseClaimableAccount.
	}, nil
}

// enforce100Percent ensures that the sum of revenue shares in revShareMap equals 100%,
// adjusting the owner's share if necessary.
// If the owner isn't present, it's added to the map with a share that balances the total to 100%.
// Returns the updated map or an error.
func enforce100Percent(owner string, revShareMap map[string]uint64) (map[string]uint64, error) {
	totalShare := uint64(0)
	ownerFound := false

	for address, share := range revShareMap {
		totalShare += share
		if strings.EqualFold(address, owner) {
			ownerFound = true
		}
	}

	// if the owner is not part of the list, add it with the difference.
	// this avoids the need for the user to add it.
	if !ownerFound && totalShare < 100 && totalShare > 0 {
		revShareMap[owner] = 100 - totalShare
	}

	return revShareMap, nil
}
