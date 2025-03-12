package cmd

import (
	"fmt"
	"os"
	"strconv"

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
	Use:   "populate [num_accounts]",
	Short: "Create new accounts for relay spam",
	Long:  `Create new accounts and add them to the configuration file.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Get config file from flag
		configFile, err := cmd.Flags().GetString("config")
		if err != nil || configFile == "" {
			configFile = "config.yml"
		}

		// Load config
		cfg, err := config.LoadConfig(configFile)
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

		// Determine number of accounts to create
		numAccounts := viper.GetInt("num_accounts") // Default from config
		if len(args) > 0 {
			// If argument is provided, use it instead
			parsedNum, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Invalid number of accounts: %v\n", err)
				os.Exit(1)
			}
			numAccounts = parsedNum
		}

		fmt.Printf("Creating %d new accounts...\n", numAccounts)

		newApps, err := accountManager.CreateAccounts(numAccounts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create accounts: %v\n", err)
			os.Exit(1)
		}

		// Instead of modifying the entire config, we'll only update the applications section without changing the rest of the config structure
		// Read the original config file to preserve its structure
		originalConfigBytes, err := os.ReadFile(configFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read original config file: %v\n", err)
			os.Exit(1)
		}

		// Parse the original config as a map to preserve structure
		var originalConfig map[string]interface{}
		if err := yaml.Unmarshal(originalConfigBytes, &originalConfig); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse original config: %v\n", err)
			os.Exit(1)
		}

		// Get the existing applications from the original config
		existingApps, ok := originalConfig["applications"].([]interface{})
		if !ok {
			// If applications key doesn't exist or is not a list, create a new list
			existingApps = []interface{}{}
		}

		// Convert new applications to map format for YAML
		for _, app := range newApps {
			appMap := map[string]interface{}{
				"name":           app.Name,
				"address":        app.Address,
				"mnemonic":       app.Mnemonic,
				"serviceidgoal":  app.ServiceIdGoal,
				"delegateesgoal": app.DelegateesGoal,
			}
			existingApps = append(existingApps, appMap)
		}

		// Update only the applications section in the original config
		originalConfig["applications"] = existingApps

		// Ensure the global ApplicationStakeGoal is set if not already present
		if _, exists := originalConfig["application_stake_goal"]; !exists && cfg.ApplicationStakeGoal != "" {
			originalConfig["application_stake_goal"] = cfg.ApplicationStakeGoal
		}

		// Marshal the updated config back to YAML
		updatedConfigBytes, err := yaml.Marshal(originalConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to marshal updated config: %v\n", err)
			os.Exit(1)
		}

		// Write the updated config back to the file
		err = os.WriteFile(configFile, updatedConfigBytes, 0644)
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

		// Only print commands for the newly created accounts
		if len(commands) >= numAccounts {
			for _, cmd := range commands[len(commands)-numAccounts:] {
				fmt.Println(cmd)
			}
		} else {
			// If we have fewer commands than expected, just print all of them
			for _, cmd := range commands {
				fmt.Println(cmd)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(populateCmd)

	// Add config flag
	populateCmd.Flags().String("config", "", "Path to the config file")
}
