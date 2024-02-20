package application_test

import (
	"fmt"
	"testing"

	sdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/testutil"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/testutil/network"
	application "github.com/pokt-network/poktroll/x/application/module"
	"github.com/pokt-network/poktroll/x/application/types"
)

func TestCLI_UnstakeApplication(t *testing.T) {
	net, _ := networkWithApplicationObjects(t, 2)
	val := net.Validators[0]
	ctx := val.ClientCtx

	// Create a keyring and add an account for the application to be unstaked
	kr := ctx.Keyring
	accounts := testutil.CreateKeyringAccounts(t, kr, 1)
	appAccount := accounts[0]

	// Update the context with the new keyring
	ctx = ctx.WithKeyring(kr)

	// Common args used for all requests
	commonArgs := []string{
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(net.Config.BondDenom, math.NewInt(10))).String()),
	}

	tests := []struct {
		desc    string
		address string
		err     *sdkerrors.Error
	}{
		{
			desc:    "unstake application: valid",
			address: appAccount.Address.String(),
		},
		{
			desc: "unstake application: missing address",
			// address:     "explicitly missing",
			err: types.ErrAppInvalidAddress,
		},
		{
			desc:    "unstake application: invalid address",
			address: "invalid",
			err:     types.ErrAppInvalidAddress,
		},
	}

	// Initialize the App Account by sending it some funds from the validator account that is part of genesis
	network.InitAccount(t, net, appAccount.Address)

	// Run the tests
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			// Wait for a new block to be committed
			require.NoError(t, net.WaitForNextBlock())

			// Prepare the arguments for the CLI command
			args := []string{
				fmt.Sprintf("--%s=%s", flags.FlagFrom, test.address),
			}
			args = append(args, commonArgs...)

			// Execute the command
			outUnstake, err := clitestutil.ExecTestCLICmd(ctx, application.CmdUnstakeApplication(), args)

			// Validate the error if one is expected
			if test.err != nil {
				stat, ok := status.FromError(test.err)
				require.True(t, ok)
				require.Contains(t, stat.Message(), test.err.Error())
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
