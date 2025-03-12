package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"cosmossdk.io/math"
	comettypes "github.com/cometbft/cometbft/types"
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

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/either"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/tools/relay-spam/config"
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

// We CANNOT use CLI TX BANK SEND! We MUST USE TXCLIENT and send MANY MSGS in ONE transaction!

// Simple implementation of the Block interface
type simpleBlock struct {
	height int64
	hash   []byte
}

func (b *simpleBlock) Height() int64 {
	return b.height
}

func (b *simpleBlock) Hash() []byte {
	return b.hash
}

func (b *simpleBlock) Txs() []comettypes.Tx {
	return []comettypes.Tx{}
}

// Simple implementation of the BlockReplayObservable interface
type simpleBlockReplayObservable struct {
	block client.Block
}

func newSimpleBlockReplayObservable(block client.Block) *simpleBlockReplayObservable {
	return &simpleBlockReplayObservable{
		block: block,
	}
}

func (o *simpleBlockReplayObservable) Subscribe(ctx context.Context) observable.Observer[client.Block] {
	// Create a channel that will emit the block once and then close
	ch := make(chan client.Block, 1)
	ch <- o.block
	close(ch)

	// Return an observer that will read from the channel
	// Create a no-op unsubscribe function
	unsubscribe := func(toRemove observable.Observer[client.Block]) {}
	return channel.NewObserver[client.Block](ctx, unsubscribe)
}

func (o *simpleBlockReplayObservable) Last(ctx context.Context, n int) []client.Block {
	// Always return the same block regardless of n
	return []client.Block{o.block}
}

// GetReplayBufferSize returns the number of elements in the replay buffer
func (o *simpleBlockReplayObservable) GetReplayBufferSize() int {
	// We always have 1 block in our simple implementation
	return 1
}

// SubscribeFromLatestBufferedOffset returns an observer which is initially notified of
// values in the replay buffer, starting from the latest buffered value at index 'offset'.
func (o *simpleBlockReplayObservable) SubscribeFromLatestBufferedOffset(ctx context.Context, offset int) observable.Observer[client.Block] {
	// For our simple implementation, we ignore the offset and just return the same as Subscribe
	return o.Subscribe(ctx)
}

// UnsubscribeAll unsubscribes and removes all observers from the observable.
func (o *simpleBlockReplayObservable) UnsubscribeAll() {
	// No-op for our simple implementation as we don't maintain a list of observers
}

// Simple implementation of the BlockClient interface
type simpleBlockClient struct {
	block client.Block
}

func newSimpleBlockClient() *simpleBlockClient {
	return &simpleBlockClient{
		block: &simpleBlock{
			height: 1,
			hash:   []byte("simple_block_hash"),
		},
	}
}

func (c *simpleBlockClient) CommittedBlocksSequence(ctx context.Context) client.BlockReplayObservable {
	// Return a simple implementation of BlockReplayObservable
	return newSimpleBlockReplayObservable(c.block)
}

func (c *simpleBlockClient) LastBlock(ctx context.Context) client.Block {
	return c.block
}

func (c *simpleBlockClient) Close() {
	// Nothing to close
}

// Simple implementation of the EventsQueryClient interface
type simpleEventsQueryClient struct{}

func newSimpleEventsQueryClient() *simpleEventsQueryClient {
	return &simpleEventsQueryClient{}
}

func (c *simpleEventsQueryClient) EventsBytes(
	ctx context.Context,
	query string,
) (client.EventsBytesObservable, error) {
	// Create a new observable that will never emit any events
	obs, _ := channel.NewObservable[either.Bytes]()
	return obs, nil
}

func (c *simpleEventsQueryClient) Close() {
	// Nothing to close
}

// mockAccountRetriever is a simple implementation of the AccountRetriever interface
// that always returns a fixed account number and sequence.
type mockAccountRetriever struct{}

// GetAccount implements the AccountRetriever interface.
func (ar mockAccountRetriever) GetAccount(clientCtx cosmosclient.Context, addr sdk.AccAddress) (cosmosclient.Account, error) {
	return nil, nil
}

// GetAccountWithHeight implements the AccountRetriever interface.
func (ar mockAccountRetriever) GetAccountWithHeight(clientCtx cosmosclient.Context, addr sdk.AccAddress) (cosmosclient.Account, int64, error) {
	return nil, 0, nil
}

// EnsureExists implements the AccountRetriever interface.
func (ar mockAccountRetriever) EnsureExists(clientCtx cosmosclient.Context, addr sdk.AccAddress) error {
	return nil
}

// GetAccountNumberSequence implements the AccountRetriever interface.
func (ar mockAccountRetriever) GetAccountNumberSequence(clientCtx cosmosclient.Context, addr sdk.AccAddress) (uint64, uint64, error) {
	return 1, 1, nil
}

// fundCmd represents the fund command
var fundCmd = &cobra.Command{
	Use:   "fund",
	Short: "Fund accounts",
	Long:  `Fund accounts by sending transactions directly, only funding the difference needed to reach the target balance.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Load config
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
			os.Exit(1)
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
		banktypes.RegisterInterfaces(registry)

		// Create the codec with the registry
		cdc := codec.NewProtoCodec(registry)

		// Create a legacy Amino codec for address encoding
		amino := codec.NewLegacyAmino()
		sdk.RegisterLegacyAminoCodec(amino)
		cryptocodec.RegisterCrypto(amino)

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
			WithChainID(chainID).
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
			WithChainID(chainID).
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

		// Fund accounts
		err = fundAccountsWithCosmosClient(ctx, cfg, clientCtx, txFactory, faucetAddrStr, faucetAddr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to fund accounts: %v\n", err)
			os.Exit(1)
		}
	},
}

// fundAccountsWithCosmosClient funds accounts using the Cosmos SDK client
func fundAccountsWithCosmosClient(
	ctx context.Context,
	cfg *config.Config,
	clientCtx cosmosclient.Context,
	txFactory cosmostx.Factory,
	faucetAddrStr string,
	faucetKeyName string,
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

	// Connect to the GRPC endpoint
	conn, err := grpc.Dial(grpcEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to GRPC endpoint: %w", err)
	}
	defer conn.Close()

	// Create a bank query client
	bankQueryClient := banktypes.NewQueryClient(conn)

	// Check each application's balance
	for _, app := range cfg.Applications {
		balance, err := getBalance(ctx, bankQueryClient, app.Address)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to get balance for %s: %v\n", app.Address, err)
			// Add to funding list with full amount if we can't check balance
			addressesToFund = append(addressesToFund, app.Address)
			fundingAmounts = append(fundingAmounts, targetFund)
			continue
		}

		// If balance is less than target, add to funding list
		if balance.Amount.LT(targetFund) {
			// Calculate the amount needed to reach the target
			amountNeeded := targetFund.Sub(balance.Amount)
			addressesToFund = append(addressesToFund, app.Address)
			fundingAmounts = append(fundingAmounts, amountNeeded)
			fmt.Fprintf(os.Stderr, "Account %s needs funding. Current balance: %s, Target: %s, Funding: %s\n",
				app.Address, balance.Amount.String(), targetFund.String(), amountNeeded.String())
		} else {
			fmt.Fprintf(os.Stderr, "Account %s has sufficient balance. Current: %s, Target: %s\n",
				app.Address, balance.Amount.String(), targetFund.String())
		}
	}

	// If no addresses need funding, return
	if len(addressesToFund) == 0 {
		fmt.Println("No accounts need funding. All balances are at or above the target.")
		return nil
	}

	// Process addresses in batches
	batchSize := 1000 // Smaller batch size for testing
	for i := 0; i < len(addressesToFund); i += batchSize {
		end := i + batchSize
		if end > len(addressesToFund) {
			end = len(addressesToFund)
		}

		batchAddresses := addressesToFund[i:end]
		batchAmounts := fundingAmounts[i:end]

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
			fmt.Printf("Adding message to send %s to %s\n", coinUpokt.String(), addr)
		}

		// Sign and broadcast the transaction with all messages
		fmt.Printf("Sending batch transaction with %d messages...\n", len(msgs))

		// Create a transaction builder
		txBuilder := clientCtx.TxConfig.NewTxBuilder()

		// Set the messages
		if err := txBuilder.SetMsgs(msgs...); err != nil {
			return fmt.Errorf("failed to set messages: %w", err)
		}

		// Set gas limit - using a high value to ensure it goes through
		txBuilder.SetGasLimit(1000000000000)

		// Set fee amount based on gas limit and gas prices
		gasPrices := txFactory.GasPrices()
		fees := sdk.NewCoins()
		for _, gasPrice := range gasPrices {
			fee := gasPrice.Amount.MulInt(math.NewInt(int64(txBuilder.GetTx().GetGas()))).RoundInt()
			fees = fees.Add(sdk.NewCoin(gasPrice.Denom, fee))
		}
		txBuilder.SetFeeAmount(fees)

		// Get account number and sequence
		faucetAddr, err := sdk.AccAddressFromBech32(faucetAddrStr)
		if err != nil {
			return fmt.Errorf("failed to parse faucet address: %w", err)
		}

		accNum, accSeq, err := clientCtx.AccountRetriever.GetAccountNumberSequence(clientCtx, faucetAddr)
		if err != nil {
			return fmt.Errorf("failed to get account number and sequence: %w", err)
		}

		// Set the account number and sequence
		txFactory = txFactory.WithAccountNumber(accNum).WithSequence(accSeq)

		// Sign the transaction
		err = cosmostx.Sign(ctx, txFactory, faucetKeyName, txBuilder, true)
		if err != nil {
			return fmt.Errorf("failed to sign transaction: %w", err)
		}

		// Encode the transaction
		txBytes, err := clientCtx.TxConfig.TxEncoder()(txBuilder.GetTx())
		if err != nil {
			return fmt.Errorf("failed to encode transaction: %w", err)
		}

		// Broadcast the transaction
		res, err := clientCtx.BroadcastTxSync(txBytes)
		if err != nil {
			return fmt.Errorf("failed to broadcast transaction: %w", err)
		}

		// Check for errors in the response
		if res.Code != 0 {
			return fmt.Errorf("transaction failed: %s", res.RawLog)
		}

		fmt.Printf("Successfully funded %d accounts in batch. Transaction hash: %s\n", len(batchAddresses), res.TxHash)

		// Add a small delay between batches to avoid overwhelming the node
		if i+batchSize < len(addressesToFund) {
			time.Sleep(1 * time.Second)
		}
	}

	return nil
}

// getBalance queries the balance of an address
func getBalance(ctx context.Context, bankQueryClient banktypes.QueryClient, address string) (*sdk.Coin, error) {
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

	// Add chain-id flag
	fundCmd.Flags().String("chain-id", "poktroll", "Chain ID of the blockchain")

	// Add faucet flag
	fundCmd.Flags().String("faucet", "faucet", "Name or address of the faucet account to send funds from")
}
