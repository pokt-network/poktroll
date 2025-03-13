package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	cosmoskeyring "github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/pokt-network/poktroll/tools/relay-spam/application"
	"github.com/pokt-network/poktroll/tools/relay-spam/config"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
)

// Initialize SDK configuration
func init() {
	// Set prefixes
	config := sdk.GetConfig()
	accountAddressPrefix := "pokt"
	accountPubKeyPrefix := accountAddressPrefix + "pub"
	validatorAddressPrefix := accountAddressPrefix + "valoper"
	validatorPubKeyPrefix := accountAddressPrefix + "valoperpub"
	consNodeAddressPrefix := accountAddressPrefix + "valcons"
	consNodePubKeyPrefix := accountAddressPrefix + "valconspub"

	// Set and seal config
	config.SetBech32PrefixForAccount(accountAddressPrefix, accountPubKeyPrefix)
	config.SetBech32PrefixForValidator(validatorAddressPrefix, validatorPubKeyPrefix)
	config.SetBech32PrefixForConsensusNode(consNodeAddressPrefix, consNodePubKeyPrefix)
}

// stakeCmd represents the stake command
var stakeCmd = &cobra.Command{
	Use:   "stake",
	Short: "Stake applications",
	Long:  `Stake applications to services.`,
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

		// Validate required config settings
		if cfg.ApplicationStakeGoal == "" {
			fmt.Fprintf(os.Stderr, "ApplicationStakeGoal is required in config\n")
			os.Exit(1)
		}

		// Set default data directory if not specified
		if cfg.DataDir == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to get user home directory: %v\n", err)
				os.Exit(1)
			}
			cfg.DataDir = filepath.Join(homeDir, ".poktroll")
		}

		// Ensure data directory exists
		if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create data directory: %v\n", err)
			os.Exit(1)
		}

		// Get keyring backend from flag
		keyringBackend, err := cmd.Flags().GetString("keyring-backend")
		if err != nil {
			keyringBackend = "test"
		}

		// Create codec and registry for keyring
		registry := codectypes.NewInterfaceRegistry()
		cryptocodec.RegisterInterfaces(registry)
		sdk.RegisterInterfaces(registry)
		authtypes.RegisterInterfaces(registry)
		apptypes.RegisterInterfaces(registry)

		// Create a legacy Amino codec for address encoding
		amino := codec.NewLegacyAmino()
		sdk.RegisterLegacyAminoCodec(amino)
		cryptocodec.RegisterCrypto(amino)

		// Create the codec with the registry
		cdc := codec.NewProtoCodec(registry)

		// Create a keyring
		var kr cosmoskeyring.Keyring
		if keyringBackend == "inmemory" {
			kr = cosmoskeyring.NewInMemory(cdc)
		} else {
			// Create the keyring
			kr, err = cosmoskeyring.New(
				"poktroll",
				keyringBackend,
				cfg.DataDir,
				os.Stdin,
				cdc,
			)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create keyring: %v\n", err)
				os.Exit(1)
			}
		}

		// Set default RPC endpoint if not provided
		rpcEndpoint := "http://localhost:26657"
		if cfg.RpcEndpoint != "" {
			rpcEndpoint = cfg.RpcEndpoint
		}

		// Create a TxConfig
		txConfig := authtx.NewTxConfig(cdc, authtx.DefaultSignModes)

		// Create a client context
		clientCtx := cosmosclient.Context{}.
			WithKeyring(kr).
			WithChainID(cfg.ChainID).
			WithCodec(cdc).
			WithInterfaceRegistry(registry).
			WithTxConfig(txConfig).
			WithAccountRetriever(authtypes.AccountRetriever{})

		// Set the RPC endpoint for transaction broadcasting
		clientCtx = clientCtx.WithNodeURI(rpcEndpoint)

		// Initialize the client context with a client
		client, err := cosmosclient.NewClientFromNode(rpcEndpoint)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create client: %v\n", err)
			os.Exit(1)
		}
		clientCtx = clientCtx.WithClient(client)

		// Create a Staker instance
		// We need to create a new client context with the GRPC endpoint
		// instead of the RPC endpoint for the staker to use for querying
		stakerClientCtx := cosmosclient.Context{}.
			WithKeyring(clientCtx.Keyring).
			WithChainID(clientCtx.ChainID).
			WithCodec(clientCtx.Codec).
			WithInterfaceRegistry(clientCtx.InterfaceRegistry).
			WithTxConfig(clientCtx.TxConfig).
			WithAccountRetriever(clientCtx.AccountRetriever).
			WithClient(clientCtx.Client)

		// Use the GRPC endpoint for the staker's client context
		if cfg.GrpcEndpoint != "" {
			stakerClientCtx = stakerClientCtx.WithNodeURI(cfg.GrpcEndpoint)
		} else {
			fmt.Println("Warning: GRPC endpoint not specified in config, using RPC endpoint for GRPC connections")
			stakerClientCtx = stakerClientCtx.WithNodeURI(rpcEndpoint)
		}
		staker := application.NewStaker(stakerClientCtx, cfg)

		// Stake applications
		fmt.Println("Staking applications...")
		if err := staker.StakeApplications(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to stake applications: %v\n", err)
			os.Exit(1)
		}

		// Delegate applications to gateways
		fmt.Println("Delegating applications to gateways...")
		if err := staker.DelegateToGateway(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to delegate applications: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Staking and delegation completed successfully.")
	},
}

func init() {
	rootCmd.AddCommand(stakeCmd)

	// Add keyring-backend flag
	stakeCmd.Flags().String("keyring-backend", "test", "Keyring backend to use (os, file, test, inmemory)")

	// Add config flag
	stakeCmd.Flags().String("config", "", "Path to the config file")

	// Add debug flag
	stakeCmd.Flags().Bool("debug", false, "Enable debug output")
}
