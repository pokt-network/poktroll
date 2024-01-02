package cli_test

import (
	"context"
	"fmt"
	"testing"

	sdkerrors "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client/flags"
	testcli "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/testutil/network"
	"github.com/pokt-network/poktroll/testutil/network/gatewaynet"
	"github.com/pokt-network/poktroll/x/gateway/client/cli"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
)

func TestCLI_UnstakeGateway(t *testing.T) {
	ctx := context.Background()
	memnet := gatewaynet.NewInMemoryNetworkWithGateways(
		t, &network.InMemoryNetworkConfig{
			NumGateways: 5,
		},
	)
	memnet.Start(ctx, t)

	clientCtx := memnet.GetClientCtx(t)
	net := memnet.GetNetwork(t)
	gatewayAccount := network.GetGenesisState[*gatewaytypes.GenesisState](t, gatewaytypes.ModuleName, memnet).GatewayList[0]

	// Common args used for all requests
	commonArgs := []string{
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(net.Config.BondDenom, sdkmath.NewInt(10))).String()),
	}

	tests := []struct {
		desc    string
		address string
		err     *sdkerrors.Error
	}{
		{
			desc:    "unstake gateway: valid",
			address: gatewayAccount.GetAddress(),
		},
		{
			desc: "unstake gateway: missing address",
			// address: intentionally omitted,
			err: gatewaytypes.ErrGatewayInvalidAddress,
		},
		{
			desc:    "unstake gateway: invalid address",
			address: "invalid",
			err:     gatewaytypes.ErrGatewayInvalidAddress,
		},
	}

	// Run the tests
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			// Wait for a new block to be committed
			require.NoError(t, net.WaitForNextBlock())

			// Prepare the arguments for the CLI command
			args := []string{
				fmt.Sprintf("--%s=%s", flags.FlagFrom, tt.address),
			}
			args = append(args, commonArgs...)

			// Execute the command
			outUnstake, err := testcli.ExecTestCLICmd(clientCtx, cli.CmdUnstakeGateway(), args)

			// Validate the error if one is expected
			if tt.err != nil {
				stat, ok := status.FromError(tt.err)
				require.True(t, ok)
				require.Contains(t, stat.Message(), tt.err.Error())
				return
			}
			require.NoError(t, err)

			// Check the response
			var resp sdk.TxResponse
			require.NoError(t, net.Config.Codec.UnmarshalJSON(outUnstake.Bytes(), &resp))
			require.NotNil(t, resp)
			require.NotNil(t, resp.TxHash)
			require.Equal(t, uint32(0), resp.Code)
		})
	}
}
