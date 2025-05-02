package cmd

import (
	"github.com/spf13/cobra"

	relayerconfig "github.com/pokt-network/poktroll/pkg/relayer/config"
)

// RelayerCmd returns the Cobra root command for the relayminer CLI.
func RelayerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "relayminer",
		Short: "RelayMiner Subcommands (i.e. Supplier Operation)",
		Long: `RelayMiner Subcommands to start, test and operate a RelayMiner.

A Supplier is just an onchain record advertising to provide a service.
A RelayMiner is the coprocessor that runs offchain to handle relays, provide a service and earn rewards.`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(startCmd())
	cmd.AddCommand(relayCmd())
	return cmd
}

// uniqueSigningKeyNames returns a list of unique operator signing key names from the RelayMiner config.
func uniqueSigningKeyNames(relayMinerConfig *relayerconfig.RelayMinerConfig) []string {
	uniqueKeyMap := make(map[string]bool)
	for _, server := range relayMinerConfig.Servers {
		for _, supplier := range server.SupplierConfigsMap {
			for _, signingKeyName := range supplier.SigningKeyNames {
				uniqueKeyMap[signingKeyName] = true
			}
		}
	}

	uniqueKeyNames := make([]string, 0, len(uniqueKeyMap))
	for key := range uniqueKeyMap {
		uniqueKeyNames = append(uniqueKeyNames, key)
	}

	return uniqueKeyNames
}
