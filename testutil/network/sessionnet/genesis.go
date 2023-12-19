package sessionnet

import (
	"fmt"
	"reflect"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// TODO_IN_THIS_COMMIT: godoc comment
func GetGenesisState[T proto.Message](t *testing.T, moduleName string, net *inMemoryNetworkWithSessions) T {
	t.Helper()

	require.NotEmptyf(t, net.network, "in-memory network not started yet, call inMemoryNetworkWithSessions#Start() first")

	var genesisState T
	genesisStateType := reflect.TypeOf(genesisState)
	genesisStateValue := reflect.New(genesisStateType.Elem())
	genesisStatePtr := genesisStateValue.Interface().(proto.Message)

	genesisStateJSON := net.config.CosmosCfg.GenesisState[moduleName]
	err := net.config.CosmosCfg.Codec.UnmarshalJSON(genesisStateJSON, genesisStatePtr)
	//err := json.Unmarshal(genesisStateJSON, &genesisState)
	require.NoError(t, err)

	return genesisStatePtr.(T)
}

// TODO_IN_THIS_COMMIT: godoc comment ... each supplier is staked for a unique service.
//
// ConfigureDefaultSupplierModuleGenesisState generates a GenesisState object with a given number of suppliers,
// populates the genesis state for the supplier module with it, and then returns the included suppliers
// as pre-generated account objects.
func (memnet *inMemoryNetworkWithSessions) configureSupplierModuleGenesisState(t *testing.T) *suppliertypes.GenesisState {
	t.Helper()

	require.NotEmptyf(t, memnet.config.CosmosCfg, "memnet cosmos config not initialized, call inMemoryNetworkWithSessions#Start() first")
	require.NotEmptyf(t, memnet.preGeneratedAccounts, "memnet pre-generated accounts not initialized, call inMemoryNetworkWithSessions#Start() first")

	// Create a supplier for each session in numClaimsSessions.
	state := suppliertypes.DefaultGenesis()
	for i := 0; i < memnet.config.NumSuppliers; i++ {
		preGenerateAcct, ok := memnet.preGeneratedAccounts.Next()
		require.True(t, ok)

		supplier := sharedtypes.Supplier{
			Address: preGenerateAcct.Address.String(),
			Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(10000)},
			Services: []*sharedtypes.SupplierServiceConfig{
				{
					Service: &sharedtypes.Service{Id: fmt.Sprintf("svc%d", i)},
					Endpoints: []*sharedtypes.SupplierEndpoint{
						{
							Url:     "http://localhost:1",
							RpcType: sharedtypes.RPCType_JSON_RPC,
						},
					},
				},
			},
		}
		state.SupplierList = append(state.SupplierList, supplier)
	}

	supplierGenesisBuffer, err := memnet.config.CosmosCfg.Codec.MarshalJSON(state)
	require.NoError(t, err)

	// Add supplier module genesis state to the network config.
	memnet.config.CosmosCfg.GenesisState[suppliertypes.ModuleName] = supplierGenesisBuffer

	return state
}

// ConfigureDefaultApplicationModuleGenesisState generates a GenesisState object with a given number of applications.
// It returns the populated GenesisState object.
func (memnet *inMemoryNetworkWithSessions) configureAppModuleGenesisState(t *testing.T) *apptypes.GenesisState {
	t.Helper()

	require.NotEmptyf(t, memnet.preGeneratedAccounts, "inMemoryNetworkWithSessions#preGeneratedAccounts not initialized, call inMemoryNetworkWithSessions#Start() first")
	require.NotEmptyf(t, memnet.preGeneratedAccounts, "memnet pre-generated accounts not initialized, call inMemoryNetworkWithSessions#Start() first")

	state := apptypes.DefaultGenesis()
	for i := 0; i < memnet.config.NumApplications; i++ {
		preGeneratedAcct, ok := memnet.preGeneratedAccounts.Next()
		require.True(t, ok)

		application := apptypes.Application{
			Address: preGeneratedAcct.Address.String(),
			Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(10000)},
			ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
				{Service: &sharedtypes.Service{Id: fmt.Sprintf("svc%d", i)}},
				// NB: applications are staked for a service which no supplier is staked for.
				{Service: &sharedtypes.Service{Id: fmt.Sprintf("nosvc%d", i)}},
			},
		}
		state.ApplicationList = append(state.ApplicationList, application)
	}
	appGenesisState := state
	appGenesisBuffer, err := memnet.config.CosmosCfg.Codec.MarshalJSON(appGenesisState)
	require.NoError(t, err)

	// Add supplier and application module genesis state to the network config.
	memnet.config.CosmosCfg.GenesisState[apptypes.ModuleName] = appGenesisBuffer

	return state
}
