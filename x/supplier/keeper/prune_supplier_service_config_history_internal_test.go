package keeper

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

// zeroSessionLengthSharedKeeper returns params with a zero session length at
// every height, which only an unvalidated genesis params history can contain.
type zeroSessionLengthSharedKeeper struct {
	types.SharedKeeper
}

func (zeroSessionLengthSharedKeeper) GetParamsAtHeight(context.Context, int64) sharedtypes.Params {
	return sharedtypes.Params{}
}

func (zeroSessionLengthSharedKeeper) IterateParamsHistoryReverse(context.Context, int64, func(int64, sharedtypes.Params) bool) {
}

// A zero session length keeps the service config instead of panicking.
func TestLastProofWindowCloseHeight_ZeroSessionLengthKeepsConfig(t *testing.T) {
	k, ctx := newMinimalKeeper(t)
	k.sharedKeeper = zeroSessionLengthSharedKeeper{}

	require.Equal(t, int64(math.MaxInt64), k.lastProofWindowCloseHeight(ctx, 1, 10))
}
