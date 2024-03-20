package keeper

import (
	"context"
)

// EndBlocker called at every block and settles all pending claims.
func (k Keeper) EndBlocker(ctx context.Context) error {
	return k.SettlePendingClaims(ctx, k.environment)
}
