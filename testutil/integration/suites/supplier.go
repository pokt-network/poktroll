package suites

import (
	"testing"

	"cosmossdk.io/depinject"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/pocket/app/volatile"
	"github.com/pokt-network/pocket/pkg/client"
	"github.com/pokt-network/pocket/pkg/client/query"
	"github.com/pokt-network/pocket/pkg/polylog/polyzero"
	"github.com/pokt-network/pocket/testutil/testcache"
	sharedtypes "github.com/pokt-network/pocket/x/shared/types"
	suppliertypes "github.com/pokt-network/pocket/x/supplier/types"
)

var _ IntegrationSuite = (*SupplierModuleSuite)(nil)

// SupplierModuleSuite is a test suite which abstracts common supplier module
// functionality. It is intended to be embedded in dependent integration test suites.
type SupplierModuleSuite struct {
	BaseIntegrationSuite
}

// GetSupplierQueryClient constructs and returns a query client for the supplier
// module of the integration app.
func (s *SupplierModuleSuite) GetSupplierQueryClient(t *testing.T) client.SupplierQueryClient {
	deps := depinject.Supply(
		s.GetApp().QueryHelper(),
		polyzero.NewLogger(),
		testcache.NewNoopKeyValueCache[sharedtypes.Supplier](),
	)
	supplierQueryClient, err := query.NewSupplierQuerier(deps)
	require.NoError(t, err)

	return supplierQueryClient
}

// StakeSupplier sends a MsgStakeSupplier with the given bech32 address,
// stake amount, and service IDs.
func (s *SupplierModuleSuite) StakeSupplier(
	t *testing.T,
	supplierAddress string,
	stakeAmtUpokt int64,
	serviceIds []string,
) *suppliertypes.MsgStakeSupplierResponse {
	t.Helper()

	serviceConfigs := make([]*sharedtypes.SupplierServiceConfig, len(serviceIds))
	for serviceIdx, serviceId := range serviceIds {
		serviceConfigs[serviceIdx] = &sharedtypes.SupplierServiceConfig{
			ServiceId: serviceId,
			Endpoints: []*sharedtypes.SupplierEndpoint{
				{
					Url:     "http://test.example:1234",
					RpcType: sharedtypes.RPCType_JSON_RPC,
				},
			},
			RevShare: []*sharedtypes.ServiceRevenueShare{
				{
					Address:            supplierAddress,
					RevSharePercentage: 100,
				},
			},
		}
	}

	stakeSupplierMsg := suppliertypes.NewMsgStakeSupplier(
		supplierAddress,
		supplierAddress,
		supplierAddress,
		cosmostypes.NewInt64Coin(volatile.DenomuPOKT, stakeAmtUpokt),
		serviceConfigs,
	)

	txMsgRes, err := s.GetApp().RunMsg(t, stakeSupplierMsg)
	require.NoError(t, err)

	return txMsgRes.(*suppliertypes.MsgStakeSupplierResponse)
}
