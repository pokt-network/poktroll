package app

import (
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/pokt-network/poktroll/app/upgrades"
)

// allUpgrades includes all upgrades that have upgrade strategy implemented. Only after a new upgrade is added here,
// we can create a new release (https://github.com/pokt-network/poktroll/releases) and schedule an upgrade on chain.
var allUpgrades = []upgrades.Upgrade{
	upgrades.Upgrade_0_0_4,
}

// setUpgrades sets upgrade handlers for all upgrades and executes KVStore migration if an upgrade plan file exists.
// Upgrade plans are submitted on chain, and the full-node/validator creates the upgrade plan file for cosmovisor.
func (app *App) setUpgrades() error {
	// Set upgrade handlers for all upgrades
	for _, upgrade := range allUpgrades {
		app.Keepers.UpgradeKeeper.SetUpgradeHandler(
			upgrade.PlanName,
			upgrade.CreateUpgradeHandler(app.ModuleManager, &app.Keepers, app.Configurator()),
		)
	}

	// Reads the upgrade info from disk.
	// The previous binary is expected to have read the plan from on-chain and saved it locally.
	upgradePlan, err := app.Keepers.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		return err
	}

	// Find the planned upgrade by name.
	// If nothing is found, assume there's nothing to upgrade since `ReadUpgradeInfoFromDisk()`
	// would have returned an error if the file was corrupted or there was OS permissions issue.
	plannedUpgrade, found := findPlannedUpgrade(upgradePlan.Name, allUpgrades)
	if !found {
		return nil
	}

	// Allows to skip the store upgrade if `--unsafe-skip-upgrades` is provided and the height matches.
	// Makes it possible for social consensus to overrule the upgrade in case it has a bug.
	// NOTE: if 2/3 of the consensus has this configured for upgrade heights, the chain will continue climbing
	// without performing upgrades.
	shouldSkipStoreUpgrade := app.Keepers.UpgradeKeeper.IsSkipHeight(upgradePlan.Height)
	if !shouldSkipStoreUpgrade {
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradePlan.Height, &plannedUpgrade.StoreUpgrades))
	}

	return nil
}

// findPlannedUpgrade returns the planned upgrade by name.
func findPlannedUpgrade(upgradePlanName string, upgrades []upgrades.Upgrade) (*upgrades.Upgrade, bool) {
	for _, upgrade := range upgrades {
		if upgrade.PlanName == upgradePlanName {
			return &upgrade, true
		}
	}
	return nil, false
}
