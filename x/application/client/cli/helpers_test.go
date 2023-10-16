package cli_test

import (
	"pocket/cmd/pocketd/cmd"
	"pocket/testutil/network"
	"pocket/testutil/nullify"
	"pocket/testutil/sample"
	"pocket/x/application/types"
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

var _ = strconv.IntSize

func init() {
	cmd.InitSDKConfig()
}

func networkWithApplicationObjects(t *testing.T, n int) (*network.Network, []types.Application) {
	t.Helper()
	cfg := network.DefaultConfig()
	state := applicationModuleGenesis(t, n)
	buf, err := cfg.Codec.MarshalJSON(state)
	require.NoError(t, err)
	cfg.GenesisState[types.ModuleName] = buf
	return network.New(t, cfg), state.ApplicationList
}

func applicationModuleGenesis(t *testing.T, n int) *types.GenesisState {
	t.Helper()
	state := types.DefaultGenesis()
	for i := 0; i < n; i++ {
		stake := sdk.NewCoin("upokt", sdk.NewInt(int64(i)))
		application := types.Application{
			Address: sample.AccAddress(),
			Stake:   &stake,
		}
		nullify.Fill(&application)
		state.ApplicationList = append(state.ApplicationList, application)
	}
	return state
}
