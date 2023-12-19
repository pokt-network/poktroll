package cli_test

import (
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
	memnet := sessionnet.NewInMemoryNetworkWithSessions(
		&network.InMemoryNetworkConfig{
			NumApplications: 2,
		},
	)
	memnet.Start(t)

	appGenesisState := sessionnet.GetGenesisState[*apptypes.GenesisState](t, apptypes.ModuleName, memnet)
	applications := appGenesisState.ApplicationList

	net := memnet.GetNetwork(t)
	ctx := memnet.GetClientCtx(t)
	appBech32 := applications[0].GetAddress()
	//appAddr, err := sdk.AccAddressFromBech32(appBech32)
	//require.NoError(t, err)

	gatewayBech32 := applications[1].GetAddress()
	//gatewayAddr, err := sdk.AccAddressFromBech32(gatewayBech32)
	//require.NoError(t, err)

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

	//// Initialize the App and Gateway Accounts by sending it some funds from the validator account that is part of genesis
	//sessionnet.InitAccountWithSequence(t, net, appAddr, 1)
	//sessionnet.InitAccountWithSequence(t, net, gatewayAddr, 2)

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
			undelegateOutput, err := clitestutil.ExecTestCLICmd(ctx, cli.CmdUndelegateFromGateway(), args)

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
