//go:build integration

package application

import (
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
	appFundAmount = int64(100000000)
	stakeAmount   = int64(100)

	service1Id = "svc1"
	service2Id = "svc2"
	service3Id = "svc3"
)

type AppTransferSuite struct {
	suites.ApplicationModuleSuite
	gatewaySuite suites.GatewayModuleSuite

	gateway1Bech32 string
	gateway2Bech32 string
	gateway3Bech32 string

	app1Bech32 string
	app2Bech32 string
	app3Bech32 string
}

func (s *AppTransferSuite) SetupTest() {
	// Construct a new integration app for each test.
	s.NewApp(s.T())
	s.gatewaySuite.SetApp(s.GetApp())

	// Ensure gateways and apps have bank balances.
	s.setupTestAddresses()

	// Stake gateways for applications to delegate to.
	s.setupStakeGateways()

	// Stake app1 and app2.
	s.setupStakeApps(map[string][]string{
		s.app1Bech32: {service1Id, service3Id},
		s.app2Bech32: {service1Id, service2Id},
	})

	// Delegate app 1 to gateway 1 and app2 to gateways 1 and 2.
	s.setupDelegateAppsToGateway(map[string][]string{
		s.app1Bech32: {s.gateway1Bech32, s.gateway3Bech32},
		s.app2Bech32: {s.gateway1Bech32, s.gateway2Bech32},
	})

	// Assert the on-chain state shows the application 3 as NOT staked.
	_, queryErr := s.GetAppQueryClient().GetApplication(s.SdkCtx(), s.app3Bech32)
	require.ErrorContains(s.T(), queryErr, "application not found")
	require.ErrorContains(s.T(), queryErr, s.app3Bech32)
}

func (s *AppTransferSuite) TestSingleSourceToNonexistentDestinationSucceeds() {
	sharedParams := sharedtypes.DefaultParams()
	sessionEndHeight := shared.GetSessionEndHeight(&sharedParams, s.SdkCtx().BlockHeight())

	transferBeginHeight := s.SdkCtx().BlockHeight()

	// Transfer app1 to app3
	transferRes := s.Transfer(s.T(), s.app1Bech32, s.app3Bech32)
	srcApp := transferRes.GetApplication()

	// Assert application pending transfer field updated in the msg response.
	pendingTransfer := srcApp.GetPendingTransfer()
	require.NotNil(s.T(), pendingTransfer)

	expectedPendingTransfer := &apptypes.PendingApplicationTransfer{
		DestinationAddress: s.app3Bech32,
		SessionEndHeight:   uint64(sessionEndHeight),
	}
	require.EqualValues(s.T(), expectedPendingTransfer, pendingTransfer)

	// Query and assert application pending transfer field updated in the store.
	foundApp1, err := s.GetAppQueryClient().GetApplication(s.SdkCtx(), s.app1Bech32)
	require.NoError(s.T(), err)
	require.EqualValues(s.T(), expectedPendingTransfer, foundApp1.GetPendingTransfer())

	// Assert that the "message" type event (tx result event) is observed which
	// corresponds to the MsgTransferApplication message.
	msgTypeURL := cosmostypes.MsgTypeURL(&apptypes.MsgTransferApplication{})
	msgEvent := s.LatestMatchingEvent(events2.NewMsgEventMatchFn(msgTypeURL))
	require.NotNil(s.T(), msgEvent, "expected transfer application message event")

	// Assert that the transfer begin event (tx result event) is observed.
	s.shouldObserveTransferBeginEvent(&foundApp1, s.app3Bech32)

	// Continue until transfer end commit height - 1.
	transferEndHeight := apptypes.GetApplicationTransferHeight(&sharedParams, &foundApp1)
	blocksUntilTransferEndHeight := transferEndHeight - transferBeginHeight
	s.GetApp().NextBlocks(s.T(), int(blocksUntilTransferEndHeight)-1)

	// Assert that app1 is in transfer period.
	foundApp1, err = s.GetAppQueryClient().GetApplication(s.SdkCtx(), s.app1Bech32)
	require.NoError(s.T(), err)

	require.Equal(s.T(), s.app1Bech32, foundApp1.GetAddress())
	require.Equal(s.T(), expectedPendingTransfer, foundApp1.GetPendingTransfer())

	// Continue to transfer end height.
	s.GetApp().NextBlock(s.T())

	// Query for and assert that the destination application was created.
	foundApp3, err := s.GetAppQueryClient().GetApplication(s.SdkCtx(), s.app3Bech32)
	require.NoError(s.T(), err)

	// Assert that the application was created with the correct address and stake amount.
	require.Equal(s.T(), s.app3Bech32, foundApp3.GetAddress())
	require.Equal(s.T(), stakeAmount, foundApp3.GetStake().Amount.Int64())

	// Assert that the transfer end event (end block event) is observed.
	s.shouldObserveTransferEndEvent(&foundApp3, s.app1Bech32)

	// Assert that app1 is unstaked.
	_, err = s.GetAppQueryClient().GetApplication(s.SdkCtx(), s.app1Bech32)
	require.ErrorContains(s.T(), err, "application not found")
	require.ErrorContains(s.T(), err, s.app1Bech32)

	// Assert that app1's bank balance has not changed.
	balanceRes, err := s.GetBankQueryClient().Balance(s.SdkCtx(), &banktypes.QueryBalanceRequest{
		Address: s.app1Bech32,
		Denom:   volatile.DenomuPOKT,
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), balanceRes)

	require.EqualValues(s.T(),
		cosmostypes.NewInt64Coin(volatile.DenomuPOKT, appFundAmount-stakeAmount),
		*balanceRes.GetBalance(),
	)
}

func (s *AppTransferSuite) TestMultipleSourceToSameNonexistentDestinationMergesSources() {
	sharedParams := sharedtypes.DefaultParams()
	msgTransferAppTypeURL := cosmostypes.MsgTypeURL(&apptypes.MsgTransferApplication{})
	sessionEndHeight := shared.GetSessionEndHeight(&sharedParams, s.SdkCtx().BlockHeight())

	transferBeginHeight := s.SdkCtx().BlockHeight()

	// Transfer app1 & app2 to app3 in the same session (and tx).
	srcToDstTransferMap := map[string]string{
		s.app1Bech32: s.app3Bech32,
		s.app2Bech32: s.app3Bech32,
	}
	transferResps := s.MultiTransfer(s.T(), srcToDstTransferMap)

	transferResSrcIndices := []string{
		s.app1Bech32,
		s.app2Bech32,
	}
	var (
		transferEndHeight       int64
		expectedPendingTransfer *apptypes.PendingApplicationTransfer
	)
	for transferResIdx, transferRes := range transferResps {
		expectedSrcBech32 := transferResSrcIndices[transferResIdx]
		expectedDstBech32 := srcToDstTransferMap[expectedSrcBech32]

		srcApp := transferRes.GetApplication()

		// Assert application pending transfer field updated in the msg response.
		pendingTransfer := srcApp.GetPendingTransfer()
		require.NotNil(s.T(), pendingTransfer)

		nextExpectedPendingTransfer := &apptypes.PendingApplicationTransfer{
			DestinationAddress: expectedDstBech32,
			SessionEndHeight:   uint64(sessionEndHeight),
		}
		if expectedPendingTransfer != nil {
			require.EqualValues(s.T(), expectedPendingTransfer, nextExpectedPendingTransfer)
		}
		expectedPendingTransfer = nextExpectedPendingTransfer
		require.EqualValues(s.T(), expectedPendingTransfer, pendingTransfer)

		// Query and assert application pending transfer field updated in the store.
		foundSrcApp, err := s.GetAppQueryClient().GetApplication(s.SdkCtx(), expectedSrcBech32)
		require.NoError(s.T(), err)
		require.EqualValues(s.T(), expectedPendingTransfer, foundSrcApp.GetPendingTransfer())

		// Assert that the "message" type event (tx result event) is observed which
		// corresponds to the MsgTransferApplication message.
		msgEvent := s.LatestMatchingEvent(events2.NewMsgEventMatchFn(msgTransferAppTypeURL))
		require.NotNil(s.T(), msgEvent, "expected transfer application message event")

		// Assert that the transfer begin event (tx result event) is observed.
		s.shouldObserveTransferBeginEvent(&foundSrcApp, expectedDstBech32)

		nextTransferEndHeight := apptypes.GetApplicationTransferHeight(&sharedParams, &foundSrcApp)
		if transferEndHeight != 0 {
			require.Equal(s.T(), transferEndHeight, nextTransferEndHeight)
		}
		transferEndHeight = nextTransferEndHeight
	}

	// Assert that the "message" type event (tx result event) is observed which
	// corresponds to the MsgTransferApplication message.
	msgEvents := s.FilterLatestEvents(events2.NewMsgEventMatchFn(msgTransferAppTypeURL))
	require.Equal(s.T(), 2, len(msgEvents), "expected 2 application transfer message events")

	// Continue until transfer end commit height - 1.
	blocksUntilTransferEndHeight := transferEndHeight - transferBeginHeight
	s.GetApp().NextBlocks(s.T(), int(blocksUntilTransferEndHeight)-1)

	// Assert that app1 is in transfer period.
	foundApp1, err := s.GetAppQueryClient().GetApplication(s.SdkCtx(), s.app1Bech32)
	require.NoError(s.T(), err)

	require.Equal(s.T(), s.app1Bech32, foundApp1.GetAddress())
	require.Equal(s.T(), expectedPendingTransfer, foundApp1.GetPendingTransfer())

	// Continue to transfer end height.
	s.GetApp().NextBlock(s.T())

	// Assert that app3 is staked with the sum amount: app1 + app2.
	foundApp3, err := s.GetAppQueryClient().GetApplication(s.SdkCtx(), s.app3Bech32)
	require.NoError(s.T(), err)

	require.Equal(s.T(), s.app3Bech32, foundApp3.GetAddress())
	require.Equal(s.T(), stakeAmount*2, foundApp3.GetStake().Amount.Int64())

	// Assert that delegations were merged.
	gatewayBech32s := []string{
		s.gateway1Bech32,
		s.gateway2Bech32,
		s.gateway3Bech32,
	}
	for _, gatewayBech32 := range gatewayBech32s {
		require.Contains(s.T(), foundApp3.GetDelegateeGatewayAddresses(), gatewayBech32)
	}
	require.Equal(s.T(), len(gatewayBech32s), len(foundApp3.GetDelegateeGatewayAddresses()))

	// Assert that services were merged.
	serviceIds := []string{
		service1Id,
		service2Id,
		service3Id,
	}
	for _, serviceId := range serviceIds {
		require.Contains(s.T(),
			foundApp3.GetServiceConfigs(),
			&sharedtypes.ApplicationServiceConfig{
				ServiceId: serviceId,
			},
		)
	}
	require.Equal(s.T(), len(serviceIds), len(foundApp3.GetServiceConfigs()))

	// Assert that app1 is unstaked.
	foundApp1, err = s.GetAppQueryClient().GetApplication(s.SdkCtx(), s.app1Bech32)
	require.ErrorContains(s.T(), err, "application not found")
	require.ErrorContains(s.T(), err, s.app1Bech32)

	// Assert that app2 is unstaked.
	_, err = s.GetAppQueryClient().GetApplication(s.SdkCtx(), s.app2Bech32)
	require.ErrorContains(s.T(), err, "application not found")
	require.ErrorContains(s.T(), err, s.app2Bech32)

	// Assert that app1's bank balance has not changed
	balRes, err := s.GetBankQueryClient().Balance(s.SdkCtx(), &banktypes.QueryBalanceRequest{
		Address: s.app1Bech32,
		Denom:   volatile.DenomuPOKT,
	})
	require.NoError(s.T(), err)
	require.Equal(s.T(), appFundAmount-stakeAmount, balRes.GetBalance().Amount.Int64())

	// Assert that app2's bank balance has not changed
	balRes, err = s.GetBankQueryClient().Balance(s.SdkCtx(), &banktypes.QueryBalanceRequest{
		Address: s.app2Bech32,
		Denom:   volatile.DenomuPOKT,
	})
	require.NoError(s.T(), err)
	require.Equal(s.T(), appFundAmount-stakeAmount, balRes.GetBalance().Amount.Int64())
}

// TODO_TEST:
//func (s *AppTransferSuite) TestSequentialTransfersSucceed() {}

// TODO_TEST: Scenario: User cannot start an Application stake transfer from Application which has a pending transfer
// TODO_TEST: Scenario: The user cannot unstake an Application which has a pending transfer
// TODO_TEST: Scenario: The user can (un/re-)delegate an Application which has a pending transfer

// setupTestAddresses sets up the required addresses for the test suite using
// pre-generated accounts.
func (s *AppTransferSuite) setupTestAddresses() {
	s.gateway1Bech32 = s.setupTestAccount().Address.String()
	s.gateway2Bech32 = s.setupTestAccount().Address.String()
	s.gateway3Bech32 = s.setupTestAccount().Address.String()
	s.app1Bech32 = s.setupTestAccount().Address.String()
	s.app2Bech32 = s.setupTestAccount().Address.String()
	s.app3Bech32 = s.setupTestAccount().Address.String()
}

func (s *AppTransferSuite) setupTestAccount() *testkeyring.PreGeneratedAccount {
	appAccount, ok := s.GetApp().GetPreGeneratedAccounts().Next()
	require.Truef(s.T(), ok, "insufficient pre-generated accounts available")

	s.FundAddress(s.T(), appAccount.Address, appFundAmount)

	return appAccount
}

// TODO_IN_THIS_COMMIT: move & godoc...
func (s *AppTransferSuite) setupStakeGateways() {
	gatewayBech32s := []string{
		s.gateway1Bech32,
		s.gateway2Bech32,
		s.gateway3Bech32,
	}

	for _, gatewayBech32 := range gatewayBech32s {
		gwStakeRes := s.gatewaySuite.StakeGateway(s.T(), gatewayBech32, stakeAmount)
		require.Equal(s.T(), gatewayBech32, gwStakeRes.GetGateway().GetAddress())
		require.Equal(s.T(), stakeAmount, gwStakeRes.GetGateway().GetStake().Amount.Int64())
	}
}

// TODO_IN_THIS_COMMIT: move & godoc...
func (s *AppTransferSuite) setupStakeApps(appBech32ToServiceIdsMap map[string][]string) {
	// Stake application.
	for appBech32, serviceIds := range appBech32ToServiceIdsMap {
		stakeAppRes := s.StakeApp(s.T(), appBech32, stakeAmount, serviceIds)
		require.Equal(s.T(), appBech32, stakeAppRes.GetApplication().GetAddress())
		require.Equal(s.T(), stakeAmount, stakeAppRes.GetApplication().GetStake().Amount.Int64())

		// Assert the on-chain state shows the application as staked.
		foundApp, queryErr := s.GetAppQueryClient().GetApplication(s.SdkCtx(), appBech32)
		require.NoError(s.T(), queryErr)
		require.Equal(s.T(), appBech32, foundApp.GetAddress())
		require.Equal(s.T(), stakeAmount, foundApp.GetStake().Amount.Int64())
	}
}

// TODO_IN_THIS_COMMIT: move & godoc...
func (s *AppTransferSuite) setupDelegateAppsToGateway(appBech32ToServiceIdsMap map[string][]string) {
	for appBech32, gatewayBech32s := range appBech32ToServiceIdsMap {
		for _, gatewayBech32 := range gatewayBech32s {
			delegateRes := s.DelegateAppToGateway(s.T(), appBech32, gatewayBech32)
			require.Equal(s.T(), appBech32, delegateRes.GetApplication().GetAddress())
			require.Contains(s.T(), delegateRes.GetApplication().GetDelegateeGatewayAddresses(), gatewayBech32)
		}
	}
}

// shouldObserveTransferBeginEvent asserts that the transfer begin event from
// expectedSrcApp to expectedDstAppBech32 is observed.
func (s *AppTransferSuite) shouldObserveTransferBeginEvent(
	expectedSrcApp *apptypes.Application,
	expectedDstAppBech32 string,
) {
	eventTypeURL := cosmostypes.MsgTypeURL(&apptypes.EventTransferBegin{})
	transferBeginEvents := s.FilterLatestEvents(events2.NewEventTypeMatchFn(eventTypeURL))
	require.NotEmpty(s.T(), transferBeginEvents)

	transferBeginEvent := new(cosmostypes.Event)
	for _, event := range transferBeginEvents {
		eventSrcAddr, hasSrcAddr := s.GetAttributeValue(event, "source_address")
		require.True(s.T(), hasSrcAddr)

		if eventSrcAddr == expectedSrcApp.GetAddress() {
			transferBeginEvent = event
			break
		}
	}
	require.NotNil(s.T(), transferBeginEvent)

	evtSrcAddr, hasSrcAddr := s.GetAttributeValue(transferBeginEvent, "source_address")
	require.True(s.T(), hasSrcAddr)
	require.Equal(s.T(), expectedSrcApp.GetAddress(), evtSrcAddr)

	evtDstAddr, hasDstAddr := s.GetAttributeValue(transferBeginEvent, "destination_address")
	require.True(s.T(), hasDstAddr)
	require.Equal(s.T(), expectedDstAppBech32, evtDstAddr)

	evtSrcApp := new(apptypes.Application)
	evtSrcAppStr, hasSrcApp := s.GetAttributeValue(transferBeginEvent, "source_application")
	require.True(s.T(), hasSrcApp)

	evtSrcAppBz := []byte(evtSrcAppStr)
	err := s.GetApp().GetCodec().UnmarshalJSON(evtSrcAppBz, evtSrcApp)
	require.NoError(s.T(), err)
	require.EqualValues(s.T(), expectedSrcApp.GetPendingTransfer(), evtSrcApp.GetPendingTransfer())
}

// shouldObserveTransferEndEvent asserts that the transfer end event from
// expectedSrcAppBech32 to expectedDstApp is observed.
func (s *AppTransferSuite) shouldObserveTransferEndEvent(
	expectedDstApp *apptypes.Application,
	expectedSrcAppBech32 string,
) {
	eventTypeURL := cosmostypes.MsgTypeURL(&apptypes.EventTransferEnd{})
	transferEndEvent := s.LatestMatchingEvent(events2.NewEventTypeMatchFn(eventTypeURL))
	require.NotNil(s.T(), transferEndEvent)

	evtSrcAddr, hasSrcAddrAttr := s.GetAttributeValue(transferEndEvent, "source_address")
	require.True(s.T(), hasSrcAddrAttr)
	require.Equal(s.T(), expectedSrcAppBech32, evtSrcAddr)

	evtDstAddr, hasDstAddrAttr := s.GetAttributeValue(transferEndEvent, "destination_address")
	require.True(s.T(), hasDstAddrAttr)
	require.Equal(s.T(), expectedDstApp.GetAddress(), evtDstAddr)

	evtDstApp := new(apptypes.Application)
	evtDstAppStr, hasDstAppAttr := s.GetAttributeValue(transferEndEvent, "destination_application")
	require.True(s.T(), hasDstAppAttr)

	evtDstAppBz := []byte(evtDstAppStr)
	err := s.GetApp().GetCodec().UnmarshalJSON(evtDstAppBz, evtDstApp)
	require.NoError(s.T(), err)
	require.EqualValues(s.T(), expectedDstApp.GetPendingTransfer(), evtDstApp.GetPendingTransfer())
}

// TestAppTransferSuite runs the application transfer test suite.
func TestAppTransferSuite(t *testing.T) {
	suite.Run(t, new(AppTransferSuite))
}
