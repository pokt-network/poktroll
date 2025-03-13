package cmd

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tyler-smith/go-bip39"
	"gopkg.in/yaml.v2"

	"github.com/pokt-network/poktroll/tools/relay-spam/account"
	"github.com/pokt-network/poktroll/tools/relay-spam/config"
)

// populateCmd represents the populate command
var populateCmd = &cobra.Command{
	Use:   "populate [entity_type] [num_accounts]",
	Short: "Create new accounts for relay spam",
	Long:  `Create new accounts and add them to the configuration file. Entity type can be "application", "service", or "supplier".`,
	Args:  cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		// Get config file from flag
		configFile, err := cmd.Flags().GetString("config")
		if err != nil || configFile == "" {
			configFile = "config.yml"
		}

		// Determine entity type (default to application if not specified)
		entityType := "application"
		numAccounts := viper.GetInt("num_accounts") // Default from config

		if len(args) > 0 {
			// First argument is entity type
			entityType = strings.ToLower(args[0])

			// Validate entity type
			if entityType != "application" && entityType != "service" && entityType != "supplier" {
				fmt.Fprintf(os.Stderr, "Invalid entity type: %s. Must be 'application', 'service', or 'supplier'.\n", entityType)
				os.Exit(1)
			}

			// If second argument is provided, it's the number of accounts
			if len(args) > 1 {
				parsedNum, err := strconv.Atoi(args[1])
				if err != nil {
					fmt.Fprintf(os.Stderr, "Invalid number of accounts: %v\n", err)
					os.Exit(1)
				}
				numAccounts = parsedNum
			}
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

		fmt.Printf("Creating %d new %s accounts...\n", numAccounts, entityType)

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

		// Handle different entity types
		switch entityType {
		case "application":
			// Create application accounts
			newApps, err := accountManager.CreateAccounts(numAccounts)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create accounts: %v\n", err)
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

		case "service":
			// Create service accounts
			newAccounts, err := createGenericAccounts(kr, "relay_spam_service", numAccounts)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create service accounts: %v\n", err)
				os.Exit(1)
			}

			// Get the existing services from the original config
			existingServices, ok := originalConfig["services"].([]interface{})
			if !ok {
				// If services key doesn't exist or is not a list, create a new list
				existingServices = []interface{}{}
			}

			// Convert new services to map format for YAML
			for _, acc := range newAccounts {
				serviceMap := map[string]interface{}{
					"name":     acc.name,
					"address":  acc.address,
					"mnemonic": acc.mnemonic,
					// ServiceId is intentionally left empty as per the TODO comment
				}
				existingServices = append(existingServices, serviceMap)
			}

			// Update only the services section in the original config
			originalConfig["services"] = existingServices

			// Ensure the global ServiceStakeGoal is set if not already present
			if _, exists := originalConfig["service_stake_goal"]; !exists && cfg.ServiceStakeGoal != "" {
				originalConfig["service_stake_goal"] = cfg.ServiceStakeGoal
			}

		case "supplier":
			// Create supplier accounts
			newAccounts, err := createGenericAccounts(kr, "relay_spam_supplier", numAccounts)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create supplier accounts: %v\n", err)
				os.Exit(1)
			}

			// Get the existing suppliers from the original config
			existingSuppliers, ok := originalConfig["suppliers"].([]interface{})
			if !ok {
				// If suppliers key doesn't exist or is not a list, create a new list
				existingSuppliers = []interface{}{}
			}

			// Convert new suppliers to map format for YAML
			for _, acc := range newAccounts {
				supplierMap := map[string]interface{}{
					"name":          acc.name,
					"address":       acc.address,
					"mnemonic":      acc.mnemonic,
					"owner_address": acc.address, // Same as address
					"stake_config": map[string]interface{}{
						"owner_address":    acc.address,
						"operator_address": acc.address,
						"services":         []interface{}{},
					},
				}
				existingSuppliers = append(existingSuppliers, supplierMap)
			}

			// Update only the suppliers section in the original config
			originalConfig["suppliers"] = existingSuppliers

			// Ensure the global SupplierStakeGoal is set if not already present
			if _, exists := originalConfig["supplier_stake_goal"]; !exists && cfg.SupplierStakeGoal != "" {
				originalConfig["supplier_stake_goal"] = cfg.SupplierStakeGoal
			}
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

		fmt.Printf("Successfully created %d %s accounts and updated config file\n", numAccounts, entityType)

		// Print funding commands
		fmt.Printf("\nTo fund these %s accounts, run the following commands:\n", entityType)

		// Generate funding commands based on entity type
		var fundAmount string

		switch entityType {
		case "application":
			fundAmount = cfg.ApplicationFundGoal
			if fundAmount == "" {
				fundAmount = "1000000upokt" // Default value
			}

			// Get addresses of newly created applications
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
			return

		case "service":
			fundAmount = cfg.ServiceStakeGoal
			if fundAmount == "" {
				fundAmount = "1000000upokt" // Default value
			}

		case "supplier":
			fundAmount = cfg.SupplierStakeGoal
			if fundAmount == "" {
				fundAmount = "1000000upokt" // Default value
			}
		}

		// For services and suppliers, we need to extract addresses from the updated config
		if entityType == "service" || entityType == "supplier" {
			var entityList []interface{}
			if entityType == "service" {
				entityList = originalConfig["services"].([]interface{})
			} else {
				entityList = originalConfig["suppliers"].([]interface{})
			}

			// Get the last numAccounts entities (the ones we just created)
			startIdx := len(entityList) - numAccounts
			if startIdx < 0 {
				startIdx = 0
			}

			for i := startIdx; i < len(entityList); i++ {
				entity := entityList[i].(map[string]interface{})
				address := entity["address"].(string)
				cmd := fmt.Sprintf("poktrolld tx bank send faucet %s %s %s",
					address, fundAmount, cfg.TxFlags)
				fmt.Println(cmd)
			}
		}
	},
}

// Generic account structure for services and suppliers
type genericAccount struct {
	name     string
	address  string
	mnemonic string
}

// Helper function to create generic accounts for services and suppliers
func createGenericAccounts(kr keyring.Keyring, namePrefix string, numAccounts int) ([]genericAccount, error) {
	var accounts []genericAccount

	// Find the highest existing index
	startIndex := 0
	re := regexp.MustCompile(namePrefix + `_(\d+)`)

	// List all keys in the keyring
	keyInfos, err := kr.List()
	if err != nil {
		return nil, err
	}

	// Find the highest index
	for _, info := range keyInfos {
		matches := re.FindStringSubmatch(info.Name)
		if len(matches) > 1 {
			index, err := strconv.Atoi(matches[1])
			if err == nil && index >= startIndex {
				startIndex = index + 1
			}
		}
	}

	for i := 0; i < numAccounts; i++ {
		index := startIndex + i
		name := fmt.Sprintf("%s_%d", namePrefix, index)

		// Generate mnemonic
		entropy, err := bip39.NewEntropy(256)
		if err != nil {
			return nil, err
		}
		mnemonic, err := bip39.NewMnemonic(entropy)
		if err != nil {
			return nil, err
		}

		// Create account
		record, err := kr.NewAccount(name, mnemonic, "", "m/44'/118'/0'/0/0", hd.Secp256k1)
		if err != nil {
			return nil, err
		}

		address, err := record.GetAddress()
		if err != nil {
			return nil, err
		}

		// Create account
		acc := genericAccount{
			name:     name,
			address:  address.String(),
			mnemonic: mnemonic,
		}

		accounts = append(accounts, acc)
	}

	return accounts, nil
}

func init() {
	rootCmd.AddCommand(populateCmd)

	// Add config flag
	populateCmd.Flags().String("config", "", "Path to the config file")
}
