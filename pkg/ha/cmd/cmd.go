// Package cmd provides the command-line interface for the HA RelayMiner.
package cmd

import (
	"github.com/spf13/cobra"
)

// HARelayerCmd returns the Cobra command for the HA RelayMiner.
// This is added as a subcommand under the main relayminer command.
// Usage: pocketd relayminer ha [relayer|miner]
func HARelayerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ha",
		Short: "High-Availability RelayMiner Commands",
		Long: `High-Availability (HA) RelayMiner - Run a horizontally scalable RelayMiner.

The HA RelayMiner enables running multiple RelayMiner instances behind a load balancer
with shared state via Redis. This provides:

- Horizontal scaling for high throughput
- Automatic failover for high availability
- Shared session state across instances
- Redis Streams for relay message coordination

Architecture:

                     +----------------+
                     | Load Balancer  |
                     +----------------+
                            |
              +-------------+-------------+
              |             |             |
       +-----------+  +-----------+  +-----------+
       | HA Relay  |  | HA Relay  |  | HA Relay  |
       | er  #1    |  | er  #2    |  | er  #3    |
       +-----------+  +-----------+  +-----------+
              |             |             |
              +-------------+-------------+
                            |
                     +-------------+
                     | Redis (HA)  |
                     +-------------+
                            |
              +-------------+-------------+
              |                           |
       +-----------+               +-----------+
       | HA Miner  |               | HA Miner  |
       |  Leader   |               | Follower  |
       +-----------+               +-----------+

Components:
- HA Relayer: Stateless HTTP/WebSocket proxy that validates and forwards relays
- HA Miner: Builds SMST trees and submits claims/proofs (one leader via Redis locks)
- Redis: Shared state for sessions, WAL, and relay messages

Usage:
  pocketd relayminer ha relayer --config /path/to/config.yaml
  pocketd relayminer ha miner --supplier pokt1abc...
`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(startRelayerCmd())
	cmd.AddCommand(startMinerCmd())
	return cmd
}
