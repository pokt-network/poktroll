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
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// Upgrade_0_0_12 is the upgrade handler for v0.0.12 upgrade.
//   - Before: v0.0.11
//   - After: v0.0.12

// This upgrade introduces a type change to RevSharePercent from float32 to uint64, which is introduced as a separate
// protobuf field. As a result, we expect existing on-chain data to switch to default value.
// Investigate the impact of this change on existing on-chain data.
//
// TODO_IN_THIS_PR: decide if we need a proper module migration.

// TODO_IN_THIS_PR: WIP. Using this diff as a starting point: https://github.com/pokt-network/poktroll/compare/v0.0.11...feat/proof-endblocker
// TODO_IN_THIS_PR: Wait for https://github.com/pokt-network/poktroll/pull/1042
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

			// Add supplier module staking_fee per `config.yml`. The min stake is set to 1000000 upokt, but we avoid
			// GetParams() to avoid potential protobuf issues and all networks have the same value (no need to read).
			// Validate with: `poktrolld q supplier params --node=https://testnet-validated-validator-rpc.poktroll.com/`
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
