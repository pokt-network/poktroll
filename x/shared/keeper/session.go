package keeper

import (
	"context"

	"github.com/pokt-network/poktroll/x/shared"
)

func (k Keeper) GetSessionStartHeight(ctx context.Context, queryHeight int64) int64 {
	sharedParams := k.GetParams(ctx)
	return shared.GetSessionStartHeight(&sharedParams, queryHeight)
}

func (k Keeper) GetSessionEndHeight(ctx context.Context, queryHeight int64) int64 {
	sharedParams := k.GetParams(ctx)
	return shared.GetSessionEndHeight(&sharedParams, queryHeight)
}

func (k Keeper) GetSessionNumber(ctx context.Context, queryHeight int64) int64 {
	sharedParams := k.GetParams(ctx)
	return shared.GetSessionNumber(&sharedParams, queryHeight)
}
