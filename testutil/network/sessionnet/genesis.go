// TODO_IMPROVE: draw a mermaid diagram of the app/supplier/service session network.

package sessionnet

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// configureSupplierModuleGenesisState generates and populates the in-memory
// network's application module's GenesisState object with the number of suppliers
// specified by the InMemoryConfig, each of which is staked for a unique service.
// It returns the genesis state object.
func (memnet *inMemoryNetworkWithSessions) configureSupplierModuleGenesisState(t *testing.T) *suppliertypes.GenesisState {
	t.Helper()

	require.NotEmptyf(t, memnet.GetCosmosNetworkConfig(t), "cosmos config not initialized, call #Start() first")
	require.NotEmptyf(t, memnet.PreGeneratedAccountIterator, "pre-generated accounts not initialized, call #Start() first")

	// Create NumSuppliers number of suppliers with a service "svcX", where "X" is the supplier's index.
	var supplierGenesisState = suppliertypes.DefaultGenesis()
	for i := 0; i < memnet.Config.NumSuppliers; i++ {
		preGenerateAcct, ok := memnet.PreGeneratedAccountIterator.Next()
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
		supplierGenesisState.SupplierList = append(supplierGenesisState.SupplierList, supplier)
	}

	supplierGenesisBuffer, err := memnet.GetCosmosNetworkConfig(t).Codec.MarshalJSON(supplierGenesisState)
	require.NoError(t, err)

	// Add supplier module genesis supplierGenesisState to the network config.
	memnet.GetCosmosNetworkConfig(t).GenesisState[suppliertypes.ModuleName] = supplierGenesisBuffer

	return supplierGenesisState
}

// configureApplicationModuleGenesisState generates and populates the in-memory network's
// applicaion module's GenesisState object with a given number of applications,
// each of which is staked for a service such that
// memnet.Config.AppSupplierPairingRatio*NumSuppliers number of applications are staked
// for each genesis supplier's service (assumes that each supplier is staked for a unique
// service with no overlap).
func (memnet *inMemoryNetworkWithSessions) configureAppModuleGenesisState(t *testing.T) *apptypes.GenesisState {
	t.Helper()

	require.NotEmptyf(t, memnet.GetCosmosNetworkConfig(t), "cosmos config not initialized, call #Start() first")
	require.NotEmptyf(t, memnet.PreGeneratedAccountIterator, "pre-generated accounts not initialized, call #Start() first")

	var (
		serviceIdx      = 0
		appGenesisState = apptypes.DefaultGenesis()
	)
	for i := 0; i < memnet.Config.GetNumApplications(t); i++ {
		preGeneratedAcct, ok := memnet.PreGeneratedAccountIterator.Next()
		require.True(t, ok)

		application := apptypes.Application{
			Address: preGeneratedAcct.Address.String(),
			Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(10000)},
			ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
				{Service: &sharedtypes.Service{Id: fmt.Sprintf("svc%d", serviceIdx)}},
				// NB: applications are staked for a service which no supplier is staked for.
				{Service: &sharedtypes.Service{Id: fmt.Sprintf("nosvc%d", serviceIdx)}},
			},
		}
		appGenesisState.ApplicationList = append(appGenesisState.ApplicationList, application)

		// NB: only increment serviceIdx every AppSupplierPairingRatio applications
		// to ensure that AppSupplierPairingRatio*NumSuppliers number of applications
		// are staked for each supplier's service (ea. supplier is currently staked
		// for a unique service with no overlap).
		if (i+1)%memnet.Config.AppSupplierPairingRatio == 0 {
			serviceIdx++
		}
	}
	appGenesisBuffer, err := memnet.Config.CosmosCfg.Codec.MarshalJSON(appGenesisState)
	require.NoError(t, err)

	// Add supplier and application module genesis appGenesisState to the network memnetConfig.
	memnet.GetCosmosNetworkConfig(t).GenesisState[apptypes.ModuleName] = appGenesisBuffer

	return appGenesisState
}
