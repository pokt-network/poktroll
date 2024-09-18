package application

import (
	"strings"
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/app/volatile"
	events2 "github.com/pokt-network/poktroll/testutil/events"
	"github.com/pokt-network/poktroll/testutil/integration/suites"
	"github.com/pokt-network/poktroll/testutil/testkeyring"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	"github.com/pokt-network/poktroll/x/shared"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var (
	appFundAmount  = int64(100000000)
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

	app1Addr    cosmostypes.AccAddress
	app2Addr    cosmostypes.AccAddress
	app3Account cosmostypes.AccAddress
}

func (s *AppTransferSuite) SetupTest() {
	// Construct a new integration app for each test.
	s.NewApp(s.T())

	// Ensure app1, app2, and app3 have bank balances.
	s.setupTestAccounts()

	// Stake application 1.
	stakeApp1Res := s.StakeApp(s.T(), s.app1Addr.String(), appStakeAmount, []string{service1Config.ServiceId})
	require.Equal(s.T(), s.app1Addr.String(), stakeApp1Res.GetApplication().GetAddress())
	require.Equal(s.T(), appStakeAmount, stakeApp1Res.GetApplication().GetStake().Amount.Int64())

	// Assert the on-chain state shows the application 1 as staked.
	foundApp, queryErr := s.GetAppQueryClient().GetApplication(s.sdkCtx(), s.app1Addr.String())
	require.NoError(s.T(), queryErr)
	require.Equal(s.T(), s.app1Addr.String(), foundApp.GetAddress())
	require.Equal(s.T(), appStakeAmount, foundApp.GetStake().Amount.Int64())

	// Assert the on-chain state shows the application 2 as NOT staked.
	foundApp, queryErr = s.GetAppQueryClient().GetApplication(s.sdkCtx(), s.app2Addr.String())
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
	transferRes := s.Transfer(s.T(), s.app1Addr, s.app2Addr)
	srcApp := transferRes.GetApplication()

	transferBeginHeight := s.sdkCtx().BlockHeight()

	// Assert application pending transfer field updated in the msg response.
	pendingTransfer := srcApp.GetPendingTransfer()
	require.NotNil(s.T(), pendingTransfer)

	expectedPendingTransfer := &apptypes.PendingApplicationTransfer{
		DestinationAddress: s.app2Addr.String(),
		SessionEndHeight:   uint64(sessionEndHeight),
	}
	require.EqualValues(s.T(), expectedPendingTransfer, pendingTransfer)

	// Query and assert application pending transfer field updated in the store.
	foundApp1, err := s.GetAppQueryClient().GetApplication(s.sdkCtx(), s.app1Addr.String())
	require.NoError(s.T(), err)
	require.EqualValues(s.T(), expectedPendingTransfer, foundApp1.GetPendingTransfer())

	// Assert that the "message" type event (tx result event) is observed which
	// corresponds to the MsgTransferApplication message.
	msgTypeURL := cosmostypes.MsgTypeURL(&apptypes.MsgTransferApplication{})
	msgEvent := s.LatestMatchingEvent(events2.NewMsgEventMatchFn(msgTypeURL))
	require.NotNil(s.T(), msgEvent, "expected transfer application message event")

	// Assert that the transfer begin event (tx result event) is observed.
	s.shouldObserveTransferBeginEvent(&foundApp1)

	// wait for transfer end commit height - 1
	// TODO_IN_THIS_COMMIT: comment regarding why transferEndHeight is correct...
	transferEndHeight := apptypes.GetApplicationTransferHeight(&sharedParams, &foundApp1)
	blocksUntilTransferEndHeight := transferEndHeight - transferBeginHeight
	s.GetApp().NextBlocks(s.T(), int(blocksUntilTransferEndHeight))

	// assert that app1 is in transfer period
	foundApp1, err = s.GetAppQueryClient().GetApplication(s.sdkCtx(), s.app1Addr.String())
	require.NoError(s.T(), err)

	require.Equal(s.T(), s.app1Addr.String(), foundApp1.GetAddress())
	require.Equal(s.T(), expectedPendingTransfer, foundApp1.GetPendingTransfer())

	// wait for end block event (end)
	s.GetApp().NextBlock(s.T())

	// Query for and assert that the destination application was created.
	foundApp2, err := s.GetAppQueryClient().GetApplication(s.sdkCtx(), s.app2Addr.String())
	require.NoError(s.T(), err)

	// Assert that the application was created with the correct address and stake amount.
	require.Equal(s.T(), s.app2Addr.String(), foundApp2.GetAddress())
	require.Equal(s.T(), appStakeAmount, foundApp2.GetStake().Amount.Int64())

	// Assert that the transfer end event (end block event) is observed.
	s.shouldObserveTransferEndEvent(&foundApp2)

	// assert that app1 is unstaked
	_, err = s.GetAppQueryClient().GetApplication(s.sdkCtx(), s.app1Addr.String())
	require.ErrorContains(s.T(), err, "application not found")

	// assert that app1's bank balance has not changed
	balanceRes, err := s.GetBankQueryClient().Balance(s.sdkCtx(), &banktypes.QueryBalanceRequest{
		Address: s.app1Addr.String(),
		Denom:   volatile.DenomuPOKT,
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), balanceRes)

	require.EqualValues(s.T(),
		cosmostypes.NewInt64Coin(volatile.DenomuPOKT, appFundAmount-appStakeAmount),
		*balanceRes.GetBalance(),
	)
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
	s.app1Addr = s.setupTestAccount().Address
	s.app2Addr = s.setupTestAccount().Address
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

	s.FundAddress(s.T(), appAccount.Address, appFundAmount)

	return appAccount
}

// TODO_IN_THIS_COMMIT: godoc...
func (s *AppTransferSuite) shouldObserveTransferBeginEvent(expectedSrcApp *apptypes.Application) {
	eventTypeURL := cosmostypes.MsgTypeURL(&apptypes.EventTransferBegin{})
	transferBeginEvent := s.LatestMatchingEvent(events2.NewEventTypeMatchFn(eventTypeURL))
	require.NotNil(s.T(), transferBeginEvent)

	evtSrcAddr := GetTrimmedEventAttribute(s.T(), transferBeginEvent, "source_address")
	require.Equal(s.T(), s.app1Addr.String(), evtSrcAddr)

	evtDstAddr := GetTrimmedEventAttribute(s.T(), transferBeginEvent, "destination_address")
	require.Equal(s.T(), s.app2Addr.String(), evtDstAddr)

	evtSrcApp := new(apptypes.Application)
	evtSrcAppBz := []byte(GetTrimmedEventAttribute(s.T(), transferBeginEvent, "source_application"))
	err := s.GetApp().GetCodec().UnmarshalJSON(evtSrcAppBz, evtSrcApp)
	require.NoError(s.T(), err)
	require.EqualValues(s.T(), expectedSrcApp.GetPendingTransfer(), evtSrcApp.GetPendingTransfer())
}

// TODO_IN_THIS_COMMIT: godoc...
func (s *AppTransferSuite) shouldObserveTransferEndEvent(expectedDstApp *apptypes.Application) {
	eventTypeURL := cosmostypes.MsgTypeURL(&apptypes.EventTransferEnd{})
	transferEndEvent := s.LatestMatchingEvent(events2.NewEventTypeMatchFn(eventTypeURL))
	require.NotNil(s.T(), transferEndEvent)

	evtSrcAddr := GetTrimmedEventAttribute(s.T(), transferEndEvent, "source_address")
	require.Equal(s.T(), s.app1Addr.String(), evtSrcAddr)

	evtDstAddr := GetTrimmedEventAttribute(s.T(), transferEndEvent, "destination_address")
	require.Equal(s.T(), s.app2Addr.String(), evtDstAddr)

	evtDstApp := new(apptypes.Application)
	evtDstAppBz := []byte(GetTrimmedEventAttribute(s.T(), transferEndEvent, "destination_application"))
	err := s.GetApp().GetCodec().UnmarshalJSON(evtDstAppBz, evtDstApp)
	require.NoError(s.T(), err)
	require.EqualValues(s.T(), expectedDstApp.GetPendingTransfer(), evtDstApp.GetPendingTransfer())
}

// TODO_IN_THIS_COMMIT: move...
func GetTrimmedEventAttribute(t *testing.T, event *cosmostypes.Event, key string) string {
	attr, hasAttr := event.GetAttribute(key)
	require.Truef(t, hasAttr, "expected %q attribute in %q event", key, event.Type)

	return strings.Trim(attr.GetValue(), "\"")
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
