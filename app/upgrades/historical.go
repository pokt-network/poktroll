package upgrades

// This file is intended to keep old, historical upgrades in one place. It is advised to keep the future upgrades in the
// separate file, and then move them to `historical.go` after a successful upgrade so the new nodes can still sync from
// the genesis.

// TODO_CONSIDERATION: after we verify `State Sync` is fully functional, we can hypothetically remove old upgrades from
// the codebase, as the nodes won't have to execute upgrades and will download the "snapshot" instead. Some other
// blockchain networks (such as `evmos`), remove the old upgrades from the codebase.

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	"github.com/pokt-network/poktroll/app/keepers"
)

// defaultMigrationsOnlyUpgradeHandler creates an update handler that only performs module's `ConsensusVersion`
// change in blockchain state. Useful for performing upgrades that do no require additional state modifications, such as
// parameter changes, data migrations, authz authorizations, etc. If **any** of these are needed, a new, version-specific,
// upgrade handler should be created.
func defaultMigrationsOnlyUpgradeHandler(
	mm *module.Manager,
	_ *keepers.Keepers,
	configurator module.Configurator) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		return mm.RunMigrations(ctx, configurator, vm)
	}
}

// An example of an upgrade that performs additional state changes.
// Even when not changing `ConsensusVersion` of any modules, it still might be beneficial to create an upgrade
// to signal to node runners utilizing `Cosmovisor` to automatically download and install the new binary.
var Upgrade_Example = Upgrade{
	PlanName:             "v0.0.0-Example",
	CreateUpgradeHandler: defaultMigrationsOnlyUpgradeHandler,

	// We can also add, rename and delete KVStores.
	StoreUpgrades: storetypes.StoreUpgrades{},
}

// Upgrade_0_0_4 is an example of an upgrade that increases the block size.
// This example demonstrates how to change the block size using an upgrade.
var Upgrade_0_0_4 = Upgrade{
	PlanName: "v0.0.4",
	CreateUpgradeHandler: func(mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator) upgradetypes.UpgradeHandler {
		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			// Get current consensus module parameters
			currentParams, err := keepers.ConsensusParamsKeeper.ParamsStore.Get(ctx)
			if err != nil {
				return vm, err
			}

			// Supply all params even when changing just one, as `ToProtoConsensusParams` requires them to be present.
			newParams := consensusparamtypes.MsgUpdateParams{
				Authority: keepers.ConsensusParamsKeeper.GetAuthority(),
				Block:     currentParams.Block,
				Evidence:  currentParams.Evidence,
				Validator: currentParams.Validator,

				// This seems to be deprecated/not needed, but it's fine as we're copying the existing data.
				Abci: currentParams.Abci,
			}

			// Increase block size two-fold, 22020096 is the default value.
			newParams.Block.MaxBytes = 22020096 * 2

			// Update the chain state
			_, err = keepers.ConsensusParamsKeeper.UpdateParams(ctx, &newParams)
			if err != nil {
				return vm, err
			}

			return mm.RunMigrations(ctx, configurator, vm)
		}
	},
	// No changes to the KVStore in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},
}
