//go:build integration

package suites

import (
	"testing"

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
}
