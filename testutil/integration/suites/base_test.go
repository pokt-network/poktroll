package suites

import (
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/testutil/events"
	"github.com/pokt-network/poktroll/testutil/sample"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
)

// baseIntegrationSuiteTestSuite is a test suite which embeds BaseIntegrationSuite.
// **in order to test it**. It is NOT intended to be embedded in other test suites.
type baseIntegrationSuiteTestSuite struct {
	BaseIntegrationSuite
}

func (s *baseIntegrationSuiteTestSuite) SetupTest() {
	// Reset app to nil before each test.
	s.app = nil
}

func (s *baseIntegrationSuiteTestSuite) TestGetApp_PanicsIfNil() {
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

func (s *baseIntegrationSuiteTestSuite) TestNewApp() {
	require.Nil(s.T(), s.app)

	app := s.NewApp(s.T())
	require.Same(s.T(), app, s.app)
}

func (s *baseIntegrationSuiteTestSuite) TestGetApp_ReturnsApp() {
	app := s.NewApp(s.T())
	require.Same(s.T(), app, s.GetApp())
}

func (s *baseIntegrationSuiteTestSuite) TestSetApp() {
	// Construct an app.
	app := s.NewApp(s.T())

	// Reset s.app to nil.
	s.app = nil

	s.SetApp(app)
	require.Same(s.T(), app, s.app)
}

func (s *baseIntegrationSuiteTestSuite) TestGetPoktrollModuleNames() {
	moduleNames := s.GetPoktrollModuleNames()
	require.Greater(s.T(), len(moduleNames), 0, "expected non-empty module names")
	require.ElementsMatch(s.T(), s.poktrollModuleNames, moduleNames)
}

func (s *baseIntegrationSuiteTestSuite) TestGetCosmosModuleNames() {
	moduleNames := s.GetCosmosModuleNames()
	require.Greater(s.T(), len(moduleNames), 0, "expected non-empty module names")
	require.ElementsMatch(s.T(), s.cosmosModuleNames, moduleNames)
}

func (s *baseIntegrationSuiteTestSuite) TestSdkCtx() {
	s.NewApp(s.T())
	sdkCtx := s.SdkCtx()

	require.NotNil(s.T(), sdkCtx)
	require.Greater(s.T(), sdkCtx.BlockHeight(), int64(0))
}

func (s *baseIntegrationSuiteTestSuite) TestFundAddressAndGetBankQueryClient() {
	s.NewApp(s.T())
	fundAmount := int64(1000)
	fundAddr, err := cosmostypes.AccAddressFromBech32(sample.AccAddress())
	require.NoError(s.T(), err)

	// Assert that the balance is zero before funding.
	bankClient := s.GetBankQueryClient(s.T())
	balance, err := bankClient.GetBalance(s.SdkCtx(), fundAddr.String())
	require.NoError(s.T(), err)
	require.Equal(s.T(), int64(0), balance.Amount.Int64())

	// Fund the address.
	s.FundAddress(s.T(), fundAddr, fundAmount)

	// Assert that the balance amount is equal to fundAmount.
	balance, err = bankClient.GetBalance(s.SdkCtx(), fundAddr.String())
	require.NoError(s.T(), err)
	require.Equal(s.T(), fundAmount, balance.Amount.Int64())
}

func (s *baseIntegrationSuiteTestSuite) TestFilterLatestEventsWithNewMsgEventMatchFn() {
	expectedNumEvents := 3

	s.NewApp(s.T())
	s.emitBankMsgSendEvents(expectedNumEvents)

	// Filter for the "message" type event with "action" attribute value
	// equal to the MsgSend TypeURL.
	msgSendTypeURL := cosmostypes.MsgTypeURL(&banktypes.MsgSend{})
	matchedEvents := s.FilterEvents(events.NewMsgEventMatchFn(msgSendTypeURL))

	require.Equal(s.T(), expectedNumEvents, len(matchedEvents), "unexpected number of matched events")

	// Assert that the events are cleared on the next block.
	s.GetApp().NextBlock(s.T())
	require.Equal(s.T(), 0, len(s.SdkCtx().EventManager().Events()), "expected no events in the next block")
}

func (s *baseIntegrationSuiteTestSuite) TestFilterLatestEventsWithNewEventTypeMatchFn() {
	expectedNumEvents := 3
	s.NewApp(s.T())

	// Assert that the event manager is empty before emitting events.
	require.Equal(s.T(), 0, len(s.SdkCtx().EventManager().Events()))

	// Emit the expected number of EventGatewayUnbondingBegin events.
	s.emitPoktrollGatewayUnbondingBeginEvents(expectedNumEvents)

	// Filter for the event with type equal to the EventGatewayUnstaked TypeURL.
	eventGatewayUnbondingBeginTypeURL := cosmostypes.MsgTypeURL(&gatewaytypes.EventGatewayUnbondingBegin{})
	matchedEvents := s.FilterEvents(events.NewEventTypeMatchFn(eventGatewayUnbondingBeginTypeURL))

	require.Equal(s.T(), expectedNumEvents, len(matchedEvents), "unexpected number of matched events")

	// Assert that the events are cleared on the next block.
	s.GetApp().NextBlock(s.T())
	require.Equal(s.T(), 0, len(s.SdkCtx().EventManager().Events()), "expected no events in the next block")
}

func (s *baseIntegrationSuiteTestSuite) TestGetAttributeValue() {
	s.NewApp(s.T())
	s.emitBankMsgSendEvents(1)

	testEvents := s.SdkCtx().EventManager().Events()
	// NB: 5 events are emitted for a single MsgSend:
	// - message
	// - coin_spent
	// - coin_received
	// - transfer
	// - coinbase
	require.Equal(s.T(), 5, len(testEvents), "expected 5 events")

	// Get the "message" event and check its "module" attribute. Cosmos-sdk emits
	// a "message" event alongside other txResult events for each message in a tx.
	event := testEvents[0]
	value, hasAttr := events.GetAttributeValue(&event, "module")
	require.True(s.T(), hasAttr)
	require.Equal(s.T(), banktypes.ModuleName, value)
}

// emitBankMsgSendEvents causes the bank module to emit events as the result
// of handling a MsgSend message which are intended to be used to make assertions
// in tests.
func (s *baseIntegrationSuiteTestSuite) emitBankMsgSendEvents(expectedNumEvents int) {
	msgs := make([]cosmostypes.Msg, 0)

	for i := 0; i < expectedNumEvents; i++ {
		faucetAddr, err := cosmostypes.AccAddressFromBech32(s.GetApp().GetFaucetBech32())
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

	_, err := s.GetApp().RunMsgs(s.T(), msgs...)
	require.NoError(s.T(), err)
}

// emitPoktrollGatewayUnbondingBeginEvents emits the given number of EventGatewayUnbondingBegin
// events to the event manager. These events are intended to be used to make
// assertions in tests.
func (s *baseIntegrationSuiteTestSuite) emitPoktrollGatewayUnbondingBeginEvents(expectedNumEvents int) {
	for i := 0; i < expectedNumEvents; i++ {
		err := s.SdkCtx().EventManager().EmitTypedEvent(&gatewaytypes.EventGatewayUnbondingBegin{
			Gateway: &gatewaytypes.Gateway{
				Address: sample.AccAddress(),
			},
		})
		require.NoError(s.T(), err)
	}
}

// Run the test suite.
func TestBaseIntegrationSuite(t *testing.T) {
	suite.Run(t, new(baseIntegrationSuiteTestSuite))
}
