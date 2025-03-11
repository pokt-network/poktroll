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

// importCmd represents the import command
var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import accounts from config",
	Long:  `Import accounts from the configuration file into the keyring.`,
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

		// Import accounts
		fmt.Println("Importing accounts from config...")
		err = accountManager.ImportAccounts()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to import accounts: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Successfully imported all accounts")
	},
}

func init() {
	rootCmd.AddCommand(importCmd)
}
