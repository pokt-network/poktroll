package faucet

import (
	"context"
	"fmt"
	"net"
	"strings"

	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"google.golang.org/grpc"

	"github.com/pokt-network/poktroll/pkg/client"
)

// Config defines the configuration for a faucet server.
// Loaded via viper. Can be provided through:
// - A config file matching this structure (uses mapstructure tags as keys)
// - Environment variables, prefixed with "FAUCET_" (e.g. FAUCET_SIGNING_KEY_NAME)
type Config struct {

	// ############################################################
	// ### Public Configs - Settable via config file or env ###
	// ############################################################

	// SigningKeyName specifies the key (by name) used to sign transactions.
	// - Must exist in the keyring.
	SigningKeyName string `mapstructure:"signing_key_name"`

	// SupportedSendCoins lists cosmos-sdk coin strings (amount + denom).
	// - Specifies which coins the faucet can send to users.
	SupportedSendCoins []string `mapstructure:"supported_send_coins"`

	// CreateAccountsOnly controls faucet behavior:
	// - true: Only service requests for accounts not yet on-chain (i.e. new accounts without an onchain public key)
	// - false: Service all requests (i.e. do not reject requests from any)
	CreateAccountsOnly bool `mapstructure:"create_accounts_only"`

	// ListenAddress specifies the network address for the faucet HTTP server.
	// - Format: "host:port"
	ListenAddress string `mapstructure:"listen_address"`

	// ############################################################
	// ### Internal Configs - Ignored if present in config file ###
	// ############################################################

	// signingAddress holds the address of the signing key.
	// - Initialized in NewConfig
	// - Ignored if present in config file
	signingAddress cosmostypes.AccAddress `mapstructure:"-"`

	// txClient is used by the faucet to sign and broadcast transactions.
	// - Set via FaucetOptionFn (WithTxClient)
	// - Ignored if present in config file
	txClient client.TxClient `mapstructure:"-"`

	// bankQueryClient is used by the faucet to query account balances.
	// - Set via FaucetOptionFn (WithBankQueryClient)
	// - Ignored if present in config file
	bankQueryClient bankGRPCQueryClient `mapstructure:"-"`
}

// FaucetOptionFn defines a function that configures a faucet server instance.
type FaucetOptionFn func(server *Server)

// NewFaucetConfig creates a new faucet server configuration from the provided arguments.
func NewFaucetConfig(
	clientCtx cosmosclient.Context,
	signingKeyName string,
	listenAddress string,
	sendTokens []string,
	createAccountOnly bool,
) (*Config, error) {
	if len(sendTokens) == 0 {
		return nil, fmt.Errorf("send tokens MUST contain at least one token (e.g. 1upokt)")
	}

	config := &Config{
		SigningKeyName:     signingKeyName,
		ListenAddress:      listenAddress,
		SupportedSendCoins: sendTokens,
		CreateAccountsOnly: createAccountOnly,
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	if err := config.LoadSigningKey(clientCtx); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate checks faucet server configuration for correctness and minimum requirements:
// - SigningKeyName must be set
// - SupportedSendCoins must include at least one valid coin string (e.g. "1upokt")
// - ListenAddress must be a valid "host:port" string
func (config *Config) Validate() error {
	if config.SigningKeyName == "" {
		return fmt.Errorf("signing key name MUST be set")
	}

	// At least one valid & support coin token must be specified.
	if err := config.validateSupportedSendCoins(config.SupportedSendCoins); err != nil {
		return err
	}

	if err := config.validateListenAddress(config.ListenAddress); err != nil {
		return err
	}
	return nil
}

// GetSupportedSendCoins returns SupportedSendCoins as a cosmos-sdk Coins object.
func (config *Config) GetSupportedSendCoins() cosmostypes.Coins {
	sendCoins, _ := cosmostypes.ParseCoinsNormalized(strings.Join(config.SupportedSendCoins, ","))
	return sendCoins
}

// GetSigningAddress returns the address of the configured signing key.
func (config *Config) GetSigningAddress() cosmostypes.AccAddress {
	return config.signingAddress
}

// validateSupportedSendCoins checks that SupportedSendCoins are valid coin strings.
// - Ensures no parsing errors will occur when used later.
func (config *Config) validateSupportedSendCoins(supportedSendTokens []string) error {
	if _, err := cosmostypes.ParseCoinsNormalized(strings.Join(supportedSendTokens, ",")); err != nil {
		return fmt.Errorf("unable to parse send coins %w", err)
	}
	return nil
}

// validateListenAddress checks that ListenAddress matches the "host:port" format.
func (config *Config) validateListenAddress(listenAddress string) error {
	// Ensure that the listen address is the expected format.
	if _, _, err := net.SplitHostPort(listenAddress); err != nil {
		return fmt.Errorf("listen address MUST be in the form of host:port (e.g. 127.0.0.1:42069)")
	}
	return nil
}

// LoadSigningKey loads the signing key from the keyring in the given client context.
// - Uses the configured SigningKeyName.
func (config *Config) LoadSigningKey(clientCtx cosmosclient.Context) error {
	// Load the faucet key, by name, from the keyring.
	// NOTE: DOES respect the --keyring-backend and --home flags.
	keyRecord, err := clientCtx.Keyring.Key(config.SigningKeyName)
	if err != nil {
		return err
	}

	// Set the faucet send account address.
	config.signingAddress, err = keyRecord.GetAddress()
	if err != nil {
		return err
	}

	return nil
}

// WithConfig sets the faucet server's configuration object.
func WithConfig(config *Config) FaucetOptionFn {
	return func(faucet *Server) {
		faucet.config = config
	}
}

// WithTxClient sets the faucet server's transaction (tx) client.
func WithTxClient(txClient client.TxClient) FaucetOptionFn {
	return func(faucet *Server) {
		faucet.config.txClient = txClient
	}
}

// bankGRPCQueryClient defines the interface to the bank module's gRPC query client.
// - Exposes only the methods required by the faucet.
type bankGRPCQueryClient interface {
	AllBalances(ctx context.Context, in *banktypes.QueryAllBalancesRequest, opts ...grpc.CallOption) (*banktypes.QueryAllBalancesResponse, error)
}

// WithBankQueryClient sets the faucet server's bank query client.
func WithBankQueryClient(bankQueryClient bankGRPCQueryClient) FaucetOptionFn {
	return func(faucet *Server) {
		faucet.config.bankQueryClient = bankQueryClient
	}
}
