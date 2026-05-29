package keeper

import (
	"context"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
)

// EndBlocker maintains the anchored-session-grid invariant `live params == currently-
// effective epoch` (#543, Option B). At each block, if a params-history entry became
// effective at exactly the current height, that entry is promoted to the live params via
// SetParams. The common path (no epoch becomes effective at this height) is a pure no-op
// with no state write, so it does not alter app hashes on blocks without a promotion.
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

	k.logger.Info("promoting params epoch to live (anchored session grid)",
		"effective_height", currentHeight,
		"num_blocks_per_session", newParams.GetNumBlocksPerSession(),
		"session_grid_anchor_height", newParams.GetSessionGridAnchorHeight(),
	)

	return k.SetParams(ctx, newParams)
}
