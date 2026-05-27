package upgrades

import (
	"context"

	cosmoslog "cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/pokt-network/poktroll/app/keepers"
)

const (
	Upgrade_0_1_34_PlanName = "v0.1.34"
)

// Upgrade_0_1_34 handles the upgrade to release `v0.1.34`.
// This upgrade adds:
//   - Deduplicate supplier rev share addresses in service config history.
//   - Backfill (issue #1846): mark below-min_stake applications as unbonding.
//     The settlement auto-unstake check now uses the on-chain min_stake param
//     instead of the hardcoded DefaultMinStake; this sweep clears applications
//     that dropped below min_stake before the fix and were never force-unbonded.
//
// NOTE: Application service config history (added in this release for
// deterministic historical session queries) requires NO migration: an empty
// history means the application never changed its service config, and
// GetActiveServiceConfigs falls back to the flat ServiceConfigs snapshot for
// such apps. History is written lazily, only when an app actually swaps service.
//
// CONSENSUS-BREAKING (anchored session grid, #543):
// num_blocks_per_session can now be changed to ANY value via governance without
// misaligning in-flight sessions. Boundary math (GetSessionStartHeight/EndHeight/Number)
// is computed relative to a per-epoch grid anchor stored in shared Params
// (session_grid_anchor_height / session_number_at_anchor) instead of a single modulo from
// block 1. A shared EndBlocker promotes each params epoch to live at its effective height,
// so live params always describe the currently-effective epoch (Option B).
// This handler seeds the genesis epoch: it stamps the current live params with anchor=1,
// number=1 (which makes the new epoch-relative math reduce EXACTLY to the legacy block-1
// grid — no boundary moves at the upgrade) AND records that genesis epoch in params history
// at effective_height=1. The history seed is what lets F1/F2 at-height reads resolve N=60
// for pre-upgrade heights (protecting actors already mid-unbonding) with no new proto field
// and no backfill. See docs/session_length_anchored_grid_spec.md §4.6 / §11.3.
//
// CONSENSUS-BREAKING (validator commission on settlement rewards):
// Settlement reward distribution now applies each validator's commission rate
// before splitting the post-commission remainder among its delegators
// (DistributeValidatorRewards in x/tokenomics/token_logic_module/distribution_validator.go).
// Previously the "proposer" bucket was split purely by network-wide stake weight,
// ignoring commission entirely (validators earned only their self-bonded share).
// This changes the bank operations emitted for the same settled claims, so it is
// consensus-breaking and activates atomically when validators run the v0.1.34
// binary at the upgrade height. It requires NO KVStore migration or upgrade-handler
// logic: the new code path simply takes effect from the upgrade height onward.
var Upgrade_0_1_34 = Upgrade{
	PlanName: Upgrade_0_1_34_PlanName,
	// No KVStore migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},

	// Upgrade Handler
	CreateUpgradeHandler: func(
		mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {

		deduplicateSupplierRevShareAddresses := func(ctx context.Context, logger cosmoslog.Logger) error {
			logger.Info("Deduplicating supplier rev share addresses")

			count, err := keepers.SupplierKeeper.DeduplicateSupplierRevShareAddresses(ctx)
			if err != nil {
				logger.Error("Failed to deduplicate supplier rev share addresses", "error", err)
				return err
			}

			logger.Info("Deduplicated supplier rev share addresses",
				"modified_suppliers", count,
			)
			return nil
		}

		// Backfill for issue #1846: before v0.1.34 the settlement auto-unstake check
		// compared application stake against the hardcoded DefaultMinStake (1 POKT)
		// instead of the on-chain min_stake param, so applications that dropped below
		// the real min_stake were never force-unbonded. v0.1.34 fixes the check; this
		// sweep clears the pre-upgrade backlog of below-min_stake applications.
		unbondBelowMinStakeApplications := func(ctx context.Context, logger cosmoslog.Logger) error {
			logger.Info("Marking below-min_stake applications as unbonding")

			count, err := keepers.ApplicationKeeper.MarkBelowMinStakeApplicationsUnbonding(ctx)
			if err != nil {
				logger.Error("Failed to mark below-min_stake applications as unbonding", "error", err)
				return err
			}

			logger.Info("Marked below-min_stake applications as unbonding",
				"unbonding_applications", count,
			)
			return nil
		}

		// Seed the anchored session grid (#543). Stamp the current live shared params with
		// the genesis-grid anchor (block 1, session 1) and record that epoch in params
		// history at effective_height=1. With anchor=1 the new epoch-relative boundary math
		// is bit-identical to the legacy block-1 grid, so no in-flight session moves at the
		// upgrade. See docs/session_length_anchored_grid_spec.md §4.6 / §11.3.
		seedAnchoredSessionGrid := func(ctx context.Context, logger cosmoslog.Logger) error {
			logger.Info("Seeding anchored session grid (anchor=1, session_number_at_anchor=1)")

			liveParams := keepers.SharedKeeper.GetParams(ctx)

			// Live params idempotency guard: only stamp anchor=1 if the live
			// params don't already carry an anchor. On mainnet (and any chain
			// upgrading from v0.1.33) the anchor field is zero in live params
			// pre-upgrade, so the stamp proceeds normally. On testnets where a
			// previous rehearsal already pinned a non-default anchor, we must
			// NOT clobber that customization — symmetric with the history-seed
			// guard below.
			if liveParams.SessionGridAnchorHeight == 0 {
				stampedParams := liveParams
				stampedParams.SessionGridAnchorHeight = 1
				stampedParams.SessionNumberAtAnchor = 1
				if err := keepers.SharedKeeper.SetParams(ctx, stampedParams); err != nil {
					logger.Error("Failed to set anchored shared params", "error", err)
					return err
				}
				logger.Info("Stamped live shared params with genesis-grid anchor",
					"anchor_height", stampedParams.SessionGridAnchorHeight,
					"session_number_at_anchor", stampedParams.SessionNumberAtAnchor,
				)
			} else {
				logger.Info("Skipping live shared params anchor stamp — anchor already set",
					"anchor_height", liveParams.SessionGridAnchorHeight,
					"session_number_at_anchor", liveParams.SessionNumberAtAnchor,
				)
			}

			// Compute the params snapshot the history would seed with: even if the
			// live-params stamp above was skipped (testnet re-run), we still want
			// the height=1 history entry to carry the anchor-stamped shape if it
			// is absent. This keeps the two seeds in lock-step.
			historySeedParams := keepers.SharedKeeper.GetParams(ctx)
			if historySeedParams.SessionGridAnchorHeight == 0 {
				historySeedParams.SessionGridAnchorHeight = 1
				historySeedParams.SessionNumberAtAnchor = 1
			}

			// Seed params history at height 1 so pre-upgrade heights resolve to N=60.
			// Idempotency guard: only write if no entry exists at height 1. On mainnet
			// the entry is fresh (history is empty pre-upgrade). On testnets where a
			// previous rehearsal may have already seeded a different grid, we must NOT
			// clobber that customization — the operator's existing pinning takes
			// precedence over the handler's default.
			if _, exists := keepers.SharedKeeper.GetParamsHistoryEntry(ctx, 1); exists {
				logger.Info("Skipping shared params history seed at height 1 — entry already exists")
			} else if err := keepers.SharedKeeper.SetParamsAtHeight(ctx, 1, historySeedParams); err != nil {
				logger.Error("Failed to seed shared params history at height 1", "error", err)
				return err
			}

			logger.Info("Seeded anchored session grid",
				"num_blocks_per_session", historySeedParams.GetNumBlocksPerSession(),
			)
			return nil
		}

		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
			logger := sdkCtx.Logger()

			if err := deduplicateSupplierRevShareAddresses(ctx, logger); err != nil {
				return vm, err
			}

			// Seed the anchored session grid BEFORE marking below-min-stake applications
			// for unbonding. The unbond pass calls `GetSessionEndHeight` which routes
			// through `sessionGridAnchor` — with an unstamped (anchor=0) live params
			// snapshot the call resolves to the legacy block-1 grid via the fallback,
			// which is correct but order-dependent on that fallback continuing to behave
			// that way. Seeding first means the unbond pass reads a stamped (anchor=1,
			// session=1) live params and never relies on the fallback. The output state
			// is IDENTICAL — the fallback path and the stamped path return the same
			// session end height on mainnet first run — but the ordering makes the
			// dependency explicit instead of latent. Audit pass 3 MED1.
			if err := seedAnchoredSessionGrid(ctx, logger); err != nil {
				return vm, err
			}

			if err := unbondBelowMinStakeApplications(ctx, logger); err != nil {
				return vm, err
			}

			return vm, nil
		}
	},
}
