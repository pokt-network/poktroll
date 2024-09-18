//go:build integration

package suites

import (
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/pokt-network/poktroll/testutil/integration"
)

// IntegrationSuite is an interface intended to be used within test suites which
// exercise an integration.App.
type IntegrationSuite interface {
	// NewApp constructs a new integration app and sets it on the suite.
	NewApp(*testing.T) *integration.App
	// SetApp sets the integration app on the suite.
	SetApp(*integration.App)
	// GetApp returns the integration app from the suite.
	GetApp() *integration.App
	// GetModuleNames returns the list of all poktroll modules names in the integration app.
	GetModuleNames() []string
	// SdkCtx returns the integration app's SDK context.
	SdkCtx() *cosmostypes.Context

	// FundAddress sends amtUpokt coins from the faucet to the given address.
	FundAddress(t *testing.T, addr cosmostypes.AccAddress, amtUpokt int64, opts ...integration.RunOption)
	// GetBankQueryClient returns a query client for the bank module of the integration app
	GetBankQueryClient() banktypes.QueryClient

	// TODO_IN_THIS_COMMIT: godoc...
	FilterLatestEvents(matchFn func(*cosmostypes.Event) bool) []*cosmostypes.Event
	// TODO_IN_THIS_COMMIT: godoc...
	LatestMatchingEvent(matchFn func(*cosmostypes.Event) bool) *cosmostypes.Event
	// TODO_IN_THIS_COMMIT: godoc...
	GetAttributeValue(event *cosmostypes.Event, key string) (value string, hasAttr bool)
}
