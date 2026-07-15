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

// TODO_NEXT_UPGRADE: Rename NEXT with the appropriate next
// upgrade version number and update comment versions.

const (
	Upgrade_NEXT_PlanName = "vNEXT"
)

// Upgrade_NEXT handles the upgrade to release `vNEXT`.
// This upgrade adds:
//   - Settlement budget redistribution: the per-supplier head-split cap
//     (appStake / numPendingSessions / actualNumSuppliers) becomes a guaranteed FLOOR
//     rather than a hard ceiling. Budget left unused by idle/light suppliers is
//     redistributed to suppliers that served above their floor, bounded by the
//     application's committed per-session budget B. Introduces the new tokenomics param
//     `overservicing_bonus_multiplier`, seeded to 1 here so the consensus change ships as
//     an exact no-op (m == 1 reproduces the legacy head-split cap byte-for-byte);
//     governance opens redistribution afterwards by raising / zeroing the multiplier.
//     Also enforces the anti-collusion invariant
//     `mint_ratio * mint_equals_burn_claim_distribution.supplier < 1` in params validation.
var Upgrade_NEXT = Upgrade{
	PlanName: Upgrade_NEXT_PlanName,
	// No KVStore migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},

	// Upgrade Handler
	CreateUpgradeHandler: func(
		mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		// Add new parameters by:
		// 1. Inspecting the diff between vPREV..vNEXT
		// 2. Manually inspect changes in ignite's config.yml
		// 3. Update the upgrade handler here accordingly
		// Ref: https://github.com/pokt-network/poktroll/compare/vPREV..vNEXT

		// Seed the new overservicing_bonus_multiplier tokenomics param.
		//
		// On upgrade the previously-stored tokenomics params deserialize with the new field at
		// its proto3 zero value (0). The settlement path already treats 0 as 1 (the legacy
		// floor cap), so this upgrade is economically inert even without any migration — the
		// zero-value is benign by design. We still set it to 1 explicitly so the stored param
		// is self-documenting rather than relying on the read-time coercion. Governance later
		// raises the multiplier to enable redistribution — no second upgrade required.
		applyNewParameters := func(ctx context.Context, logger cosmoslog.Logger) (err error) {
			logger.Info("Starting settlement budget redistribution parameter updates",
				"upgrade_plan_name", Upgrade_NEXT_PlanName)

			tokenomicsParams := keepers.TokenomicsKeeper.GetParams(ctx)

			// Set overservicing_bonus_multiplier to its default (1 = no-op / legacy
			// head-split cap) if unset, so redistribution stays OFF until governance opts in.
			if tokenomicsParams.OverservicingBonusMultiplier == 0 {
				tokenomicsParams.OverservicingBonusMultiplier = tokenomicstypes.DefaultOverservicingBonusMultiplier
				logger.Info("Setting default overservicing_bonus_multiplier to 1 (no-op; governance can enable redistribution)")
			}

			// Ensure the new parameter set is valid (also checks the anti-collusion invariant).
			if err = tokenomicsParams.ValidateBasic(); err != nil {
				logger.Error("Failed to validate tokenomics params", "error", err)
				return err
			}

			if err = keepers.TokenomicsKeeper.SetParams(ctx, tokenomicsParams); err != nil {
				logger.Error("Failed to set tokenomics params", "error", err)
				return err
			}
			logger.Info("Successfully seeded overservicing_bonus_multiplier", "new_params", tokenomicsParams)

			return nil
		}

		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
			logger := sdkCtx.Logger()

			if err := applyNewParameters(ctx, logger); err != nil {
				return vm, err
			}

			return vm, nil
		}
	},
}
