package gateway_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/network"
	"github.com/pokt-network/poktroll/x/gateway/types"
)

// Dummy variable to avoid unused import error.
var _ = strconv.IntSize

// networkWithGatewayObjects creates a network with a populated gateway state of n gateway objects
func networkWithGatewayObjects(t *testing.T, n int) (*network.Network, []types.Gateway) {
	t.Helper()

	// Configure the testing network
	cfg := network.DefaultConfig()
	gatewayGenesisState := network.DefaultGatewayModuleGenesisState(t, n)
	buf, err := cfg.Codec.MarshalJSON(gatewayGenesisState)
	require.NoError(t, err)
	cfg.GenesisState[types.ModuleName] = buf

	// Start the network
	net := network.New(t, cfg)

	// Wait for the network to be fully initialized to avoid race conditions
	// with consensus reactor goroutines
	require.NoError(t, net.WaitForNextBlock())

	// Additional wait to ensure all consensus components are fully initialized
	require.NoError(t, net.WaitForNextBlock())

	return net, gatewayGenesisState.GatewayList
}
