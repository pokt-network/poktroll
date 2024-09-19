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
	NewApp(*testing.T, ...integration.IntegrationAppOption) *integration.App
	// SetApp sets the integration app on the suite.
	SetApp(*integration.App)
	// GetApp returns the integration app from the suite.
	GetApp() *integration.App
	// GetPoktrollModuleNames returns the list of the names of all poktroll modules
	// in the integration app.
	GetPoktrollModuleNames() []string
	// GetCosmosModuleNames returns the list of the names of all cosmos-sdk modules
	// in the integration app.
	GetCosmosModuleNames() []string
	// SdkCtx returns the integration app's SDK context.
	SdkCtx() *cosmostypes.Context

	// FundAddress sends amtUpokt coins from the faucet to the given address.
	FundAddress(t *testing.T, addr cosmostypes.AccAddress, amtUpokt int64)
	// GetBankQueryClient constructs and returns a query client for the bank module
	// of the integration app.
	GetBankQueryClient() banktypes.QueryClient

	// FilterLatestEvents returns the most recent events in the event manager that
	// match the given matchFn.
	FilterLatestEvents(matchFn func(*cosmostypes.Event) bool) []*cosmostypes.Event
	// LatestMatchingEvent returns the most recent event in the event manager that
	// matches the given matchFn.
	LatestMatchingEvent(matchFn func(*cosmostypes.Event) bool) *cosmostypes.Event
	// GetAttributeValue returns the value of the attribute with the given key in the
	// event. The returned attribute value is trimmed of any quotation marks. If the
	// attribute does not exist, hasAttr is false.
	GetAttributeValue(event *cosmostypes.Event, key string) (value string, hasAttr bool)
}
