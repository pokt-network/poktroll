package network

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/testutil/sample"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

// GatewayModuleGenesisStateWithAddresses generates a GenesisState object with
// a gateway list full of gateways with the given addresses.
func GatewayModuleGenesisStateWithAddresses(t *testing.T, addresses []string) *gatewaytypes.GenesisState {
	t.Helper()

	state := gatewaytypes.DefaultGenesis()
	for _, addr := range addresses {
		gateway := gatewaytypes.Gateway{
			Address: addr,
			Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(10000)},
		}
		state.GatewayList = append(state.GatewayList, gateway)
	}
	return state
}

// DefaultGatewayModuleGenesisState generates a GenesisState object with a given number of gateways.
// It returns the populated GenesisState object.
func DefaultGatewayModuleGenesisState(t *testing.T, n int) *gatewaytypes.GenesisState {
	t.Helper()

	state := gatewaytypes.DefaultGenesis()
	for i := 0; i < n; i++ {
		stake := sdk.NewCoin("upokt", sdk.NewInt(int64(i)))
		gateway := gatewaytypes.Gateway{
			Address: sample.AccAddress(),
			Stake:   &stake,
		}
		// TODO_CONSIDERATION: Evaluate whether we need `nullify.Fill` or if we should enforce `(gogoproto.nullable) = false` everywhere
		// nullify.Fill(&gateway)
		state.GatewayList = append(state.GatewayList, gateway)
	}
	return state
}

// TODO_IN_THIS_COMMIT: still need this?
//
// DefaultSupplierModuleGenesisState generates a GenesisState object with a given number of suppliers.
// It returns the populated GenesisState object.
func DefaultSupplierModuleGenesisState(t *testing.T, n int) *types.GenesisState {
	t.Helper()

	state := types.DefaultGenesis()
	for i := 0; i < n; i++ {
		stake := sdk.NewCoin("upokt", sdk.NewInt(int64(i)))
		supplier := sharedtypes.Supplier{
			Address: sample.AccAddress(),
			Stake:   &stake,
			Services: []*sharedtypes.SupplierServiceConfig{
				{
					Service: &sharedtypes.Service{Id: fmt.Sprintf("svc%d", i)},
					Endpoints: []*sharedtypes.SupplierEndpoint{
						{
							Url:     fmt.Sprintf("http://localhost:%d", i),
							RpcType: sharedtypes.RPCType_JSON_RPC,
						},
					},
				},
			},
		}
		// TODO_CONSIDERATION: Evaluate whether we need `nullify.Fill` or if we should enforce `(gogoproto.nullable) = false` everywhere
		// nullify.Fill(&supplier)
		state.SupplierList = append(state.SupplierList, supplier)
	}
	return state
}
