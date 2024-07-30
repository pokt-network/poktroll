package app

import (
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/pokt-network/poktroll/app/upgrades"
)

// allUpgrades includes all upgrades that have been created, but not necessarily submitted on-chain
var allUpgrades = []upgrades.Upgrade{
	upgrades.Upgrade_0_0_4,
}

// setUpgrades sets upgrade handlers for all upgrades and executes KVStore migration if upgrade plan file exists
func (app *App) setUpgrades() error {
	// Set upgrade handlers for all upgrades
	for _, u := range allUpgrades {
		app.Keepers.UpgradeKeeper.SetUpgradeHandler(
			u.PlanName,
			u.CreateUpgradeHandler(app.ModuleManager, &app.Keepers, app.Configurator()),
		)
	}

	// Reads the upgrade info from disk (was put there by the old binary using on-chain upgrade `Plan`).
	upgradePlan, err := app.Keepers.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		return err
	}

	// Find the planned upgrade by name. If not found, assume there's nothing to upgrade, as `ReadUpgradeInfoFromDisk()`
	// would have returned an error if the file was corrupted or there's OS permissions issue.
	plannedUpgrade, found := findPlannedUpgrade(upgradePlan.Name)
	if !found {
		return nil
	}

	// Allows to skip the store upgrade if `--unsafe-skip-upgrades` is provided and the height matches.
	shouldSkipStoreUpgrade := app.Keepers.UpgradeKeeper.IsSkipHeight(upgradePlan.Height)

	if !shouldSkipStoreUpgrade {
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradePlan.Height, &plannedUpgrade.StoreUpgrades))
	}

	return nil
}

// findPlannedUpgrade returns the planned upgrade by name.
func findPlannedUpgrade(upgradePlanName string) (upgrades.Upgrade, bool) {
	for _, u := range allUpgrades {
		if u.PlanName == upgradePlanName {
			return u, true
		}
	}
	return upgrades.Upgrade{}, false
}
