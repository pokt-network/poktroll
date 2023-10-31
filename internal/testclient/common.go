package testclient

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"

	"pocket/app"
	"pocket/cmd/pocketd/cmd"
)

// CometLocalWebsocketURL provides a default URL pointing to the localnet websocket endpoint.
const CometLocalWebsocketURL = "ws://localhost:36657/websocket"

// EncodingConfig encapsulates encoding configurations for the Pocket application.

var EncodingConfig = app.MakeEncodingConfig()

// init initializes the SDK configuration upon package import.
func init() {
	cmd.InitSDKConfig()
}

// NewLocalnetClientCtx creates a client context specifically tailored for localnet
// environments. The returned client context is initialized with encoding
// configurations, a default home directory, a default account retriever, and
// command flags.
//
// Parameters:
// - t: The testing.T instance used for the current test.
// - flagSet: The set of flags to be read for initializing the client context.
//
// Returns:
// - A pointer to a populated client.Context instance suitable for localnet usage.
func NewLocalnetClientCtx(t *testing.T, flagSet *pflag.FlagSet) *client.Context {
	homedir := DefaultHomeDir()
	clientCtx := client.Context{}.
		WithCodec(EncodingConfig.Marshaler).
		WithTxConfig(EncodingConfig.TxConfig).
		WithHomeDir(homedir).
		WithAccountRetriever(authtypes.AccountRetriever{}).
		WithInterfaceRegistry(EncodingConfig.InterfaceRegistry)

	clientCtx, err := client.ReadPersistentCommandFlags(clientCtx, flagSet)
	require.NoError(t, err)
	return &clientCtx
}

// NewLocalnetFlagSet creates a set of predefined flags suitable for a localnet
// testing environment.
//
// Parameters:
// - t: The testing.T instance used for the current test.
//
// Returns:
// - A flag set populated with flags tailored for localnet environments.
func NewLocalnetFlagSet(t *testing.T) *pflag.FlagSet {
	mockFlagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
	mockFlagSet.String(flags.FlagNode, "tcp://127.0.0.1:36657", "use localnet poktrolld node")
	mockFlagSet.String(flags.FlagHome, "", "use localnet poktrolld node")
	mockFlagSet.String(flags.FlagKeyringBackend, "test", "use test keyring")
	err := mockFlagSet.Parse([]string{})
	require.NoError(t, err)

	return mockFlagSet
}

// DefaultHomeDir computes the default home directory for the Pocket application
// by appending the application name to the current user's home directory.
//
// Returns:
// - A string representation of the default home directory path.
func DefaultHomeDir() string {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	return filepath.Join(userHomeDir, "."+app.Name)
}
