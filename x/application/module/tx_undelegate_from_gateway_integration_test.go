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

type UndelegateFromGatewayIntegrationTestSuite struct {
	suites.ApplicationModuleSuite
	gatewaySuite suites.GatewayModuleSuite
}

func TestUndelegateFromGatewayIntegrationSuite(t *testing.T) {
	suite.Run(t, new(UndelegateFromGatewayIntegrationTestSuite))
}

func (s *UndelegateFromGatewayIntegrationTestSuite) SetupTest() {
	s.NewApp(s.T())
	s.gatewaySuite.SetApp(s.GetApp())
}

func (s *UndelegateFromGatewayIntegrationTestSuite) TestUndelegateFromGateway_Valid() {
	// Prepare addresses
	appAddr := sample.AccAddress()
	gatewayAddr := sample.AccAddress()
	
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
	
	// First delegate app to gateway
	s.DelegateAppToGateway(s.T(), appAddr, gatewayAddr)
	
	// Verify delegation exists
	appQueryClient := s.GetAppQueryClient(s.T())
	app, err := appQueryClient.GetApplication(s.SdkCtx(), appAddr)
	require.NoError(s.T(), err)
	require.Contains(s.T(), app.DelegateeGatewayAddresses, gatewayAddr)
	
	// Undelegate app from gateway
	res := s.UndelegateAppFromGateway(s.T(), appAddr, gatewayAddr)
	require.NotNil(s.T(), res)
	
	// Verify undelegation - check pending undelegations
	app, err = appQueryClient.GetApplication(s.SdkCtx(), appAddr)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), app.PendingUndelegations)
	
	// Check if gateway is in any of the pending undelegations
	found := false
	for _, undelegatingGateways := range app.PendingUndelegations {
		for _, gw := range undelegatingGateways.GatewayAddresses {
			if gw == gatewayAddr {
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	require.True(s.T(), found, "Gateway should be in pending undelegations")
}

func (s *UndelegateFromGatewayIntegrationTestSuite) TestUndelegateFromGateway_InvalidAddress() {
	// Test with invalid application address
	validGatewayAddr := sample.AccAddress()
	
	msg := types.NewMsgUndelegateFromGateway("invalid", validGatewayAddr)
	_, err := s.GetApp().RunMsg(s.T(), msg)
	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "invalid application address")
}

func (s *UndelegateFromGatewayIntegrationTestSuite) TestUndelegateFromGateway_InvalidGatewayAddress() {
	// Test with invalid gateway address
	validAppAddr := sample.AccAddress()
	
	// Fund and stake app
	appAccAddr, err := cosmostypes.AccAddressFromBech32(validAppAddr)
	require.NoError(s.T(), err)
	s.FundAddress(s.T(), appAccAddr, 10000000)
	s.StakeApp(s.T(), validAppAddr, 1000000, []string{"svc1"})
	
	msg := types.NewMsgUndelegateFromGateway(validAppAddr, "invalid")
	_, err = s.GetApp().RunMsg(s.T(), msg)
	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "invalid gateway address")
}

func (s *UndelegateFromGatewayIntegrationTestSuite) TestUndelegateFromGateway_AppNotStaked() {
	// Test undelegation with unstaked application
	appAddr := sample.AccAddress()
	gatewayAddr := sample.AccAddress()
	
	// Fund and stake only gateway
	gatewayAccAddr, err := cosmostypes.AccAddressFromBech32(gatewayAddr)
	require.NoError(s.T(), err)
	s.FundAddress(s.T(), gatewayAccAddr, 10000000)
	s.gatewaySuite.StakeGateway(s.T(), gatewayAddr, 1000)
	
	// Try to undelegate unstaked app
	msg := types.NewMsgUndelegateFromGateway(appAddr, gatewayAddr)
	_, err = s.GetApp().RunMsg(s.T(), msg)
	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "not found")
}

func (s *UndelegateFromGatewayIntegrationTestSuite) TestUndelegateFromGateway_NotDelegated() {
	// Test undelegation when app is not delegated to gateway
	appAddr := sample.AccAddress()
	gatewayAddr := sample.AccAddress()
	
	// Fund and stake both
	appAccAddr, err := cosmostypes.AccAddressFromBech32(appAddr)
	require.NoError(s.T(), err)
	s.FundAddress(s.T(), appAccAddr, 10000000)
	s.StakeApp(s.T(), appAddr, 1000000, []string{"svc1"})
	
	gatewayAccAddr, err := cosmostypes.AccAddressFromBech32(gatewayAddr)
	require.NoError(s.T(), err)
	s.FundAddress(s.T(), gatewayAccAddr, 10000000)
	s.gatewaySuite.StakeGateway(s.T(), gatewayAddr, 1000)
	
	// Try to undelegate without delegation
	msg := types.NewMsgUndelegateFromGateway(appAddr, gatewayAddr)
	_, err = s.GetApp().RunMsg(s.T(), msg)
	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "not delegated")
}