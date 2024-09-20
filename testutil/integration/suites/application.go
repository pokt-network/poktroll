//go:build integration

package suites

import (
	"testing"

	"cosmossdk.io/depinject"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/query"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ IntegrationSuite = (*ApplicationModuleSuite)(nil)

// ApplicationModuleSuite is a test suite which abstracts common application module
// functionality. It is intended to be embedded in dependent integration test suites.
type ApplicationModuleSuite struct {
	BaseIntegrationSuite
}

// GetAppQueryClient constructs and returns a query client for the application
// module of the integration app.
func (s *ApplicationModuleSuite) GetAppQueryClient() client.ApplicationQueryClient {
	deps := depinject.Supply(s.GetApp().QueryHelper())
	appQueryClient, err := query.NewApplicationQuerier(deps)
	require.NoError(s.T(), err)

	return appQueryClient
}

// StakeApp sends a MsgStakeApplication with the given bech32 address,
// stake amount, and service IDs.
func (s *ApplicationModuleSuite) StakeApp(
	t *testing.T,
	bech32 string,
	stakeAmtUpokt int64,
	serviceIds []string,
) *apptypes.MsgStakeApplicationResponse {
	t.Helper()

	serviceConfigs := make([]*sharedtypes.ApplicationServiceConfig, len(serviceIds))
	for serviceIdx, serviceId := range serviceIds {
		serviceConfigs[serviceIdx] = &sharedtypes.ApplicationServiceConfig{ServiceId: serviceId}
	}

	stakeAppMsg := apptypes.NewMsgStakeApplication(
		bech32,
		cosmostypes.NewInt64Coin(volatile.DenomuPOKT, stakeAmtUpokt),
		serviceConfigs,
	)

	txMsgRes, err := s.GetApp().RunMsg(t, stakeAppMsg)
	require.NoError(t, err)

	return txMsgRes.(*apptypes.MsgStakeApplicationResponse)
}

// Transfer sends a MsgApplicationTransfer to begin an application transfer
// from srcBech32 to dstDst32.
func (s *ApplicationModuleSuite) Transfer(
	t *testing.T,
	srcBech32, dstBech32 string,
) *apptypes.MsgTransferApplicationResponse {
	t.Helper()

	msgTransferApp := &apptypes.MsgTransferApplication{
		SourceAddress:      srcBech32,
		DestinationAddress: dstBech32,
	}

	txMsgRes, err := s.GetApp().RunMsg(t, msgTransferApp)
	require.NoError(t, err)

	return txMsgRes.(*apptypes.MsgTransferApplicationResponse)
}

// MultiTransfer sends multiple MsgTransferApplication messages to transfer
// applications from the source to the destination addresses specified in the
// srcToDstBech32Map. All transfer messages are included in a single transaction.
func (s *ApplicationModuleSuite) MultiTransfer(
	t *testing.T,
	srcToDstBech32Map map[string]string,
) (transferResps []*apptypes.MsgTransferApplicationResponse) {
	t.Helper()

	var msgs []cosmostypes.Msg
	for srcBech32, dstBech32 := range srcToDstBech32Map {
		msgs = append(msgs, &apptypes.MsgTransferApplication{
			SourceAddress:      srcBech32,
			DestinationAddress: dstBech32,
		})
	}

	txMsgResps, err := s.GetApp().RunMsgs(t, msgs...)
	require.NoError(t, err)

	for _, txMsgRes := range txMsgResps {
		transferRes, ok := txMsgRes.(*apptypes.MsgTransferApplicationResponse)
		require.Truef(t, ok, "unexpected txMsgRes type: %T", txMsgRes)
		transferResps = append(transferResps, transferRes)
	}

	return transferResps
}

// DelegateAppToGateway sends a MsgDelegateToGateway to delegate the application
// with the given bech32 address to the gateway with the given bech32 address.
func (s *ApplicationModuleSuite) DelegateAppToGateway(
	t *testing.T,
	appBech32, gatewayBech32 string,
) *apptypes.MsgDelegateToGatewayResponse {
	t.Helper()

	delegateAppToGatewayMsg := apptypes.NewMsgDelegateToGateway(appBech32, gatewayBech32)
	txMsgRes, err := s.GetApp().RunMsg(t, delegateAppToGatewayMsg)
	require.NoError(t, err)

	return txMsgRes.(*apptypes.MsgDelegateToGatewayResponse)
}
