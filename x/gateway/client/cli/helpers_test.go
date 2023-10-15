package cli_test

import (
	"strconv"
	"testing"

	"pocket/testutil/network"
	"pocket/testutil/nullify"
	"pocket/x/gateway/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/stretchr/testify/require"
)

// networkWithGatewayObjects creates a network with a populated gateway state of n gateway objects
func networkWithGatewayObjects(t *testing.T, n int) (*network.Network, []types.Gateway) {
	t.Helper()
	cfg := network.DefaultConfig()
	state := gatewayModuleGenesis(t, n)
	buf, err := cfg.Codec.MarshalJSON(state)
	require.NoError(t, err)
	cfg.GenesisState[types.ModuleName] = buf
	return network.New(t, cfg), state.GatewayList
}

// gatewayModuleGenesis generates a default genesis state for the gateway module and then
// populates it with n gateway objects
func gatewayModuleGenesis(t *testing.T, n int) *types.GenesisState {
	t.Helper()
	state := types.DefaultGenesis()
	for i := 0; i < n; i++ {
		stake := sdk.NewCoin("upokt", sdk.NewInt(int64(i)))
		gateway := types.Gateway{
			Address: strconv.Itoa(i),
			Stake:   &stake,
		}
		nullify.Fill(&gateway)
		state.GatewayList = append(state.GatewayList, gateway)
	}
	return state
}
