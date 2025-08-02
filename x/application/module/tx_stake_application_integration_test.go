package application_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/testutil/integration/suites"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

type StakeApplicationIntegrationTestSuite struct {
	suites.ApplicationModuleSuite
}

func TestStakeApplicationIntegrationSuite(t *testing.T) {
	suite.Run(t, new(StakeApplicationIntegrationTestSuite))
}

func (s *StakeApplicationIntegrationTestSuite) SetupTest() {
	s.NewApp(s.T())
}

func (s *StakeApplicationIntegrationTestSuite) TestStakeApplication_Valid() {
	// Test valid stake application
	appAddr := sample.AccAddress()
	stakeAmount := int64(1000000) // 1 POKT minimum stake
	serviceIds := []string{"svc1"}

	// Fund the account
	appAccAddr, err := sdk.AccAddressFromBech32(appAddr)
	require.NoError(s.T(), err)
	s.FundAddress(s.T(), appAccAddr, stakeAmount*2)

	// Stake the application
	res := s.StakeApp(s.T(), appAddr, stakeAmount, serviceIds)
	require.NotNil(s.T(), res)

	// Verify application was staked
	appQueryClient := s.GetAppQueryClient(s.T())
	app, err := appQueryClient.GetApplication(s.SdkCtx(), appAddr)
	require.NoError(s.T(), err)
	require.Equal(s.T(), appAddr, app.Address)
	require.Equal(s.T(), sdk.NewCoin("upokt", math.NewInt(stakeAmount)), *app.Stake)
	require.Len(s.T(), app.ServiceConfigs, 1)
	require.Equal(s.T(), serviceIds[0], app.ServiceConfigs[0].ServiceId)
}

func (s *StakeApplicationIntegrationTestSuite) TestStakeApplication_InvalidAddress() {
	// Test with invalid address
	invalidAddr := "invalid"
	stakeAmount := int64(1000000) // 1 POKT minimum stake
	serviceIds := []string{"svc1"}

	msg := types.NewMsgStakeApplication(
		invalidAddr,
		sdk.NewCoin("upokt", math.NewInt(stakeAmount)),
		[]*sharedtypes.ApplicationServiceConfig{{ServiceId: serviceIds[0]}},
	)

	_, err := s.GetApp().RunMsg(s.T(), msg)
	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "invalid address")
}

func (s *StakeApplicationIntegrationTestSuite) TestStakeApplication_InvalidStake() {
	appAddr := sample.AccAddress()

	testCases := []struct {
		name        string
		stakeAmount int64
		stakeDenom  string
		expectError string
	}{
		{
			name:        "zero stake",
			stakeAmount: 0,
			stakeDenom:  "upokt",
			expectError: "invalid stake",
		},
		{
			name:        "invalid denom",
			stakeAmount: 1000,
			stakeDenom:  "invalid",
			expectError: "invalid stake",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			msg := types.NewMsgStakeApplication(
				appAddr,
				sdk.NewCoin(tc.stakeDenom, math.NewInt(tc.stakeAmount)),
				[]*sharedtypes.ApplicationServiceConfig{{ServiceId: "svc1"}},
			)

			_, err := s.GetApp().RunMsg(s.T(), msg)
			require.Error(s.T(), err)
			require.Contains(s.T(), err.Error(), tc.expectError)
		})
	}
}

func (s *StakeApplicationIntegrationTestSuite) TestStakeApplication_InvalidService() {
	appAddr := sample.AccAddress()
	stakeAmount := int64(1000000) // 1 POKT minimum stake

	testCases := []struct {
		name           string
		serviceConfigs []*sharedtypes.ApplicationServiceConfig
		expectError    string
	}{
		{
			name:           "empty services",
			serviceConfigs: []*sharedtypes.ApplicationServiceConfig{},
			expectError:    "invalid service configs",
		},
		{
			name: "service with spaces",
			serviceConfigs: []*sharedtypes.ApplicationServiceConfig{
				{ServiceId: "svc1 svc1_part2 svc1_part3"},
			},
			expectError: "invalid service configs",
		},
		{
			name: "multiple services",
			serviceConfigs: []*sharedtypes.ApplicationServiceConfig{
				{ServiceId: "svc1"},
				{ServiceId: "svc2"},
			},
			expectError: "invalid service configs",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Fund the account
			appAccAddr, err := sdk.AccAddressFromBech32(appAddr)
			require.NoError(s.T(), err)
			s.FundAddress(s.T(), appAccAddr, stakeAmount*2)

			msg := types.NewMsgStakeApplication(
				appAddr,
				sdk.NewCoin("upokt", math.NewInt(stakeAmount)),
				tc.serviceConfigs,
			)

			_, err = s.GetApp().RunMsg(s.T(), msg)
			require.Error(s.T(), err)
			require.Contains(s.T(), err.Error(), tc.expectError)
		})
	}
}
