package upgrades

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	appKeeper "github.com/pokt-network/poktroll/x/application/keeper"
)

// defaultMigrationsOnlyUpgradeHandler creates an update handler that only performs module's `ConsensusVersion`
// change in blockchain state. Useful for performing upgrades that do no require additional state modifications, such as
// parameter changes, data migrations, authz authorizations, etc. If **any** of these are needed, a new, version-specific,
// upgrade handler should be created.
func defaultMigrationsOnlyUpgradeHandler(
	mm *module.Manager,
	_ appKeeper.Keeper,
	configurator module.Configurator) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		return mm.RunMigrations(ctx, configurator, vm)
	}
}

// An example of an upgrade that performs additional state changes.
// Even when not changing `ConsensusVersion` of any modules, it still might be beneficial to create an upgrade
// to signal to node runners utilizing `Cosmovisor` to automatically download and install the new binary.
// TODO_IN_THIS_PR: link to the document that explains Cosmovisor usage and its benefits for node runners.
var Upgrade_0_4_0 = Upgrade{
	VersionName:          "v0.4.0",
	CreateUpgradeHandler: defaultMigrationsOnlyUpgradeHandler,
	StoreUpgrades:        storetypes.StoreUpgrades{},
}
