//go:build integration

package suites

import (
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/testutil/events"
	"github.com/pokt-network/poktroll/testutil/integration"
	"github.com/pokt-network/poktroll/testutil/sample"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
)

var gatewayStakeAmount = int64(1000)

type BaseIntegrationSuiteTestSuite struct {
	BaseIntegrationSuite
}

func (s *BaseIntegrationSuite) SetupTest() {
	// Reset app to nil before each test.
	s.app = nil
}

func (s *BaseIntegrationSuiteTestSuite) TestGetApp_PanicsIfNil() {
	require.Nil(s.T(), s.app)

	// Expect the call to GetApp() to panic, defer recovery to check.
	didPanic := false
	defer func() {
		if r := recover(); r != nil {
			didPanic = true
		}

		// Assertion MUST follow recovery; i.e., run last.
		require.True(s.T(), didPanic, "expected call to s.GetApp() to panic when app is nil")
	}()

	// Trigger panic. 🚨
	s.GetApp()
}

func (s *BaseIntegrationSuiteTestSuite) TestNewApp() {
	require.Nil(s.T(), s.app)

	app := s.NewApp(s.T())
	require.Same(s.T(), app, s.app)
}

func (s *BaseIntegrationSuiteTestSuite) TestGetApp_ReturnsApp() {
	app := s.NewApp(s.T())
	require.Same(s.T(), app, s.GetApp())
}
func (s *BaseIntegrationSuiteTestSuite) TestSetApp() {
	// Construct an app.
	app := s.NewApp(s.T())

	// Reset s.app to nil.
	s.app = nil

	s.SetApp(app)
	require.Same(s.T(), app, s.app)
}

func (s *BaseIntegrationSuiteTestSuite) TestGetModuleNames() {
	moduleNames := s.GetModuleNames()
	require.ElementsMatch(s.T(), allPoktrollModuleNames, moduleNames)
}

func (s *BaseIntegrationSuiteTestSuite) TestSdkCtx() {
	s.NewApp(s.T())
	sdkCtx := s.SdkCtx()

	require.NotNil(s.T(), sdkCtx)
	require.Greater(s.T(), sdkCtx.BlockHeight(), int64(0))
}

func (s *BaseIntegrationSuiteTestSuite) TestFundAddressAndGetBankQueryClient() {
	s.NewApp(s.T())
	fundAmount := int64(1000)
	fundAddr, err := cosmostypes.AccAddressFromBech32(sample.AccAddress())
	require.NoError(s.T(), err)

	// Assert that the balance is zero before funding.
	bankQueryClient := s.GetBankQueryClient()
	balRes, err := bankQueryClient.Balance(s.SdkCtx(), &banktypes.QueryBalanceRequest{
		Address: fundAddr.String(),
		Denom:   volatile.DenomuPOKT,
	})
	require.NoError(s.T(), err)
	require.Equal(s.T(), int64(0), balRes.GetBalance().Amount.Int64())

	// Fund the address.
	s.FundAddress(s.T(), fundAddr, fundAmount)

	// Assert that the balance amount is equal to fundAmount.
	balRes, err = bankQueryClient.Balance(s.SdkCtx(), &banktypes.QueryBalanceRequest{
		Address: fundAddr.String(),
		Denom:   volatile.DenomuPOKT,
	})
	require.NoError(s.T(), err)
	require.Equal(s.T(), fundAmount, balRes.GetBalance().Amount.Int64())
}

func (s *BaseIntegrationSuiteTestSuite) TestFilterLatestEventsWithNewMsgEventMatchFn() {
	expectedNumEvents := 3

	s.NewApp(s.T())
	s.generateBankMsgSendEvents(expectedNumEvents)

	// Filter for the "message" type event with "action" attribute value
	// equal to the MsgSend TypeURL.
	msgSendTypeURL := cosmostypes.MsgTypeURL(&banktypes.MsgSend{})
	matchedEvents := s.FilterLatestEvents(events.NewMsgEventMatchFn(msgSendTypeURL))

	require.Equal(s.T(), expectedNumEvents, len(matchedEvents), "unexpected number of matched events")

	// Assert that the events are cleared on the next block.
	s.GetApp().NextBlock(s.T())
	require.Equal(s.T(), 0, len(s.SdkCtx().EventManager().Events()), "expected no events in the next block")
}

func (s *BaseIntegrationSuiteTestSuite) TestFilterLatestEventsWithNewEventTypeMatchFn() {
	expectedNumEvents := 3

	s.NewApp(s.T())
	s.emitPoktrollGatewayUnstakedEvents(expectedNumEvents)

	// Filter for the event with type equal to the EventGatewayUnstaked TypeURL.
	eventGatewayUnstakedTypeURL := cosmostypes.MsgTypeURL(&gatewaytypes.EventGatewayUnstaked{})
	matchedEvents := s.FilterLatestEvents(events.NewEventTypeMatchFn(eventGatewayUnstakedTypeURL))

	require.Equal(s.T(), expectedNumEvents, len(matchedEvents), "unexpected number of matched events")

	// Assert that the events are cleared on the next block.
	s.GetApp().NextBlock(s.T())
	require.Equal(s.T(), 0, len(s.SdkCtx().EventManager().Events()), "expected no events in the next block")
}

func (s *BaseIntegrationSuiteTestSuite) TestGetAttributeValue() {
	s.NewApp(s.T())
	s.generateBankMsgSendEvents(1)

	testEvents := s.SdkCtx().EventManager().Events()
	// NB: 5 events are emitted for a single MsgSend:
	// - message
	// - coin_spent
	// - coin_received
	// - transfer
	// - coinbase
	require.Equal(s.T(), 5, len(testEvents), "expected 5 events")
	event := testEvents[0]

	// Get the matched event and check its attributes.
	value, hasAttr := s.GetAttributeValue(&event, "module")
	require.True(s.T(), hasAttr)
	require.Equal(s.T(), banktypes.ModuleName, value)
}

// TODO_IN_THIS_COMMIT: godoc...
func (s *BaseIntegrationSuiteTestSuite) generateBankMsgSendEvents(expectedNumEvents int) {
	msgs := make([]cosmostypes.Msg, 0)

	for i := 0; i < expectedNumEvents; i++ {
		faucetAddr, err := cosmostypes.AccAddressFromBech32(integration.FaucetAddrStr)
		require.NoError(s.T(), err)

		randomAddr, err := cosmostypes.AccAddressFromBech32(sample.AccAddress())
		require.NoError(s.T(), err)

		sendMsg := banktypes.NewMsgSend(
			faucetAddr,
			randomAddr,
			cosmostypes.NewCoins(cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 100)),
		)
		msgs = append(msgs, sendMsg)
	}

	s.GetApp().RunMsg(s.T(), nil, msgs...)
}

// TODO_IN_THIS_COMMIT: godoc...
func (s *BaseIntegrationSuiteTestSuite) emitPoktrollGatewayUnstakedEvents(expectedNumEvents int) {
	for i := 0; i < expectedNumEvents; i++ {
		err := s.SdkCtx().EventManager().EmitTypedEvent(&gatewaytypes.EventGatewayUnstaked{
			Address: sample.AccAddress(),
		})
		require.NoError(s.T(), err)
	}
}

func TestBaseIntegrationSuite(t *testing.T) {
	suite.Run(t, new(BaseIntegrationSuiteTestSuite))
}
