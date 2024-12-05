package query_test

import (
	"context"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/query"
	_ "github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/testutil/mockclient"
	"github.com/pokt-network/poktroll/testutil/mockgrpc"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func setupTest(t *testing.T) (client.SharedQueryClient, *mockgrpc.MockClientConn, *mockclient.MockCometRPC) {
	ctrl := gomock.NewController(t)

	mockConn := mockgrpc.NewMockClientConn(ctrl)
	mockBlock := mockclient.NewMockCometRPC(ctrl)

	cfg := depinject.Supply(mockConn, mockBlock)

	querier, err := query.NewSharedQuerier(cfg)
	require.NoError(t, err)
	require.NotNil(t, querier)

	return querier, mockConn, mockBlock
}

func TestSharedQuerier_ParamsHistoricalValues(t *testing.T) {
	ctx := context.Background()
	querier, mockConn, _ := setupTest(t)

	// Helper function to create params with specific values
	createParamsWithMultiplier := func(multiplier uint64) *sharedtypes.Params {
		return &sharedtypes.Params{
			NumBlocksPerSession:             100,
			GracePeriodEndOffsetBlocks:      10,
			ClaimWindowOpenOffsetBlocks:     20,
			ClaimWindowCloseOffsetBlocks:    30,
			ProofWindowOpenOffsetBlocks:     40,
			ProofWindowCloseOffsetBlocks:    50,
			SupplierUnbondingPeriodSessions: 5,
			ComputeUnitsToTokensMultiplier:  multiplier,
		}
	}

	t.Run("retrieves and caches params values", func(t *testing.T) {
		// First query - params with multiplier 1000
		mockConn.EXPECT().
			Invoke(
				gomock.Any(),
				"/poktroll.shared.Query/Params",
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).
			DoAndReturn(func(_ context.Context, _ string, _ interface{}, reply interface{}, _ ...grpc.CallOption) error {
				resp := reply.(*sharedtypes.QueryParamsResponse)
				resp.Params = *createParamsWithMultiplier(1000)
				return nil
			}).Times(1)

		// Initial query should fetch from chain
		params1, err := querier.GetParams(ctx)
		require.NoError(t, err)
		require.Equal(t, uint64(1000), params1.ComputeUnitsToTokensMultiplier)

		// Second query - should use cache, no mock expectation needed
		params2, err := querier.GetParams(ctx)
		require.NoError(t, err)
		require.Equal(t, uint64(1000), params2.ComputeUnitsToTokensMultiplier)

		// Third query after small delay - should still use cache
		time.Sleep(100 * time.Millisecond)
		params3, err := querier.GetParams(ctx)
		require.NoError(t, err)
		require.Equal(t, uint64(1000), params3.ComputeUnitsToTokensMultiplier)
	})

	t.Run("handles cache expiration", func(t *testing.T) {
		// First query
		mockConn.EXPECT().
			Invoke(
				gomock.Any(),
				"/poktroll.shared.Query/Params",
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).
			DoAndReturn(func(_ context.Context, _ string, _ interface{}, reply interface{}, _ ...grpc.CallOption) error {
				resp := reply.(*sharedtypes.QueryParamsResponse)
				resp.Params = *createParamsWithMultiplier(2000)
				return nil
			}).Times(1)

		params1, err := querier.GetParams(ctx)
		require.NoError(t, err)
		require.Equal(t, uint64(2000), params1.ComputeUnitsToTokensMultiplier)

		// Wait for cache to expire
		time.Sleep(2 * time.Hour)

		// Next query should hit the chain again
		mockConn.EXPECT().
			Invoke(
				gomock.Any(),
				"/poktroll.shared.Query/Params",
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).
			DoAndReturn(func(_ context.Context, _ string, _ interface{}, reply interface{}, _ ...grpc.CallOption) error {
				resp := reply.(*sharedtypes.QueryParamsResponse)
				resp.Params = *createParamsWithMultiplier(3000)
				return nil
			}).Times(1)

		params2, err := querier.GetParams(ctx)
		require.NoError(t, err)
		require.Equal(t, uint64(3000), params2.ComputeUnitsToTokensMultiplier)
	})
}
