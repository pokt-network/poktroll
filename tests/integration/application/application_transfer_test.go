package application

import (
	"encoding/json"
	"testing"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/testutil/integration"
	"github.com/pokt-network/poktroll/testutil/integration/suites"
	"github.com/pokt-network/poktroll/testutil/testkeyring"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	"github.com/pokt-network/poktroll/x/shared"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var (
	appStakeAmount = int64(100)

	service1Config = &sharedtypes.ApplicationServiceConfig{
		ServiceId: "svc1",
	}

	service2Config = &sharedtypes.ApplicationServiceConfig{
		ServiceId: "svc2",
	}
)

type AppTransferSuite struct {
	suites.ApplicationModuleSuite

	app1Account cosmostypes.AccAddress
	app2Account cosmostypes.AccAddress
	app3Account cosmostypes.AccAddress
}

func (s *AppTransferSuite) SetupTest() {
	// Construct a new integration app for each test.
	s.NewApp(s.T())
	s.ApplicationModuleSuite.SetupTest()

	var err error

	// Ensure app1, app2, and app3 have bank balances.
	s.setupTestAccounts()

	stakeApp1Msg := apptypes.NewMsgStakeApplication(
		s.app1Account.String(),
		cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(appStakeAmount)),
		// TODO_IN_THIS_COMMIT: add services and assert services were merged.
		[]*sharedtypes.ApplicationServiceConfig{service1Config},
	)

	bankQueryClient := banktypes.NewQueryClient(s.GetApp().QueryHelper())
	bankParamsRes, err := bankQueryClient.Params(s.sdkCtx(), &banktypes.QueryParamsRequest{})
	require.NoError(s.T(), err)

	s.T().Logf(">>> bankParamsRes: %v", bankParamsRes)

	// Stake application 1.
	anyRes := s.GetApp().RunMsg(
		s.T(),
		stakeApp1Msg,
		integration.RunUntilNextBlockOpts...,
	)
	require.NotNil(s.T(), anyRes)

	stakeApp1Res := new(apptypes.MsgStakeApplicationResponse)
	err = s.GetApp().GetCodec().UnpackAny(anyRes, &stakeApp1Res)
	require.NoError(s.T(), err)
	require.Equal(s.T(), s.app1Account.String(), stakeApp1Res.GetApplication().GetAddress())
	require.Equal(s.T(), appStakeAmount, stakeApp1Res.GetApplication().GetStake().Amount.Int64())

	// Assert the on-chain state shows the application 1 as staked.
	foundApp, queryErr := s.GetAppQueryClient().GetApplication(s.sdkCtx(), s.app1Account.String())
	require.NoError(s.T(), queryErr)
	require.Equal(s.T(), s.app1Account.String(), foundApp.GetAddress())
	require.Equal(s.T(), appStakeAmount, foundApp.GetStake().Amount.Int64())

	// Assert the on-chain state shows the application 2 as NOT staked.
	foundApp, queryErr = s.GetAppQueryClient().GetApplication(s.sdkCtx(), s.app2Account.String())
	require.Error(s.T(), queryErr)

	// Assert the on-chain state shows the application 3 as NOT staked.
	foundApp, queryErr = s.GetAppQueryClient().GetApplication(s.sdkCtx(), s.app3Account.String())
	require.Error(s.T(), queryErr)
}

func (s *AppTransferSuite) TestSingleSourceToNonexistentDestinationSucceeds() {
	// TODO_IN_THIS_COMMIT: comment - assume default shared params
	sharedParams := sharedtypes.DefaultParams()
	sessionEndHeight := shared.GetSessionEndHeight(&sharedParams, s.sdkCtx().BlockHeight())

	// transfer app1 to app2
	transferRes := s.Transfer(s.T(), s.app1Account, s.app2Account)
	srcApp := transferRes.GetApplication()

	transferBeginHeight := s.sdkCtx().BlockHeight()

	// assert application pending transfer field updated
	pendingTransfer := srcApp.GetPendingTransfer()
	require.NotNil(s.T(), pendingTransfer)

	expectedPendingTransfer := &apptypes.PendingApplicationTransfer{
		DestinationAddress: s.app2Account.String(),
		SessionEndHeight:   uint64(sessionEndHeight),
	}
	require.EqualValues(s.T(), expectedPendingTransfer, pendingTransfer)

	// wait for tx result event (msg)
	events := s.GetApp().GetSdkCtx().EventManager().ABCIEvents()
	for _, event := range events {
		if event.Type != "message" {
			continue
		}

		eventsJSON, err := json.MarshalIndent(event, "", "  ")
		require.NoError(s.T(), err)
		s.T().Logf(">>> %s", eventsJSON)
	}
	//events2 := s.GetApp().GetSdkCtx().EventManager().Events()
	//events2JSON, err := json.MarshalIndent(events2, "", "  ")
	//s.T().Logf(">>> events2: %s", events2JSON)
	//txResultEvents := s.GetApp().GetSdkCtx().EventManager().Events()
	//for _, event := range txResultEvents {
	//	//if event.Type == "message" {
	//	//	require.Contains(s.T(), event.Attributes, cosmostypes.NewAttribute("action", "transfer_application"))
	//	//}
	//}
	// wait for tx result event (begin)

	// wait for transfer begin block + 1
	s.GetApp().NextBlock(s.T())

	// assert that app1 is in transfer period
	foundApp1, err := s.GetAppQueryClient().GetApplication(s.sdkCtx(), s.app1Account.String())
	require.NoError(s.T(), err)

	require.Equal(s.T(), s.app1Account.String(), foundApp1.GetAddress())
	require.Equal(s.T(), expectedPendingTransfer, foundApp1.GetPendingTransfer())

	// wait for transfer end block - 1
	// TODO_IN_THIS_COMMIT: comment or consider doing something better...
	transferEndHeight := apptypes.GetApplicationTransferHeight(&sharedParams, &foundApp1)
	blocksUntilTransferEndHeight := transferEndHeight - transferBeginHeight
	s.GetApp().NextBlocks(s.T(), int(blocksUntilTransferEndHeight-1))

	// assert that app1 is in transfer period
	foundApp1, err = s.GetAppQueryClient().GetApplication(s.sdkCtx(), s.app1Account.String())
	require.NoError(s.T(), err)

	require.Equal(s.T(), s.app1Account.String(), foundApp1.GetAddress())
	require.Equal(s.T(), expectedPendingTransfer, foundApp1.GetPendingTransfer())

	// wait for end block event (end)
	s.GetApp().NextBlock(s.T())
	// TODO_IN_THIS_COMMIT: assert event...

	// assert that app1 is unstaked
	_, err = s.GetAppQueryClient().GetApplication(s.sdkCtx(), s.app1Account.String())
	require.ErrorContains(s.T(), err, "application not found")

	// assert that app1's bank balance has not changed

	// assert that app2 is staked (w/ correct amount)
	foundApp2, err := s.GetAppQueryClient().GetApplication(s.sdkCtx(), s.app2Account.String())
	require.NoError(s.T(), err)

	require.Equal(s.T(), s.app2Account.String(), foundApp2.GetAddress())
	require.Nil(s.T(), foundApp2.GetPendingTransfer())
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
	s.app1Account = s.setupTestAccount().Address
	s.app2Account = s.setupTestAccount().Address
	s.app3Account = s.setupTestAccount().Address
}

func (s *AppTransferSuite) setupTestAccount() *testkeyring.PreGeneratedAccount {
	appAccount, ok := s.GetApp().GetPreGeneratedAccounts().Next()
	require.Truef(s.T(), ok, "insufficient pre-generated accounts available")

	//sendToAppMsg := &banktypes.MsgSend{
	//	FromAddress: integration.FaucetAddrStr,
	//	ToAddress:   appAccount.Address.String(),
	//	// TODO_IN_THIS_PR: move amount to a constant.
	//	Amount: cosmostypes.NewCoins(cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 100000000)),
	//}
	//s.GetApp().RunMsg(s.T(), sendToAppMsg)

	s.FundAddress(s.T(), appAccount.Address, 100000000)

	return appAccount
}

// TODO_IN_THIS_COMMIT: consider promoting to BaseIntegrationSuite.
// sdkCtx returns the integration app's SDK context.
func (s *AppTransferSuite) sdkCtx() *cosmostypes.Context {
	return s.GetApp().GetSdkCtx()

}

// TestAppTransferSuite runs the application transfer test suite.
func TestAppTransferSuite(t *testing.T) {
	suite.Run(t, new(AppTransferSuite))
}
