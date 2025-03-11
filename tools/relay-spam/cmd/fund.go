package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/tools/relay-spam/config"
)

// Default batch size for multi-send commands
const defaultBatchSize = 1000

// fundCmd represents the fund command
var fundCmd = &cobra.Command{
	Use:   "fund",
	Short: "Generate funding commands",
	Long:  `Generate commands to fund accounts from the faucet using bank multi-send.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Load config
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
			os.Exit(1)
		}

		// Ensure data directory exists
		if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create data directory: %v\n", err)
			os.Exit(1)
		}

		// Get batch size from flag or use default
		batchSize, err := cmd.Flags().GetInt("batch-size")
		if err != nil {
			batchSize = defaultBatchSize
		}

		// Generate funding commands using multi-send
		commands, err := generateMultiSendCommands(cfg.Applications, batchSize, cfg.TxFlags)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate funding commands: %v\n", err)
			os.Exit(1)
		}

		for _, cmd := range commands {
			fmt.Println(cmd)
		}
	},
}

// generateMultiSendCommands generates bank multi-send commands in batches
func generateMultiSendCommands(applications []config.Application, batchSize int, txFlags string) ([]string, error) {
	var commands []string

	// Process applications in batches
	for i := 0; i < len(applications); i += batchSize {
		end := i + batchSize
		if end > len(applications) {
			end = len(applications)
		}

		batch := applications[i:end]

		// Build the list of addresses for this batch
		var addresses []string
		for _, app := range batch {
			addresses = append(addresses, app.Address)
		}

		// Create the multi-send command
		// Format: poktrolld tx bank multi-send [from_key_or_address] [to_address_1 to_address_2 ...] [amount] [flags]
		cmd := fmt.Sprintf("poktrolld tx bank multi-send faucet %s 1000000upokt %s",
			strings.Join(addresses, " "), txFlags)

		commands = append(commands, cmd)
	}

	return commands, nil
}

func init() {
	rootCmd.AddCommand(fundCmd)

	// Add batch-size flag
	fundCmd.Flags().Int("batch-size", defaultBatchSize, "Number of addresses to include in each multi-send batch")
}
