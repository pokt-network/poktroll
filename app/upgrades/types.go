package upgrades

import (
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/pokt-network/poktroll/app/keepers"
)

// Upgrade represents a protocol upgrade in code. Once a `MsgSoftwareUpgrade` is submitted to the chain, and
// `VersionName` matches the `Name` of the `Plan` inside the upgrade message, the upgrade will be scheduled for execution.
type Upgrade struct {
	VersionName          string
	CreateUpgradeHandler func(*module.Manager, *keepers.Keepers, module.Configurator) upgradetypes.UpgradeHandler
	StoreUpgrades        storetypes.StoreUpgrades
}
