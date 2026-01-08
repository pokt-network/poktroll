package upgrades

import (
	"context"

	cosmoslog "cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/pokt-network/poktroll/app/keepers"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

const (
	Upgrade_0_1_31_PlanName = "v0.1.31"
)

// Upgrade_0_1_31 handles the upgrade to release `v0.1.31`.
// This upgrade implements:
//
// 1. PIP-41: Deflationary Mint Mechanism
//   - Adds new `mint_ratio` parameter to tokenomics module
//   - Default value is 1.0 (no deflation - backward compatible)
//   - Governance can later set to 0.975 to enable 2.5% deflation
//   - See: https://forum.pokt.network/t/pip-41-introducing-a-deflationary-mint-mechanism-for-shannon-tokenomics/5622
//
// 2. Historical Parameter Tracking
//   - Initializes param history for shared and session modules at upgrade height
//   - Ensures session boundary calculations remain correct after param changes
//   - Fixes claim/proof validation failures when params change mid-session
//   - See: https://github.com/pokt-network/poktroll/issues/543
var Upgrade_0_1_31 = Upgrade{
	PlanName: Upgrade_0_1_31_PlanName,
	// No KVStore migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},

	// Upgrade Handler
	CreateUpgradeHandler: func(
		mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		// PIP-41: Add mint_ratio parameter with default value 1.0 (no deflation)
		// Governance can later update to 0.975 for 2.5% deflation
		// Ref: https://github.com/pokt-network/poktroll/compare/v0.1.30..v0.1.31
		applyNewParameters := func(ctx context.Context, logger cosmoslog.Logger) (err error) {
			logger.Info("Starting PIP-41 parameter updates", "upgrade_plan_name", Upgrade_0_1_31_PlanName)

			// Get the current tokenomics params
			tokenomicsParams := keepers.TokenomicsKeeper.GetParams(ctx)

			// Set mint_ratio to default (1.0 = no deflation) if it's zero
			// This ensures backward compatibility for existing chains
			if tokenomicsParams.MintRatio == 0 {
				tokenomicsParams.MintRatio = tokenomicstypes.DefaultMintRatio
				logger.Info("PIP-41: Setting default mint_ratio to 1.0 (governance can activate deflation)")
			}

			// Ensure that the new parameters are valid
			if err = tokenomicsParams.ValidateBasic(); err != nil {
				logger.Error("Failed to validate tokenomics params", "error", err)
				return err
			}

			// Set the updated parameters
			err = keepers.TokenomicsKeeper.SetParams(ctx, tokenomicsParams)
			if err != nil {
				logger.Error("Failed to set tokenomics params", "error", err)
				return err
			}
			logger.Info("Successfully updated tokenomics params with PIP-41 mint_ratio", "new_params", tokenomicsParams)

			return nil
		}

		// Initialize historical parameter tracking for session and shared modules.
		// This ensures GetParamsAtHeight returns correct params for any height >= upgrade height.
		initializeParamsHistory := func(ctx context.Context, logger cosmoslog.Logger, upgradeHeight int64) error {
			logger.Info("Initializing historical params tracking", "upgrade_plan_name", Upgrade_0_1_31_PlanName)

			// Initialize shared params history at the upgrade height.
			sharedParams := keepers.SharedKeeper.GetParams(ctx)
			if err := keepers.SharedKeeper.SetParamsAtHeight(ctx, upgradeHeight, sharedParams); err != nil {
				logger.Error("Failed to initialize shared params history", "error", err)
				return err
			}
			logger.Info("Initialized shared params history",
				"effective_height", upgradeHeight,
				"num_blocks_per_session", sharedParams.NumBlocksPerSession,
			)

			// Initialize session params history at the upgrade height.
			sessionParams := keepers.SessionKeeper.GetParams(ctx)
			if err := keepers.SessionKeeper.SetParamsAtHeight(ctx, upgradeHeight, sessionParams); err != nil {
				logger.Error("Failed to initialize session params history", "error", err)
				return err
			}
			logger.Info("Initialized session params history",
				"effective_height", upgradeHeight,
				"num_suppliers_per_session", sessionParams.NumSuppliersPerSession,
			)

			return nil
		}

		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
			logger := sdkCtx.Logger()

			if err := applyNewParameters(ctx, logger); err != nil {
				return vm, err
			}

			if err := initializeParamsHistory(ctx, logger, sdkCtx.BlockHeight()); err != nil {
				return vm, err
			}

			return vm, nil
		}
	},
}
