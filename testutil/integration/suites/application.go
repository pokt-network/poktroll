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

type ApplicationModuleSuite struct {
	BaseIntegrationSuite
}

func (s *ApplicationModuleSuite) GetAppQueryClient() client.ApplicationQueryClient {
	deps := depinject.Supply(s.GetApp().QueryHelper())
	appQueryClient, err := query.NewApplicationQuerier(deps)
	require.NoError(s.T(), err)

	return appQueryClient
}

// TODO_IN_THIS_COMMIT: godoc
func (s *ApplicationModuleSuite) StakeApp(
	t *testing.T,
	appBech32 string,
	appStakeAmount int64,
	serviceIds []string,
) *apptypes.MsgStakeApplicationResponse {
	t.Helper()

	serviceConfigs := make([]*sharedtypes.ApplicationServiceConfig, len(serviceIds))
	for serviceIdx, serviceId := range serviceIds {
		serviceConfigs[serviceIdx] = &sharedtypes.ApplicationServiceConfig{ServiceId: serviceId}
	}

	stakeAppMsg := apptypes.NewMsgStakeApplication(
		appBech32,
		cosmostypes.NewInt64Coin(volatile.DenomuPOKT, appStakeAmount),
		serviceConfigs,
	)

	txMsgRes, err := s.GetApp().RunMsg(t, stakeAppMsg)
	require.NoError(t, err)

	return txMsgRes.(*apptypes.MsgStakeApplicationResponse)
}

// TODO_IN_THIS_COMMIT: godoc
func (s *ApplicationModuleSuite) Transfer(
	t *testing.T,
	srcAddr, dstAddr cosmostypes.AccAddress,
) *apptypes.MsgTransferApplicationResponse {
	t.Helper()

	msgTransferApp := &apptypes.MsgTransferApplication{
		SourceAddress:      srcAddr.String(),
		DestinationAddress: dstAddr.String(),
	}

	txMsgRes, err := s.GetApp().RunMsg(t, msgTransferApp)
	require.NoError(t, err)

	return txMsgRes.(*apptypes.MsgTransferApplicationResponse)
}

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
