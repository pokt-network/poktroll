package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

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
	"github.com/pokt-network/poktroll/tools/relay-spam/service"
	"github.com/pokt-network/poktroll/tools/relay-spam/supplier"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
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
	Use:   "stake [entity_type]",
	Short: "Stake entities",
	Long:  `Stake applications to services or add services to the network.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Get entity type from args
		entityType := args[0]

		// Validate entity type
		if entityType != "application" && entityType != "service" && entityType != "supplier" {
			fmt.Fprintf(os.Stderr, "Invalid entity type: %s. Must be 'application', 'service', or 'supplier'\n", entityType)
			os.Exit(1)
		}

		// Get config file from flag
		configFile, err := cmd.Flags().GetString("config")
		if err != nil || configFile == "" {
			configFile = "config.yml"
		}

		// Check if dry-run mode is enabled
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		// Load config
		cfg, err := config.LoadConfig(configFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
			os.Exit(1)
		}

		// Validate required config settings based on entity type
		if entityType == "application" && cfg.ApplicationStakeGoal == "" {
			fmt.Fprintf(os.Stderr, "ApplicationStakeGoal is required in config for staking applications\n")
			os.Exit(1)
		} else if entityType == "supplier" && cfg.SupplierStakeGoal == "" {
			fmt.Fprintf(os.Stderr, "SupplierStakeGoal is required in config for staking suppliers\n")
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
		servicetypes.RegisterInterfaces(registry)
		suppliertypes.RegisterInterfaces(registry)

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

		// Create a client context with the GRPC endpoint for querying
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

		if entityType == "application" {
			// Handle application staking
			appStaker := application.NewStaker(stakerClientCtx, cfg)

			if dryRun {
				// In dry-run mode, just print what would be done
				fmt.Println("DRY RUN MODE: No transactions will be broadcast")
				fmt.Println("\n=== Applications that would be staked ===")

				// Parse the stake amount from the global application stake goal
				stakeAmount, err := sdk.ParseCoinNormalized(cfg.ApplicationStakeGoal)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to parse stake amount: %v\n", err)
					os.Exit(1)
				}

				for _, app := range cfg.Applications {
					fmt.Printf("Application: %s (%s)\n", app.Name, app.Address)
					fmt.Printf("  - Stake Amount: %s\n", stakeAmount.String())
					fmt.Printf("  - Service ID: %s\n", app.ServiceIdGoal)

					// Check if the application is already staked (if querier is available)
					if appStaker.Querier() != nil {
						isStaked, err := appStaker.Querier().IsStaked(context.Background(), app.Address)
						if err != nil {
							fmt.Printf("  - Status: Unknown (error checking stake status: %v)\n", err)
						} else if isStaked {
							isStakedWithAmount, err := appStaker.Querier().IsStakedWithAmount(context.Background(), app.Address, stakeAmount)
							if err != nil {
								fmt.Printf("  - Status: Staked (error checking stake amount: %v)\n", err)
							} else if isStakedWithAmount {
								isStakedForService, err := appStaker.Querier().IsStakedForService(context.Background(), app.Address, app.ServiceIdGoal)
								if err != nil {
									fmt.Printf("  - Status: Staked with correct amount (error checking service: %v)\n", err)
								} else if isStakedForService {
									fmt.Printf("  - Status: Already staked with %s for service %s (would be skipped)\n", stakeAmount.String(), app.ServiceIdGoal)
								} else {
									fmt.Printf("  - Status: Staked with correct amount but for different service (would be staked)\n")
								}
							} else {
								fmt.Printf("  - Status: Staked but with different amount (would be staked)\n")
							}
						} else {
							fmt.Printf("  - Status: Not staked (would be staked)\n")
						}
					} else {
						fmt.Printf("  - Status: Unknown (querier not available)\n")
					}
				}

				fmt.Println("\n=== Applications that would be delegated ===")
				for _, app := range cfg.Applications {
					if len(app.DelegateesGoal) == 0 {
						fmt.Printf("Application: %s (%s)\n", app.Name, app.Address)
						fmt.Printf("  - Status: No gateways specified for delegation (would be skipped)\n")
						continue
					}

					fmt.Printf("Application: %s (%s)\n", app.Name, app.Address)
					fmt.Printf("  - Would be delegated to gateways: %s\n", strings.Join(app.DelegateesGoal, ", "))

					// Check if the application is staked (if querier is available)
					if appStaker.Querier() != nil {
						isStaked, err := appStaker.Querier().IsStaked(context.Background(), app.Address)
						if err != nil {
							fmt.Printf("  - Status: Unknown (error checking stake status: %v)\n", err)
						} else if !isStaked {
							fmt.Printf("  - Status: Not staked (delegation would be skipped)\n")
						} else {
							fmt.Printf("  - Status: Staked (would be delegated)\n")
						}
					} else {
						fmt.Printf("  - Status: Unknown (querier not available)\n")
					}
				}

				fmt.Println("\nDRY RUN COMPLETE: No transactions were broadcast")
			} else {
				// Get concurrency level from flag
				concurrency, _ := cmd.Flags().GetInt("concurrency")
				if concurrency < 1 {
					concurrency = 1
				}

				// Stake applications concurrently
				fmt.Printf("Staking applications with %d concurrent workers...\n", concurrency)
				if err := stakeApplicationsConcurrently(appStaker, concurrency); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to stake applications: %v\n", err)
					os.Exit(1)
				}

				// Delegate applications to gateways concurrently
				fmt.Printf("Delegating applications to gateways with %d concurrent workers...\n", concurrency)
				if err := delegateApplicationsConcurrently(appStaker, concurrency); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to delegate applications: %v\n", err)
					os.Exit(1)
				}

				fmt.Println("Application staking and delegation completed successfully.")
			}
		} else if entityType == "service" {
			// Handle service staking
			serviceStaker := service.NewStaker(stakerClientCtx, cfg)

			if dryRun {
				// In dry-run mode, just print what would be done
				fmt.Println("DRY RUN MODE: No transactions will be broadcast")
				fmt.Println("\n=== Services that would be added ===")

				for _, svc := range cfg.Services {
					fmt.Printf("Service: %s (%s)\n", svc.Name, svc.Address)
					fmt.Printf("  - Service ID: %s\n", svc.ServiceId)

					// Check if the service already exists (if querier is available)
					if serviceStaker.Querier() != nil {
						serviceExists, err := serviceStaker.Querier().ServiceExists(context.Background(), svc.ServiceId)
						if err != nil {
							fmt.Printf("  - Status: Unknown (error checking service existence: %v)\n", err)
						} else if serviceExists {
							fmt.Printf("  - Status: Service already exists (would be skipped)\n")
						} else {
							fmt.Printf("  - Status: Service does not exist (would be added)\n")
						}
					} else {
						fmt.Printf("  - Status: Unknown (querier not available)\n")
					}
				}

				fmt.Println("\nDRY RUN COMPLETE: No transactions were broadcast")
			} else {
				// Get concurrency level from flag
				concurrency, _ := cmd.Flags().GetInt("concurrency")
				if concurrency < 1 {
					concurrency = 1
				}

				// Add services concurrently
				fmt.Printf("Adding services with %d concurrent workers...\n", concurrency)
				if err := addServicesConcurrently(serviceStaker, concurrency); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to add services: %v\n", err)
					os.Exit(1)
				}

				fmt.Println("Service addition completed successfully.")
			}
		} else if entityType == "supplier" {
			// Handle supplier staking
			supplierStaker := supplier.NewStaker(stakerClientCtx, cfg)

			if dryRun {
				// In dry-run mode, just print what would be done
				fmt.Println("DRY RUN MODE: No transactions will be broadcast")
				fmt.Println("\n=== Suppliers that would be staked ===")

				// Parse the stake amount from the global supplier stake goal
				stakeAmount, err := sdk.ParseCoinNormalized(cfg.SupplierStakeGoal)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to parse stake amount: %v\n", err)
					os.Exit(1)
				}

				for _, sup := range cfg.Suppliers {
					fmt.Printf("Supplier: %s (%s)\n", sup.Name, sup.Address)
					fmt.Printf("  - Stake Amount: %s\n", stakeAmount.String())
					fmt.Printf("  - Owner Address: %s\n", sup.OwnerAddress)
					fmt.Printf("  - Services: %d\n", len(sup.StakeConfig.Services))

					// Check if the supplier already exists (if querier is available)
					if supplierStaker.Querier() != nil {
						supplierExists, err := supplierStaker.Querier().SupplierExists(context.Background(), sup.Address)
						if err != nil {
							fmt.Printf("  - Status: Unknown (error checking supplier existence: %v)\n", err)
						} else if supplierExists {
							fmt.Printf("  - Status: Supplier already exists (would be skipped)\n")
						} else {
							fmt.Printf("  - Status: Supplier does not exist (would be staked)\n")
						}
					} else {
						fmt.Printf("  - Status: Unknown (querier not available)\n")
					}
				}

				fmt.Println("\nDRY RUN COMPLETE: No transactions were broadcast")
			} else {
				// Get concurrency level from flag
				concurrency, _ := cmd.Flags().GetInt("concurrency")
				if concurrency < 1 {
					concurrency = 1
				}

				// Stake suppliers concurrently
				fmt.Printf("Staking suppliers with %d concurrent workers...\n", concurrency)
				if err := stakeSuppliersConcurrently(supplierStaker, concurrency); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to stake suppliers: %v\n", err)
					os.Exit(1)
				}

				fmt.Println("Supplier staking completed successfully.")
			}
		}
	},
}

// stakeApplicationsConcurrently stakes applications concurrently using the specified number of workers
func stakeApplicationsConcurrently(appStaker *application.Staker, concurrency int) error {
	// Get the list of applications to stake
	applications := appStaker.GetConfig().Applications
	if len(applications) == 0 {
		return nil
	}

	// Create a channel to receive applications to stake
	appChan := make(chan config.Application, len(applications))

	// Create a channel to receive errors
	errChan := make(chan error, len(applications))

	// Create a wait group to wait for all workers to finish
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for app := range appChan {
				fmt.Printf("Worker %d: Staking application %s...\n", workerID, app.Name)
				if err := appStaker.StakeApplication(app); err != nil {
					errChan <- fmt.Errorf("failed to stake application %s: %w", app.Name, err)
					return
				}
			}
		}(i)
	}

	// Send applications to the channel
	for _, app := range applications {
		appChan <- app
	}

	// Close the channel to signal that no more applications will be sent
	close(appChan)

	// Wait for all workers to finish
	wg.Wait()

	// Check for errors
	select {
	case err := <-errChan:
		return err
	default:
		return nil
	}
}

// delegateApplicationsConcurrently delegates applications to gateways concurrently
func delegateApplicationsConcurrently(appStaker *application.Staker, concurrency int) error {
	// Get the list of applications to delegate
	applications := appStaker.GetConfig().Applications
	if len(applications) == 0 {
		return nil
	}

	// Create a channel to receive applications to delegate
	appChan := make(chan config.Application, len(applications))

	// Create a channel to receive errors
	errChan := make(chan error, len(applications))

	// Create a wait group to wait for all workers to finish
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for app := range appChan {
				// Skip if no gateways are specified for delegation
				if len(app.DelegateesGoal) == 0 {
					fmt.Printf("Worker %d: No gateways specified for delegation for application %s, skipping\n", workerID, app.Name)
					continue
				}

				fmt.Printf("Worker %d: Delegating application %s to gateways...\n", workerID, app.Name)
				if err := appStaker.DelegateApplicationToGateway(app); err != nil {
					errChan <- fmt.Errorf("failed to delegate application %s: %w", app.Name, err)
					return
				}
			}
		}(i)
	}

	// Send applications to the channel
	for _, app := range applications {
		appChan <- app
	}

	// Close the channel to signal that no more applications will be sent
	close(appChan)

	// Wait for all workers to finish
	wg.Wait()

	// Check for errors
	select {
	case err := <-errChan:
		return err
	default:
		return nil
	}
}

// addServicesConcurrently adds services concurrently using the specified number of workers
func addServicesConcurrently(serviceStaker *service.Staker, concurrency int) error {
	// Get the list of services to add
	services := serviceStaker.GetConfig().Services
	if len(services) == 0 {
		return nil
	}

	// Create a channel to receive services to add
	svcChan := make(chan config.Service, len(services))

	// Create a channel to receive errors
	errChan := make(chan error, len(services))

	// Create a wait group to wait for all workers to finish
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for svc := range svcChan {
				fmt.Printf("Worker %d: Adding service %s with ID %s...\n", workerID, svc.Name, svc.ServiceId)
				if err := serviceStaker.AddService(svc); err != nil {
					errChan <- fmt.Errorf("failed to add service %s: %w", svc.Name, err)
					return
				}
			}
		}(i)
	}

	// Send services to the channel
	for _, svc := range services {
		svcChan <- svc
	}

	// Close the channel to signal that no more services will be sent
	close(svcChan)

	// Wait for all workers to finish
	wg.Wait()

	// Check for errors
	select {
	case err := <-errChan:
		return err
	default:
		return nil
	}
}

// stakeSuppliersConcurrently stakes suppliers concurrently using the specified number of workers
func stakeSuppliersConcurrently(supplierStaker *supplier.Staker, concurrency int) error {
	// Get the list of suppliers to stake
	suppliers := supplierStaker.GetConfig().Suppliers
	if len(suppliers) == 0 {
		return nil
	}

	// Create a channel to receive suppliers to stake
	supChan := make(chan config.Supplier, len(suppliers))

	// Create a channel to receive errors
	errChan := make(chan error, len(suppliers))

	// Create a wait group to wait for all workers to finish
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for sup := range supChan {
				fmt.Printf("Worker %d: Staking supplier %s...\n", workerID, sup.Name)
				if err := supplierStaker.StakeSupplier(sup); err != nil {
					errChan <- fmt.Errorf("failed to stake supplier %s: %w", sup.Name, err)
					return
				}
			}
		}(i)
	}

	// Send suppliers to the channel
	for _, sup := range suppliers {
		supChan <- sup
	}

	// Close the channel to signal that no more suppliers will be sent
	close(supChan)

	// Wait for all workers to finish
	wg.Wait()

	// Check for errors
	select {
	case err := <-errChan:
		return err
	default:
		return nil
	}
}

func init() {
	rootCmd.AddCommand(stakeCmd)

	// Add help message for entity_type argument
	stakeCmd.SetHelpTemplate(stakeCmd.UsageTemplate() + `
Entity Types:
  application    Stake application accounts to services
  service        Add service definitions to the network
  supplier       Stake supplier accounts to provide services
`)

	// Add keyring-backend flag
	stakeCmd.Flags().String("keyring-backend", "test", "Keyring backend to use (os, file, test, inmemory)")

	// Add config flag
	stakeCmd.Flags().String("config", "", "Path to the config file")

	// Add debug flag
	stakeCmd.Flags().Bool("debug", false, "Enable debug output")

	// Add dry-run flag
	stakeCmd.Flags().Bool("dry-run", false, "Show what would be staked and delegated without performing transactions")

	// Add concurrency flag
	stakeCmd.Flags().Int("concurrency", 10, "Number of concurrent workers for staking operations")
}
