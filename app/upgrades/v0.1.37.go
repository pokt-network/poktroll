package upgrades

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/pokt-network/poktroll/app/keepers"
)

const (
	Upgrade_0_1_37_PlanName = "v0.1.37"
)

// Upgrade_0_1_37 handles the upgrade to release `v0.1.37`.
// Consensus-breaking changes in this release:
//   - Addresses in application, gateway, service and supplier messages (owners, operators,
//     signers, delegatees, revenue shares, allowlist entries) and in claim and proof messages
//     must be in canonical bech32 form.
//   - State migrations at the upgrade height: suppliers stored under a non-canonical operator
//     address are removed without refund and non-canonical supplier owners are rewritten
//     (MigrateNonCanonicalSupplierAddresses); non-canonical service owners are rewritten
//     (CanonicalizeServiceOwnerAddresses); applications and gateways stored under a
//     non-canonical address begin unbonding, and pending application transfers to a
//     non-canonical destination are canonicalized or dropped (MigrateNonCanonicalApplications,
//     MigrateNonCanonicalGateways).
//   - Supplier unbonding removes records by their stored key and skips, instead of failing on,
//     an index entry without a record; gateway refund failures and application index
//     inconsistencies are logged and skipped in the EndBlockers.
//   - Relay-mining difficulty is left unchanged when the relay EMA is zero.
//   - The Morse claim gas-fee waiver applies only to transactions whose claims can succeed:
//     claiming enabled, distinct existing unclaimed accounts, supplier claims signed by the
//     node or its output address.
//   - Permissioned services: `MsgUpdateServiceSupplierAllowlist`; a service with a non-empty
//     supplier allowlist admits only listed suppliers (stake, restake and Morse supplier
//     claims); an owner change deactivates services the new owner is not allowed on; the
//     supplier EndBlocker sweeps unlisted suppliers (before the service cap sweep).
//   - Settlement counts only claims that can be priced when dividing an application's
//     session budget among its suppliers.
//   - MsgStakeSupplier reads one more store key per service (allowlist check), about +1k gas.
//   - Dependency updates: smt (stricter proof validation), iavl, cometbft gRPC stack.
//   - Supplier owner recovery: the owner address of the suppliers listed in
//     v0.1.37_recovery.go, keyed by operator address, is rewritten to the address their
//     owner supplied (RecoverSupplierOwners).
var Upgrade_0_1_37 = Upgrade{
	PlanName: Upgrade_0_1_37_PlanName,
	// No KVStore migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},

	// Upgrade Handler
	CreateUpgradeHandler: func(
		mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		// Add new parameters by:
		// 1. Inspecting the diff between v0.1.36..v0.1.37
		// 2. Manually inspect changes in ignite's config.yml
		// 3. Update the upgrade handler here accordingly
		// Ref: https://github.com/pokt-network/poktroll/compare/v0.1.36..v0.1.37

		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			logger := cosmostypes.UnwrapSDKContext(ctx).Logger().With("upgrade_plan_name", Upgrade_0_1_37_PlanName)

			numRemoved, numOwnersCanonicalized := keepers.SupplierKeeper.MigrateNonCanonicalSupplierAddresses(ctx)
			logger.Info("Migrated non-canonical supplier addresses",
				"suppliers_removed", numRemoved,
				"owners_canonicalized", numOwnersCanonicalized,
			)

			numServiceOwnersCanonicalized := keepers.ServiceKeeper.CanonicalizeServiceOwnerAddresses(ctx)
			logger.Info("Canonicalized service owner addresses",
				"service_owners_canonicalized", numServiceOwnersCanonicalized,
			)

			numAppsUnbondingBegun, numTransferDestinationsCanonicalized, numTransfersDropped := keepers.ApplicationKeeper.MigrateNonCanonicalApplications(ctx)
			logger.Info("Migrated non-canonical application addresses",
				"applications_unbonding_begun", numAppsUnbondingBegun,
				"transfer_destinations_canonicalized", numTransferDestinationsCanonicalized,
				"transfers_dropped", numTransfersDropped,
			)

			numGatewaysUnbondingBegun := keepers.GatewayKeeper.MigrateNonCanonicalGateways(ctx)
			logger.Info("Migrated non-canonical gateway addresses",
				"gateways_unbonding_begun", numGatewaysUnbondingBegun,
			)

			recoveries := SupplierOwnerRecoveries()
			numRecovered, numSkipped := keepers.SupplierKeeper.RecoverSupplierOwners(ctx, recoveries, RecoveryCancelsUnbonding)
			logger.Info("Recovered supplier owners",
				"suppliers_recovered", numRecovered,
				"suppliers_skipped", numSkipped,
			)
			if len(recoveries) > 0 && numRecovered == 0 {
				// Every listed supplier was skipped: none exists at this height (the
				// unbondings already ended) or the destination is unusable.
				logger.Error("Supplier owner recovery recovered nothing", "suppliers_listed", len(recoveries))
			}

			return vm, nil
		}
	},
}
