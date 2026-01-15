package upgrades

import (
	"context"
	"fmt"
	"time"

	cosmoslog "cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/authz"

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
//
// 3. PNF Authz Grants (Mainnet Only)
//   - Grants 12 missing authz permissions to PNF on mainnet
//   - Enables PNF to manage authorizations, upgrades, governance, and admin recovery
//   - Prepares for transition from Grove to PNF as primary authority
//   - Includes new MsgAdminRecoverMorseAccount for fast recovery without allowlist
//   - Note: Betanet PNF already has all permissions, no action needed
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
		initializeDifficultyHistory := func(ctx context.Context, logger cosmoslog.Logger, upgradeHeight int64) error {
			logger.Info("Initializing relay mining difficulty history", "upgrade_plan_name", Upgrade_0_1_31_PlanName)

			// Get all existing difficulties
			allDifficulties := keepers.ServiceKeeper.GetAllRelayMiningDifficulty(ctx)

			for _, difficulty := range allDifficulties {
				// Check if history already exists for this service
				existingHistory := keepers.ServiceKeeper.GetRelayMiningDifficultyHistoryForService(ctx, difficulty.ServiceId)

				if len(existingHistory) == 0 {
					// Initialize history with current difficulty at upgrade height
					if err := keepers.ServiceKeeper.SetRelayMiningDifficultyAtHeight(ctx, upgradeHeight, difficulty); err != nil {
						logger.Error("Failed to initialize difficulty history",
							"service_id", difficulty.ServiceId,
							"error", err,
						)
						return err
					}
					logger.Info("Initialized difficulty history",
						"service_id", difficulty.ServiceId,
						"effective_height", upgradeHeight,
					)
				}
			}

			return nil
		}

		// Grant missing authz permissions to PNF on mainnet only.
		// This ensures PNF has full operational control for governance operations.
		//
		// Context:
		// - Mainnet PNF: Missing 12 critical grants (needs this upgrade)
		// - Betanet PNF: Already has 37 grants (no action needed)
		// - Grove: Has 36 grants on mainnet (will be manually revoked post-upgrade)
		//
		// Note: We grant to PNF directly (not via NetworkAuthzGranteeAddress) because:
		// - NetworkAuthzGranteeAddress points to Grove on mainnet for migration purposes
		// - We want to grant these permissions to PNF to prepare for post-migration
		// - This allows both Grove and PNF to have operational permissions during transition
		// - Grove's permissions will be manually revoked after validating PNF works
		grantPnfAuthzPermissions := func(ctx context.Context, logger cosmoslog.Logger) error {
			sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
			chainID := sdkCtx.ChainID()

			// Only grant on mainnet - betanet PNF already has all permissions
			if chainID != "pocket" {
				logger.Info("Skipping PNF authz grants (not mainnet - betanet PNF already has full permissions)", "chain_id", chainID)
				return nil
			}

			logger.Info("Granting missing authz permissions to MainNet PNF", "upgrade_plan_name", Upgrade_0_1_31_PlanName)

			pnfAddress := MainNetPnfAddress
			logger.Info("Granting to MainNet PNF", "address", pnfAddress)

			// Define the missing messages that PNF needs for full operational control
			missingAuthzMessages := []string{
				// Authz Self-Management - Critical for autonomy
				"/cosmos.authz.v1beta1.MsgExec",
				"/cosmos.authz.v1beta1.MsgGrant",
				"/cosmos.authz.v1beta1.MsgRevoke",
				"/cosmos.authz.v1beta1.MsgRevokeAll",
				"/cosmos.authz.v1beta1.MsgPruneExpiredGrants",

				// Upgrade Management - Critical for protocol upgrades
				"/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade",
				"/cosmos.upgrade.v1beta1.MsgCancelUpgrade",

				// Governance Management
				"/cosmos.gov.v1.MsgCancelProposal",

				// Migration Operations
				"/pocket.migration.MsgImportMorseClaimableAccounts",
				"/pocket.migration.MsgRecoverMorseAccount",
				"/pocket.migration.MsgAdminRecoverMorseAccount", // NEW: Admin recovery without allowlist
				"/pocket.service.MsgRecoverMorseAccount",
			}

			// Grant permissions from gov module authority to PNF
			expiration, err := time.Parse(time.RFC3339, "2500-01-01T00:00:00Z")
			if err != nil {
				return fmt.Errorf("failed to parse expiration time: %w", err)
			}

			granterAddr := keepers.MigrationKeeper.GetAuthority()
			granterCosmosAddr := cosmostypes.MustAccAddressFromBech32(granterAddr)
			pnfCosmosAddr := cosmostypes.MustAccAddressFromBech32(pnfAddress)

			for _, msg := range missingAuthzMessages {
				err = keepers.AuthzKeeper.SaveGrant(
					ctx,
					pnfCosmosAddr,
					granterCosmosAddr,
					authz.NewGenericAuthorization(msg),
					&expiration,
				)
				if err != nil {
					return fmt.Errorf("failed to save grant for message %s: %w", msg, err)
				}
				logger.Info("Granted authorization", "msg", msg, "to", pnfAddress)
			}

			logger.Info("Successfully granted missing authz permissions to PNF", "total_grants", len(missingAuthzMessages))
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

			if err := grantPnfAuthzPermissions(ctx, logger); err != nil {
				return vm, err
			}

			return vm, nil
		}
	},
}
