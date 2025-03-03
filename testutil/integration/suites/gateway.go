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

// StakeGateway stakes a gateway with the given address and stake amount.
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

// GetGateway returns the gateway with the given address, if it exists; otherwise an error is returned.
func (s *GatewayModuleSuite) GetGateway(t *testing.T, gatewayAddr string) (*gatewaytypes.Gateway, error) {
	t.Helper()

	gatewyQueryClient := gatewaytypes.NewQueryClient(s.GetApp().QueryHelper())
	res, err := gatewyQueryClient.Gateway(s.SdkCtx(), &gatewaytypes.QueryGetGatewayRequest{
		Address: gatewayAddr,
	})
	if err != nil {
		return nil, err
	}

	return &res.Gateway, nil
}
