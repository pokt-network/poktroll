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

type UnstakeApplicationIntegrationTestSuite struct {
	suites.ApplicationModuleSuite
}

func TestUnstakeApplicationIntegrationSuite(t *testing.T) {
	suite.Run(t, new(UnstakeApplicationIntegrationTestSuite))
}

func (s *UnstakeApplicationIntegrationTestSuite) SetupTest() {
	s.NewApp(s.T())
}

func (s *UnstakeApplicationIntegrationTestSuite) TestUnstakeApplication_Valid() {
	// Prepare address
	appAddr := sample.AccAddressBech32()

	// Fund and stake application
	appAccAddr, err := cosmostypes.AccAddressFromBech32(appAddr)
	require.NoError(s.T(), err)
	s.FundAddress(s.T(), appAccAddr, 10000000)
	s.StakeApp(s.T(), appAddr, 1000000, []string{"svc1"})

	// Verify app is staked
	appQueryClient := s.GetAppQueryClient(s.T())
	app, err := appQueryClient.GetApplication(s.SdkCtx(), appAddr)
	require.NoError(s.T(), err)
	require.Equal(s.T(), appAddr, app.Address)

	// Unstake application
	unstakeMsg := types.NewMsgUnstakeApplication(appAddr)
	_, err = s.GetApp().RunMsg(s.T(), unstakeMsg)
	require.NoError(s.T(), err)

	// Verify app is marked for unstaking
	app, err = appQueryClient.GetApplication(s.SdkCtx(), appAddr)
	require.NoError(s.T(), err)
	require.True(s.T(), app.IsUnbonding())
}

func (s *UnstakeApplicationIntegrationTestSuite) TestUnstakeApplication_InvalidAddress() {
	// Test with invalid address
	msg := types.NewMsgUnstakeApplication("invalid")
	_, err := s.GetApp().RunMsg(s.T(), msg)
	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "invalid application address")
}

func (s *UnstakeApplicationIntegrationTestSuite) TestUnstakeApplication_NotStaked() {
	// Test unstaking non-existent application
	appAddr := sample.AccAddressBech32()

	msg := types.NewMsgUnstakeApplication(appAddr)
	_, err := s.GetApp().RunMsg(s.T(), msg)
	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "not found")
}

func (s *UnstakeApplicationIntegrationTestSuite) TestUnstakeApplication_AlreadyUnstaking() {
	// Prepare address
	appAddr := sample.AccAddressBech32()

	// Fund and stake application
	appAccAddr, err := cosmostypes.AccAddressFromBech32(appAddr)
	require.NoError(s.T(), err)
	s.FundAddress(s.T(), appAccAddr, 10000000)
	s.StakeApp(s.T(), appAddr, 1000000, []string{"svc1"})

	// Unstake application first time
	unstakeMsg := types.NewMsgUnstakeApplication(appAddr)
	_, err = s.GetApp().RunMsg(s.T(), unstakeMsg)
	require.NoError(s.T(), err)

	// Try to unstake again
	_, err = s.GetApp().RunMsg(s.T(), unstakeMsg)
	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "unbonding period")
}
