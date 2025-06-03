package app

import (
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/pokt-network/poktroll/app/upgrades"
)

// allUpgrades includes all upgrades that have upgrade strategy implemented.
// A new upgrade MUST be added BEFORE a new release is created; https://github.com/pokt-network/poktroll/releases).
// The chain upgrade can be scheduled AFTER the new version (with upgrade strategy implemented) is released,
// so `cosmovisor` can automatically pull the binary from GitHub.
var allUpgrades = []upgrades.Upgrade{
	// v0.0.4 was the first upgrade we implemented and tested on network that is no longer used.
	// upgrades.Upgrade_0_0_4,

	// v0.0.10 was the first upgrade we implemented on Alpha TestNet.
	// upgrades.Upgrade_0_0_10,

	// v0.0.11 was the Alpha TestNet exclusive upgrade to bring it on par with Beta TestNet.
	// upgrades.Upgrade_0_0_11,

	// v0.0.12 - the first upgrade going live on both Alpha and Beta TestNets.
	// upgrades.Upgrade_0_0_12,

	// v0.0.13 - this upgrade introduces morse migration module and websocket service handling.
	// upgrades.Upgrade_0_0_13,

	// v0.0.14 - upgrade to release latest features on TestNets to perform more load testing prior to MainNet launch.
	// upgrades.Upgrade_0_0_14,

	// v0.1.2 - upgrade to release morse migration capabilities
	// upgrades.Upgrade_0_1_2,

	// v0.1.3 - upgrade to reduce network and memory footprint of session suppliers
	// upgrades.Upgrade_0_1_3,

	// v0.1.4 - upgrade to reduce network and memory footprint of session suppliers
	// upgrades.Upgrade_0_1_4,

	// v0.1.5 - upgrade to reduce memory footprint when iterating over Suppliers and Applications.
	// upgrades.Upgrade_0_1_5,

	// v0.1.6 - upgrade to reduce the memory by avoiding unnecessary marshaling of the supplier object when iterating over the suppliers.
	// upgrades.Upgrade_0_1_6,

	// v0.1.7 - upgrade to mint and distribute Morse Account Claimer Tokens.
	// upgrades.Upgrade_0_1_7,

	// v0.1.8 - upgrade to enable:
	// - Application indexing
	// - Suppliers service config indexing and more granular hydration
	// upgrades.Upgrade_0_1_8,

	// v0.1.9 - upgrade to cache claim settlement context
	// upgrades.Upgrade_0_1_9,

	// v0.1.10 - upgrade to fix chain halts caused by the previous upgrade.
	// upgrades.Upgrade_0_1_10,

	// v0.1.11 - upgrade to add allow_morse_account_import_overwrite param.
	// upgrades.Upgrade_0_1_11,

	// v0.1.12 - upgrade to add allow_morse_account_import_overwrite param.
	// upgrades.Upgrade_0_1_12,

	// v0.1.13 - upgrade to:
	// - add morse_account_claiming_enabled migration module param
	// - add compute_unit_cost_granularity shared module param
	// - fix chain halt caused by zero relay claims
	// upgrades.Upgrade_0_1_14,

	// v0.1.14 - upgrade to:
	// - Add Morse supplier claiming non-custodial Morse owner check (#1317)
	// upgrades.Upgrade_0_1_14,

	// v0.1.15 - upgrade to:
	// - Add compute units validation in claim settlement to prevent chain halts when CUPR params change (#1407)
	upgrades.Upgrade_0_1_15,
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
	// The previous binary is expected to have read the plan from onchain and saved it locally.
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
	// NOTE: if 2/3 of the consensus has this configured (i.e. skip upgrade at a specific height),
	// the chain will continue climbing without performing the upgrade.
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
