package application

import (
	"fmt"
	"math"
	"testing"

	"cosmossdk.io/core/appmodule"
	"github.com/cosmos/cosmos-sdk/codec"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/bank"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/testutil/integration"
	"github.com/pokt-network/poktroll/testutil/integration/suites"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	"github.com/pokt-network/poktroll/x/shared"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var (
	appStakeAmount = int64(100)

	service1Config = &sharedtypes.ApplicationServiceConfig{
		Service: &sharedtypes.Service{Id: "svc1"},
	}

	service2Config = &sharedtypes.ApplicationServiceConfig{
		Service: &sharedtypes.Service{Id: "svc2"},
	}
)

type AppTransferSuite struct {
	suites.ApplicationModuleSuite

	app1Addr cosmostypes.AccAddress
	app2Addr cosmostypes.AccAddress
	app3Addr cosmostypes.AccAddress
}

// TODO_IN_THIS_COMMIT: move
var faucetCoins = cosmostypes.NewCoins(cosmostypes.NewInt64Coin(volatile.DenomuPOKT, math.MaxInt64))

// TODO_INVESTIGATE: why doesn't this work? This genesis state seems to be ignored. 🤔
// TODO_IN_THIS_COMMIT: move
func faucetAcctGenesisOpt(sdkCtx cosmostypes.Context, cdc codec.Codec, mod appmodule.AppModule) (isGenesisImported bool) {
	hasGenesis := mod.(module.HasGenesis)
	if hasGenesis == nil {
		fmt.Printf(">>> hasGenesis: %v\n", hasGenesis != nil)
		return false
	}
	//fmt.Printf(">>> hasGenesis: %v\n", hasGenesis != nil)

	bankModule, isBankMod := mod.(bank.AppModule)
	//fmt.Printf(">>> isBankMod: %v\n", isBankMod)

	if !isBankMod {
		return false
	}

	genesisState := &banktypes.GenesisState{
		Params: banktypes.Params{
			DefaultSendEnabled: true,
		},
		Balances: []banktypes.Balance{
			{
				Address: integration.FaucetAddrStr,
				Coins:   faucetCoins,
			},
		},
		//Supply:        faucetCoins,
		//DenomMetadata: []banktypes.Metadata{},
		//SendEnabled: []banktypes.SendEnabled{
		//	{Denom: volatile.DenomuPOKT, Enabled: true},
		//},
	}
	genesisStateJSON := cdc.MustMarshalJSON(genesisState)
	bankModule.InitGenesis(sdkCtx, cdc, genesisStateJSON)
	exportGenesisJSON := bankModule.ExportGenesis(sdkCtx, cdc)
	fmt.Printf(">>> exportGenesisJSON: %s\n", string(exportGenesisJSON))
	return true
	//return false
}

// TODO_IN_THIS_COMMIT: godoc
func (s *AppTransferSuite) SetupTest() {
	// Construct a fresh integration app for each test.
	s.NewApp(s.T(), faucetAcctGenesisOpt)

	s.ApplicationModuleSuite.SetupTest()

	// TODO_IN_THIS_COMMIT: finish comment...
	// Ensure app1, app2, and app3 are funded with ...
	s.setupTestAccounts()

	app1Bech32 := s.app1Addr.String()
	app2Bech32 := s.app2Addr.String()
	app3Bech32 := s.app3Addr.String()

	// Stake application 1.
	stakeApp1Res := s.StakeApp(s.T(), app1Bech32, appStakeAmount, service1Config.GetService())
	require.Equal(s.T(), app1Bech32, stakeApp1Res.GetApplication().GetAddress())
	require.Equal(s.T(), appStakeAmount, stakeApp1Res.GetApplication().GetStake().Amount.Int64())

	// Assert the on-chain state shows the application 1 as staked.
	foundApp, queryErr := s.GetAppQueryClient().GetApplication(s.GetApp(s.T()).GetSdkCtx(), app1Bech32)
	require.NoError(s.T(), queryErr)
	require.Equal(s.T(), app1Bech32, foundApp.GetAddress())
	require.Equal(s.T(), appStakeAmount, foundApp.GetStake().Amount.Int64())

	sdkCtx := s.GetApp(s.T()).GetSdkCtx()

	// Assert the on-chain state shows the application 2 as NOT staked.
	foundApp, queryErr = s.GetAppQueryClient().GetApplication(sdkCtx, app2Bech32)
	require.Error(s.T(), queryErr)

	// Assert the on-chain state shows the application 3 as NOT staked.
	foundApp, queryErr = s.GetAppQueryClient().GetApplication(sdkCtx, app3Bech32)
	require.Error(s.T(), queryErr)
}

// TODO_IN_THIS_COMMIT: godoc
func (s *AppTransferSuite) TestSingleSourceToNonexistentDestinationSucceeds() {
	// TODO_IN_THIS_COMMIT: comment - assume default shared params
	sharedParams := sharedtypes.DefaultParams()
	sessionEndHeight := shared.GetSessionEndHeight(&sharedParams, s.GetApp(s.T()).GetSdkCtx().BlockHeight())

	// transfer app1 to app2
	srcAddr := s.app1Addr
	dstAddr := s.app2Addr
	transferRes := s.Transfer(s.T(), srcAddr, dstAddr)
	srcApp := transferRes.GetApplication()

	// assert application pending transfer field updated
	pendingTransfer := srcApp.GetPendingTransfer()
	require.NotNil(s.T(), pendingTransfer)

	expectedPendingTransfer := &apptypes.PendingApplicationTransfer{
		DestinationAddress: dstAddr.String(),
		SessionEndHeight:   uint64(sessionEndHeight),
	}
	require.EqualValues(s.T(), expectedPendingTransfer, pendingTransfer)

	// wait for tx result event (msg)
	// wait for tx result event (begin)

	// wait for transfer begin block + 1
	// assert that app1 is in transfer period

	// wait for transfer end block - 1
	// assert that app1 is in transfer period

	// wait for end block event (end)

	// assert that app1 is unstaked
	// assert that app1's bank balance has not changed
	// assert that app2 is staked (w/ correct amount)
}

//func (suite *AppTransferSuite) TestMultipleSourceToSameNonexistentDestinationSucceedsForFirst() {
//	// transfer app1 to app3
//	// assert transfer begin success
//
//	// transfer app2 to app3
//	// assert transfer begin success
//
//	// wait for tx result event (msg)
//	// wait for tx result event (begin)
//
//	// wait for transfer begin block + 1
//	// assert that app1 is in transfer period
//	// assert that app2 is in transfer period
//
//	// wait for transfer end block - 1
//	// assert that app1 is in transfer period
//	// assert that app2 is in transfer period
//
//	// wait for end block event (end)
//
//	// assert that app1 is unstaked
//	// assert that app2 is unstaked
//	// assert that app1's bank balance has not changed
//	// assert that app2's bank balance has not changed
//	// assert that app3 is staked (w/ sum amount: app1 + app2)
//	// assert that delegations were merged
//}

// TODO_TEST:
//func (suite *AppTransferSuite) TestSequentialTransfersSucceed() {
//
//}

// setupTestAccounts sets up the pre-generated accounts for the test suite.
func (s *AppTransferSuite) setupTestAccounts() {
	s.app1Addr = s.setupTestAccount()
	s.app2Addr = s.setupTestAccount()
	s.app3Addr = s.setupTestAccount()
}

// TODO_IN_THIS_COMMIT: godoc
func (s *AppTransferSuite) setupTestAccount() cosmostypes.AccAddress {
	appAccount, ok := s.GetApp(s.T()).GetPreGeneratedAccounts().Next()
	require.Truef(s.T(), ok, "insufficient pre-generated accounts available")

	sdkCtx := s.GetApp(s.T()).GetSdkCtx()
	bankQueryClient := banktypes.NewQueryClient(s.GetApp(s.T()).QueryHelper())
	bankParamsRes, err := bankQueryClient.Params(sdkCtx, &banktypes.QueryParamsRequest{})
	require.NoError(s.T(), err)
	s.T().Logf(">>> bankParamsRes: %+v", bankParamsRes)

	bankBalsRes, err := bankQueryClient.AllBalances(sdkCtx, &banktypes.QueryAllBalancesRequest{
		Address: integration.FaucetAddrStr,
		//Pagination:   nil,
		//ResolveDenom: false,
	})
	require.NoError(s.T(), err)

	bankBalRes, err := bankQueryClient.Balance(sdkCtx, &banktypes.QueryBalanceRequest{
		Address: integration.FaucetAddrStr,
		Denom:   volatile.DenomuPOKT,
		//Pagination:   nil,
		//ResolveDenom: false,
	})
	require.NoError(s.T(), err)

	bankSupRes, err := bankQueryClient.TotalSupply(sdkCtx, &banktypes.QueryTotalSupplyRequest{})
	require.NoError(s.T(), err)

	s.T().Logf(">>> faucetAddr: %s", integration.FaucetAddrStr)
	s.T().Logf(">>> bankBalsRes: %+v", bankBalsRes)
	s.T().Logf(">>> bankBalRes: %+v", bankBalRes)
	s.T().Logf(">>> bankSupRes: %+v", bankSupRes)

	addr := appAccount.Address
	s.FundAddress(s.T(), addr, 99999999999)

	return addr
}

// TestAppTransferSuite runs the application transfer test suite.
func TestAppTransferSuite(t *testing.T) {
	suite.Run(t, new(AppTransferSuite))
}
