package upgrades

import (
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/pokt-network/poktroll/app/keepers"
)

type Upgrade struct {
	VersionName          string
	CreateUpgradeHandler func(*module.Manager, *keepers.Keepers, module.Configurator) upgradetypes.UpgradeHandler
	StoreUpgrades        storetypes.StoreUpgrades
}
