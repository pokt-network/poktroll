package upgrades

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"

	"github.com/pokt-network/poktroll/app/keepers"
)

// Upgrade_0_0_4 is an example of an upgrade that increases the block size.
// This example demonstrates how to change the block size using an upgrade.
var Upgrade_0_0_4 = Upgrade{
	PlanName: "v0.0.4",
	CreateUpgradeHandler: func(mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
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
			if _, err = keepers.ConsensusParamsKeeper.UpdateParams(ctx, &newParams); err != nil {
				return vm, err
			}

			return mm.RunMigrations(ctx, configurator, vm)
		}
	},
	// No changes to the KVStore in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},
}
