package gateway

import (
	"context"

	"github.com/pokt-network/poktroll/proto/types/gateway"
	"github.com/pokt-network/poktroll/x/gateway/keeper"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx context.Context, k keeper.Keeper, genState gateway.GenesisState) {
	// Set all the gateway
	for _, gateway := range genState.GatewayList {
		k.SetGateway(ctx, gateway)
	}
	// this line is used by starport scaffolding # genesis/module/init
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(err)
	}
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx context.Context, k keeper.Keeper) *gateway.GenesisState {
	genesis := gateway.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	genesis.GatewayList = k.GetAllGateways(ctx)
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
