package v2

import (
	"context"
	"fmt"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	appKeeper "github.com/pokt-network/poktroll/x/application/keeper"
	appTypes "github.com/pokt-network/poktroll/x/application/types"
)

func CreateUpgradeHandler(
	mm *module.Manager,
	appk appKeeper.Keeper,
	configurator module.Configurator,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {

		fmt.Println("Plan:")
		fmt.Println(plan.Name, plan.Height, plan.Info)

		newParams := appTypes.NewParams(69)
		err := appk.SetParams(ctx, newParams)
		// TODO_IN_THIS_PR: change a parameter somwhere
		//     application:
		// params:
		// max_delegated_gateways: "7"
		if err != nil {
			fmt.Println("Unable to set new app params")
			return vm, err
		}
		// 6:44PM INF applying upgrade "v2" at height: 50 module=x/upgrade
		// Plan:
		// v2 50 Software upgrade to version 2
		return mm.RunMigrations(ctx, configurator, vm)
	}
}
