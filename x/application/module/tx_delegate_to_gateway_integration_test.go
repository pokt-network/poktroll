package application_test

import (
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/testutil/integration/suites"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/application/types"
)

type DelegateToGatewayIntegrationTestSuite struct {
	suites.ApplicationModuleSuite
	gatewaySuite suites.GatewayModuleSuite
}

func TestDelegateToGatewayIntegrationSuite(t *testing.T) {
	suite.Run(t, new(DelegateToGatewayIntegrationTestSuite))
}

func (s *DelegateToGatewayIntegrationTestSuite) SetupTest() {
	s.NewApp(s.T())
	s.gatewaySuite.SetApp(s.GetApp())
}

func (s *DelegateToGatewayIntegrationTestSuite) TestDelegateToGateway_Valid() {
	// Prepare addresses
	appAddr := sample.AccAddressBech32()
	gatewayAddr := sample.AccAddressBech32()

	// Fund accounts
	appAccAddr, err := cosmostypes.AccAddressFromBech32(appAddr)
	require.NoError(s.T(), err)
	s.FundAddress(s.T(), appAccAddr, 10000000)

	gatewayAccAddr, err := cosmostypes.AccAddressFromBech32(gatewayAddr)
	require.NoError(s.T(), err)
	s.FundAddress(s.T(), gatewayAccAddr, 10000000)

	// Stake application and gateway
	s.StakeApp(s.T(), appAddr, 1000000, []string{"svc1"})
	s.gatewaySuite.StakeGateway(s.T(), gatewayAddr, 1000)

	// Delegate app to gateway
	res := s.DelegateAppToGateway(s.T(), appAddr, gatewayAddr)
	require.NotNil(s.T(), res)

	// Verify delegation
	appQueryClient := s.GetAppQueryClient(s.T())
	app, err := appQueryClient.GetApplication(s.SdkCtx(), appAddr)
	require.NoError(s.T(), err)
	require.Contains(s.T(), app.DelegateeGatewayAddresses, gatewayAddr)
}

func (s *DelegateToGatewayIntegrationTestSuite) TestDelegateToGateway_InvalidAddress() {
	// Test with invalid application address
	validGatewayAddr := sample.AccAddressBech32()

	msg := types.NewMsgDelegateToGateway("invalid", validGatewayAddr)
	_, err := s.GetApp().RunMsg(s.T(), msg)
	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "invalid application address")
}

func (s *DelegateToGatewayIntegrationTestSuite) TestDelegateToGateway_InvalidGatewayAddress() {
	// Test with invalid gateway address
	validAppAddr := sample.AccAddressBech32()

	// Fund and stake app
	appAccAddr, err := cosmostypes.AccAddressFromBech32(validAppAddr)
	require.NoError(s.T(), err)
	s.FundAddress(s.T(), appAccAddr, 10000000)
	s.StakeApp(s.T(), validAppAddr, 1000000, []string{"svc1"})

	msg := types.NewMsgDelegateToGateway(validAppAddr, "invalid")
	_, err = s.GetApp().RunMsg(s.T(), msg)
	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "invalid gateway address")
}

func (s *DelegateToGatewayIntegrationTestSuite) TestDelegateToGateway_AppNotStaked() {
	// Test delegation with unstaked application
	appAddr := sample.AccAddressBech32()
	gatewayAddr := sample.AccAddressBech32()

	// Fund and stake only gateway
	gatewayAccAddr, err := cosmostypes.AccAddressFromBech32(gatewayAddr)
	require.NoError(s.T(), err)
	s.FundAddress(s.T(), gatewayAccAddr, 10000000)
	s.gatewaySuite.StakeGateway(s.T(), gatewayAddr, 1000)

	// Try to delegate unstaked app
	msg := types.NewMsgDelegateToGateway(appAddr, gatewayAddr)
	_, err = s.GetApp().RunMsg(s.T(), msg)
	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "not found")
}

func (s *DelegateToGatewayIntegrationTestSuite) TestDelegateToGateway_GatewayNotStaked() {
	// Test delegation to unstaked gateway
	appAddr := sample.AccAddressBech32()
	gatewayAddr := sample.AccAddressBech32()

	// Fund and stake only app
	appAccAddr, err := cosmostypes.AccAddressFromBech32(appAddr)
	require.NoError(s.T(), err)
	s.FundAddress(s.T(), appAccAddr, 10000000)
	s.StakeApp(s.T(), appAddr, 1000000, []string{"svc1"})

	// Try to delegate to unstaked gateway
	msg := types.NewMsgDelegateToGateway(appAddr, gatewayAddr)
	_, err = s.GetApp().RunMsg(s.T(), msg)
	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "not found")
}
