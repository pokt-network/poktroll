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

const CometLocalWebsocketURL = "ws://localhost:36657/websocket"

var (
	EncodingConfig = app.MakeEncodingConfig()
)

func init() {
	cmd.InitSDKConfig()
}

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

func NewLocalnetFlagSet(t *testing.T) *pflag.FlagSet {
	mockFlagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
	mockFlagSet.String(flags.FlagNode, "tcp://127.0.0.1:36657", "use localnet poktrolld node")
	mockFlagSet.String(flags.FlagHome, "", "use localnet poktrolld node")
	mockFlagSet.String(flags.FlagKeyringBackend, "test", "use test keyring")
	err := mockFlagSet.Parse([]string{})
	require.NoError(t, err)

	return mockFlagSet
}

func DefaultHomeDir() string {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	return filepath.Join(userHomeDir, "."+app.Name)
}
