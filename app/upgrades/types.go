package upgrades

import (
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/pokt-network/poktroll/app/keepers"
)

const (
	// The default PNF/DAO address in the genesis file for Alpha TestNet. Used to create new authz authorizations.
	AlphaTestNetPnfAddress = "pokt1r6ja6rz6rpae58njfrsgs5n5sp3r36r2q9j04h"

	// This is the authority address used to create new authz authorizations. Defaults to x/gov module account address.
	// DEV_NOTE: This hard-coded address is kept for record-keeping historical purposes.
	// Use `keepers.UpgradeKeeper.Authority(ctx, &upgradetypes.QueryAuthorityRequest{})` to query the authority address of the current Alpha Network.
	AlphaTestNetAuthorityAddress = "pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t"

	// The default PNF/DAO address in the genesis file for Beta TestNet. Used to create new authz authorizations.
	BetaTestNetPnfAddress = "pokt1f0c9y7mahf2ya8tymy8g4rr75ezh3pkklu4c3e"

	// The PNF/DAO address in the genesis file for MainNet.
	// Will be used to create new authz authorizations in the future.
	MainNetPnfAddress = "pokt1hv3xrylxvwd7hfv03j50ql0ttp3s5hqqelegmv"

	// The Grove address in the genesis file for MainNet.
	// This is the current address that is used to create new authz authorizations for the time being.
	MainnetGroveAddress = "pokt18808wvw0h4t450t06uvauny8lvscsxjfyua7vh"
)

// NetworkAuthzGranteeAddress is a map of network names (i.e chain-id) to their respective PNF addresses.
var NetworkAuthzGranteeAddress = map[string]string{
	"pocket-alpha": AlphaTestNetPnfAddress,
	"pocket-beta":  BetaTestNetPnfAddress,
	// Currently grove address is the one being authorized to update mainnet parameters.
	// TODO_POST_MAINNET: This needs to be updated to the PNF address once it becomes the
	// entity that will be updating parameters on mainnet.
	"pocket": MainnetGroveAddress,
}

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
