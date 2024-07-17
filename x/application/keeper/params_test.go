package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/proto/types/application"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
)

func TestGetParams(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	params := application.DefaultParams()

	require.NoError(t, k.SetParams(ctx, params))
	require.EqualValues(t, params, k.GetParams(ctx))
}
