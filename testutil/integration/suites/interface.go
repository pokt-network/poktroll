//go:build integration

package suites

import (
	"testing"

	"github.com/pokt-network/poktroll/testutil/integration"
)

// TODO_IN_THIS_COMMIT: godoc
type IntegrationSuite interface {
	NewApp(t *testing.T) *integration.App
	GetApp(t *testing.T) *integration.App
	GetModuleNames() []string
}
