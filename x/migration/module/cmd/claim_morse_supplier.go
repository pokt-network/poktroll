package cmd

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"

	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/pokt-network/poktroll/cmd/flags"
	"github.com/pokt-network/poktroll/cmd/logger"
	"github.com/pokt-network/poktroll/pkg/deps/config"
	"github.com/pokt-network/poktroll/x/migration/types"
	supplierconfig "github.com/pokt-network/poktroll/x/supplier/config"
)

// TODO_MAINNET_MIGRATION: Add a few examples,
func ClaimSupplierCmd() *cobra.Command {
	claimSupplierCmd := &cobra.Command{
		Use:   "claim-supplier [morse_node_address] [morse_private_key_export_path] [path_to_supplier_stake_config] --from [shannon_dest_key_name]",
		Args:  cobra.ExactArgs(3),
		Short: "Claim 1 onchain MorseClaimableAccount as a staked supplier account",
		Long: `
Claim 1 onchain MorseClaimableAccount as a staked supplier account.

morse_node_address: Hex-encoded address of the Morse node account to be claimed

morse_private_key_export_path: Path to the Morse private key for ONE of the following:
  - Morse node account (operator) — custodial
  - Morse output account (owner) — non-custodial

What happens:
  - The unstaked balance of the onchain MorseClaimableAccount will be minted to the Shannon account specified by --from
  - The Shannon account will also be staked as a supplier with a stake equal to the supplier stake the MorseClaimableAccount had on Morse
  - A transaction with MsgClaimMorseSupplier will be constructed, signed, and broadcast

More info: https://dev.poktroll.com/operate/morse_migration/claiming`,

		RunE: runClaimSupplier,
	}

	// Add a string flag for providing a passphrase to decrypt the Morse keyfile.
	claimSupplierCmd.Flags().StringVarP(
		&morseKeyfileDecryptPassphrase,
		flags.FlagPassphrase,
		flags.FlagPassphraseShort,
		"",
		flags.FlagPassphraseUsage,
	)

	// Add a bool flag indicating whether to skip the passphrase prompt.
	claimSupplierCmd.Flags().BoolVar(
		&noPassphrase,
		flags.FlagNoPassphrase,
		false,
		flags.FlagNoPassphraseUsage,
	)

	// This command depends on the conventional cosmos-sdk CLI tx flags.
	cosmosflags.AddTxFlagsToCmd(claimSupplierCmd)

	return claimSupplierCmd
}

// runClaimSupplier performs the following sequence:
// - Load the Morse private key from the morse_key_export_path argument (arg 0).
// - Load and validate the supplier service staking config from the path_to_supplier_stake_config argument pointing to a local config file (arg 1).
// - Sign and broadcast the MsgClaimMorseSupplier message using the Shannon key named by the `--from` flag.
// - Wait until the tx is committed onchain for either a synchronous or asynchronous error.
func runClaimSupplier(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	morseNodeAddr := args[0]
	if _, err := hex.DecodeString(morseNodeAddr); err != nil {
		return fmt.Errorf("expected morse operating address to be hex-encoded, got: %q", morseNodeAddr)
	}

	// Retrieve and validate the morse key based on the provided argument.
	morseKeyExportPath := args[1]
	morsePrivKey, err := LoadMorsePrivateKey(morseKeyExportPath, morseKeyfileDecryptPassphrase, noPassphrase)
	if err != nil {
		return err
	}

	// Conventionally derive a cosmos-sdk client context from the cobra command.
	clientCtx, err := cosmosclient.GetClientTxContext(cmd)
	if err != nil {
		return err
	}

	// Load the supplier stake config from the YAML file.
	supplierStakeConfigPath := args[2]
	supplierStakeConfig, err := loadSupplierStakeConfigYAML(supplierStakeConfigPath)
	if err != nil {
		return err
	}

	// Check and warn if the signing account doesn't match either the configured owner or operator address.
	shannonSigningAddr := clientCtx.GetFromAddress().String()
	shannonOwnerAddr := supplierStakeConfig.OwnerAddress
	shannonOperatorAddr := supplierStakeConfig.OperatorAddress
	switch shannonSigningAddr {
	case shannonOwnerAddr, shannonOperatorAddr:
		// All good.
	default:
		logger.Logger.Warn().
			Str("signer_address", shannonSigningAddr).
			Str("owner_address", shannonOwnerAddr).
			Str("operator_address", shannonOperatorAddr).
			Msg("signer address matches NEITHER owner NOR operator address")
	}

	// Construct a MsgClaimMorseSupplier message.
	msgClaimMorseSupplier, err := types.NewMsgClaimMorseSupplier(
		shannonOwnerAddr,
		shannonOperatorAddr,
		morseNodeAddr,
		morsePrivKey,
		supplierStakeConfig.Services,
		shannonSigningAddr,
	)
	if err != nil {
		return err
	}

	// Print the claim message according to the --output format.
	if err = clientCtx.PrintProto(msgClaimMorseSupplier); err != nil {
		return err
	}

	// Last chance for the user to abort.
	skipConfirmation, err := cmd.Flags().GetBool(cosmosflags.FlagSkipConfirmation)
	if err != nil {
		return err
	}

	if !skipConfirmation {
		// DEV_NOTE: Intentionally using fmt instead of logger.Logger to receive user input on the same line.
		fmt.Printf("Confirm MsgClaimMorseSupplier: y/[n]: ")
		stdinReader := bufio.NewReader(os.Stdin)

		// This call to ReadLine() will block until the user sends a new line to stdin.
		inputLine, _, readErr := stdinReader.ReadLine()
		if readErr != nil {
			return err
		}

		// Terminate the confirmation prompt output line.
		fmt.Println()

		// Abort unless some affirmative confirmation is given.
		switch string(inputLine) {
		case "Yes", "yes", "Y", "y":
		default:
			return nil
		}
	}

	// Construct a tx client.
	txClient, err := config.GetTxClientFromFlags(ctx, cmd)
	if err != nil {
		return err
	}

	// Sign and broadcast the claim Morse account message.
	txResponse, eitherErr := txClient.SignAndBroadcast(ctx, msgClaimMorseSupplier)
	if err, _ = eitherErr.SyncOrAsyncError(); err != nil {
		return err
	}

	// Print the TxResponse according to the --output format.
	if err = clientCtx.PrintProto(txResponse); err != nil {
		return err
	}

	return nil
}

// loadSupplierStakeConfigYAML loads, parses, and validates the supplier stake
// config from configYAMLPath.
func loadSupplierStakeConfigYAML(configYAMLPath string) (*supplierconfig.SupplierStakeConfig, error) {
	// Read the YAML file from the provided path.
	yamlStakeConfigBz, err := os.ReadFile(configYAMLPath)
	if err != nil {
		return nil, err
	}

	// Unmarshal the YAML into a config.YAMLStakeConfig struct.
	var yamlStakeConfig supplierconfig.YAMLStakeConfig
	if err = yaml.Unmarshal(yamlStakeConfigBz, &yamlStakeConfig); err != nil {
		return nil, err
	}

	// Validate that the stake amount is not set in the YAML config.
	if len(yamlStakeConfig.StakeAmount) != 0 {
		return nil, supplierconfig.ErrSupplierConfigInvalidStake.Wrapf("stake_amount MUST NOT be set in the supplier config YAML; it is automatically determined by the onchain MorseClaimableAccount state")
	}

	// Validate the owner and operator addresses.
	if err = yamlStakeConfig.ValidateAndNormalizeAddresses(logger.Logger); err != nil {
		return nil, err
	}

	// Validate the default revenue share map.
	defaultRevShareMap, err := yamlStakeConfig.ValidateAndNormalizeDefaultRevShare()
	if err != nil {
		return nil, err
	}

	// Validate and parse the service configs.
	supplierServiceConfigs, err := yamlStakeConfig.ValidateAndParseServiceConfigs(defaultRevShareMap)
	if err != nil {
		return nil, err
	}

	return &supplierconfig.SupplierStakeConfig{
		OwnerAddress:    yamlStakeConfig.OwnerAddress,
		OperatorAddress: yamlStakeConfig.OperatorAddress,
		Services:        supplierServiceConfigs,
		// StakeAmount:  (intentionally omitted),
		// The stake amount is determined by the sum of any existing supplier stake
		// and the supplier stake amount of the associated MorseClaimableAccount.
	}, nil
}
