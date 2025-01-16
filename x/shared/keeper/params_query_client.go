package keeper

import (
	"context"
	"fmt"

	"github.com/pokt-network/poktroll/pkg/client"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ client.ParamsQuerier[*sharedtypes.Params] = (*keeperParamsQuerier[sharedtypes.Params, Keeper])(nil)

// DEV_NOTE: Can't use cosmostypes.Msg instead of any because P
// would be a pointer but GetParams() returns a value. 🙄
type paramsKeeperIface[P any] interface {
	GetParams(context.Context) P
}

// keeperParamsQuerier provides a base implementation of ParamsQuerier for keeper-based clients
type keeperParamsQuerier[P any, K paramsKeeperIface[P]] struct {
	keeper K
}

// NewKeeperParamsQuerier creates a new keeperParamsQuerier instance
func NewKeeperParamsQuerier[P any, K paramsKeeperIface[P]](
	keeper K,
) (*keeperParamsQuerier[P, K], error) {
	return &keeperParamsQuerier[P, K]{
		keeper: keeper,
	}, nil
}

// GetParams retrieves current parameters from the keeper
func (kpq *keeperParamsQuerier[P, K]) GetParams(ctx context.Context) (*P, error) {
	params := kpq.keeper.GetParams(ctx)
	return &params, nil
}

// GetParamsAtHeight retrieves parameters as they were at a specific height
//
// TODO_MAINNET(@bryanchriswhite, #931): Integrate with indexer module/mixin once available.
// Currently, this method is (and MUST) NEVER called on-chain and only exists to satisfy the
// client.ParamsQuerier interface. However, it will be needed as part of #931 to support
// querying for params at historical heights, so it's short-circuited for now to always
// return an error.
func (kpq *keeperParamsQuerier[P, K]) GetParamsAtHeight(_ context.Context, _ int64) (*P, error) {
	return nil, fmt.Errorf("TODO(#931): Support on-chain historical queries")
}

// TODO_IN_THIS_COMMIT: godoc...
func (kpq *keeperParamsQuerier[P, K]) SetParamsAtHeight(_ context.Context, height int64, params *P) error {
	// TODO_IN_THIS_COMMIT: this will be called on-chain once we have on-chain historical data but it will reference the historical keeper/mix-in method(s).
	return fmt.Errorf("TODO(#931): Support on-chain historical caching")
}
