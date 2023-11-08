package cli_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/cmd/pocketd/cmd"
	"github.com/pokt-network/poktroll/testutil/network"
	"github.com/pokt-network/poktroll/x/gateway/types"
)

// Dummy variable to avoid unused import error.
var _ = strconv.IntSize

// init initializes the SDK configuration.
func init() {
	cmd.InitSDKConfig()
}

// networkWithGatewayObjects creates a network with a populated gateway state of n gateway objects
func networkWithGatewayObjects(t *testing.T, n int) (*network.Network, []types.Gateway) {
	t.Helper()
	cfg := network.DefaultConfig()
	gatewayGenesisState := network.DefaultGatewayModuleGenesisState(t, n)
	buf, err := cfg.Codec.MarshalJSON(gatewayGenesisState)
	require.NoError(t, err)
	cfg.GenesisState[types.ModuleName] = buf
	return network.New(t, cfg), gatewayGenesisState.GatewayList
}
