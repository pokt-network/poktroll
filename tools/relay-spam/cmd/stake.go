package cmd

import (
	"fmt"
	"os"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/pokt-network/poktroll/tools/relay-spam/application"
	"github.com/pokt-network/poktroll/tools/relay-spam/config"
)

var delegateFlag bool

// stakeCmd represents the stake command
var stakeCmd = &cobra.Command{
	Use:   "stake",
	Short: "Stake applications and delegate to gateways",
	Long:  `Stake applications and optionally delegate them to gateways.`,
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

		// Create client context
		clientCtx := client.Context{
			Keyring: kr,
			// Other client context fields would be set here in a real implementation
		}

		// Create application staker
		staker := application.NewStaker(clientCtx, cfg)

		// Stake applications
		fmt.Println("Staking applications...")
		err = staker.StakeApplications()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to stake applications: %v\n", err)
			os.Exit(1)
		}

		// Delegate to gateways if requested
		if delegateFlag {
			fmt.Println("Delegating applications to gateways...")
			err = staker.DelegateToGateway()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to delegate applications: %v\n", err)
				os.Exit(1)
			}
		}

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

		fmt.Println("Successfully staked applications and updated config file")
	},
}

func init() {
	rootCmd.AddCommand(stakeCmd)
	stakeCmd.Flags().BoolVarP(&delegateFlag, "delegate", "d", false, "Delegate applications to gateways after staking")
}
