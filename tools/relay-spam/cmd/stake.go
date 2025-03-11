package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/tools/relay-spam/account"
	"github.com/pokt-network/poktroll/tools/relay-spam/config"
)

// stakeCmd represents the stake command
var stakeCmd = &cobra.Command{
	Use:   "stake",
	Short: "Stake applications",
	Long:  `Stake applications to services.`,
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

		// Import accounts
		fmt.Println("Importing accounts from config...")
		err = accountManager.ImportAccounts()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to import accounts: %v\n", err)
			os.Exit(1)
		}

		// Generate stake commands
		fmt.Println("Generating stake commands...")
		for _, app := range cfg.Applications {
			stakeCmd := fmt.Sprintf("poktrolld tx application stake %s %d%s %s %s",
				app.Address,
				app.StakeGoal,
				"upokt",
				app.ServiceIdGoal,
				cfg.TxFlags)

			fmt.Println(stakeCmd)

			// Generate delegate commands if needed
			if len(app.DelegateesGoal) > 0 {
				delegateCmd := fmt.Sprintf("poktrolld tx application delegate %s %s %s",
					app.Address,
					strings.Join(app.DelegateesGoal, ","),
					cfg.TxFlags)

				fmt.Println(delegateCmd)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(stakeCmd)
}
