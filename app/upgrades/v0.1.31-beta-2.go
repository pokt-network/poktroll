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
	Upgrade_0_1_31_Beta_2_PlanName = "v0.1.31-beta-2"
)

// Upgrade_0_1_31_Beta_2 handles the betanet upgrade to test v0.1.31 fixes.
// This upgrade implements the same changes as v0.1.31 plus critical bug fixes:
//
// 1. PIP-41: Deflationary Mint Mechanism
//   - Adds new `mint_ratio` parameter to tokenomics module
//   - Default value is 1.0 (no deflation - backward compatible)
//
// 2. Historical Parameter Tracking
//   - Initializes param history for shared and session modules at upgrade height
//   - Ensures session boundary calculations remain correct after param changes
//
// 3. Orphaned Service Config Index Cleanup (NEW - Bug Fix)
//   - Cleans up orphaned index entries from activation/deactivation/supplier indexes
//   - Fixes root cause: re-indexing now properly removes old entries before adding new ones
//   - See: x/supplier/keeper/supplier_index.go
//
// 4. Defensive Logging Improvements
//   - Changed orphaned entry warnings to debug level to reduce log noise
//
// Note: This upgrade is for betanet only. Mainnet will use v0.1.31 or v0.1.32
// depending on betanet test results.
var Upgrade_0_1_31_Beta_2 = Upgrade{
	PlanName: Upgrade_0_1_31_Beta_2_PlanName,
	// No KVStore migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},

	// Upgrade Handler
	CreateUpgradeHandler: func(
		mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		// PIP-41: Add mint_ratio parameter with default value 1.0 (no deflation)
		applyNewParameters := func(ctx context.Context, logger cosmoslog.Logger) (err error) {
			logger.Info("Starting PIP-41 parameter updates", "upgrade_plan_name", Upgrade_0_1_31_Beta_2_PlanName)

			// Get the current tokenomics params
			tokenomicsParams := keepers.TokenomicsKeeper.GetParams(ctx)

			// Set mint_ratio to default (1.0 = no deflation) if it's zero
			if tokenomicsParams.MintRatio == 0 {
				tokenomicsParams.MintRatio = tokenomicstypes.DefaultMintRatio
				logger.Info("PIP-41: Setting default mint_ratio to 1.0")
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
		initializeParamsHistory := func(ctx context.Context, logger cosmoslog.Logger, upgradeHeight int64) error {
			logger.Info("Initializing historical params tracking", "upgrade_plan_name", Upgrade_0_1_31_Beta_2_PlanName)

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

		// Initialize relay mining difficulty history for all services.
		// CRITICAL FIX: This upgrade handler is idempotent and ensures ALL nodes have
		// difficulty history initialized at THIS specific upgrade height.
		//
		// Previous issue: v0.1.31 (height 666078) crashed after some nodes initialized
		// history, causing divergence. This fix ensures consistent state by always
		// initializing at the current upgrade height, regardless of existing history.
		initializeDifficultyHistory := func(ctx context.Context, logger cosmoslog.Logger, upgradeHeight int64) error {
			logger.Info("Initializing relay mining difficulty history", "upgrade_plan_name", Upgrade_0_1_31_Beta_2_PlanName)

			// Get all existing difficulties
			allDifficulties := keepers.ServiceKeeper.GetAllRelayMiningDifficulty(ctx)

			for _, difficulty := range allDifficulties {
				serviceId := difficulty.ServiceId

				// Check if history already exists for this service
				existingHistory := keepers.ServiceKeeper.GetRelayMiningDifficultyHistoryForService(ctx, serviceId)

				if len(existingHistory) > 0 {
					logger.Info("Found existing difficulty history, will ensure entry exists at upgrade height",
						"service_id", serviceId,
						"num_existing_entries", len(existingHistory),
					)
				}

				// ALWAYS set difficulty at this upgrade height to ensure consistency.
				// If an entry already exists at this height, it will be overwritten with current values.
				// This is idempotent: running the upgrade multiple times produces the same result.
				if err := keepers.ServiceKeeper.SetRelayMiningDifficultyAtHeight(ctx, upgradeHeight, difficulty); err != nil {
					logger.Error("Failed to initialize difficulty history",
						"service_id", serviceId,
						"error", err,
					)
					return err
				}
				logger.Info("Initialized difficulty history at upgrade height",
					"service_id", serviceId,
					"effective_height", upgradeHeight,
					"target_hash", difficulty.TargetHash,
				)
			}

			return nil
		}

		// Clean up orphaned service config index entries.
		// These are index entries that point to non-existent primary store records,
		// which can occur when suppliers are unstaked and their service configs are deleted
		// but some index entries remain due to historical bugs or incomplete deletions.
		//
		// This cleanup runs during upgrade to remove any accumulated orphaned entries
		// from the activation, deactivation, and supplier indexes. The pruning logic
		// already handles these defensively (see prune_supplier_service_config_history.go),
		// but cleaning them up improves performance and reduces log noise.
		//
		// Root cause has been fixed in x/supplier/keeper/supplier_index.go by ensuring
		// all old indexes are removed before re-indexing during supplier updates.
		cleanupOrphanedServiceConfigIndexes := func(ctx context.Context, logger cosmoslog.Logger) error {
			logger.Info("Cleaning up orphaned service config indexes")

			actCount, deactCount, supplierCount, err := keepers.SupplierKeeper.CleanupOrphanedServiceConfigIndexes(ctx)
			if err != nil {
				logger.Error("Failed to cleanup orphaned indexes", "error", err)
				return err
			}

			logger.Info("Cleaned up orphaned service config indexes",
				"activation_cleaned", actCount,
				"deactivation_cleaned", deactCount,
				"supplier_cleaned", supplierCount,
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

			if err := initializeDifficultyHistory(ctx, logger, sdkCtx.BlockHeight()); err != nil {
				return vm, err
			}

			if err := cleanupOrphanedServiceConfigIndexes(ctx, logger); err != nil {
				return vm, err
			}

			logger.Info("Successfully completed v0.1.31-beta-2 upgrade for betanet")
			return vm, nil
		}
	},
}
