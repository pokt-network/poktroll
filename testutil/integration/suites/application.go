package suites

import (
	"testing"

	"cosmossdk.io/depinject"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/query"
	"github.com/pokt-network/poktroll/testutil/integration"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ IntegrationSuite = (*ApplicationModuleSuite)(nil)

type ApplicationModuleSuite struct {
	BaseIntegrationSuite

	appQueryClient client.ApplicationQueryClient
}

func (s *ApplicationModuleSuite) SetupTest() {
	// Construct a new application query client.
	var err error
	deps := depinject.Supply(s.GetApp().QueryHelper())
	s.appQueryClient, err = query.NewApplicationQuerier(deps)
	require.NoError(s.T(), err)
}

func (s *ApplicationModuleSuite) GetAppQueryClient() client.ApplicationQueryClient {
	return s.appQueryClient
}

// TODO_IN_THIS_COMMIT: godoc
func (s *ApplicationModuleSuite) StakeApp(
	t *testing.T,
	appBech32 string,
	appStakeAmount int64,
	services ...*sharedtypes.Service,
) *apptypes.MsgStakeApplicationResponse {
	t.Helper()

	serviceConfigs := make([]*sharedtypes.ApplicationServiceConfig, len(services))
	for serviceIdx, service := range services {
		serviceConfigs[serviceIdx] = &sharedtypes.ApplicationServiceConfig{ServiceId: service.GetId()}
	}

	stakeAppMsg := apptypes.NewMsgStakeApplication(
		appBech32,
		cosmostypes.NewInt64Coin(volatile.DenomuPOKT, appStakeAmount),
		serviceConfigs,
	)

	anyRes := s.GetApp().RunMsg(t,
		stakeAppMsg,
		integration.RunUntilNextBlockOpts...,
	)
	require.NotNil(t, anyRes)

	stakeAppRes := new(apptypes.MsgStakeApplicationResponse)
	err := s.GetApp().GetCodec().UnpackAny(anyRes, &stakeAppRes)
	require.NoError(t, err)

	return stakeAppRes
}

// TODO_IN_THIS_COMMIT: godoc
func (s *ApplicationModuleSuite) Transfer(
	t *testing.T,
	srcAddr, dstAddr cosmostypes.AccAddress,
	opts ...integration.RunOption,
) *apptypes.MsgTransferApplicationResponse {
	msgTransferApp := &apptypes.MsgTransferApplication{
		SourceAddress:      srcAddr.String(),
		DestinationAddress: dstAddr.String(),
	}

	opts = integration.RunUntilNextBlockOpts.Append(opts...)
	anyRes := s.GetApp().RunMsg(t, msgTransferApp, opts...)

	transferRes := new(apptypes.MsgTransferApplicationResponse)
	err := s.GetApp().GetCodec().UnpackAny(anyRes, &transferRes)
	require.NoError(t, err)

	return transferRes
}
