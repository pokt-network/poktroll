package app

import (
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/pokt-network/poktroll/app/upgrades"
)

// allUpgrades includes all upgrades that have been created (in the code)
var allUpgrades = []upgrades.Upgrade{
	upgrades.Upgrade_0_4_0,
}

func (app *App) setUpgrades() error {
	// Set upgrade handlers for all upgrades
	for _, u := range allUpgrades {
		app.UpgradeKeeper.SetUpgradeHandler(
			u.VersionName,
			u.CreateUpgradeHandler(app.ModuleManager, app.ApplicationKeeper, app.Configurator()),
		)
	}

	// Reads the upgrade info from disk (was put there by the old binary using on-chain upgrade `Plan`).
	upgradePlan, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		return err
	}

	// Find the planned upgrade by name. Assume there's nothing to upgrade, as `ReadUpgradeInfoFromDisk()` would have
	// returned an error if the file was missing/corrupted.
	planedUpgrade, found := findPlannedUpgrade(upgradePlan.Name)
	if !found {
		return nil
	}

	// Allows to skip the store upgrade if `--unsafe-skip-upgrades` is provided and the height matches.
	shouldSkipStoreUpgrade := app.UpgradeKeeper.IsSkipHeight(upgradePlan.Height)

	if !shouldSkipStoreUpgrade {
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradePlan.Height, &planedUpgrade.StoreUpgrades))
	}

	return nil
}

// findPlannedUpgrade returns the planned upgrade by name.
func findPlannedUpgrade(upgradePlanName string) (upgrades.Upgrade, bool) {
	for _, u := range allUpgrades {
		if u.VersionName == upgradePlanName {
			return u, true
		}
	}
	return upgrades.Upgrade{}, false
}
