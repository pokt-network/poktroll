package upgrades

import (
	"context"
	"strings"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cosmosTypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/pokt-network/pocket/app/keepers"
	sharedtypes "github.com/pokt-network/pocket/x/shared/types"
)

const Upgrade_0_0_12_PlanName = "v0.0.12"

// Upgrade_0_0_12 handles the upgrade to release `v0.0.12`.
// This is planned to be issued on both Pocket Network's Shannon Alpha & Beta TestNets.
var Upgrade_0_0_12 = Upgrade{
	PlanName: Upgrade_0_0_12_PlanName,
	CreateUpgradeHandler: func(mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		// Parameter configurations aligned with repository config.yml specifications.
		// These values reflect the delta between v0.0.11 and the main branch as of #1043.
		// Reference:
		// - Comparison: https://github.com/pokt-network/pocket/compare/v0.0.11..7541afd6d89a12d61e2c32637b535f24fae20b58
		// - Direct diff: `git diff v0.0.11..7541afd6d89a12d61e2c32637b535f24fae20b58 -- config.yml`
		//
		// DEV_NOTE: These parameter updates are derived from config.yml in the root directory
		// of this repository, which serves as the source of truth for all parameter changes.
		const (
			supplierStakingFee                = 1000000 // uPOKT
			serviceTargetNumRelays            = 100     // num relays
			tokenomicsGlobalInflationPerClaim = 0.1     // % of the claim amount
		)

		applyNewParameters := func(ctx context.Context) (err error) {
			logger := cosmosTypes.UnwrapSDKContext(ctx).Logger()
			logger.Info("Starting parameter updates", "upgrade_plan_name", Upgrade_0_0_12_PlanName)

			// Set supplier module staking_fee to 1000000upokt, in line with the config.yml in the repo.
			// Verify via:
			// $ pocketd q supplier params --node=...
			supplierParams := keepers.SupplierKeeper.GetParams(ctx)
			supplierParams.MinStake = &cosmosTypes.Coin{
				Denom:  "upokt",
				Amount: math.NewInt(supplierStakingFee),
			}
			err = keepers.SupplierKeeper.SetParams(ctx, supplierParams)
			if err != nil {
				logger.Error("Failed to set supplier params", "error", err)
				return err
			}
			logger.Info("Successfully updated supplier params", "new_params", supplierParams)

			// Add service module `target_num_relays` parameter, in line with the config.yml in the repo.
			// Verify via:
			// $ pocketd q service params --node=...
			serviceParams := keepers.ServiceKeeper.GetParams(ctx)
			serviceParams.TargetNumRelays = serviceTargetNumRelays
			err = keepers.ServiceKeeper.SetParams(ctx, serviceParams)
			if err != nil {
				logger.Error("Failed to set service params", "error", err)
				return err
			}
			logger.Info("Successfully updated service params", "new_params", serviceParams)

			// Add tokenomics module `global_inflation_per_claim` parameter, in line with the config.yml in the repo.
			// Verify via:
			// $ pocketd q tokenomics params --node=...
			tokenomicsParams := keepers.TokenomicsKeeper.GetParams(ctx)
			tokenomicsParams.GlobalInflationPerClaim = tokenomicsGlobalInflationPerClaim
			err = keepers.TokenomicsKeeper.SetParams(ctx, tokenomicsParams)
			if err != nil {
				logger.Error("Failed to set tokenomics params", "error", err)
				return err
			}
			logger.Info("Successfully updated tokenomics params", "new_params", tokenomicsParams)
			return nil
		}

		// Helper function to update all suppliers' RevShare to 100%.
		// This is necessary to ensure that we have that value populated before suppliers are connected.
		//
		updateSuppliersRevShare := func(ctx context.Context) error {
			logger := cosmosTypes.UnwrapSDKContext(ctx).Logger()
			suppliers := keepers.SupplierKeeper.GetAllSuppliers(ctx)
			logger.Info("Updating (overriding) all suppliers to delegate 100% revenue share to the supplier's operator address",
				"num_suppliers", len(suppliers))

			for _, supplier := range suppliers {
				for _, service := range supplier.Services {
					if len(service.RevShare) > 1 {
						// WARNING: Overwriting existing revshare settings without preserving history.
						// NOTE: While the canonical approach would be using Module Upgrade (docs.cosmos.network/v0.46/building-modules/upgrade)
						// to handle protobuf type changes (see: github.com/cosmos/cosmos-sdk/blob/v0.46.0-rc1/x/bank/migrations/v043/store.go#L50-L71),
						// we've opted for direct overwrite because:
						// 1. No active revenue shares are impacted at time of writing
						// 2. Additional protobuf and repo structure changes would be required for proper (though unnecessary) migration

						// Create a string representation of just the revenue share addresses
						addresses := make([]string, len(service.RevShare))
						for i, rs := range service.RevShare {
							addresses[i] = rs.Address
						}
						revShareAddressesStr := "[" + strings.Join(addresses, ",") + "]"
						logger.Warn(
							"Overwriting existing revenue share configuration",
							"supplier_operator", supplier.OperatorAddress,
							"supplier_owner", supplier.OwnerAddress,
							"service", service.ServiceId,
							"previous_revshare_count", len(service.RevShare),
							"previous_revshare_addresses", revShareAddressesStr,
						)
						service.RevShare = []*sharedtypes.ServiceRevenueShare{
							{
								Address:            supplier.OperatorAddress,
								RevSharePercentage: uint64(100),
							},
						}
					} else if len(service.RevShare) == 1 {
						// If there is only one revshare setting, we can safely overwrite it (because it has 100%
						// revenue share), keeping the existing address.
						logger.Info("Updating supplier's revenue share configuration",
							"supplier_operator", supplier.OperatorAddress,
							"supplier_owner", supplier.OwnerAddress,
							"service", service.ServiceId,
							"previous_revshare_address", service.RevShare[0].Address,
						)
						currentRevShare := service.RevShare[0]
						service.RevShare = []*sharedtypes.ServiceRevenueShare{
							{
								Address:            currentRevShare.Address,
								RevSharePercentage: uint64(100),
							},
						}
					} else {
						logger.Warn("That shouldn't happen: no revenue share configuration found for supplier",
							"supplier_operator", supplier.OperatorAddress,
							"supplier_owner", supplier.OwnerAddress,
							"service", service.ServiceId,
						)
					}
				}
				keepers.SupplierKeeper.SetSupplier(ctx, supplier)
				logger.Info("Updated supplier",
					"supplier_operator", supplier.OperatorAddress,
					"supplier_owner", supplier.OwnerAddress)
			}
			return nil
		}

		// Returns the upgrade handler for v0.0.12
		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			logger := cosmosTypes.UnwrapSDKContext(ctx).Logger()
			logger.Info("Starting upgrade handler", "upgrade_plan_name", Upgrade_0_0_12_PlanName)

			logger.Info("Starting parameter updates section", "upgrade_plan_name", Upgrade_0_0_12_PlanName)
			// Update all governance parameter changes.
			// This includes adding params, removing params and changing values of existing params.
			err := applyNewParameters(ctx)
			if err != nil {
				logger.Error("Failed to apply new parameters",
					"upgrade_plan_name", Upgrade_0_0_12_PlanName,
					"error", err)
				return vm, err
			}

			logger.Info("Starting supplier RevShare updates section", "upgrade_plan_name", Upgrade_0_0_12_PlanName)
			// Override all suppliers' RevShare to be 100% delegate to the supplier's operator address
			err = updateSuppliersRevShare(ctx)
			if err != nil {
				logger.Error("Failed to update suppliers RevShare",
					"upgrade_plan_name", Upgrade_0_0_12_PlanName,
					"error", err)
				return vm, err
			}

			logger.Info("Starting module migrations section", "upgrade_plan_name", Upgrade_0_0_12_PlanName)
			vm, err = mm.RunMigrations(ctx, configurator, vm)
			if err != nil {
				logger.Error("Failed to run migrations",
					"upgrade_plan_name", Upgrade_0_0_12_PlanName,
					"error", err)
				return vm, err
			}

			logger.Info("Successfully completed upgrade handler", "upgrade_plan_name", Upgrade_0_0_12_PlanName)
			return vm, nil
		}
	},
	// No changes to the KVStore in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},
}
