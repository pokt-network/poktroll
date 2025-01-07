package upgrades

import (
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/pokt-network/poktroll/app/keepers"
)

// TODO_MAINNET_DISCUSSION(@Olshansk): different networks should have the same gov module address, but might have different DAO addresses,
// unless we specifically write in these addresses in the genesis file.
// Should we use the same address/wallet for DAO or find a way to detect the network the upgrade is being applied to,
// to pick different addresses depending on the name of the network? (e.g chain-id)

const (
	// The default PNF/DAO address in the genesis file for Alpha TestNet. Used to create new authz authorizations.
	AlphaTestNetPnfAddress = "pokt1r6ja6rz6rpae58njfrsgs5n5sp3r36r2q9j04h"
	// Authority address. Defaults to gov module address. Used to create new authz authorizations.
	AlphaTestNetAuthorityAddress = "pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t"
)

// Upgrade represents a protocol upgrade in code.
// Once a `MsgSoftwareUpgrade` is submitted onchain, and `Upgrade.PlanName` matches the `Plan.Name`,
// the upgrade will be scheduled for execution at the corresponding height.
type Upgrade struct {
	// PlanName is a name an upgrade is matched to from the onchain `upgradetypes.Plan`.
	PlanName string

	// CreateUpgradeHandler returns an upgrade handler that will be executed at the time of the upgrade.
	// State changes and protocol version upgrades should be performed here.
	CreateUpgradeHandler func(*module.Manager, *keepers.Keepers, module.Configurator) upgradetypes.UpgradeHandler

	// StoreUpgrades adds, renames and deletes KVStores in the state to prepare for a protocol upgrade.
	StoreUpgrades storetypes.StoreUpgrades
}
