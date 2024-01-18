package keeper

import (
	"context"

	abci "github.com/cometbft/cometbft/abci/types"
)

// BeginBlocker is called at the start of each block and  will execute any
// Redelegation's from the queue that should be processed in this block,
// returning any error from doing so.
func (k *Keeper) BeginBlocker(ctx context.Context) error {
	return k.ProcessRedelegations(ctx)
}
