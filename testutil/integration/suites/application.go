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
func (s *ApplicationModuleSuite) GetAppQueryClient(t *testing.T) client.ApplicationQueryClient {
	deps := depinject.Supply(s.GetApp().QueryHelper())
	appQueryClient, err := query.NewApplicationQuerier(deps)
	require.NoError(t, err)

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
// from srcBech32 to dstBech32.
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

// DelegateAppsToGateways sends multiple MsgDelegateToGateway messages to delegate
// applications to gateways. The applications and gateways are specified in the
// appToGatewayBech32Map. All delegate messages are included in a single transaction.
func (s *ApplicationModuleSuite) DelegateAppsToGateways(
	t *testing.T,
	appToGatewayBech32Map map[string][]string,
) (delegateResps []*apptypes.MsgDelegateToGatewayResponse) {
	t.Helper()

	var delegateAppToGatewayMsgs []cosmostypes.Msg
	for appBech32, gatewayBech32s := range appToGatewayBech32Map {
		for _, gatewayBech32 := range gatewayBech32s {
			delegateAppToGatewayMsgs = append(
				delegateAppToGatewayMsgs,
				apptypes.NewMsgDelegateToGateway(appBech32, gatewayBech32),
			)
		}
	}

	txMsgResps, err := s.GetApp().RunMsgs(t, delegateAppToGatewayMsgs...)
	require.NoError(t, err)

	for _, txMsgRes := range txMsgResps {
		delegateRes, ok := txMsgRes.(*apptypes.MsgDelegateToGatewayResponse)
		require.Truef(t, ok, "unexpected txMsgRes type: %T", txMsgRes)
		delegateResps = append(delegateResps, delegateRes)
	}

	return delegateResps
}

// UndelegateAppFromGateway sends a MsgUndelegateFromGateway to undelegate the
// application with address appBech32 from the gateway with address gatewayBech32.
func (s *ApplicationModuleSuite) UndelegateAppFromGateway(
	t *testing.T,
	appBech32, gatewayBech32 string,
) *apptypes.MsgUndelegateFromGatewayResponse {
	t.Helper()

	undelegateAppFromGatewayMsg := apptypes.NewMsgUndelegateFromGateway(appBech32, gatewayBech32)
	txMsgRes, err := s.GetApp().RunMsg(t, undelegateAppFromGatewayMsg)
	require.NoError(t, err)

	return txMsgRes.(*apptypes.MsgUndelegateFromGatewayResponse)
}

// UndelegateAppsFromGateways sends multiple MsgUndelegateFromGateway messages to
// undelegate applications from gateways. The applications and gateways are specified
// in the appToGatewayBech32Map. All undelegate messages are included in a single transaction.
func (s *ApplicationModuleSuite) UndelegateAppsFromGateways(
	t *testing.T,
	appToGatewayBech32Map map[string][]string,
) (undelegateResps []*apptypes.MsgUndelegateFromGatewayResponse) {
	t.Helper()

	var undelegateAppFromGatewayMsgs []cosmostypes.Msg
	for appBech32, gatewayBech32s := range appToGatewayBech32Map {
		for _, gatewayBech32 := range gatewayBech32s {
			undelegateAppFromGatewayMsgs = append(
				undelegateAppFromGatewayMsgs,
				apptypes.NewMsgUndelegateFromGateway(appBech32, gatewayBech32),
			)
		}
	}

	txMsgResps, err := s.GetApp().RunMsgs(t, undelegateAppFromGatewayMsgs...)
	require.NoError(t, err)

	for _, txMsgRes := range txMsgResps {
		undelegateRes, ok := txMsgRes.(*apptypes.MsgUndelegateFromGatewayResponse)
		require.Truef(t, ok, "unexpected txMsgRes type: %T", txMsgRes)
		undelegateResps = append(undelegateResps, undelegateRes)
	}

	return undelegateResps
}
