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
	Short: "Generate funding commands for accounts",
	Long:  `Generate commands to fund accounts in the configuration file.`,
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
		kr := keyring.NewInMemory(cdc)

		// Create account manager
		accountManager := account.NewManager(kr, cfg)

		// Generate funding commands
		fmt.Println("Generating funding commands...")
		commands, err := accountManager.GenerateFundingCommands()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate funding commands: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Run the following commands to fund your accounts:")
		for _, cmd := range commands {
			fmt.Println(cmd)
		}
	},
}

func init() {
	rootCmd.AddCommand(fundCmd)
}
