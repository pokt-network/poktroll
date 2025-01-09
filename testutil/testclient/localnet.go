package testclient

import (
	"fmt"
	"os"

	"cosmossdk.io/depinject"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/regen-network/gocuke"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app"
	"github.com/pokt-network/poktroll/cmd/poktrolld/cmd"
)

var (
	// CometLocalTCPURL provides a default URL pointing to the localnet TCP endpoint.
	CometLocalTCPURL = "tcp://localhost:26657"

	// CometLocalWebsocketURL provides a default URL pointing to the localnet websocket endpoint.
	CometLocalWebsocketURL = "ws://localhost:26657/websocket"

	// TxConfig provided by app.AppConfig(), intended as a convenience for use in tests.
	TxConfig client.TxConfig
	// Marshaler provided by app.AppConfig(), intended as a convenience for use in tests.
	Marshaler codec.Codec
	// InterfaceRegistry provided by app.AppConfig(), intended as a convenience for use in tests.
	InterfaceRegistry codectypes.InterfaceRegistry
)

// init initializes the SDK configuration upon package import.
func init() {
	cmd.InitSDKConfig()

	deps := depinject.Configs(
		app.AppConfig(),
		depinject.Supply(
			log.NewLogger(os.Stderr),
		),
	)

	// Ensure that the global variables are initialized.
	if err := depinject.Inject(
		deps,
		&TxConfig,
		&Marshaler,
		&InterfaceRegistry,
	); err != nil {
		panic(err)
	}

	// If VALIDATOR_RPC_ENDPOINT environment variable is set, use it to override the default localnet endpoint.
	if endpoint := os.Getenv("VALIDATOR_RPC_ENDPOINT"); endpoint != "" {
		CometLocalTCPURL = fmt.Sprintf("tcp://%s", endpoint)
		CometLocalWebsocketURL = fmt.Sprintf("ws://%s/websocket", endpoint)
	}
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
func NewLocalnetClientCtx(t gocuke.TestingT, flagSet *pflag.FlagSet) *client.Context {
	t.Helper()

	homedir := app.DefaultNodeHome
	clientCtx := client.Context{}.
		WithCodec(Marshaler).
		WithTxConfig(TxConfig).
		WithHomeDir(homedir).
		WithAccountRetriever(authtypes.AccountRetriever{}).
		WithInterfaceRegistry(InterfaceRegistry)

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
func NewLocalnetFlagSet(t gocuke.TestingT) *pflag.FlagSet {
	t.Helper()

	return NewFlagSet(t, CometLocalTCPURL)
}

// TODO_IN_THIS_COMMIT: godoc...
func NewFlagSet(t gocuke.TestingT, cometTCPURL string) *pflag.FlagSet {
	t.Helper()

	mockFlagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
	// TODO_IMPROVE: It would be nice if the value could be set correctly based
	// on whether the test using it is running in tilt or not.
	mockFlagSet.Bool(flags.FlagGRPCInsecure, true, "use insecure grpc connection")
	mockFlagSet.String(flags.FlagGRPC, cometTCPURL, "use localnet poktrolld node")
	//mockFlagSet.String(flags.FlagNode, cometTCPURL, "use localnet poktrolld node")
	mockFlagSet.String(flags.FlagHome, "", "use localnet poktrolld node")
	mockFlagSet.String(flags.FlagKeyringBackend, "test", "use test keyring")
	mockFlagSet.String(flags.FlagChainID, app.Name, "use poktroll chain-id")
	err := mockFlagSet.Parse([]string{})
	require.NoError(t, err)

	return mockFlagSet
}
