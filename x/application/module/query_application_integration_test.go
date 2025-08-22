package application_test

import (
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/testutil/integration/suites"
	"github.com/pokt-network/poktroll/testutil/sample"
)

type QueryApplicationIntegrationTestSuite struct {
	suites.ApplicationModuleSuite
}

func TestQueryApplicationIntegrationSuite(t *testing.T) {
	suite.Run(t, new(QueryApplicationIntegrationTestSuite))
}

func (s *QueryApplicationIntegrationTestSuite) SetupTest() {
	s.NewApp(s.T())
}

func (s *QueryApplicationIntegrationTestSuite) TestQueryApplication_Show() {
	// Prepare test applications
	app1Addr := sample.AccAddressBech32()
	app2Addr := sample.AccAddressBech32()

	// Fund and stake applications
	app1AccAddr, err := cosmostypes.AccAddressFromBech32(app1Addr)
	require.NoError(s.T(), err)
	s.FundAddress(s.T(), app1AccAddr, 10000000)
	s.StakeApp(s.T(), app1Addr, 1000000, []string{"svc1"})

	app2AccAddr, err := cosmostypes.AccAddressFromBech32(app2Addr)
	require.NoError(s.T(), err)
	s.FundAddress(s.T(), app2AccAddr, 10000000)
	s.StakeApp(s.T(), app2Addr, 1000000, []string{"svc2"})

	// Get query client
	appQueryClient := s.GetAppQueryClient(s.T())

	// Test: found
	app, err := appQueryClient.GetApplication(s.SdkCtx(), app1Addr)
	require.NoError(s.T(), err)
	require.Equal(s.T(), app1Addr, app.Address)
	require.Len(s.T(), app.ServiceConfigs, 1)
	require.Equal(s.T(), "svc1", app.ServiceConfigs[0].ServiceId)

	// Test: not found
	nonExistentAddr := sample.AccAddressBech32()
	_, err = appQueryClient.GetApplication(s.SdkCtx(), nonExistentAddr)
	require.Error(s.T(), err)
	require.Contains(s.T(), err.Error(), "not found")
}

func (s *QueryApplicationIntegrationTestSuite) TestQueryApplication_List() {
	// Prepare and stake 3 applications
	var appAddresses []string
	for i := 0; i < 3; i++ {
		appAddr := sample.AccAddressBech32()
		appAddresses = append(appAddresses, appAddr)

		// Fund and stake
		appAccAddr, err := cosmostypes.AccAddressFromBech32(appAddr)
		require.NoError(s.T(), err)
		s.FundAddress(s.T(), appAccAddr, 10000000)
		s.StakeApp(s.T(), appAddr, 1000000, []string{"svc1"})
	}

	// Get query client
	appQueryClient := s.GetAppQueryClient(s.T())

	// Test: Get all applications
	apps, err := appQueryClient.GetAllApplications(s.SdkCtx())
	require.NoError(s.T(), err)
	require.GreaterOrEqual(s.T(), len(apps), len(appAddresses))

	// Verify our staked apps are present
	foundCount := 0
	for _, app := range apps {
		for _, expectedAddr := range appAddresses {
			if app.Address == expectedAddr {
				foundCount++
				require.Len(s.T(), app.ServiceConfigs, 1)
				require.Equal(s.T(), "svc1", app.ServiceConfigs[0].ServiceId)
				require.Equal(s.T(), int64(1000000), app.Stake.Amount.Int64())
				break
			}
		}
	}
	require.Equal(s.T(), len(appAddresses), foundCount, "All staked apps should be found")
}
