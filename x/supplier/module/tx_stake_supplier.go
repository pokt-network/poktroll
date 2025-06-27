package supplier

import (
	"os"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/x/supplier/config"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

const (
	flagConfig      = "config"
	flagConfigUsage = "Path to the supplier configuration file (YAML format)"

	flagStakeOnly      = "stake-only"
	flagStakeOnlyUsage = "Update only the supplier stake amount. Config file should contain stake_amount but the services section must be empty. Can be signed by owner or operator."

	flagServicesOnly      = "services-only"
	flagServicesOnlyUsage = "Update only the supplier service configurations. Config file should contain services but stake_amount must be empty. Must be signed by operator."
)

var (
	configFlagValue       string
	stakeOnlyFlagValue    bool
	servicesOnlyFlagValue bool
	_                     = strconv.Itoa(0) // Part of the default ignite imports
)

func CmdStakeSupplier() *cobra.Command {
	// fromAddress & signature is retrieved via `flags.FlagFrom` in the `clientCtx`
	cmd := &cobra.Command{
		Use:   "stake-supplier --config <config_file.yaml> [--stake-only | --services-only]",
		Short: "Stake a supplier or update supplier configuration",
		Long: `Stake a supplier or update supplier configuration using the specified configuration file.
This command supports flexible staking operations:

• Initial staking: Requires TODO_IN_THIS_PR.
• Stake-only updates: Update stake amount without changing service configurations (--stake-only)
• Service configuration updates: Update services without changing stake (--services-only)

The command supports both custodial and non-custodial staking workflows and sources
the necessary information from the provided configuration file.

For more details on the staking process, please refer to the supplier staking documentation at:
https://dev.poktroll.com/operate/configs/supplier_staking_config`,
		Example: `
  # Initial supplier staking by Operator: Required stake and optional services
  $ pocketd tx supplier stake-supplier --config stake_config.yaml --from $(OPERATOR_ADDRESS)

  # Initial supplier staking by Owner: Required stake and services section must be empty
  $ pocketd tx supplier stake-supplier --config stake_config.yaml --from $(OWNER_ADDRESS)

  # Update only the stake amount by Owner: Services section must be empty
  $ pocketd tx supplier stake-supplier --config stake_config.yaml --stake-only --from $(OWNER_ADDRESS)

  TODO_IN_THIS_PR: Add more examples`,

		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, _ []string) (err error) {
			configContent, err := os.ReadFile(configFlagValue)
			if err != nil {
				return err
			}

			supplierStakeConfigs, err := config.ParseSupplierConfigs(cmd.Context(), configContent)
			if err != nil {
				return err
			}

			signingKeyNameOrAddress, err := cmd.Flags().GetString(flags.FlagFrom)
			if err != nil {
				return err
			}

			// Ensure the --from flag is set before getting the client context.
			// Default to owner/operator signer in this order::
			// 1. owner address - rationale: typical stake escrow source
			// 2. operator address - rationale: --services-only, operator-custodial workflow, etc. use cases
			if signingKeyNameOrAddress == "" {
				switch {
				case supplierStakeConfigs.OwnerAddress != "":
					signingKeyNameOrAddress = supplierStakeConfigs.OwnerAddress
				case supplierStakeConfigs.OperatorAddress != "":
					signingKeyNameOrAddress = supplierStakeConfigs.OperatorAddress
				default:
					return types.ErrSupplierInvalidAddress.Wrap("unable to determine signer address: config must specify owner_address or operator_address, or use --from flag")
				}

				if err = cmd.Flags().Set(flags.FlagFrom, signingKeyNameOrAddress); err != nil {
					return err
				}
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgStakeSupplier(
				clientCtx.GetFromAddress().String(),
				supplierStakeConfigs.OwnerAddress,
				supplierStakeConfigs.OperatorAddress,
				supplierStakeConfigs.StakeAmount,
				supplierStakeConfigs.Services,
			)

			if err = msg.ValidateBasic(); err != nil {
				return err
			}

			// Ensure mutually exclusive flags
			if stakeOnlyFlagValue && servicesOnlyFlagValue {
				return types.ErrSupplierInvalidServiceConfig.Wrap("--stake-only and --services-only flags are mutually exclusive")
			}

			// Validate flag-specific requirements
			if stakeOnlyFlagValue {
				if len(msg.GetServices()) > 0 {
					return types.ErrSupplierInvalidServiceConfig.Wrap("--stake-only flag specified but config contains service configurations; remove services section from config")
				}
				if msg.GetStake() == nil {
					return types.ErrSupplierInvalidStake.Wrap("--stake-only flag requires stake_amount in config file")
				}
			} else if servicesOnlyFlagValue {
				if msg.GetStake() != nil {
					return types.ErrSupplierInvalidStake.Wrap("--services-only flag specified but config contains stake_amount; remove stake_amount from config")
				}
				if len(msg.GetServices()) == 0 {
					return types.ErrSupplierInvalidServiceConfig.Wrap("--services-only flag requires services section in config file")
				}
				if !msg.IsSigner(msg.GetOperatorAddress()) {
					return types.ErrSupplierInvalidServiceConfig.Wrap(
						"--services-only flag requires operator to be the transaction signer",
					)
				}
			} else {
				// Default behavior: require both stake and services for new suppliers
				if len(msg.GetServices()) == 0 {
					return types.ErrSupplierInvalidServiceConfig.Wrap("no service configurations provided in config file; either provide services or use --stake-only flag")
				}
				if msg.GetStake() == nil {
					return types.ErrSupplierInvalidStake.Wrap("stake amount is required; either provide stake_amount in config or use --services-only flag")
				}
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().StringVar(&configFlagValue, flagConfig, "", flagConfigUsage)
	cmd.Flags().BoolVar(&stakeOnlyFlagValue, flagStakeOnly, false, flagStakeOnlyUsage)
	cmd.Flags().BoolVar(&servicesOnlyFlagValue, flagServicesOnly, false, flagServicesOnlyUsage)
	flags.AddTxFlagsToCmd(cmd)

	if err := cmd.MarkFlagRequired(flagConfig); err != nil {
		panic(err)
	}

	return cmd
}
