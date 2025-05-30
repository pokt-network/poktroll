package suites

import (
	"testing"

	"cosmossdk.io/depinject"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/query"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/testutil/testcache"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ IntegrationSuite = (*ServiceModuleSuite)(nil)

// ServiceModuleSuite is a test suite which abstracts common service module
// functionality. It is intended to be embedded in dependent integration test suites.
type ServiceModuleSuite struct {
	BaseIntegrationSuite
}

// SetupService adds or updates an on-chain service.
func (s *ServiceModuleSuite) SetupService(
	t *testing.T,
	serviceId,
	ownerAddress string,
	computeUnitsPerRelay uint64,
) {
	t.Helper()

	msgSetupService := servicetypes.MsgSetupService{
		Signer: ownerAddress,
		Service: sharedtypes.Service{
			Id:                   serviceId,
			OwnerAddress:         ownerAddress,
			ComputeUnitsPerRelay: computeUnitsPerRelay,
		},
	}
	_, err := s.GetApp().RunMsg(t, &msgSetupService)
	require.NoError(t, err)
}

// GetServiceQueryClient returns a query client for the service module.
func (s *ServiceModuleSuite) GetServiceQueryClient(t *testing.T) client.ServiceQueryClient {
	t.Helper()

	deps := depinject.Supply(
		s.GetApp().QueryHelper(),
		polyzero.NewLogger(),
		testcache.NewNoopKeyValueCache[sharedtypes.Service](),
		testcache.NewNoopKeyValueCache[servicetypes.RelayMiningDifficulty](),
		testcache.NewNoopParamsCache[servicetypes.Params](),
	)
	serviceClient, err := query.NewServiceQuerier(deps)
	require.NoError(t, err)

	return serviceClient
}
