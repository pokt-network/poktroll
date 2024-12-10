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
	ctrl            *gomock.Controller
	ctx             context.Context
	querier         client.SharedQueryClient
	TTL             time.Duration
	clientConnMock  *mockclient.MockClientConn
	blockClientMock *mockclient.MockCometRPC
}

func TestSharedQuerierSuite(t *testing.T) {
	suite.Run(t, new(SharedQuerierTestSuite))
}

func (s *SharedQuerierTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.ctx = context.Background()
	s.clientConnMock = mockclient.NewMockClientConn(s.ctrl)
	s.blockClientMock = mockclient.NewMockCometRPC(s.ctrl)
	s.TTL = 200 * time.Millisecond

	deps := depinject.Supply(s.clientConnMock, s.blockClientMock)

	// Create a shared querier with test-specific cache settings.
	querier, err := query.NewSharedQuerier(deps,
		query.WithQueryCacheOptions(
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

	s.expectMockConnToReturnParamsWithMultiplierOnce(multiplier)

	// Initial get should be a cache miss.
	params1, err := s.querier.GetParams(s.ctx)
	s.NoError(err)
	s.Equal(multiplier, params1.ComputeUnitsToTokensMultiplier)

	// Second get should be a cache hit.
	params2, err := s.querier.GetParams(s.ctx)
	s.NoError(err)
	s.Equal(multiplier, params2.ComputeUnitsToTokensMultiplier)

	// Third get, after 90% of the TTL - should still be a cache hit.
	time.Sleep(time.Duration(float64(s.TTL) * .9))
	params3, err := s.querier.GetParams(s.ctx)
	s.NoError(err)
	s.Equal(multiplier, params3.ComputeUnitsToTokensMultiplier)
}

func (s *SharedQuerierTestSuite) TestHandlesCacheExpiration() {
	s.expectMockConnToReturnParamsWithMultiplierOnce(2000)

	params1, err := s.querier.GetParams(s.ctx)
	s.NoError(err)
	s.Equal(uint64(2000), params1.ComputeUnitsToTokensMultiplier)

	// Wait for cache to expire
	time.Sleep(300 * time.Millisecond)

	// Next query should be a cache miss again.
	s.expectMockConnToReturnParamsWithMultiplierOnce(3000)

	params2, err := s.querier.GetParams(s.ctx)
	s.NoError(err)
	s.Equal(uint64(3000), params2.ComputeUnitsToTokensMultiplier)
}

// expectMockConnToReturnParamsWithMultiplerOnce registers an expectation on s.clientConnMock
// such that this test will fail if the mock connection doesn't see exactly one params request.
// When it does see the params request, it will respond with a sharedtypes.Params object where
// the ComputeUnitsToTokensMultiplier field is set to the given multiplier.
func (s *SharedQuerierTestSuite) expectMockConnToReturnParamsWithMultiplierOnce(multiplier uint64) {
	s.clientConnMock.EXPECT().
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
