//go:build integration

package suites

import (
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/volatile"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
)

var _ IntegrationSuite = (*GatewayModuleSuite)(nil)

type GatewayModuleSuite struct {
	BaseIntegrationSuite
}

func (s *GatewayModuleSuite) StakeGateway(
	t *testing.T,
	bech32 string,
	stakeAmtUpokt int64,
) *gatewaytypes.MsgStakeGatewayResponse {
	t.Helper()

	stakeGatewayMsg := gatewaytypes.NewMsgStakeGateway(
		bech32,
		cosmostypes.NewInt64Coin(volatile.DenomuPOKT, stakeAmtUpokt),
	)

	txMsgRes, err := s.GetApp().RunMsg(t, stakeGatewayMsg)
	require.NoError(t, err)

	return txMsgRes.(*gatewaytypes.MsgStakeGatewayResponse)
}
