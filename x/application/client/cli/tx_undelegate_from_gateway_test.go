package cli_test

import (
	"context"
	"fmt"
	"testing"

	sdkerrors "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client/flags"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/testutil/network"
	"github.com/pokt-network/poktroll/testutil/network/sessionnet"
	"github.com/pokt-network/poktroll/x/application/client/cli"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
)

func TestCLI_UndelegateFromGateway(t *testing.T) {
	ctx := context.Background()
	memnet := sessionnet.NewInMemoryNetworkWithSessions(
		t, &network.InMemoryNetworkConfig{
			NumSuppliers:            2,
			AppSupplierPairingRatio: 1,
		},
	)
	memnet.Start(ctx, t)

	appGenesisState := network.GetGenesisState[*apptypes.GenesisState](t, apptypes.ModuleName, memnet)
	applications := appGenesisState.ApplicationList

	net := memnet.GetNetwork(t)
	appBech32 := applications[0].GetAddress()
	gatewayBech32 := applications[1].GetAddress()

	// Common args used for all requests
	commonArgs := []string{
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(net.Config.BondDenom, sdkmath.NewInt(10))).String()),
	}

	tests := []struct {
		desc           string
		appAddress     string
		gatewayAddress string
		err            *sdkerrors.Error
	}{
		{
			desc:           "undelegate from gateway: valid",
			appAddress:     appBech32,
			gatewayAddress: gatewayBech32,
		},
		{
			desc: "invalid - missing app address",
			// appAddress:  intentionally omitted
			gatewayAddress: gatewayBech32,
			err:            apptypes.ErrAppInvalidAddress,
		},
		{
			desc:           "invalid - invalid app address",
			appAddress:     "invalid address",
			gatewayAddress: gatewayBech32,
			err:            apptypes.ErrAppInvalidAddress,
		},
		{
			desc:       "invalid - missing gateway address",
			appAddress: appBech32,
			// gatewayAddress: intentionally omitted
			err: apptypes.ErrAppInvalidGatewayAddress,
		},
		{
			desc:           "invalid - invalid gateway address",
			appAddress:     appBech32,
			gatewayAddress: "invalid address",
			err:            apptypes.ErrAppInvalidGatewayAddress,
		},
	}

	// Run the tests
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			// Wait for a new block to be committed
			require.NoError(t, net.WaitForNextBlock())

			// Prepare the arguments for the CLI command
			args := []string{
				tt.gatewayAddress,
				fmt.Sprintf("--%s=%s", flags.FlagFrom, tt.appAddress),
			}
			args = append(args, commonArgs...)

			// Execute the command
			undelegateOutput, err := clitestutil.ExecTestCLICmd(memnet.GetClientCtx(t), cli.CmdUndelegateFromGateway(), args)

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
			require.NoError(t, net.Config.Codec.UnmarshalJSON(undelegateOutput.Bytes(), &resp))
			require.NotNil(t, resp)
			require.NotNil(t, resp.TxHash)
			require.Equal(t, uint32(0), resp.Code)
		})
	}
}
