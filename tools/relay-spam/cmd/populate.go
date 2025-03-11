package cmd

import (
	"fmt"
	"os"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"

	"github.com/pokt-network/poktroll/tools/relay-spam/account"
	"github.com/pokt-network/poktroll/tools/relay-spam/config"
)

// populateCmd represents the populate command
var populateCmd = &cobra.Command{
	Use:   "populate",
	Short: "Create new accounts for relay spam",
	Long:  `Create new accounts and add them to the configuration file.`,
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

		// Create accounts
		numAccounts := viper.GetInt("num_accounts")
		fmt.Printf("Creating %d new accounts...\n", numAccounts)

		newApps, err := accountManager.CreateAccounts(numAccounts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create accounts: %v\n", err)
			os.Exit(1)
		}

		// Add new accounts to config
		cfg.Applications = append(cfg.Applications, newApps...)

		// Save updated config
		configBytes, err := yaml.Marshal(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to marshal config: %v\n", err)
			os.Exit(1)
		}

		err = os.WriteFile(configFile, configBytes, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write config file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Successfully created %d accounts and updated config file\n", numAccounts)

		// Print funding commands
		fmt.Println("\nTo fund these accounts, run the following commands:")
		commands, err := accountManager.GenerateFundingCommands()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate funding commands: %v\n", err)
			os.Exit(1)
		}

		for _, cmd := range commands[len(commands)-numAccounts:] {
			fmt.Println(cmd)
		}
	},
}

func init() {
	rootCmd.AddCommand(populateCmd)
}
