//go:build integration

package suites

import "github.com/pokt-network/poktroll/testutil/integration"

// TODO_IN_THIS_COMMIT: godoc
type IntegrationSuite interface {
	GetApp() *integration.App
	GetModuleNames() []string
}
