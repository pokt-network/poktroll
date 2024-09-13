//go:build integration

package suites

import (
	"github.com/stretchr/testify/suite"

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

var _ IntegrationSuite = (*BaseIntegrationSuite)(nil)

// TODO_IN_THIS_COMMIT: godoc
type BaseIntegrationSuite struct {
	suite.Suite
	app *integration.App
}

// TODO_IN_THIS_COMMIT: godoc
func (s *BaseIntegrationSuite) GetApp() *integration.App {
	// Construct and assign a new app on first call.
	if s.app == nil {
		s.app = integration.NewCompleteIntegrationApp(s.T())
	}
	return s.app
}

func (s *BaseIntegrationSuite) GetModuleNames() []string {
	return allPoktrollModuleNames
}

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
