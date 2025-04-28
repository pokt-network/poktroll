package upgrades

import (
	"context"
	"fmt"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/pokt-network/poktroll/app/keepers"
)

const (
	Upgrade_0_1_8_PlanName = "v0.1.8"
)

// Upgrade_0_1_8 handles the upgrade to release `v0.1.8`.
// This is planned to be issued on both Pocket Network's Shannon Alpha, Beta TestNets
// It is an upgrade intended to enable application indexing.
// TODO_FOLLOWUP: Update the github link from main to v0.1.8 once the upgrade is released.
// https://github.com/pokt-network/poktroll/compare/v0.1.6..main
var Upgrade_0_1_8 = Upgrade{
	PlanName: Upgrade_0_1_8_PlanName,
	// No migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},

	// Upgrade Handler
	CreateUpgradeHandler: func(
		mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			logger := cosmostypes.UnwrapSDKContext(ctx).Logger().With("upgrade_plan_name", Upgrade_0_1_8_PlanName)
			logger.Info("Starting upgrade handler")

			logger.Info("Indexing applications")
			if err := indexApplications(ctx, keepers, logger); err != nil {
				return vm, err
			}

			return vm, nil
		}
	},
}

// indexApplications triggers application indexing.
// It iterates over all applications in the store and re-stores them to trigger the indexer.
func indexApplications(ctx context.Context, keepers *keepers.Keepers, logger log.Logger) error {
	// Get all deprecated suppliers from the store.
	applicationsIterator := keepers.ApplicationKeeper.GetAllApplicationsIterator(ctx)
	defer applicationsIterator.Close()

	for ; applicationsIterator.Valid(); applicationsIterator.Next() {
		application, err := applicationsIterator.Value()
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to get application with key %s from iterator: %v", string(applicationsIterator.Key()), err))
			return err
		}

		// Re-store the application to trigger the indexer.
		keepers.ApplicationKeeper.SetApplication(ctx, application)
	}

	return nil
}
