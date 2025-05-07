package cmd

import (
	"github.com/spf13/cobra"
)

// RelayerCmd returns the Cobra root command for the relayminer CLI.
func RelayerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "relayminer",
		Short: "RelayMiner Subcommands (i.e. Supplier Operation)",
		Long: `RelayMiner Subcommands to start, test and operate a RelayMiner.

A Supplier is an **onchain record** advertising a service (e.g. API to access ETH data).
A RelayMiner is an **offchain coprocessor** that provides a service, proxies requests, validates relays, and ensures the Supplier earns rewards.

Relay flow overview:

    +------+      +--------------+      +-----------------+
    | User | <--> |  RelayMiner  | <--> |   Backend API   |
    +------+      +--------------+      +-----------------+
                      |
                      v
                +----------------+
                |   Supplier     |
                | (onchain rec.) |
                +----------------+

1. **User** sends a relay request
2. **RelayMiner** proxies, validates, signs, and forwards the request
3. **Backend API** is the actual service (e.g. ETH node)
4. **Supplier** is the onchain record that the RelayMiner is operating for

This structure allows:
- Secure, auditable, and rewardable relays
- Clear separation between onchain identity (Supplier) and offchain execution (RelayMiner)

For more info, run 'relayminer --help'.`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(startCmd())
	cmd.AddCommand(relayCmd())
	return cmd
}
