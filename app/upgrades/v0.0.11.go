package upgrades

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cosmosTypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/pokt-network/poktroll/app/keepers"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// Upgrade_0_0_11 is the upgrade handler for v0.0.11 Alpha TestNet upgrade
// Beta TestNet was launched with v0.0.11, so this upgrade is exclusively for Alpha TestNet.
//   - Before: v0.0.10
//   - After: v0.0.11
var Upgrade_0_0_11 = Upgrade{
	PlanName: "v0.0.11",
	CreateUpgradeHandler: func(mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		// Adds new parameters using ignite's config.yml as a reference. Assuming we don't need any other parameters.
		// https://github.com/pokt-network/poktroll/compare/v0.0.10...v0.0.11-rc
		applyNewParameters := func(ctx context.Context) (err error) {
			logger := cosmosTypes.UnwrapSDKContext(ctx).Logger()
			logger.Info("Starting parameter updates for v0.0.11")

			// Set num_suppliers_per_session to 15
			// Validate with: `pocketd q session params --node=https://testnet-validated-validator-rpc.poktroll.com/`
			sessionParams := sessiontypes.Params{
				NumSuppliersPerSession: uint64(15),
			}

			// ALL parameters must be present when setting params.
			err = keepers.SessionKeeper.SetParams(ctx, sessionParams)
			if err != nil {
				logger.Error("Failed to set session params", "error", err)
				return err
			}
			logger.Info("Successfully updated session params", "new_params", sessionParams)

			// Set tokenomics params. The values are based on default values for LocalNet/Beta TestNet.
			// Validate with: `pocketd q tokenomics params --node=https://testnet-validated-validator-rpc.poktroll.com/`
			tokenomicsParams := tokenomicstypes.Params{
				MintAllocationPercentages: tokenomicstypes.MintAllocationPercentages{
					Dao:         0.1,
					Proposer:    0.05,
					Supplier:    0.7,
					SourceOwner: 0.15,
					Application: 0.0,
				},
				DaoRewardAddress: AlphaTestNetPnfAddress,
			}

			// ALL parameters must be present when setting params.
			err = keepers.TokenomicsKeeper.SetParams(ctx, tokenomicsParams)
			if err != nil {
				logger.Error("Failed to set tokenomics params", "error", err)
				return err
			}
			logger.Info("Successfully updated tokenomics params", "new_params", tokenomicsParams)

			return
		}

		// The diff shows that the only new authz authorization is for the `pocket.session.MsgUpdateParam` message.
		// However, this message is already authorized for the `pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t` address.
		// See here: pocketd q authz grants-by-granter pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t --node=https://shannon-testnet-grove-seed-rpc.alpha.poktroll.com
		// If this upgrade would have been applied to other networks, we could have added a separate upgrade handler for each network.

		// Returns the upgrade handler for v0.0.11
		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			logger := cosmosTypes.UnwrapSDKContext(ctx).Logger()
			logger.Info("Starting v0.0.11 upgrade handler")

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

			logger.Info("Successfully completed v0.0.11 upgrade handler")
			return vm, nil
		}
	},
	// No changes to the KVStore in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},
}
