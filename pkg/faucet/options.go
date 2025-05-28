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

// Config is the configuration for a faucet server.
// It is loaded via viper and therefore can be provided via:
// - config file, conforming to this structure (mapstructure tags as keys)
// - environment variables, prefixed with "FAUCET_" (e.g. FAUCET_SIGNING_KEY_NAME)
type Config struct {
	// SigningKeyName is the name of the key to use for signing transactions.
	// This key is expected to be present in the keyring.
	SigningKeyName string `mapstructure:"signing_key_name"`

	// SupportedSendCoins is a list of cosmos-sdk coin strings (i.e. amount + denom)
	// which the faucet server will send when a user requests tokens of that denomination.
	SupportedSendCoins []string `mapstructure:"supported_send_coins"`

	// CreateAccountsOnly determines whether the faucet will service all requests (when false),
	// or only those for recipient addresses which do not already exist onchain (when true).
	CreateAccountsOnly bool `mapstructure:"create_accounts_only"`

	// ListenAddress is the network address that the faucet HTTP server will listen on.
	// It is expected to be in the form of "host:port".
	ListenAddress string `mapstructure:"listen_address"`

	// signingAddress is the address of the signing key. It initialized in the NewConfig.
	// It SHOULD NOT be included in the config file, will be ignored if it is.
	signingAddress cosmostypes.AccAddress `mapstructure:"-"`

	// txClient is the tx client used by the faucet server to sign and broadcast transactions.
	// It is configured via a FaucetOptionFn (WithTxClient) and SHOULD NOT be included in the
	// config file. If it is, it will be ignored.
	txClient client.TxClient `mapstructure:"-"`

	// bankQueryClient is the bank query client used by the faucet server to query balances.
	// It is configured via a FaucetOptionFn (WithBankQueryClient) and SHOULD NOT be included
	// in the config file. If it is, it will be ignored.
	bankQueryClient bankGRPCQueryClient `mapstructure:"-"`
}

// FaucetOptionFn is a function that receives the faucet for configuration.
type FaucetOptionFn func(server *Server)

// NewConfig constructs a new faucet server configuration from the given arguments.
func NewConfig(
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

// Validate performs validation of the faucet server configuration:
// - SigningKeyName MUST be set
// - SupportedSendCoins MUST be set with at least one valid coin string (e.g. "1upokt")
// - ListenAddress MUST be set with a valid host:port format
func (config *Config) Validate() error {
	if config.SigningKeyName == "" {
		return fmt.Errorf("signing key name MUST be set")
	}

	if err := config.validateSupportedSendCoins(config.SupportedSendCoins); err != nil {
		return err
	}

	if err := config.validateListenAddress(config.ListenAddress); err != nil {
		return err
	}
	return nil
}

// GetSupportedSendCoins returns the configured supported_send_coins as a cosmos-sdk coins object.
func (config *Config) GetSupportedSendCoins() cosmostypes.Coins {
	sendCoins, _ := cosmostypes.ParseCoinsNormalized(strings.Join(config.SupportedSendCoins, ","))
	return sendCoins
}

// GetSigningAddress returns the address of the configured signing key.
func (config *Config) GetSigningAddress() cosmostypes.AccAddress {
	return config.signingAddress
}

// validateSupportedSendCoins ensures that the configured supported_send_coins are
// valid such that there's no possibility that they return an error when parsing later.
func (config *Config) validateSupportedSendCoins(supportedSendTokens []string) error {
	if _, err := cosmostypes.ParseCoinsNormalized(strings.Join(supportedSendTokens, ",")); err != nil {
		return fmt.Errorf("unable to parse send coins %w", err)
	}
	return nil
}

// validateListenAddress ensures that the configured listen_address conforms to the expected "host:port" format.
func (config *Config) validateListenAddress(listenAddress string) error {
	// Ensure that the listen address is the expected format.
	if _, _, err := net.SplitHostPort(listenAddress); err != nil {
		return fmt.Errorf("listen address MUST be in the form of host:port (e.g. 127.0.0.1:42069)")
	}
	return nil
}

// LoadSigningKey loads the key from the keyring provided by the given client context, using the configured signing_key_name.
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

// WithConfig sets the faucet server configuration.
func WithConfig(config *Config) FaucetOptionFn {
	return func(faucet *Server) {
		faucet.config = config
	}
}

// WithTxClient sets the faucet server's transaction client.
func WithTxClient(txClient client.TxClient) FaucetOptionFn {
	return func(faucet *Server) {
		faucet.config.txClient = txClient
	}
}

// bankGRPCQueryClient is an interface to the protobuf generated bank module gRPC query client which exposes the necessary methods for the faucet.
type bankGRPCQueryClient interface {
	AllBalances(ctx context.Context, in *banktypes.QueryAllBalancesRequest, opts ...grpc.CallOption) (*banktypes.QueryAllBalancesResponse, error)
}

// WithBankQueryClient sets the faucet server's bank query client.
func WithBankQueryClient(bankQueryClient bankGRPCQueryClient) FaucetOptionFn {
	return func(faucet *Server) {
		faucet.config.bankQueryClient = bankQueryClient
	}
}
