package cli_test

import (
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
	"github.com/pokt-network/poktroll/testutil/network/sessionnet"
	"github.com/pokt-network/poktroll/x/application/client/cli"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
)

func TestCLI_UnstakeApplication(t *testing.T) {
	memnet := sessionnet.NewInMemoryNetworkWithSessions(
		&network.InMemoryNetworkConfig{
			NumApplications: 2,
		},
	)
	memnet.Start(t)

	appGenesisState := sessionnet.GetGenesisState[*apptypes.GenesisState](t, apptypes.ModuleName, memnet)
	applications := appGenesisState.ApplicationList
	appBech32 := applications[0].GetAddress()
	appAddr, err := sdk.AccAddressFromBech32(appBech32)
	require.NoError(t, err)

	net := memnet.GetNetwork(t)

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
			desc:    "unstake application: valid",
			address: appBech32,
		},
		{
			desc: "unstake application: missing address",
			// address:     "explicitly missing",
			err: apptypes.ErrAppInvalidAddress,
		},
		{
			desc:    "unstake application: invalid address",
			address: "invalid",
			err:     apptypes.ErrAppInvalidAddress,
		},
	}

	// Initialize the App Account by sending it some funds from the validator account that is part of genesis
	sessionnet.InitAccount(t, net, appAddr)

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
			outUnstake, err := testcli.ExecTestCLICmd(memnet.GetClientCtx(t), cli.CmdUnstakeApplication(), args)

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
