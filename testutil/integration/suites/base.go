package suites

import (
	"reflect"
	"strings"
	"testing"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/depinject"
	"github.com/cosmos/cosmos-sdk/codec"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/query"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	_ "github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/testutil/integration"
	"github.com/pokt-network/poktroll/testutil/testcache"
)

var _ IntegrationSuite = (*BaseIntegrationSuite)(nil)

// BaseIntegrationSuite is a base implementation of IntegrationSuite.
// It is intended to be embedded in other integration test suites.
type BaseIntegrationSuite struct {
	suite.Suite
	app *integration.App

	pocketModuleNames []string
	cosmosModuleNames []string
}

// NewApp constructs a new integration app and sets it on the suite.
func (s *BaseIntegrationSuite) NewApp(t *testing.T, opts ...integration.IntegrationAppOptionFn) *integration.App {
	t.Helper()

	s.pocketModuleNames = nil
	s.cosmosModuleNames = nil

	defaultIntegrationAppOption := integration.WithInitChainerModuleFn(newInitChainerCollectModuleNamesFn(s))
	opts = append([]integration.IntegrationAppOptionFn{defaultIntegrationAppOption}, opts...)
	s.app = integration.NewCompleteIntegrationApp(t, opts...)
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

// GetPocketModuleNames returns the list of the names of all pocket modules
// in the integration app.
func (s *BaseIntegrationSuite) GetPocketModuleNames() []string {
	return s.pocketModuleNames
}

// GetCosmosModuleNames returns the list of the names of all cosmos-sdk modules
// in the integration app.
func (s *BaseIntegrationSuite) GetCosmosModuleNames() []string {
	return s.cosmosModuleNames
}

// SdkCtx returns the integration app's SDK context.
func (s *BaseIntegrationSuite) SdkCtx() *cosmostypes.Context {
	return s.GetApp().GetSdkCtx()

}

// FundAddress sends amountUpokt coins from the faucet to the given address.
func (s *BaseIntegrationSuite) FundAddress(
	t *testing.T,
	addr cosmostypes.AccAddress,
	amountUpokt int64,
) {
	coinUpokt := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, amountUpokt)
	sendMsg := &banktypes.MsgSend{
		FromAddress: s.GetApp().GetFaucetBech32(),
		ToAddress:   addr.String(),
		Amount:      cosmostypes.NewCoins(coinUpokt),
	}

	txMsgRes, err := s.GetApp().RunMsg(t, sendMsg)
	require.NoError(t, err)
	require.NotNil(t, txMsgRes)

	// NB: no use in returning sendRes because it has no fields.
}

// GetBankQueryClient constructs and returns a query client for the bank module
// of the integration app.
func (s *BaseIntegrationSuite) GetBankQueryClient(t *testing.T) client.BankQueryClient {
	t.Helper()

	deps := depinject.Supply(
		polyzero.NewLogger(),
		testcache.NewNoopKeyValueCache[query.Balance](),
		s.GetApp().QueryHelper(),
	)
	bankQueryClient, err := query.NewBankQuerier(deps)
	require.NoError(t, err)

	return bankQueryClient
}

// FilterEvents returns the events from the event manager which match the given
// matchFn. Events are returned in reverse order, i.e. the most recent event is
// first.
func (s *BaseIntegrationSuite) FilterEvents(
	matchFn func(*cosmostypes.Event) bool,
) (matchedEvents []*cosmostypes.Event) {
	return s.filterEvents(matchFn, false)
}

// LatestMatchingEvent returns the most recent event in the event manager that
// matches the given matchFn.
func (s *BaseIntegrationSuite) LatestMatchingEvent(
	matchFn func(*cosmostypes.Event) bool,
) (matchedEvent *cosmostypes.Event) {
	filteredEvents := s.filterEvents(matchFn, true)

	if len(filteredEvents) < 1 {
		return nil
	}

	return filteredEvents[0]
}

// filterEvents returns the events from the event manager that match the given matchFn.
// If latestOnly is true, then only the most recent matching event is returned.
//
// TODO_IMPROVE: consolidate with testutil/events/filter.go
func (s *BaseIntegrationSuite) filterEvents(
	matchFn func(*cosmostypes.Event) bool,
	latestOnly bool,
) (matchedEvents []*cosmostypes.Event) {
	events := s.GetApp().GetSdkCtx().EventManager().Events()

	// NB: Iterate in reverse to get the latest events first.
	for i := len(events) - 1; i >= 0; i-- {
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

// newInitChainerCollectModuleNamesFn returns an InitChainerModuleFn that collects
// the names of cosmos and pocket modules in their respective suite field slices.
func newInitChainerCollectModuleNamesFn(suite *BaseIntegrationSuite) integration.InitChainerModuleFn {
	return func(ctx cosmostypes.Context, cdc codec.Codec, mod appmodule.AppModule) {
		modName, hasName := mod.(module.HasName)
		if !hasName {
			polylog.DefaultContextLogger.Warn().Msg("unable to get module name")
		}

		modType := reflect.TypeOf(mod)
		// TODO_POST_MAINNET: replace "poktroll" with "pocket" once the repo rename is complete.
		if strings.Contains(modType.PkgPath(), "poktroll") {
			suite.pocketModuleNames = append(suite.pocketModuleNames, modName.Name())
			return
		}

		// NB: We can assume that any non-pocket module is a cosmos-sdk module
		// so long as we're not importing any third-party modules; in which case,
		// we would have to add another check above.
		suite.cosmosModuleNames = append(suite.cosmosModuleNames, modName.Name())
	}
}
