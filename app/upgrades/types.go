package upgrades

import (
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/pokt-network/poktroll/app/keepers"
)

// Visit the pocket-network-genesis repo for the source of truth for these addresses.
// https://github.com/pokt-network/pocket-network-genesis/tree/master/shannon
const (
	// Useful for local testing & development
	LocalNetPnfAddress = "pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw"

	// The default PNF/DAO address in the genesis file for Alpha TestNet. Used to create new authz authorizations.
	AlphaTestNetPnfAddress = "pokt1r6ja6rz6rpae58njfrsgs5n5sp3r36r2q9j04h"

	// This is the authority address used to create new authz authorizations. Defaults to x/gov module account address.
	// DEV_NOTE: This hard-coded address is kept for record-keeping historical purposes.
	// Use `keepers.<target_module>.GetAuthority()` to get the authority address for the module.
	AlphaTestNetAuthorityAddress = "pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t"

	// The default PNF/DAO address in the genesis file for Beta TestNet. Used to create new authz authorizations.
	BetaTestNetPnfAddress = "pokt1f0c9y7mahf2ya8tymy8g4rr75ezh3pkklu4c3e"

	// PNF/DAO address specified in MainNet genesis.
	// Used to create new authz authorizations after migration is complete.
	MainNetPnfAddress = "pokt1hv3xrylxvwd7hfv03j50ql0ttp3s5hqqelegmv"

	// Grove address specified in MainNet genesis.
	// Used to create new authz authorizations throughout the migration process.
	MainnetGroveAddress = "pokt18808wvw0h4t450t06uvauny8lvscsxjfyua7vh"
)

// NetworkAuthzGranteeAddress is a map of network names (i.e chain-id) to their
// respective authorization (i.e. PNF/DAO) addresses.
var NetworkAuthzGranteeAddress = map[string]string{
	"pocket-alpha": AlphaTestNetPnfAddress,
	"pocket-beta":  BetaTestNetPnfAddress,

	// Grove's address is used as of #1191 to authorize updates to mainnet parameters.
	// TODO_POST_MAINNET: Update to PNF address once the migration is complete.
	"pocket": MainnetGroveAddress,

	// TODO_TECHDEBT: We currently use "pocket" for the local network environment,
	// which interferes with the mainnet address. Streamline using `pocket-local` for the localnet
	// and uncomment the line below in the meantime during testing.
	// "pocket": LocalNetPnfAddress,
	// "pocket-local": LocalNetPnfAddress,
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

// An example of an upgrade that uses the default upgrade handler and also performs additional state changes.
// For example, even if `ConsensusVersion` is not modified for any modules, it still might be beneficial to create
// an upgrade so node runners are signaled to start utilizing `Cosmovisor` for new binaries.
var UpgradeExample = Upgrade{
	// PlanName can be any string.
	// This code is executed when the upgrade with this plan name is submitted to the network.
	// This does not necessarily need to be a version, but it's usually the case with consensus-breaking changes.
	PlanName:             "v0.0.0-Example",
	CreateUpgradeHandler: defaultUpgradeHandler,

	// We can also add, rename and delete KVStores.
	// More info in cosmos-sdk docs: https://docs.cosmos.network/v0.50/learn/advanced/upgrade#add-storeupgrades-for-new-modules
	StoreUpgrades: storetypes.StoreUpgrades{
		// Added: []string{"wowsuchrelay"},
	},
}
