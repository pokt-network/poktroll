package query_test

import (
	"context"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/query"
	"github.com/pokt-network/poktroll/pkg/client/query/cache"
	_ "github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/testutil/mockclient"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

type SharedQuerierTestSuite struct {
	suite.Suite
	ctrl      *gomock.Controller
	ctx       context.Context
	querier   client.SharedQueryClient
	mockConn  *mockclient.MockClientConn
	mockBlock *mockclient.MockCometRPC
	TTL       time.Duration
}

func TestSharedQuerierSuite(t *testing.T) {
	suite.Run(t, new(SharedQuerierTestSuite))
}

func (s *SharedQuerierTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.ctx = context.Background()
	s.mockConn = mockclient.NewMockClientConn(s.ctrl)
	s.mockBlock = mockclient.NewMockCometRPC(s.ctrl)
	s.TTL = 200 * time.Millisecond

	deps := depinject.Supply(s.mockConn, s.mockBlock)

	// Create querier with test-specific cache settings
	querier, err := query.NewSharedQuerier(deps,
		query.WithCacheOptions(
			cache.WithTTL(s.TTL),
			cache.WithHistoricalMode(100),
		),
	)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), querier)

	s.querier = querier
}

func (s *SharedQuerierTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *SharedQuerierTestSuite) TestRetrievesAndCachesParamsValues() {
	multiplier := uint64(1000)

	// First query - params with multiplier 1000
	s.expectMockConnToReturnParamsWithMultiplierOnce(multiplier)

	// Initial query should fetch from chain.
	params1, err := s.querier.GetParams(s.ctx)
	s.NoError(err)
	s.Equal(multiplier, params1.ComputeUnitsToTokensMultiplier)

	// Second query - should use cache, no mock expectation needed, this is
	// asserted here due to the mock expectation calling Times(1).
	params2, err := s.querier.GetParams(s.ctx)
	s.NoError(err)
	s.Equal(multiplier, params2.ComputeUnitsToTokensMultiplier)

	// Third query after 90% of the TTL - should still use cache.
	time.Sleep(time.Duration(float64(s.TTL) * .9))
	params3, err := s.querier.GetParams(s.ctx)
	s.NoError(err)
	s.Equal(multiplier, params3.ComputeUnitsToTokensMultiplier)
}

func (s *SharedQuerierTestSuite) TestHandlesCacheExpiration() {
	// First query
	s.expectMockConnToReturnParamsWithMultiplierOnce(2000)

	params1, err := s.querier.GetParams(s.ctx)
	s.NoError(err)
	s.Equal(uint64(2000), params1.ComputeUnitsToTokensMultiplier)

	// Wait for cache to expire
	time.Sleep(300 * time.Millisecond)

	// Next query should hit the chain again
	s.expectMockConnToReturnParamsWithMultiplierOnce(3000)

	params2, err := s.querier.GetParams(s.ctx)
	s.NoError(err)
	s.Equal(uint64(3000), params2.ComputeUnitsToTokensMultiplier)
}

func (s *SharedQuerierTestSuite) expectMockConnToReturnParamsWithMultiplierOnce(multiplier uint64) {
	s.mockConn.EXPECT().
		Invoke(
			gomock.Any(),
			"/poktroll.shared.Query/Params",
			gomock.Any(),
			gomock.Any(),
			gomock.Any(),
		).
		DoAndReturn(func(_ context.Context, _ string, _, reply any, _ ...grpc.CallOption) error {
			resp := reply.(*sharedtypes.QueryParamsResponse)
			params := sharedtypes.DefaultParams()
			params.ComputeUnitsToTokensMultiplier = multiplier

			resp.Params = params
			return nil
		}).Times(1)
}
