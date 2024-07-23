package upgrades

import (
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	appKeeper "github.com/pokt-network/poktroll/x/application/keeper"
)

type Upgrade struct {
	VersionName          string
	CreateUpgradeHandler func(*module.Manager, appKeeper.Keeper, module.Configurator) upgradetypes.UpgradeHandler
	StoreUpgrades        storetypes.StoreUpgrades
}
