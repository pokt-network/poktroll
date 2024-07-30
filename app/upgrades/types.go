package upgrades

import (
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/pokt-network/poktroll/app/keepers"
)

// Upgrade represents a protocol upgrade in code. Once a `MsgSoftwareUpgrade` is submitted to the chain, and
// `PlanName` matches the `Name` of the `Plan` inside the upgrade message, the upgrade will be scheduled for execution.
type Upgrade struct {
	// PlanName is a name an upgrade is matched to from the on-chain `upgradetypes.Plan`.
	PlanName string

	// CreateUpgradeHandler returns an upgrade handler that will be executed at the time of the upgrade.
	// State changes and protocol version upgrades should be performed here.
	CreateUpgradeHandler func(*module.Manager, *keepers.Keepers, module.Configurator) upgradetypes.UpgradeHandler

	// StoreUpgrades adds, renames and deletes KVStores in the state to prepare for a protocol upgrade.
	StoreUpgrades storetypes.StoreUpgrades
}
