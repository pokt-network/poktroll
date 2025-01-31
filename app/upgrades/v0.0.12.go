package upgrades

import (
	"context"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cosmosTypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/pokt-network/poktroll/app/keepers"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// Upgrade_0_0_12 handles the v0.0.12 upgrade.
//
// Versions:
//   - Before: v0.0.11 
//   - After: v0.0.12

// This upgrade changes RevSharePercent from float32  to uint64 in a new protobuf field.
// The result is existing onchain resetting to the default.
//
// TODO_IN_THIS_PR: Verify impact on existing chain data
var Upgrade_0_0_12 = Upgrade{
	PlanName: "v0.0.12",
	CreateUpgradeHandler: func(mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		// Adds new parameters using ignite's config.yml as a reference. Assuming we don't need any other parameters.
		applyNewParameters := func(ctx context.Context) (err error) {
			logger := cosmosTypes.UnwrapSDKContext(ctx).Logger()
			logger.Info("Starting parameter updates for v0.0.12")

			// Set supplier module staking_fee to 1000000 upokt to match config.yml. 
			// Using hardcoded value because:  
			//   - All networks (Alpha & Beta TestNet) share the same value
			//   - Avoids potential protobuf issues with GetParams()
			//
			// Verify via:
			// $ poktrolld q supplier params --node=https://testnet-validated-validator-rpc.poktroll.com/
			supplierParams := suppliertypes.Params{
				MinStake: &cosmosTypes.Coin{
					Denom:  "upokt",
					Amount: math.NewInt(1000000),
				},
				StakingFee: &cosmosTypes.Coin{
					Denom: "upokt",
					// TODO_IN_THIS_PR: 100upokt a good value?
					Amount: math.NewInt(100),
				},
			}

			// ALL parameters must be present when setting params.
			err = keepers.SupplierKeeper.SetParams(ctx, supplierParams)
			if err != nil {
				logger.Error("Failed to set supplier params", "error", err)
				return err
			}
			logger.Info("Successfully updated supplier params", "new_params", supplierParams)

			// Add service module `target_num_relays` parameter per `config.yml`.
			// We don't use `GetParams()` to avoid potential protobuf issues and all networks have the same value (no need to read).
			serviceParams := servicetypes.Params{
				AddServiceFee: &cosmosTypes.Coin{
					Denom:  "upokt",
					Amount: math.NewInt(1000000000),
				},
				TargetNumRelays: 100,
			}
			err = keepers.ServiceKeeper.SetParams(ctx, serviceParams)
			if err != nil {
				logger.Error("Failed to set service params", "error", err)
				return err
			}
			logger.Info("Successfully updated service params", "new_params", serviceParams)

			// Add tokenomics module `global_inflation_per_claim` parameter per `config.yml`.
			// We use GetParams() as `DaoRewardAddress` is different between networks and we don't want to hardcode it.
			tokenomicsParams := keepers.TokenomicsKeeper.GetParams(ctx)
			tokenomicsParams.GlobalInflationPerClaim = 0.1
			err = keepers.TokenomicsKeeper.SetParams(ctx, tokenomicsParams)
			if err != nil {
				logger.Error("Failed to set tokenomics params", "error", err)
				return err
			}
			return
		}

		// Returns the upgrade handler for v0.0.12
		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			logger := cosmosTypes.UnwrapSDKContext(ctx).Logger()
			logger.Info("Starting v0.0.12 upgrade handler")

			err := applyNewParameters(ctx)
			if err != nil {
				logger.Error("Failed to apply new parameters", "error", err)
				return vm, err
			}

			// Since we changed the type of RevSharePercent from float32 to uint64, we need to update all on-chain data.
			// The easiest way to do this is to iterate over all suppliers and services and set the revshare to 100 by default.
			suppliers := keepers.SupplierKeeper.GetAllSuppliers(ctx)
			logger.Info("Updating all suppliers to have a 100% revshare to the supplier", "num_suppliers", len(suppliers))
			for _, supplier := range suppliers {
				for _, service := range supplier.Services {
					// Force all services to have a 100% revshare to the supplier.
					// Not something we would do on a real mainnet, but it's a quick way to resolve the issue.
					// Currently, we don't break any existing suppliers (as all of them have a 100% revshare to the supplier).
					service.RevShare = []*sharedtypes.ServiceRevenueShare{
						{
							Address:            supplier.OperatorAddress,
							RevSharePercentage: uint64(100),
						},
					}
				}
				keepers.SupplierKeeper.SetSupplier(ctx, supplier)
				logger.Info("Updated supplier", "supplier", supplier.OperatorAddress)
			}

			logger.Info("Running module migrations")
			vm, err = mm.RunMigrations(ctx, configurator, vm)
			if err != nil {
				logger.Error("Failed to run migrations", "error", err)
				return vm, err
			}

			logger.Info("Successfully completed v0.0.12 upgrade handler")
			return vm, nil
		}
	},
	// No changes to the KVStore in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},
}
