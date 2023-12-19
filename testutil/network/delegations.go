package network

import (
	"fmt"
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client/flags"
	testcli "github.com/cosmos/cosmos-sdk/testutil/cli"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/x/application/client/cli"
)

// DelegateAppToGateway delegates the provided application to the provided gateway
func DelegateAppToGateway(
	t *testing.T,
	net *Network,
	appAddr string,
	gatewayAddr string,
) {
	t.Helper()

	val := net.Validators[0]
	ctx := val.ClientCtx
	args := []string{
		gatewayAddr,
		fmt.Sprintf("--%s=%s", flags.FlagFrom, appAddr),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, types.NewCoins(types.NewCoin(net.Config.BondDenom, math.NewInt(10))).String()),
	}
	responseRaw, err := testcli.ExecTestCLICmd(ctx, cli.CmdDelegateToGateway(), args)
	require.NoError(t, err)
	var resp types.TxResponse
	require.NoError(t, net.Config.Codec.UnmarshalJSON(responseRaw.Bytes(), &resp))
	require.NotNil(t, resp)
	require.NotNil(t, resp.TxHash)
	require.Equal(t, uint32(0), resp.Code)
}

// UndelegateAppFromGateway undelegates the provided application from the provided gateway
func UndelegateAppFromGateway(
	t *testing.T,
	net *Network,
	appAddr string,
	gatewayAddr string,
) {
	t.Helper()

	val := net.Validators[0]
	ctx := val.ClientCtx
	args := []string{
		gatewayAddr,
		fmt.Sprintf("--%s=%s", flags.FlagFrom, appAddr),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, types.NewCoins(types.NewCoin(net.Config.BondDenom, math.NewInt(10))).String()),
	}
	responseRaw, err := testcli.ExecTestCLICmd(ctx, cli.CmdUndelegateFromGateway(), args)
	require.NoError(t, err)
	var resp types.TxResponse
	require.NoError(t, net.Config.Codec.UnmarshalJSON(responseRaw.Bytes(), &resp))
	require.NotNil(t, resp)
	require.NotNil(t, resp.TxHash)
	require.Equal(t, uint32(0), resp.Code)
}
