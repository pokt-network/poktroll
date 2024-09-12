//go:build integration

package suites

import (
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/app/volatile"
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

// TODO_IN_THIS_COMMIT: godoc
type BaseIntegrationSuite struct {
	suite.Suite
	app *integration.App
}

// TODO_IN_THIS_COMMIT: godoc
func (s *BaseIntegrationSuite) NewApp(t *testing.T) *integration.App {
	t.Helper()

	s.app = integration.NewCompleteIntegrationApp(t)
	return s.app
}

// TODO_IN_THIS_COMMIT: godoc
func (s *BaseIntegrationSuite) GetApp(t *testing.T) *integration.App {
	t.Helper()

	// Implicitly construct and assign a new app on first call
	// if one was not explicitly constructed elsewhere.
	if s.app == nil {
		return s.NewApp(t)
	}
	return s.app
}

// TODO_IN_THIS_COMMIT: godoc
func (s *BaseIntegrationSuite) GetModuleNames() []string {
	return allPoktrollModuleNames
}

// TODO_IN_THIS_COMMIT: godoc
func (s *BaseIntegrationSuite) FundAddress(
	t *testing.T,
	//app *integration.App,
	addr cosmostypes.AccAddress,
	amountUpokt int64,
) {
	coinUpokt := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, amountUpokt)
	sendToAppMsg := &banktypes.MsgSend{
		FromAddress: integration.FaucetAddrStr,
		ToAddress:   addr.String(),
		Amount:      cosmostypes.NewCoins(coinUpokt),
	}

	anyRes := s.GetApp(t).RunMsg(t, sendToAppMsg, integration.RunUntilNextBlockOpts...)
	require.NotNil(t, anyRes)

	sendRes := new(banktypes.MsgSendResponse)
	err := s.GetApp(t).GetCodec().UnpackAny(anyRes, &sendRes)
	require.NoError(t, err)

	// NB: no use in returning sendRes because it has no fields.
}
