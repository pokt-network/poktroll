package suites

import (
	"math"
	"testing"

	"cosmossdk.io/depinject"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/cache/memory"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/query"
	querycache "github.com/pokt-network/poktroll/pkg/client/query/cache"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/testutil/testcache"
	"github.com/pokt-network/poktroll/testutil/testclient/testeventsquery"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ IntegrationSuite = (*ServiceModuleSuite)(nil)

// ServiceModuleSuite is a test suite which abstracts common service module
// functionality. It is intended to be embedded in dependent integration test suites.
type ServiceModuleSuite struct {
	BaseIntegrationSuite
}

// AddService adds an on-chain service.
func (s *ServiceModuleSuite) AddService(
	t *testing.T,
	serviceId,
	ownerAddress string,
	computeUnitsPerRelay uint64,
) {
	t.Helper()

	msgAddService := servicetypes.MsgAddService{
		OwnerAddress: ownerAddress,
		Service: sharedtypes.Service{
			Id:                   serviceId,
			OwnerAddress:         ownerAddress,
			ComputeUnitsPerRelay: computeUnitsPerRelay,
		},
	}
	_, err := s.GetApp().RunMsg(t, &msgAddService)
	require.NoError(t, err)
}

// GetServiceQueryClient returns a query client for the service module.
func (s *ServiceModuleSuite) GetServiceQueryClient(t *testing.T) client.ServiceQueryClient {
	t.Helper()

	paramsCache, err := querycache.NewParamsCache[servicetypes.Params](memory.WithTTL(math.MaxInt64))
	require.NoError(t, err)

	deps := depinject.Supply(
		s.GetApp().QueryHelper(),
		polyzero.NewLogger(),
		testeventsquery.NewAnyTimesEventsParamsActivationClient(t),
		testcache.NewNoopKeyValueCache[sharedtypes.Service](),
		testcache.NewNoopKeyValueCache[servicetypes.RelayMiningDifficulty](),
		paramsCache,
	)
	ctx := s.GetApp().QueryHelper().Ctx
	serviceClient, err := query.NewServiceQuerier(ctx, deps)
	require.NoError(t, err)

	return serviceClient
}
