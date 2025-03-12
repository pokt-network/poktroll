package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"cosmossdk.io/math"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	cosmostx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	cosmoskeyring "github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/tools/relay-spam/config"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
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
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
			os.Exit(1)
		}

		// Validate required config settings
		if cfg.ApplicationStakeGoal == "" {
			fmt.Fprintf(os.Stderr, "ApplicationStakeGoal is required in config\n")
			os.Exit(1)
		}

		// Parse the stake goal amount
		stakeGoalAmount, err := config.ParseAmount(cfg.ApplicationStakeGoal)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse ApplicationStakeGoal: %v\n", err)
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

		// Get chain ID from flag
		chainID, err := cmd.Flags().GetString("chain-id")
		if err != nil || chainID == "" {
			chainID = "poktroll"
		}

		// Create a context for the transaction
		ctx := context.Background()

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
			WithChainID(chainID).
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

		// Create a transaction factory
		txFactory := cosmostx.Factory{}.
			WithChainID(chainID).
			WithKeybase(kr).
			WithTxConfig(clientCtx.TxConfig).
			WithAccountRetriever(clientCtx.AccountRetriever)

		// Set default gas prices (0.01 upokt per gas unit)
		defaultGasPrice := sdk.NewDecCoinFromDec(volatile.DenomuPOKT, math.LegacyNewDecWithPrec(1, 2)) // 0.01 upokt
		gasPrices := sdk.NewDecCoins(defaultGasPrice)
		txFactory = txFactory.WithGasPrices(gasPrices.String())

		// Process each application
		fmt.Println("Staking applications...")
		for _, app := range cfg.Applications {
			// Create stake amount using the global stake goal
			stakeAmount := sdk.NewInt64Coin(volatile.DenomuPOKT, stakeGoalAmount)

			// Create service config
			serviceConfig := &sharedtypes.ApplicationServiceConfig{
				ServiceId: app.ServiceIdGoal,
			}

			// Create stake message
			stakeMsg := &apptypes.MsgStakeApplication{
				Address:  app.Address,
				Stake:    &stakeAmount,
				Services: []*sharedtypes.ApplicationServiceConfig{serviceConfig},
			}

			fmt.Printf("Staking application %s with %s to service %s\n",
				app.Address, stakeAmount.String(), app.ServiceIdGoal)

			// Create a transaction builder
			txBuilder := clientCtx.TxConfig.NewTxBuilder()

			// Set the message
			if err := txBuilder.SetMsgs(stakeMsg); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to set stake message for %s: %v\n", app.Address, err)
				continue
			}

			// Set gas limit - using a high value to ensure it goes through
			txBuilder.SetGasLimit(420069)

			// Set fee amount based on gas limit and gas prices
			gasPrices := sdk.NewDecCoins(sdk.NewDecCoinFromDec(volatile.DenomuPOKT, math.LegacyNewDecWithPrec(1, 2)))
			fees := sdk.NewCoins()
			for _, gasPrice := range gasPrices {
				fee := gasPrice.Amount.MulInt(math.NewInt(int64(txBuilder.GetTx().GetGas()))).RoundInt()
				fees = fees.Add(sdk.NewCoin(gasPrice.Denom, fee))
			}
			txBuilder.SetFeeAmount(fees)

			// Get the application address
			appAddr, err := sdk.AccAddressFromBech32(app.Address)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to parse application address %s: %v\n", app.Address, err)
				continue
			}

			// Get account number and sequence
			accNum, accSeq, err := clientCtx.AccountRetriever.GetAccountNumberSequence(clientCtx, appAddr)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to get account number and sequence for %s: %v\n", app.Address, err)
				continue
			}

			// Create a transaction factory
			txFactory := cosmostx.Factory{}.
				WithChainID(clientCtx.ChainID).
				WithKeybase(clientCtx.Keyring).
				WithTxConfig(clientCtx.TxConfig).
				WithAccountRetriever(clientCtx.AccountRetriever).
				WithAccountNumber(accNum).
				WithSequence(accSeq)

			// Sign the transaction
			err = cosmostx.Sign(ctx, txFactory, app.Name, txBuilder, true)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to sign transaction for %s: %v\n", app.Address, err)
				continue
			}

			// Encode the transaction
			txBytes, err := clientCtx.TxConfig.TxEncoder()(txBuilder.GetTx())
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to encode transaction for %s: %v\n", app.Address, err)
				continue
			}

			// Broadcast the transaction
			res, err := clientCtx.BroadcastTxSync(txBytes)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to broadcast transaction for %s: %v\n", app.Address, err)
				continue
			}

			// Check for errors in the response
			if res.Code != 0 {
				fmt.Fprintf(os.Stderr, "Transaction failed for %s: %s\n", app.Address, res.RawLog)
				continue
			}

			fmt.Printf("Successfully staked application %s. Transaction hash: %s\n", app.Address, res.TxHash)

			// Wait a bit for the transaction to be processed
			time.Sleep(5 * time.Second)

			// If delegation is needed, create and send delegation message
			if len(app.DelegateesGoal) > 0 {
				fmt.Printf("Delegating application %s to gateways: %v\n", app.Address, app.DelegateesGoal)

				// Process each gateway delegation
				for _, gatewayAddr := range app.DelegateesGoal {
					delegateMsg := &apptypes.MsgDelegateToGateway{
						AppAddress:     app.Address,
						GatewayAddress: gatewayAddr,
					}

					// Create a transaction builder for delegation
					txBuilder := clientCtx.TxConfig.NewTxBuilder()

					// Set the delegation message
					if err := txBuilder.SetMsgs(delegateMsg); err != nil {
						fmt.Fprintf(os.Stderr, "Failed to set delegation message for %s to %s: %v\n",
							app.Address, gatewayAddr, err)
						continue
					}

					// Set gas limit - using a high value to ensure it goes through
					txBuilder.SetGasLimit(1000000000000)

					// Set fee amount based on gas limit and gas prices
					gasPrices := sdk.NewDecCoins(sdk.NewDecCoinFromDec(volatile.DenomuPOKT, math.LegacyNewDecWithPrec(1, 2)))
					fees := sdk.NewCoins()
					for _, gasPrice := range gasPrices {
						fee := gasPrice.Amount.MulInt(math.NewInt(int64(txBuilder.GetTx().GetGas()))).RoundInt()
						fees = fees.Add(sdk.NewCoin(gasPrice.Denom, fee))
					}
					txBuilder.SetFeeAmount(fees)

					// Get account number and sequence - need to get fresh sequence after previous transaction
					accNum, accSeq, err := clientCtx.AccountRetriever.GetAccountNumberSequence(clientCtx, appAddr)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Failed to get account number and sequence for %s: %v\n", app.Address, err)
						continue
					}

					// Create a transaction factory
					txFactory := cosmostx.Factory{}.
						WithChainID(clientCtx.ChainID).
						WithKeybase(clientCtx.Keyring).
						WithTxConfig(clientCtx.TxConfig).
						WithAccountRetriever(clientCtx.AccountRetriever).
						WithAccountNumber(accNum).
						WithSequence(accSeq)

					// Sign the transaction
					err = cosmostx.Sign(ctx, txFactory, app.Name, txBuilder, true)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Failed to sign delegation transaction for %s to %s: %v\n",
							app.Address, gatewayAddr, err)
						continue
					}

					// Encode the transaction
					txBytes, err := clientCtx.TxConfig.TxEncoder()(txBuilder.GetTx())
					if err != nil {
						fmt.Fprintf(os.Stderr, "Failed to encode delegation transaction for %s to %s: %v\n",
							app.Address, gatewayAddr, err)
						continue
					}

					// Broadcast the transaction
					res, err := clientCtx.BroadcastTxSync(txBytes)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Failed to broadcast delegation transaction for %s to %s: %v\n",
							app.Address, gatewayAddr, err)
						continue
					}

					// Check for errors in the response
					if res.Code != 0 {
						fmt.Fprintf(os.Stderr, "Delegation transaction failed for %s to %s: %s\n",
							app.Address, gatewayAddr, res.RawLog)
						continue
					}

					fmt.Printf("Successfully delegated application %s to gateway %s. Transaction hash: %s\n",
						app.Address, gatewayAddr, res.TxHash)

					// Add a small delay between delegations
					time.Sleep(1 * time.Second)
				}
			}

			// Add a small delay between applications
			time.Sleep(2 * time.Second)
		}
	},
}

func init() {
	rootCmd.AddCommand(stakeCmd)

	// Add keyring-backend flag
	stakeCmd.Flags().String("keyring-backend", "test", "Keyring backend to use (os, file, test, inmemory)")

	// Add chain-id flag
	stakeCmd.Flags().String("chain-id", "poktroll", "Chain ID of the blockchain")

	// Add config flag
	stakeCmd.Flags().String("config", "", "Path to the config file")

	// Add debug flag
	stakeCmd.Flags().Bool("debug", false, "Enable debug output")
}
