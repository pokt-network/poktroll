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
	Upgrade_0_1_33_PlanName = "v0.1.33"
)

// Upgrade_0_1_33 handles the upgrade to release `v0.1.33`.
// This upgrade fixes:
//   - removeApplicationUndelegationIndex deleting from the wrong store (delegation
//     store instead of undelegation store), causing orphaned undelegation index entries
//     when applications with pending undelegations were removed (unstaked/transferred).
//     Bug introduced in PR #1263 (v0.1.31). The upgrade handler cleans up any
//     orphaned entries accumulated since v0.1.31.
//
// P0 audit fixes (from v0.1.31 audit):
//   - getSupplierServiceConfigUpdates now skips orphaned index entries (nil primary
//     record) instead of calling MustUnmarshal(nil) which produces zero-value structs
//     with Service == nil. Mirrors the existing nil check in
//     removeSupplierServiceConfigUpdateIndexes.
//   - shared module's MsgUpdateParams (bulk/governance) now calls recordParamsHistory
//     before SetParams, matching the singular MsgUpdateParam and session module's
//     MsgUpdateParams. Without this, governance bulk param updates bypassed history
//     recording, causing GetParamsAtHeight to return stale values.
//
// P1 audit fixes (from v0.1.31 audit):
//   - SubmitProof: Added error checks after GetClaimeduPOKT and GetNumEstimatedComputeUnits
//     calls. Previously the second := assignment silently overwrote the error from the first,
//     causing incorrect zero-value data in events when GetClaimeduPOKT failed.
//   - proof/claim handlers (CreateClaim, SubmitProof, ProofRequirementForClaim): Replaced
//     sharedKeeper.GetParams(ctx) with GetParamsAtHeight(ctx, sessionStartHeight) so that
//     claimed uPOKT and proof requirement calculations use the shared params that were
//     effective when the session started, not the current params.
//   - SharedKeeperQueryClient (GetEarliestSupplierClaimCommitHeight,
//     GetEarliestSupplierProofCommitHeight): Replaced sharedKeeper.GetParams(ctx) with
//     GetParamsAtHeight(ctx, queryHeight) so that claim/proof window calculations use
//     historical params consistent with the session-level window validation in session.go.
//
// P2 audit fixes (from v0.1.31 audit):
//   - session_hydrator: Fixed potential integer overflow in supplier sort comparison.
//     generateSupplierRandomWeight produces values across the full int64 range;
//     subtraction (weightA - weightB) could overflow, violating sort's trichotomy
//     property. Replaced with explicit comparison (<, >, ==).
//
// P3 audit fixes (from v0.1.31 audit):
//   - tokenomics: Zero-mint guard in processTokenomicsMint — returns error when
//     settlementAmount * mint_ratio truncates to 0, preventing silent claim loss.
//   - tokenomics: ValidateMintRatio now rejects 0 (range (0, 1] strictly enforced),
//     removed stale auto-correction in ValidateBasic.
//   - shared/session keepers: Added HasParamsHistory() O(1) check, replacing O(n)
//     GetAllParamsHistory() emptiness check in recordParamsHistory/RecordParamsHistory.
//   - shared/session genesis: Export/import params_history in ExportGenesis/InitGenesis,
//     ensuring param history survives chain export/import cycles.
//   - application CLI: New stake-and-delegate command sends MsgStakeApplication + N ×
//     MsgDelegateToGateway in a single transaction via extended config YAML.
//   - application: CONSENSUS-BREAKING — Fixed IsActive() OR-chain logic that always
//     returned true. Unbonding/transferring apps past their end height are now correctly
//     excluded from sessions.
//
// Settlement event improvements (event-only, no state changes):
//   - EventClaimSettled: Added `settled_upokt` (post-cap, pre-mint_ratio amount) and
//     `mint_ratio` fields so indexers can decompose overservicing loss vs deflation loss.
//   - EventApplicationOverserviced: BREAKING SEMANTIC CHANGE — `effective_burn` now
//     includes the globalInflation component, matching `expected_burn`'s basis.
//     Previously `effective_burn` excluded globalInflation, making the gap
//     (expected_burn - effective_burn) appear larger than actual overservicing.
//     Indexers (e.g. pocketdex) that compute overservicing amounts from this gap
//     will see smaller, more accurate values after this upgrade.
//   - EventApplicationOverserviced: Added `service_id` and `session_end_block_height`
//     fields to enable unambiguous joins with EventClaimSettled. Previously indexers
//     had to match on (app_addr, supplier_addr) within the same block, which is
//     ambiguous when the same pair has claims from multiple sessions settling together.
var Upgrade_0_1_33 = Upgrade{
	PlanName: Upgrade_0_1_33_PlanName,
	// No KVStore migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},

	// Upgrade Handler
	CreateUpgradeHandler: func(
		mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {

		// Clean up orphaned undelegation index entries.
		// These are index entries that reference applications which no longer exist,
		// caused by removeApplicationUndelegationIndex deleting from getDelegationStore()
		// instead of getUndelegationStore(). Introduced in PR #1263 (v0.1.31).
		cleanupOrphanedUndelegationIndexes := func(ctx context.Context, logger cosmoslog.Logger) error {
			logger.Info("Cleaning up orphaned application undelegation indexes")

			count, err := keepers.ApplicationKeeper.CleanupOrphanedUndelegationIndexes(ctx)
			if err != nil {
				logger.Error("Failed to cleanup orphaned undelegation indexes", "error", err)
				return err
			}

			logger.Info("Cleaned up orphaned application undelegation indexes",
				"orphaned_entries_removed", count,
			)
			return nil
		}

		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
			logger := sdkCtx.Logger()

			if err := cleanupOrphanedUndelegationIndexes(ctx, logger); err != nil {
				return vm, err
			}

			return vm, nil
		}
	},
}
