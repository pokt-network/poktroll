package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"crypto/tls"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

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
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"cosmossdk.io/depinject"
	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/block"
	"github.com/pokt-network/poktroll/pkg/client/events"
	"github.com/pokt-network/poktroll/pkg/client/tx"
	txtypes "github.com/pokt-network/poktroll/pkg/client/tx/types"
	"github.com/pokt-network/poktroll/tools/relay-spam/config"
	"gopkg.in/yaml.v3"
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

// fundCmd represents the fund command
var fundCmd = &cobra.Command{
	Use:   "fund",
	Short: "Fund accounts",
	Long:  `Fund accounts by sending transactions directly, only funding the difference needed to reach the target balance.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get config file from flag
		configFile, err := cmd.Flags().GetString("config")
		if err != nil || configFile == "" {
			configFile = "config.yml"
		}

		// Read config file directly
		configData, err := os.ReadFile(configFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read config file: %v\n", err)
			os.Exit(1)
		}

		// Parse YAML
		var cfg config.Config
		if err := yaml.Unmarshal(configData, &cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse config file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Using config file: %s\n", configFile)

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

		// Get faucet address from flag
		faucetAddr, err := cmd.Flags().GetString("faucet")
		if err != nil || faucetAddr == "" {
			faucetAddr = "faucet"
		}

		// Validate required config settings
		if cfg.GrpcEndpoint == "" {
			fmt.Fprintf(os.Stderr, "GRPC endpoint is required for balance checking\n")
			os.Exit(1)
		}

		// Set default RPC endpoint if not provided
		rpcEndpoint := "http://localhost:26657"
		if cfg.RpcEndpoint != "" {
			rpcEndpoint = cfg.RpcEndpoint
		}

		if cfg.ApplicationFundGoal == "" {
			fmt.Fprintf(os.Stderr, "ApplicationFundGoal is required\n")
			os.Exit(1)
		}

		// Get keyring backend from flag
		keyringBackend, err := cmd.Flags().GetString("keyring-backend")
		if err != nil {
			keyringBackend = "test"
		}

		// Create a context for the transaction
		ctx := context.Background()

		// Create codec and registry for keyring
		registry := codectypes.NewInterfaceRegistry()
		cryptocodec.RegisterInterfaces(registry)
		sdk.RegisterInterfaces(registry)
		authtypes.RegisterInterfaces(registry)
		banktypes.RegisterInterfaces(registry)

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
		// RPC endpoint should have http:// or https:// prefix
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
			WithChainID(cfg.ChainID).
			WithKeybase(kr).
			WithTxConfig(clientCtx.TxConfig).
			WithAccountRetriever(clientCtx.AccountRetriever)

		// Set default gas prices (0.01 upokt per gas unit)
		defaultGasPrice := sdk.NewDecCoinFromDec(volatile.DenomuPOKT, math.LegacyNewDecWithPrec(1, 2)) // 0.01 upokt
		gasPrices := sdk.NewDecCoins(defaultGasPrice)
		txFactory = txFactory.WithGasPrices(gasPrices.String())

		// Get the faucet key from the keyring
		faucetKey, err := kr.Key(faucetAddr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get faucet key: %v\n", err)
			os.Exit(1)
		}
		faucetAddrObj, err := faucetKey.GetAddress()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get faucet address: %v\n", err)
			os.Exit(1)
		}
		faucetAddrStr := faucetAddrObj.String()
		fmt.Printf("Using faucet address: %s\n", faucetAddrStr)

		// Get the debug flag
		debug, err := cmd.Flags().GetBool("debug")
		if err != nil {
			debug = false
		}

		// Create real clients using depinject
		// Create events query client
		eventsQueryClient := events.NewEventsQueryClient(rpcEndpoint)

		// Create block client
		cometClient, err := cosmosclient.NewClientFromNode(rpcEndpoint)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create comet client: %v\n", err)
			os.Exit(1)
		}

		// Create a BlockClient
		blockClientDeps := depinject.Supply(
			eventsQueryClient,
			cometClient,
		)
		blockClient, err := block.NewBlockClient(ctx, blockClientDeps)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create block client: %v\n", err)
			os.Exit(1)
		}

		// Create a TxContext using the existing clientCtx and txFactory
		txCtx, err := tx.NewTxContext(depinject.Supply(
			txtypes.Context(clientCtx),
			txFactory,
		))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create tx context: %v\n", err)
			os.Exit(1)
		}

		// Create depinject config with real clients
		deps := depinject.Supply(
			eventsQueryClient,
			blockClient,
			txCtx,
		)

		// Create a tx client using depinject
		txClient, err := tx.NewTxClient(
			ctx,
			deps,
			tx.WithSigningKeyName(faucetAddr),
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create tx client: %v\n", err)
			os.Exit(1)
		}

		// Fund accounts using the txClient
		err = fundAccountsWithTxClient(ctx, &cfg, txClient, clientCtx, faucetAddrStr, debug)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to fund accounts: %v\n", err)
			os.Exit(1)
		}
	},
}

// fundAccountsWithTxClient funds accounts using the TxClient
func fundAccountsWithTxClient(
	ctx context.Context,
	cfg *config.Config,
	txClient client.TxClient,
	clientCtx cosmosclient.Context,
	faucetAddrStr string,
	debug bool,
) error {
	var addressesToFund []string
	var fundingAmounts []math.Int

	// Parse the target fund goal
	targetFund, err := parseAmount(cfg.ApplicationFundGoal)
	if err != nil {
		return fmt.Errorf("failed to parse ApplicationFundGoal: %w", err)
	}

	// For GRPC, we should NOT add http:// prefix - use the raw endpoint
	grpcEndpoint := cfg.GrpcEndpoint

	// Define a struct to hold the results of balance checking
	type balanceCheckResult struct {
		address      string
		balance      *sdk.Coin
		amountNeeded math.Int
		err          error
	}

	// Create a channel to receive results
	resultChan := make(chan balanceCheckResult, len(cfg.Applications))

	// Create a worker pool to check balances concurrently
	// Use a semaphore to limit the number of concurrent requests
	maxConcurrent := 50 // Adjust this value based on what the server can handle
	sem := make(chan struct{}, maxConcurrent)

	// Create a connection pool for GRPC connections
	type connPoolItem struct {
		conn            *grpc.ClientConn
		bankQueryClient banktypes.QueryClient
	}

	// Create a buffered channel to serve as our connection pool
	connPoolSize := maxConcurrent / 5 // Adjust based on your needs
	if connPoolSize < 1 {
		connPoolSize = 1
	}
	connPool := make(chan connPoolItem, connPoolSize)

	// Initialize the connection pool
	for i := 0; i < connPoolSize; i++ {
		// Create a new GRPC connection
		var conn *grpc.ClientConn
		var err error

		// Check if the endpoint uses port 443 (HTTPS)
		if strings.Contains(grpcEndpoint, ":443") {
			// Use secure credentials for HTTPS endpoints
			conn, err = grpc.Dial(grpcEndpoint, grpc.WithTransportCredentials(
				credentials.NewTLS(&tls.Config{
					InsecureSkipVerify: false,
				}),
			))
		} else {
			// Use insecure credentials for non-HTTPS endpoints
			conn, err = grpc.Dial(grpcEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
		}

		if err != nil {
			return fmt.Errorf("failed to connect to GRPC endpoint: %w", err)
		}

		// Create a bank query client
		bankQueryClient := banktypes.NewQueryClient(conn)

		// Add to the pool
		connPool <- connPoolItem{
			conn:            conn,
			bankQueryClient: bankQueryClient,
		}
	}

	// Make sure to close all connections when we're done
	defer func() {
		// Drain the pool and close all connections
		for i := 0; i < connPoolSize; i++ {
			select {
			case item := <-connPool:
				item.conn.Close()
			default:
				// Pool is empty
				break
			}
		}
	}()

	fmt.Printf("Checking balances for %d accounts concurrently (max %d at a time, %d connections)...\n",
		len(cfg.Applications), maxConcurrent, connPoolSize)

	// Start a goroutine for each application to check its balance
	for _, app := range cfg.Applications {
		// Make a copy of app for the goroutine
		app := app

		// Acquire a semaphore slot
		sem <- struct{}{}

		go func() {
			// Release the semaphore slot when done
			defer func() { <-sem }()

			// Get a connection from the pool
			item := <-connPool

			// Make sure to return the connection to the pool when done
			defer func() { connPool <- item }()

			// Check the balance using the connection from the pool
			balance, err := getBalanceWithClient(ctx, item.bankQueryClient, app.Address)

			// Calculate amount needed if balance check was successful
			var amountNeeded math.Int
			if err == nil && balance.Amount.LT(targetFund) {
				amountNeeded = targetFund.Sub(balance.Amount)
			} else if err != nil {
				// If error, we'll fund the full amount
				amountNeeded = targetFund
			}

			// Send the result back through the channel
			resultChan <- balanceCheckResult{
				address:      app.Address,
				balance:      balance,
				amountNeeded: amountNeeded,
				err:          err,
			}
		}()
	}

	// Collect results from all goroutines
	totalAccounts := len(cfg.Applications)
	fmt.Printf("Starting balance checks for %d accounts...\n", totalAccounts)

	// Use atomic counter to track progress
	var completed int32

	// Create a ticker to periodically update progress
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	// Start a goroutine to display progress
	progressDone := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				current := atomic.LoadInt32(&completed)
				if current > 0 {
					percent := float64(current) / float64(totalAccounts) * 100
					fmt.Printf("\rProgress: %d/%d accounts checked (%.1f%%)...",
						current, totalAccounts, percent)
				}
			case <-progressDone:
				return
			}
		}
	}()

	for i := 0; i < totalAccounts; i++ {
		result := <-resultChan

		// Update progress counter
		atomic.AddInt32(&completed, 1)

		if result.err != nil {
			fmt.Fprintf(os.Stderr, "\nWarning: Failed to get balance for %s: %v\n", result.address, result.err)
			// Add to funding list with full amount if we can't check balance
			addressesToFund = append(addressesToFund, result.address)
			fundingAmounts = append(fundingAmounts, result.amountNeeded)
			continue
		}

		// If balance is less than target, add to funding list
		if result.balance.Amount.LT(targetFund) {
			addressesToFund = append(addressesToFund, result.address)
			fundingAmounts = append(fundingAmounts, result.amountNeeded)
			fmt.Fprintf(os.Stderr, "\nAccount %s needs funding. Current balance: %s, Target: %s, Funding: %s\n",
				result.address, result.balance.Amount.String(), targetFund.String(), result.amountNeeded.String())
		} else if debug {
			fmt.Fprintf(os.Stderr, "\nAccount %s has sufficient balance. Current: %s, Target: %s\n",
				result.address, result.balance.Amount.String(), targetFund.String())
		}
	}

	// Stop the progress display goroutine
	close(progressDone)

	// Print final progress and newline
	fmt.Printf("\rProgress: %d/%d accounts checked (100.0%%)...done!\n", totalAccounts, totalAccounts)

	// If no addresses need funding, return
	if len(addressesToFund) == 0 {
		fmt.Println("No accounts need funding. All balances are at or above the target.")
		return nil
	}

	// Process addresses in batches
	batchSize := 2000
	numBatches := (len(addressesToFund) + batchSize - 1) / batchSize // Ceiling division

	fmt.Printf("Processing %d addresses in %d batches of up to %d addresses each\n",
		len(addressesToFund), numBatches, batchSize)

	// Create a channel to track batch completion
	batchResults := make(chan struct {
		batchIndex int
		txHash     string
		err        error
	}, numBatches)

	// Create a semaphore to limit concurrent batches
	maxConcurrentBatches := 5 // Adjust based on what the node can handle
	batchSem := make(chan struct{}, maxConcurrentBatches)

	// Create a mutex to synchronize account sequence retrieval
	var accMutex sync.Mutex

	// Get initial account number and sequence
	faucetAddr, err := sdk.AccAddressFromBech32(faucetAddrStr)
	if err != nil {
		return fmt.Errorf("failed to parse faucet address: %w", err)
	}

	accNum, accSeq, err := clientCtx.AccountRetriever.GetAccountNumberSequence(clientCtx, faucetAddr)
	if err != nil {
		return fmt.Errorf("failed to get initial account number and sequence: %w", err)
	}

	fmt.Printf("Starting with account number %d and sequence %d\n", accNum, accSeq)

	// Start processing batches concurrently
	for i := 0; i < len(addressesToFund); i += batchSize {
		batchIndex := i / batchSize
		end := i + batchSize
		if end > len(addressesToFund) {
			end = len(addressesToFund)
		}

		batchAddresses := addressesToFund[i:end]
		batchAmounts := fundingAmounts[i:end]

		// Acquire a semaphore slot
		batchSem <- struct{}{}

		// Process this batch in a goroutine
		go func(batchIndex int, batchAddresses []string, batchAmounts []math.Int) {
			// Release the semaphore slot when done
			defer func() { <-batchSem }()

			// Create a context with timeout for this batch
			batchCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			defer cancel()

			// Create a batch of MsgSend messages
			var msgs []sdk.Msg
			for j, addr := range batchAddresses {
				// Create a MsgSend
				coinUpokt := sdk.NewCoin(volatile.DenomuPOKT, batchAmounts[j])
				sendMsg := &banktypes.MsgSend{
					FromAddress: faucetAddrStr,
					ToAddress:   addr,
					Amount:      sdk.NewCoins(coinUpokt),
				}
				msgs = append(msgs, sendMsg)
			}

			// Create a transaction builder
			txBuilder := clientCtx.TxConfig.NewTxBuilder()

			// Set the messages
			if err := txBuilder.SetMsgs(msgs...); err != nil {
				batchResults <- struct {
					batchIndex int
					txHash     string
					err        error
				}{batchIndex, "", fmt.Errorf("failed to set messages: %w", err)}
				return
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

			// Get account number and sequence
			// We already have faucetAddr from the outer scope

			// We need to get a fresh account number and sequence for each batch
			// This requires synchronization to avoid sequence conflicts
			var currentAccSeq uint64
			accMutex.Lock()
			// Use the pre-fetched account number, but get the current sequence
			currentAccSeq = accSeq
			// Increment the sequence for the next batch
			accSeq++
			accMutex.Unlock()

			// Create a transaction factory
			txFactory := cosmostx.Factory{}.
				WithChainID(clientCtx.ChainID).
				WithKeybase(clientCtx.Keyring).
				WithTxConfig(clientCtx.TxConfig).
				WithAccountRetriever(clientCtx.AccountRetriever).
				WithAccountNumber(accNum).
				WithSequence(currentAccSeq)

			// Sign the transaction
			err = cosmostx.Sign(batchCtx, txFactory, faucetAddrStr, txBuilder, true)
			if err != nil {
				batchResults <- struct {
					batchIndex int
					txHash     string
					err        error
				}{batchIndex, "", fmt.Errorf("failed to sign transaction: %w", err)}
				return
			}

			// Encode the transaction
			txBytes, err := clientCtx.TxConfig.TxEncoder()(txBuilder.GetTx())
			if err != nil {
				batchResults <- struct {
					batchIndex int
					txHash     string
					err        error
				}{batchIndex, "", fmt.Errorf("failed to encode transaction: %w", err)}
				return
			}

			// Broadcast the transaction
			res, err := clientCtx.BroadcastTxSync(txBytes)
			if err != nil {
				batchResults <- struct {
					batchIndex int
					txHash     string
					err        error
				}{batchIndex, "", fmt.Errorf("failed to broadcast transaction: %w", err)}
				return
			}

			// Check for errors in the response
			if res.Code != 0 {
				batchResults <- struct {
					batchIndex int
					txHash     string
					err        error
				}{batchIndex, "", fmt.Errorf("transaction failed: %s", res.RawLog)}
				return
			}

			// Send success result
			batchResults <- struct {
				batchIndex int
				txHash     string
				err        error
			}{batchIndex, res.TxHash, nil}

		}(batchIndex, batchAddresses, batchAmounts)
	}

	// Collect results from all batches
	successCount := 0
	failCount := 0

	fmt.Printf("Waiting for %d batches to complete...\n", numBatches)

	for i := 0; i < numBatches; i++ {
		result := <-batchResults

		if result.err != nil {
			failCount++
			fmt.Fprintf(os.Stderr, "Batch %d failed: %v\n", result.batchIndex, result.err)
		} else {
			successCount++
			fmt.Printf("Batch %d succeeded. Transaction hash: %s\n", result.batchIndex, result.txHash)
		}

		// Print progress
		fmt.Printf("Progress: %d/%d batches completed (%d succeeded, %d failed)\n",
			i+1, numBatches, successCount, failCount)
	}

	fmt.Printf("Funding complete. %d batches succeeded, %d batches failed.\n", successCount, failCount)

	return nil
}

// getBalanceWithClient queries the balance of an address using the provided client
func getBalanceWithClient(ctx context.Context, bankQueryClient banktypes.QueryClient, address string) (*sdk.Coin, error) {
	req := &banktypes.QueryBalanceRequest{
		Address: address,
		Denom:   volatile.DenomuPOKT,
	}

	res, err := bankQueryClient.Balance(ctx, req)
	if err != nil {
		return nil, err
	}

	return res.Balance, nil
}

// getBalance queries the balance of an address
func getBalance(ctx context.Context, bankQueryClient banktypes.QueryClient, address string) (*sdk.Coin, error) {
	return getBalanceWithClient(ctx, bankQueryClient, address)
}

// parseAmount parses a string amount like "1000000upokt" into an sdk.Int
func parseAmount(amount string) (math.Int, error) {
	// Remove the denomination suffix
	numStr := strings.TrimSuffix(amount, volatile.DenomuPOKT)

	// Parse the numeric part
	num, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		return math.ZeroInt(), err
	}

	return math.NewInt(num), nil
}

func init() {
	rootCmd.AddCommand(fundCmd)

	// Add keyring-backend flag
	fundCmd.Flags().String("keyring-backend", "test", "Keyring backend to use (os, file, test, inmemory)")

	// Add config flag
	fundCmd.Flags().String("config", "", "Path to the config file")

	// Add faucet flag
	fundCmd.Flags().String("faucet", "faucet", "Name or address of the faucet account to send funds from")

	// Add debug flag
	fundCmd.Flags().Bool("debug", false, "Enable debug output")
}
