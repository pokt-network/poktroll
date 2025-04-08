package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/pokt-network/poktroll/cmd/flags"
	"github.com/pokt-network/poktroll/cmd/logger"
	"github.com/pokt-network/poktroll/x/migration/types"
	"github.com/pokt-network/poktroll/x/supplier/config"
)

func ClaimSupplierCmd() *cobra.Command {
	claimSupplierCmd := &cobra.Command{
		Use:   "claim-supplier [morse_key_export_path] [path_to_supplier_stake_config] --from [shannon_dest_key_name]",
		Args:  cobra.ExactArgs(2),
		Short: "Claim an onchain MorseClaimableAccount as a staked supplier account",
		Long: `Claim an onchain MorseClaimableAccount as a staked supplier account.

The unstaked balance amount of the onchain MorseClaimableAccount will be minted to the Shannon account specified by the --from flag.
The Shannon account will also be staked as a supplier with a stake equal to the supplier stake the MorseClaimableAccount had on Morse.

This will construct, sign, and broadcast a tx containing a MsgClaimMorseSupplier message.

For more information, see: https://dev.poktroll.com/operate/morse_migration/claiming`,
		RunE:    runClaimSupplier,
		PreRunE: logger.PreRunESetup,
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

	// Retrieve and validate the morse key based on the first argument provided.
	morseKeyExportPath := args[0]
	morsePrivKey, err := loadMorsePrivateKey(morseKeyExportPath, morseKeyfileDecryptPassphrase)
	if err != nil {
		return err
	}

	// Conventionally derive a cosmos-sdk client context from the cobra command.
	clientCtx, err := cosmosclient.GetClientTxContext(cmd)
	if err != nil {
		return err
	}

	// Load the supplier stake config from the YAML file.
	supplierStakeConfigPath := args[1]
	supplierStakeConfig, err := loadSupplierStakeConfigYAML(supplierStakeConfigPath)
	if err != nil {
		return err
	}

	// Ensure that the signing account matches either the configured owner or operator address.
	signingAddr := clientCtx.GetFromAddress().String()
	ownerAddr := supplierStakeConfig.OwnerAddress
	operatorAddr := supplierStakeConfig.OperatorAddress
	switch signingAddr {
	case ownerAddr, operatorAddr:
		// All good.
	default:
		return fmt.Errorf(
			"signer address %s does not match owner address %s or supplier operator address %s",
			signingAddr, ownerAddr, operatorAddr,
		)
	}

	// Construct a MsgClaimMorseSupplier message.
	msgClaimMorseSupplier, err := types.NewMsgClaimMorseSupplier(
		ownerAddr,
		operatorAddr,
		morsePrivKey,
		supplierStakeConfig.Services,
	)
	if err != nil {
		return err
	}

	// Serialize, as JSON, and print the MsgClaimMorseSupplier for posterity and/or confirmation.
	msgClaimMorseSupplierJSON, err := json.MarshalIndent(msgClaimMorseSupplier, "", "  ")
	if err != nil {
		return err
	}

	fmt.Printf("MsgClaimMorseSupplier %s\n", string(msgClaimMorseSupplierJSON))

	// Last chance for the user to abort.
	skipConfirmation, err := cmd.Flags().GetBool(cosmosflags.FlagSkipConfirmation)
	if err != nil {
		return err
	}

	if !skipConfirmation {
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
	txClient, err := flags.GetTxClient(ctx, cmd)
	if err != nil {
		return err
	}

	// Sign and broadcast the claim Morse account message.
	_, eitherErr := txClient.SignAndBroadcast(ctx, msgClaimMorseSupplier)
	err, errCh := eitherErr.SyncOrAsyncError()
	if err != nil {
		return err
	}

	// Wait for an async error, timeout, or the errCh to close on success.
	return <-errCh
}

// loadSupplierStakeConfigYAML loads, parses, and validates the supplier stake
// config from configYAMLPath.
func loadSupplierStakeConfigYAML(configYAMLPath string) (*config.SupplierStakeConfig, error) {
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

	// Validate that the stake amount is not set in the YAML config.
	if len(yamlStakeConfig.StakeAmount) != 0 {
		return nil, config.ErrSupplierConfigInvalidStake.Wrapf("stake_amount MUST NOT be set in the supplier config YAML; it is automatically determined by the onchain MorseClaimableAccount state")
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

	return &config.SupplierStakeConfig{
		OwnerAddress:    yamlStakeConfig.OwnerAddress,
		OperatorAddress: yamlStakeConfig.OperatorAddress,
		Services:        supplierServiceConfigs,
		// StakeAmount:  (intentionally omitted),
		// The stake amount is determined by the sum of any existing supplier stake
		// and the supplier stake amount of the associated MorseClaimableAccount.
	}, nil
}
