package network

import (
	"reflect"
	"testing"

	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"
)

// GetGenesisState retrieves the genesis state for a given module from the underlying cosmos-sdk in-memory network.
func GetGenesisState[T proto.Message](t *testing.T, moduleName string, memnet InMemoryCosmosNetwork) T {
	t.Helper()

	// Ensure the in-memory network has been started.
	_ = memnet.GetNetwork(t)

	var genesisState T
	// NB: As this function is generic, it MUST use reflect in order to unmarshal
	// the genesis state as the codec requries a reference to a concrete type pointer.
	genesisStateType := reflect.TypeOf(genesisState)
	genesisStateValue := reflect.New(genesisStateType.Elem())
	genesisStatePtr := genesisStateValue.Interface().(proto.Message)

	genesisStateJSON := memnet.GetNetworkConfig(t).GenesisState[moduleName]
	err := memnet.GetNetworkConfig(t).Codec.UnmarshalJSON(genesisStateJSON, genesisStatePtr)
	require.NoError(t, err)

	return genesisStatePtr.(T)
}
