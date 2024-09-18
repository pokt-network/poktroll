//go:build integration

package suites

import (
	"strings"
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/testutil/integration"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// TODO_IMPROVE: Ideally this list should be populated during integration app construction.
var allPoktrollModuleNames = []string{
	sharedtypes.ModuleName,
	sessiontypes.ModuleName,
	servicetypes.ModuleName,
	apptypes.ModuleName,
	gatewaytypes.ModuleName,
	suppliertypes.ModuleName,
	prooftypes.ModuleName,
	tokenomicstypes.ModuleName,
}

var _ IntegrationSuite = (*BaseIntegrationSuite)(nil)

// BaseIntegrationSuite is a base implementation of IntegrationSuite.
// It is intended to be embedded in other integration test suites.
type BaseIntegrationSuite struct {
	suite.Suite
	app *integration.App

	appQueryClient client.ApplicationQueryClient
}

// NewApp constructs a new integration app and sets it on the suite.
func (s *BaseIntegrationSuite) NewApp(t *testing.T) *integration.App {
	t.Helper()

	s.app = integration.NewCompleteIntegrationApp(t)
	return s.app
}

// SetApp sets the integration app on the suite.
func (s *BaseIntegrationSuite) SetApp(app *integration.App) {
	s.app = app
}

// GetApp returns the integration app from the suite.
func (s *BaseIntegrationSuite) GetApp() *integration.App {
	if s.app == nil {
		panic("integration app is nil; use NewApp or SetApp before calling GetApp")
	}
	return s.app
}

// GetModuleNames returns the list of all poktroll modules names in the integration app.
func (s *BaseIntegrationSuite) GetModuleNames() []string {
	return allPoktrollModuleNames
}

// FundAddress sends amountUpokt coins from the faucet to the given address.
func (s *BaseIntegrationSuite) FundAddress(
	t *testing.T,
	addr cosmostypes.AccAddress,
	amountUpokt int64,
) {
	coinUpokt := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, amountUpokt)

	// TODO_IN_THIS_COMMIT: HACK: get a reference to the bank keeper directly
	// via the integration app to send tokens.

	//faucetAddr := cosmostypes.MustAccAddressFromBech32(integration.FaucetAddrStr)
	//
	//err := s.GetApp().GetBankKeeper().SendCoins(
	//	s.GetApp().GetSdkCtx(),
	//	faucetAddr,
	//	addr,
	//	cosmostypes.NewCoins(coinUpokt),
	//)
	//require.NoError(t, err)

	sendMsg := &banktypes.MsgSend{
		FromAddress: integration.FaucetAddrStr,
		ToAddress:   addr.String(),
		Amount:      cosmostypes.NewCoins(coinUpokt),
	}

	txMsgRes := s.GetApp().RunMsg(t, sendMsg, integration.RunUntilNextBlockOpts...)
	require.NotNil(t, txMsgRes)

	// NB: no use in returning sendRes because it has no fields.
}

func (s *BaseIntegrationSuite) GetBankQueryClient() banktypes.QueryClient {
	return banktypes.NewQueryClient(s.GetApp().QueryHelper())
}

// TODO_IN_THIS_COMMIT: godoc...
func (s *BaseIntegrationSuite) FilterEvents(matchFn func(*cosmostypes.Event) bool) (matchedEvents []*cosmostypes.Event) {
	return s.filterEvents(matchFn, false)
}

// TODO_IN_THIS_COMMIT: godoc...
func (s *BaseIntegrationSuite) LatestMatchingEvent(matchFn func(*cosmostypes.Event) bool) (matchedEvent *cosmostypes.Event) {
	filteredEvents := s.filterEvents(matchFn, true)

	if len(filteredEvents) < 1 {
		return nil
	}

	return filteredEvents[0]
}

// TODO_IN_THIS_COMMIT: godoc...
func (s *BaseIntegrationSuite) GetAttributeValue(event cosmostypes.Event, key string) (value string, hasAttr bool) {
	attr, hasAttr := event.GetAttribute(key)
	if !hasAttr {
		return "", false
	}

	return strings.Trim(attr.GetValue(), "\""), true
}

// TODO_IN_THIS_COMMIT: godoc...
// TODO_IN_THIS_COMMIT: consolidate with testutil/events/filter.go
func (s *BaseIntegrationSuite) filterEvents(
	matchFn func(*cosmostypes.Event) bool,
	latestOnly bool,
) (matchedEvents []*cosmostypes.Event) {
	events := s.GetApp().GetSdkCtx().EventManager().Events()

	// TODO_IN_THIS_COMMIT: comment about why reverse order and/or figure out why events accumulate across blocks.
	for i := len(events) - 1; i >= 0; i-- {
		// TODO_IN_THIS_COMMIT: double-check that there's no issue here with
		// pointing to a variable which is reused by the loop.
		event := events[i]

		if !matchFn(&event) {
			continue
		}

		matchedEvents = append(matchedEvents, &event)

		if latestOnly {
			break
		}
	}

	return matchedEvents
}
