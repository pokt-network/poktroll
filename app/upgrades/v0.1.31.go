package upgrades

import (
	"context"
	"encoding/hex"

	cosmoslog "cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/pokt-network/poktroll/app/keepers"
	servicekeeper "github.com/pokt-network/poktroll/x/service/keeper"
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
//
// 3. Relay Mining Difficulty Repair (CRITICAL FIX)
//   - Repairs corrupted/empty relay mining difficulties resulting from v0.0.10 migration
//   - The v0.0.10 upgrade skipped data migration when moving RelayMiningDifficulty
//     from tokenomics to service module, leaving 83+ services with empty difficulty objects
//   - This upgrade detects and repairs empty difficulties by creating proper defaults
//   - Initializes difficulty history for all services to support historical queries
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

		// Initialize relay mining difficulty history for all services.
		// This ensures GetRelayMiningDifficultyAtHeight returns correct difficulty for any height >= upgrade height.
		// CRITICAL FIX: This also repairs corrupted/empty difficulties that resulted from the v0.0.10 migration
		// that skipped data migration when RelayMiningDifficulty moved from tokenomics to service module.
		initializeDifficultyHistory := func(ctx context.Context, logger cosmoslog.Logger, upgradeHeight int64) error {
			logger.Info("Initializing relay mining difficulty history", "upgrade_plan_name", Upgrade_0_1_31_PlanName)

			// Get all services and service module params
			allServices := keepers.ServiceKeeper.GetAllServices(ctx)
			serviceParams := keepers.ServiceKeeper.GetParams(ctx)
			targetNumRelays := serviceParams.TargetNumRelays

			logger.Info("Processing relay mining difficulties",
				"total_services", len(allServices),
				"target_num_relays", targetNumRelays,
			)

			repairedCount := 0
			initializedCount := 0

			for _, service := range allServices {
				// Check if valid difficulty exists for this service
				difficulty, found := keepers.ServiceKeeper.GetRelayMiningDifficulty(ctx, service.Id)

				// Detect corrupted/empty difficulty: missing service_id or empty target_hash
				// This repairs the bug from v0.0.10 where migration was skipped
				isCorrupted := !found || difficulty.ServiceId == "" || len(difficulty.TargetHash) == 0

				if isCorrupted {
					// Create proper default difficulty
					difficulty = servicekeeper.NewDefaultRelayMiningDifficulty(
						ctx,
						logger,
						service.Id,
						targetNumRelays,
						targetNumRelays,
					)

					// Save the repaired difficulty to the current difficulty store
					keepers.ServiceKeeper.SetRelayMiningDifficulty(ctx, difficulty)

					repairedCount++
					logger.Info("Repaired corrupted relay mining difficulty",
						"service_id", service.Id,
						"target_hash_hex", hex.EncodeToString(difficulty.TargetHash),
						"num_relays_ema", difficulty.NumRelaysEma,
						"block_height", difficulty.BlockHeight,
					)
				}

				// Check if history already exists for this service
				existingHistory := keepers.ServiceKeeper.GetRelayMiningDifficultyHistoryForService(ctx, service.Id)

				if len(existingHistory) == 0 {
					// Initialize history with valid difficulty at upgrade height
					if err := keepers.ServiceKeeper.SetRelayMiningDifficultyAtHeight(ctx, upgradeHeight, difficulty); err != nil {
						logger.Error("Failed to initialize difficulty history",
							"service_id", service.Id,
							"error", err,
						)
						return err
					}

					initializedCount++
					logger.Info("Initialized difficulty history",
						"service_id", service.Id,
						"effective_height", upgradeHeight,
						"target_hash_hex", hex.EncodeToString(difficulty.TargetHash),
					)
				}
			}

			logger.Info("Completed relay mining difficulty initialization",
				"total_services_processed", len(allServices),
				"corrupted_repaired", repairedCount,
				"history_initialized", initializedCount,
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

			return vm, nil
		}
	},
}
