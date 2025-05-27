package cmd

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	cosmosquery "github.com/cosmos/cosmos-sdk/types/query"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/pokt-network/poktroll/cmd/logger"
	"github.com/pokt-network/poktroll/x/migration/types"
	"github.com/pokt-network/poktroll/x/supplier/config"
)

var (
	outputAddress     string
	all               bool
	nodes             []string
	nodesFile         string
	stakeTemplateFile string
)

func ClaimSupplierBulkCmd() *cobra.Command {
	claimSuppliersCmd := &cobra.Command{
		Use:   "claim-suppliers --from [shannon_dest_key_name] --output-address [morse_output_address]",
		Args:  cobra.ExactArgs(0),
		Short: "Claim many onchain MorseClaimableAccount as a staked supplier accounts> Claim multiple on-chain MorseClaimableAccounts as staked supplier accounts.",
		Long: `Claim multiple on-chain MorseClaimableAccounts as staked supplier accounts.

Flags:
--from - Specify the Shannon account that will sign the migration transaction.
--output-address - Provide the Morse output account used for the migration transaction.
--all - (Default: false) If true, claims all MorseClaimableAccounts associated with the specified Morse output address.
--nodes - Comma-separated list of Morse node addresses.
--nodes-file - Path to a file listing Morse node addresses.
--skip-verification - (Default: false) If true, skips verification of MorseClaimableAccounts before claiming.
--stake-template-file - Path to a stake template file detailing services, reward shares, etc. The owner, operator, and stake amount are automatically sourced from ClaimableAccounts.

YAML template:
owner_address: [DEDUCTED SHANNON OWNER ADDRESS] - no need to set
operator_address: [DEDUCTED SHANNON NODE ADDRESS] - no need to set
stake_amount: [DEDUCTED SHANNON NODE STAKE AMOUNT] - no need to set
default_rev_share_percent:
  <DELEGATOR_REWARDS_SHANNON_ADDRESS>: 75
  [DEDUCTED SHANNON OWNER ADDRESS]: 100 - $DELEGATOR_REWARDS_SHANNON_ADDRESS = 25
services:
  - service_id: "anvil"
    endpoints:
      - publicly_exposed_url: https://rm1.somewhere.com
        rpc_type: JSON_RPC
	rev_share_percent:
      <DELEGATOR_REWARDS_SHANNON_ADDRESS>: 90
	  [DEDUCTED SHANNON OWNER ADDRESS]: 100 - $DELEGATOR_REWARDS_SHANNON_ADDRESS = 10
  - service_id: "eth"
    endpoints:
      - publicly_exposed_url: https://rm1.somewhere.com
        rpc_type: JSON_RPC
	rev_share_percent:
      <DELEGATOR_REWARDS_SHANNON_ADDRESS>: 90
	  [DEDUCTED SHANNON OWNER ADDRESS]: 100 - $DELEGATOR_REWARDS_SHANNON_ADDRESS = 10

More info: https://dev.poktroll.com/operate/morse_migration/claiming`,

		RunE:    runClaimSuppliers,
		PreRunE: logger.PreRunESetup,
	}

	// TODO: move all this flag and descriptions into reusable variables at flags.go file?
	claimSuppliersCmd.Flags().StringVarP(
		&outputAddress,
		"output-address",
		"",
		"",
		"Provide the Morse output account used for the migration transaction.",
	)

	claimSuppliersCmd.Flags().BoolVarP(
		&all,
		"all",
		"",
		false,
		"If true, claims all MorseClaimableAccounts associated with the specified Morse output address.",
	)

	claimSuppliersCmd.Flags().StringSliceVarP(
		&nodes,
		"nodes",
		"",
		[]string{},
		"Comma-separated list of Morse node addresses.",
	)

	claimSuppliersCmd.Flags().StringVarP(
		&nodesFile,
		"nodes-file",
		"",
		"",
		"Path to a file listing Morse node addresses.",
	)

	claimSuppliersCmd.Flags().StringVarP(
		&stakeTemplateFile,
		"stake-template-file",
		"",
		"",
		"Path to a stake template file detailing services, reward shares, etc. The owner, operator, and stake amount are automatically sourced from ClaimableAccounts.",
	)

	// This command depends on the conventional cosmos-sdk CLI tx flags.
	cosmosflags.AddTxFlagsToCmd(claimSuppliersCmd)

	return claimSuppliersCmd
}

// runClaimSupplier performs the following sequence:
// - Load the Morse private key from the morse_key_export_path argument (arg 0).
// - Load and validate the supplier service staking config from the path_to_supplier_stake_config argument pointing to a local config file (arg 1).
// - Sign and broadcast the MsgClaimMorseSupplier message using the Shannon key named by the `--from` flag.
// - Wait until the tx is committed onchain for either a synchronous or asynchronous error.
func runClaimSuppliers(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Conventionally derive a cosmos-sdk client context from the cobra command.
	logger.Logger.Info().Msg("Configuring cosmos client")
	clientCtx, err := cosmosclient.GetClientTxContext(cmd)
	if err != nil {
		return err
	}

	// Load the supplier stake config template from the YAML file.
	// lets fail faster if something here is wrong (missing file for example)
	logger.Logger.Info().Msgf("Loading stake template file: %s", stakeTemplateFile)
	_, templateError := loadTemplateSupplierStakeConfigYAML(stakeTemplateFile)
	if templateError != nil {
		return templateError
	}

	logger.Logger.Info().Msgf("Loading output address: %s", outputAddress)
	_, morseOutputError := loadMorseClaimableAccount(ctx, clientCtx, outputAddress)
	if morseOutputError != nil {
		return morseOutputError
	}

	_, morseNodesError := loadMorseNodes(ctx, clientCtx)
	if morseNodesError != nil {
		return morseNodesError
	}

	// if --all then load all claimable accounts and get claimable accounts back

	// if !skipVerification then load all the claimable accounts and verify them

	// 2. need to

	// iterate over claimable

	return nil
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

	return &yamlStakeConfig, nil
}

// loadMorseClaimableAccount loads and validates the MorseClaimableAccount at the
// provided address.
//
// If the MorseClaimableAccount is not claimed yet, this function will return an
// error.
//
// If the MorseClaimableAccount is claimed, this function will return the
// MorseClaimableAccount.
func loadMorseClaimableAccount(
	ctx context.Context,
	clientCtx cosmosclient.Context,
	address string,
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

	// exists at snapshot but could or not be claimed yet
	if !res.MorseClaimableAccount.IsClaimed() {
		// morse account (unstaked) need to be claimed before attempting to claim them as supplier in shannon.
		return nil, fmt.Errorf("morse account %s if not claimed yet: %v", address, res.MorseClaimableAccount)
	}

	return &res.MorseClaimableAccount, nil
}

func loadMorseNodes(
	ctx context.Context,
	clientCtx cosmosclient.Context,
) ([]*types.MorseClaimableAccount, error) {
	addressToLoad := make([]string, 0)
	claimableNodes := make([]*types.MorseClaimableAccount, 0)

	if all {
		queryClient := types.NewQueryClient(clientCtx)

		var (
			pageKey []byte
		)
		for {
			req := &types.QueryAllMorseClaimableAccountRequest{
				Pagination: &cosmosquery.PageRequest{
					Key:        pageKey,
					Limit:      10000, // TODO: maybe allow this to be configurable?
					CountTotal: false,
				},
			}
			res, err := queryClient.MorseClaimableAccountAll(ctx, req)
			if err != nil {
				return nil, err
			}

			for i := range res.MorseClaimableAccount {
				// check if the MorseOutputAddress == outputAddress
				//  and also is already claimed, otherwise if output address match but is not claimed,
				//  return an error
				claimableNodes = append(claimableNodes, &res.MorseClaimableAccount[i])
			}

			// If the next page key is nil or empty, we've retrieved all pages
			if res.Pagination == nil || len(res.Pagination.NextKey) == 0 {
				break
			}
			pageKey = res.Pagination.NextKey
		}
	} else {
		// load the nodes from the param or file
		// fail if no one is set, and --all is false
		if len(nodes) == 0 && nodesFile == "" {
			return nil, fmt.Errorf("no nodes provided. please provide at least one node address using --nodes or --nodes-file. alternative use --all to claim all nodes associated with the output address")
		}

		if nodes != nil && len(nodes) > 0 {
			// deconstruct on it to avoid modifying by any kind the original input
			addressToLoad = append(addressToLoad, nodes...)
		}

		if nodesFile != "" {
			info, err := os.Stat(nodesFile)
			if err != nil {
				return nil, err
			}
			if info.IsDir() {
				return nil, fmt.Errorf("nodes file %s is a directory", nodesFile)
			}

			// read the file as a JSON []string
			fileContent, err := os.ReadFile(nodesFile)
			if err != nil {
				return nil, err
			}

			var readNodes []string
			if err = json.Unmarshal(fileContent, &readNodes); err != nil {
				return nil, err
			}
			addressToLoad = append(addressToLoad, readNodes...)
		}

		// TODO: figure out if there is a way to speed up this process by doing a single query instead of one per address
		//  * an option will be to query them in batches
		//  * another option will be to query them in parallel (goroutines or any job library if there is any in the project)
		for _, address := range addressToLoad {
			morseClaimableAccount, queryErr := loadMorseClaimableAccount(ctx, clientCtx, address)
			if queryErr != nil {
				return nil, queryErr
			}

			claimableNodes = append(claimableNodes, morseClaimableAccount)
		}
	}

	return claimableNodes, nil
}
