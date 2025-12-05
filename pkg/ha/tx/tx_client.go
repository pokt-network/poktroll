package tx

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	pocktclient "github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/ha/keys"
	"github.com/pokt-network/poktroll/pkg/ha/logging"
	"github.com/pokt-network/poktroll/pkg/polylog"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

const (
	// DefaultGasLimit is the default gas limit for transactions.
	DefaultGasLimit = 500000

	// DefaultGasPrice is the default gas price in upokt.
	DefaultGasPrice = "0.000001upokt"

	// DefaultTimeoutHeight is the number of blocks after which a transaction times out.
	DefaultTimeoutHeight = 100

	// DefaultChainID for the pocket network.
	DefaultChainID = "pocket"
)

// TxClientConfig contains configuration for the transaction client.
type TxClientConfig struct {
	// GRPCEndpoint is the gRPC endpoint for the full node.
	// Only used if GRPCConn is nil.
	GRPCEndpoint string

	// GRPCConn is an existing gRPC connection to reuse.
	// If provided, GRPCEndpoint and UseTLS are ignored.
	// The caller is responsible for closing this connection.
	GRPCConn *grpc.ClientConn

	// ChainID is the chain ID of the network.
	ChainID string

	// GasLimit is the gas limit for transactions.
	GasLimit uint64

	// GasPrice is the gas price for transactions.
	GasPrice cosmostypes.DecCoin

	// TimeoutBlocks is the number of blocks after which a transaction times out.
	TimeoutBlocks uint64

	// UseTLS enables TLS for the gRPC connection.
	// Set to true when connecting to endpoints on port 443 or with TLS enabled.
	// Only used if GRPCConn is nil.
	// Default: false (insecure connection)
	UseTLS bool
}

// TxClient provides transaction submission capabilities for the HA system.
// It supports multi-supplier signing using private keys from the KeyManager.
type TxClient struct {
	logger     polylog.Logger
	config     TxClientConfig
	keyManager keys.KeyManager
	grpcConn   *grpc.ClientConn
	ownsConn   bool // true if we created the connection and should close it

	// Codec for encoding/decoding transactions
	codec       codec.Codec
	txConfig    client.TxConfig
	authQuerier authtypes.QueryClient
	txClient    txtypes.ServiceClient

	// Per-supplier account info cache
	accountCache   map[string]*authtypes.BaseAccount
	accountCacheMu sync.RWMutex

	// Mutex to prevent concurrent transactions
	txMu sync.Mutex

	// Lifecycle
	closed bool
	mu     sync.RWMutex
}

// NewTxClient creates a new transaction client.
func NewTxClient(
	logger polylog.Logger,
	keyManager keys.KeyManager,
	config TxClientConfig,
) (*TxClient, error) {
	// Validate: either GRPCConn or GRPCEndpoint must be provided
	if config.GRPCConn == nil && config.GRPCEndpoint == "" {
		return nil, fmt.Errorf("either GRPCConn or GRPCEndpoint is required")
	}
	if config.ChainID == "" {
		config.ChainID = DefaultChainID
	}
	if config.GasLimit == 0 {
		config.GasLimit = DefaultGasLimit
	}
	// Check Denom instead of IsZero() since zero-value DecCoin has nil internal state
	if config.GasPrice.Denom == "" {
		gasPrice, err := cosmostypes.ParseDecCoin(DefaultGasPrice)
		if err != nil {
			return nil, fmt.Errorf("failed to parse default gas price: %w", err)
		}
		config.GasPrice = gasPrice
	}
	if config.TimeoutBlocks == 0 {
		config.TimeoutBlocks = DefaultTimeoutHeight
	}

	var grpcConn *grpc.ClientConn
	var ownsConn bool

	if config.GRPCConn != nil {
		// Use the provided connection (caller owns it)
		grpcConn = config.GRPCConn
		ownsConn = false
	} else {
		// Create our own connection
		var transportCreds credentials.TransportCredentials
		if config.UseTLS {
			transportCreds = credentials.NewTLS(&tls.Config{
				MinVersion: tls.VersionTLS12,
			})
		} else {
			transportCreds = insecure.NewCredentials()
		}

		var err error
		grpcConn, err = grpc.NewClient(
			config.GRPCEndpoint,
			grpc.WithTransportCredentials(transportCreds),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
		}
		ownsConn = true
	}

	// Create codec and tx config
	cdc, txConfig := createCodecAndTxConfig()

	tc := &TxClient{
		logger:       logging.ForComponent(logger, logging.ComponentTxClient),
		config:       config,
		keyManager:   keyManager,
		grpcConn:     grpcConn,
		ownsConn:     ownsConn,
		codec:        cdc,
		txConfig:     txConfig,
		authQuerier:  authtypes.NewQueryClient(grpcConn),
		txClient:     txtypes.NewServiceClient(grpcConn),
		accountCache: make(map[string]*authtypes.BaseAccount),
	}

	tc.logger.Info().
		Str("endpoint", config.GRPCEndpoint).
		Str("chain_id", config.ChainID).
		Bool("shared_conn", !ownsConn).
		Msg("transaction client initialized")

	return tc, nil
}

// createCodecAndTxConfig creates the codec and transaction config for signing.
func createCodecAndTxConfig() (codec.Codec, client.TxConfig) {
	registry := codectypes.NewInterfaceRegistry()

	// Register necessary interfaces
	authtypes.RegisterInterfaces(registry)
	cryptocodec.RegisterInterfaces(registry)
	prooftypes.RegisterInterfaces(registry)
	sessiontypes.RegisterInterfaces(registry)

	cdc := codec.NewProtoCodec(registry)
	txConfig := authtx.NewTxConfig(cdc, authtx.DefaultSignModes)

	return cdc, txConfig
}

// CreateClaims creates and submits claim transactions for a supplier.
func (tc *TxClient) CreateClaims(
	ctx context.Context,
	supplierOperatorAddr string,
	claims []*prooftypes.MsgCreateClaim,
) error {
	tc.mu.RLock()
	if tc.closed {
		tc.mu.RUnlock()
		return fmt.Errorf("tx client is closed")
	}
	tc.mu.RUnlock()

	if len(claims) == 0 {
		return nil
	}

	// Convert claims to Msg interface
	msgs := make([]cosmostypes.Msg, len(claims))
	for i, claim := range claims {
		msgs[i] = claim
	}

	txHash, err := tc.signAndBroadcast(ctx, supplierOperatorAddr, msgs...)
	if err != nil {
		txClaimErrors.WithLabelValues(supplierOperatorAddr, "broadcast").Inc()
		return fmt.Errorf("failed to broadcast claims: %w", err)
	}

	tc.logger.Info().
		Str("supplier", supplierOperatorAddr).
		Int("num_claims", len(claims)).
		Str("tx_hash", txHash).
		Msg("claims submitted")

	txClaimsSubmitted.WithLabelValues(supplierOperatorAddr).Add(float64(len(claims)))
	return nil
}

// SubmitProofs submits proof transactions for a supplier.
func (tc *TxClient) SubmitProofs(
	ctx context.Context,
	supplierOperatorAddr string,
	proofs []*prooftypes.MsgSubmitProof,
) error {
	tc.mu.RLock()
	if tc.closed {
		tc.mu.RUnlock()
		return fmt.Errorf("tx client is closed")
	}
	tc.mu.RUnlock()

	if len(proofs) == 0 {
		return nil
	}

	// Convert proofs to Msg interface
	msgs := make([]cosmostypes.Msg, len(proofs))
	for i, proof := range proofs {
		msgs[i] = proof
	}

	txHash, err := tc.signAndBroadcast(ctx, supplierOperatorAddr, msgs...)
	if err != nil {
		txProofErrors.WithLabelValues(supplierOperatorAddr, "broadcast").Inc()
		return fmt.Errorf("failed to broadcast proofs: %w", err)
	}

	tc.logger.Info().
		Str("supplier", supplierOperatorAddr).
		Int("num_proofs", len(proofs)).
		Str("tx_hash", txHash).
		Msg("proofs submitted")

	txProofsSubmitted.WithLabelValues(supplierOperatorAddr).Add(float64(len(proofs)))
	return nil
}

// signAndBroadcast signs and broadcasts a transaction.
func (tc *TxClient) signAndBroadcast(
	ctx context.Context,
	signerAddr string,
	msgs ...cosmostypes.Msg,
) (string, error) {
	tc.txMu.Lock()
	defer tc.txMu.Unlock()

	startTime := time.Now()
	defer func() {
		txBroadcastLatency.WithLabelValues(signerAddr).Observe(time.Since(startTime).Seconds())
	}()

	// Get signing key
	privKey, err := tc.keyManager.GetSigner(signerAddr)
	if err != nil {
		return "", fmt.Errorf("failed to get signing key: %w", err)
	}

	// Get account info
	account, err := tc.getAccount(ctx, signerAddr)
	if err != nil {
		return "", fmt.Errorf("failed to get account: %w", err)
	}

	// Build the transaction
	txBuilder := tc.txConfig.NewTxBuilder()
	if setMsgsErr := txBuilder.SetMsgs(msgs...); setMsgsErr != nil {
		return "", fmt.Errorf("failed to set messages: %w", setMsgsErr)
	}

	// Set gas and fees
	txBuilder.SetGasLimit(tc.config.GasLimit)
	feeAmount := tc.calculateFee()
	txBuilder.SetFeeAmount(feeAmount)

	// Set memo (optional)
	txBuilder.SetMemo("HA RelayMiner")

	// Sign the transaction
	err = tc.signTx(ctx, txBuilder, privKey, account)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Encode the transaction
	txBytes, err := tc.txConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return "", fmt.Errorf("failed to encode transaction: %w", err)
	}

	// Broadcast the transaction
	res, err := tc.txClient.BroadcastTx(ctx, &txtypes.BroadcastTxRequest{
		TxBytes: txBytes,
		Mode:    txtypes.BroadcastMode_BROADCAST_MODE_SYNC,
	})
	if err != nil {
		return "", fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	if res.TxResponse.Code != 0 {
		return res.TxResponse.TxHash, fmt.Errorf("transaction failed: %s", res.TxResponse.RawLog)
	}

	// Increment account sequence for next transaction
	tc.incrementSequence(signerAddr)

	txBroadcastsTotal.WithLabelValues(signerAddr, "success").Inc()
	return res.TxResponse.TxHash, nil
}

// signTx signs a transaction with the given private key.
func (tc *TxClient) signTx(
	ctx context.Context,
	txBuilder client.TxBuilder,
	privKey cryptotypes.PrivKey,
	account *authtypes.BaseAccount,
) error {
	pubKey := privKey.PubKey()
	signMode := signing.SignMode_SIGN_MODE_DIRECT

	// Set signature info placeholder
	sigV2 := signing.SignatureV2{
		PubKey: pubKey,
		Data: &signing.SingleSignatureData{
			SignMode:  signMode,
			Signature: nil,
		},
		Sequence: account.Sequence,
	}

	if err := txBuilder.SetSignatures(sigV2); err != nil {
		return fmt.Errorf("failed to set signature placeholder: %w", err)
	}

	// Build sign data
	signerData := authsigning.SignerData{
		ChainID:       tc.config.ChainID,
		AccountNumber: account.AccountNumber,
		Sequence:      account.Sequence,
		PubKey:        pubKey,
		Address:       account.Address,
	}

	// Get bytes to sign using the sign mode handler
	bytesToSign, err := authsigning.GetSignBytesAdapter(
		ctx,
		tc.txConfig.SignModeHandler(),
		signMode,
		signerData,
		txBuilder.GetTx(),
	)
	if err != nil {
		return fmt.Errorf("failed to get sign bytes: %w", err)
	}

	// Sign
	signature, err := privKey.Sign(bytesToSign)
	if err != nil {
		return fmt.Errorf("failed to sign: %w", err)
	}

	// Set the actual signature
	sigV2.Data = &signing.SingleSignatureData{
		SignMode:  signMode,
		Signature: signature,
	}

	if err := txBuilder.SetSignatures(sigV2); err != nil {
		return fmt.Errorf("failed to set signature: %w", err)
	}

	return nil
}

// getAccount retrieves account info from chain or cache.
func (tc *TxClient) getAccount(ctx context.Context, addr string) (*authtypes.BaseAccount, error) {
	// Check cache first
	tc.accountCacheMu.RLock()
	if account, ok := tc.accountCache[addr]; ok {
		tc.accountCacheMu.RUnlock()
		return account, nil
	}
	tc.accountCacheMu.RUnlock()

	// Query chain
	tc.accountCacheMu.Lock()
	defer tc.accountCacheMu.Unlock()

	// Double-check after acquiring lock
	if account, ok := tc.accountCache[addr]; ok {
		return account, nil
	}

	res, err := tc.authQuerier.Account(ctx, &authtypes.QueryAccountRequest{
		Address: addr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query account: %w", err)
	}

	var account authtypes.BaseAccount
	if err := tc.codec.UnpackAny(res.Account, &account); err != nil {
		// Try unpacking as BaseAccount directly
		if err := account.Unmarshal(res.Account.Value); err != nil {
			return nil, fmt.Errorf("failed to unpack account: %w", err)
		}
	}

	tc.accountCache[addr] = &account
	return &account, nil
}

// incrementSequence increments the cached sequence number.
func (tc *TxClient) incrementSequence(addr string) {
	tc.accountCacheMu.Lock()
	defer tc.accountCacheMu.Unlock()

	if account, ok := tc.accountCache[addr]; ok {
		account.Sequence++
	}
}

// InvalidateAccount removes an account from the cache.
func (tc *TxClient) InvalidateAccount(addr string) {
	tc.accountCacheMu.Lock()
	defer tc.accountCacheMu.Unlock()
	delete(tc.accountCache, addr)
}

// calculateFee calculates the transaction fee.
func (tc *TxClient) calculateFee() cosmostypes.Coins {
	gasLimitDec := math.LegacyNewDec(int64(tc.config.GasLimit))
	feeAmount := tc.config.GasPrice.Amount.Mul(gasLimitDec)

	// Truncate and add 1 if there's a remainder to ensure we don't underpay
	feeInt := feeAmount.TruncateInt()
	if feeAmount.Sub(math.LegacyNewDecFromInt(feeInt)).IsPositive() {
		feeInt = feeInt.Add(math.OneInt())
	}

	return cosmostypes.NewCoins(cosmostypes.NewCoin(tc.config.GasPrice.Denom, feeInt))
}

// Close closes the transaction client.
// If the client was created with a shared gRPC connection, it will not be closed.
func (tc *TxClient) Close() error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if tc.closed {
		return nil
	}
	tc.closed = true

	// Only close the connection if we created it ourselves
	if tc.ownsConn && tc.grpcConn != nil {
		if err := tc.grpcConn.Close(); err != nil {
			return fmt.Errorf("failed to close gRPC connection: %w", err)
		}
	}

	tc.logger.Info().Msg("transaction client closed")
	return nil
}

// =============================================================================
// SupplierClient wrapper for compatibility with pkg/client interfaces
// =============================================================================

// HASupplierClient wraps TxClient to implement the client.SupplierClient interface.
type HASupplierClient struct {
	txClient     *TxClient
	operatorAddr string
	logger       polylog.Logger
}

// NewHASupplierClient creates a new supplier client for a specific operator.
func NewHASupplierClient(
	txClient *TxClient,
	operatorAddr string,
	logger polylog.Logger,
) *HASupplierClient {
	return &HASupplierClient{
		txClient:     txClient,
		operatorAddr: operatorAddr,
		logger:       logger.With("supplier", operatorAddr),
	}
}

// CreateClaims implements client.SupplierClient.
func (c *HASupplierClient) CreateClaims(
	ctx context.Context,
	timeoutHeight int64,
	claimMsgs ...pocktclient.MsgCreateClaim,
) error {
	claims := make([]*prooftypes.MsgCreateClaim, len(claimMsgs))
	for i, msg := range claimMsgs {
		claim, ok := msg.(*prooftypes.MsgCreateClaim)
		if !ok {
			return fmt.Errorf("invalid claim message type: %T", msg)
		}
		claims[i] = claim
	}

	return c.txClient.CreateClaims(ctx, c.operatorAddr, claims)
}

// SubmitProofs implements client.SupplierClient.
func (c *HASupplierClient) SubmitProofs(
	ctx context.Context,
	timeoutHeight int64,
	proofMsgs ...pocktclient.MsgSubmitProof,
) error {
	proofs := make([]*prooftypes.MsgSubmitProof, len(proofMsgs))
	for i, msg := range proofMsgs {
		proof, ok := msg.(*prooftypes.MsgSubmitProof)
		if !ok {
			return fmt.Errorf("invalid proof message type: %T", msg)
		}
		proofs[i] = proof
	}

	return c.txClient.SubmitProofs(ctx, c.operatorAddr, proofs)
}

// OperatorAddress implements client.SupplierClient.
func (c *HASupplierClient) OperatorAddress() string {
	return c.operatorAddr
}
