// Package cmd provides the command-line interface for the RelayMiner.
//
// - Contains subcommands for starting, testing, and operating a RelayMiner
// - Entry point for the relayminer CLI
package cmd

import (
	"github.com/spf13/cobra"

	hacmd "github.com/pokt-network/poktroll/pkg/ha/cmd"
)

// RelayerCmd returns the Cobra root command for the relayminer CLI.
//
// - Root for all RelayMiner subcommands
// - Use 'relayminer --help' for more info
func RelayerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "relayminer",
		Short: "RelayMiner Subcommands (Supplier Operation)",
		Long: `RelayMiner CLI - Start, test, and operate a RelayMiner.

RelayMiner architecture:

- Supplier: Onchain record advertising a service (e.g., ETH API)
- RelayMiner: Offchain coprocessor providing the service, proxying requests, validating relays, and ensuring rewards

Relay flow:

    +------+      +--------------+      +-----------------+
    | User | <--> |  RelayMiner  | <--> |   Backend API   |
    +------+      +--------------+      +-----------------+
                      |
                      v
                +----------------+
                |   Supplier     |
                | (onchain rec.) |
                +----------------+

Steps:
- User sends a relay request
- RelayMiner proxies, validates, signs, and forwards the request
- Backend API is the actual service (e.g., ETH node)
- Supplier is the onchain record the RelayMiner operates for

Benefits:
- Secure, auditable, and rewardable relays
- Clear separation between onchain identity (Supplier) and offchain execution (RelayMiner)

For help, run: relayminer --help
`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(startCmd())
	cmd.AddCommand(relayCmd())
	cmd.AddCommand(hacmd.HARelayerCmd())
	return cmd
}
