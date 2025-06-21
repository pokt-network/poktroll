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
	flagConfigUsage = "Path to the stake config file"

	flagStakeOnly      = "stake-only"
	flagStakeOnlyUsage = "Only update the supplier stake. When set, it is an error to provide a config with service configurations."

	flagServicesOnly      = "services-only"
	flagServicesOnlyUsage = "Only update the supplier service configurations. When set, it is an error to provide a config with a stake amount."
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
		Use:   "stake-supplier --config <config_file.yaml>",
		Short: "Stake a supplier",
		Long: `Stake a supplier using the specified configuration file. This command
supports both custodial and non-custodial staking of the signer's tokens.
It sources the necessary information from the provided configuration file.

For more details on the staking process, please refer to the supplier staking documentation at:
https://dev.poktroll.com/operate/configs/supplier_staking_config

Example:
$ pocketd tx supplier stake-supplier --config stake_config.yaml --keyring-backend test  --from $(OWNER_ADDRESS) --network=<network> --home $(POCKETD_HOME)`,

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
					return types.ErrSupplierInvalidAddress.Wrap("no signer address provided")
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

			if !stakeOnlyFlagValue && len(msg.GetServices()) == 0 {
				return types.ErrSupplierInvalidServiceConfig.Wrap("no service configurations provided")
			}

			if servicesOnlyFlagValue {
				if !msg.IsSigner(msg.GetOperatorAddress()) {
					return types.ErrSupplierInvalidServiceConfig.Wrap(
						"only the owner account can update the service configurations",
					)
				}
			}

			if !servicesOnlyFlagValue && msg.GetStake() == nil {
				return types.ErrSupplierInvalidStake.Wrap("nil stake amount")
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
