package keeper

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
)

// EndBlocker maintains the anchored-session-grid invariant (#543, Option B). At each block,
// if a params-history entry became effective at exactly the current height, that entry is
// promoted to the live params via SetParams. The common path (no epoch becomes effective at
// this height) is a pure no-op with no state write, so it does not alter app hashes on
// blocks without a promotion.
//
// SCOPE OF THE INVARIANT — `live params == currently-effective epoch` holds for the
// SESSION-TIMING params only (num_blocks_per_session and the session/claim/proof window
// offsets; see sessionTimingParamsChanged) plus the derived grid-anchor metadata. Those are
// the fields the anchored grid is about, and UpdateParam withholds their live write until
// this promotion fires.
//
// It does NOT hold field-by-field for the rest of the params struct. UpdateParam records
// EVERY change in history at the next session boundary, but writes non-timing params
// (unbonding periods, compute-unit economics) to live IMMEDIATELY. So between such a change
// and the next boundary, live carries the new value while GetParamsAtHeight(h) for h in the
// current session still resolves to the old one. That window is intentional (legacy
// behavior) and harmless for grid math, but it means:
//
//   - Do NOT treat live params as "the epoch effective right now" for a non-timing field.
//   - Anything PRICING a claim must read GetParamsAtHeight at the claim's session start, so
//     it follows the boundary semantics that history actually records. x/proof does this in
//     all three of its sites, x/tokenomics settlement does it via
//     settlementContext.GetSharedParamsAtHeight(sessionStart), and the RelayMiner mirrors both.
//
// CRITICAL ORDERING (app/app_config.go endBlockers): the shared module MUST run AFTER every
// module that reads live shared params (service, session, proof, tokenomics, gateway,
// application, supplier). At the boundary block `anchor`, those consumers run with the OLD
// (current-epoch) live params; shared then promotes; block `anchor+1` onward sees the new
// params. Promotion fires on `effective_height == currentHeight` (NOT currentHeight+1):
// promoting one block early would make the LAST old-epoch block settle/unbond with the new
// N and lose funds. See spec §4.7.1.
func (k Keeper) EndBlocker(ctx context.Context) error {
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	newParams, found := k.GetParamsHistoryEntry(ctx, currentHeight)
	if !found {
		// Common path: no params epoch becomes effective at this height → no-op.
		return nil
	}

	// Validate before promoting. SetParams marshals and writes with NO validation, so
	// promotion is the one path that can install a live param set the MsgUpdateParam(s)
	// handlers would have rejected -- including one whose four claim/proof window offsets
	// are all zero, which makes GetNumPendingSessions() zero and divides by zero in the
	// settlement EndBlocker. Governance cannot get such an entry into history (that path
	// validates), so this guards the remaining writer: an upgrade handler calling
	// SetParamsAtHeight directly.
	//
	// Log and skip rather than return the error: this runs in the EndBlocker, so returning
	// it would halt the chain at the promotion height -- exactly the outcome the guard
	// exists to prevent. Declining to promote leaves the previous live params in place,
	// which are known-valid and keep the chain running on the old epoch.
	if err := newParams.ValidateBasic(); err != nil {
		k.logger.Error(fmt.Sprintf(
			"REFUSING to promote invalid params epoch at effective_height=%d: %s; keeping the previous live params",
			currentHeight, err,
		))
		return nil
	}

	k.logger.Info("promoting params epoch to live (anchored session grid)",
		"effective_height", currentHeight,
		"num_blocks_per_session", newParams.GetNumBlocksPerSession(),
		"session_grid_anchor_height", newParams.GetSessionGridAnchorHeight(),
	)

	return k.SetParams(ctx, newParams)
}
