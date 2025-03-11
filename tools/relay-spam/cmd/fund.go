package cmd

import (
	"fmt"
	"os"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/tools/relay-spam/account"
	"github.com/pokt-network/poktroll/tools/relay-spam/config"
)

// fundCmd represents the fund command
var fundCmd = &cobra.Command{
	Use:   "fund",
	Short: "Generate funding commands",
	Long:  `Generate commands to fund accounts from the faucet.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Load config
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
			os.Exit(1)
		}

		// Create keyring
		registry := types.NewInterfaceRegistry()
		cryptocodec.RegisterInterfaces(registry)
		cdc := codec.NewProtoCodec(registry)

		// Ensure data directory exists
		if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create data directory: %v\n", err)
			os.Exit(1)
		}

		// Create keyring with specified backend
		var kr keyring.Keyring
		if keyringBackend == "inmemory" {
			kr = keyring.NewInMemory(cdc)
		} else {
			// The Cosmos SDK keyring expects the directory structure to be:
			// <home_directory>/keyring-<backend>
			// But we want to use our own directory structure, so we need to
			// explicitly set the keyring directory
			keyringDir := cfg.DataDir

			// Create the keyring
			kr, err = keyring.New(
				"poktroll",
				keyringBackend,
				keyringDir,
				os.Stdin,
				cdc,
			)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to initialize keyring: %v\n", err)
				os.Exit(1)
			}
		}

		// Create account manager
		accountManager := account.NewManager(kr, cfg)

		// Generate funding commands
		commands, err := accountManager.GenerateFundingCommands()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate funding commands: %v\n", err)
			os.Exit(1)
		}

		for _, cmd := range commands {
			fmt.Println(cmd)
		}
	},
}

func init() {
	rootCmd.AddCommand(fundCmd)
}
