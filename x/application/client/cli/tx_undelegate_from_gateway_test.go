package cli_test

import (
	"fmt"
	"testing"

	sdkerrors "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/testutil"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/testutil/network"
	"github.com/pokt-network/poktroll/x/application/client/cli"
	"github.com/pokt-network/poktroll/x/application/types"
)

func TestCLI_UndelegateFromGateway(t *testing.T) {
	net, _ := networkWithApplicationObjects(t, 2)
	val := net.Validators[0]
	ctx := val.ClientCtx

	// Create a keyring and add an account for the application to be delegated
	// and the gateway to be delegated to
	kr := ctx.Keyring
	accounts := testutil.CreateKeyringAccounts(t, kr, 2)
	appAccount := accounts[0]
	gatewayAccount := accounts[1]

	// Update the context with the new keyring
	ctx = ctx.WithKeyring(kr)

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
			appAddress:     appAccount.Address.String(),
			gatewayAddress: gatewayAccount.Address.String(),
		},
		{
			desc: "invalid - missing app address",
			// appAddress:     appAccount.Address.String(),
			gatewayAddress: gatewayAccount.Address.String(),
			err:            types.ErrAppInvalidAddress,
		},
		{
			desc:           "invalid - invalid app address",
			appAddress:     "invalid address",
			gatewayAddress: gatewayAccount.Address.String(),
			err:            types.ErrAppInvalidAddress,
		},
		{
			desc:       "invalid - missing gateway address",
			appAddress: appAccount.Address.String(),
			// gatewayAddress: gatewayAccount.Address.String(),
			err: types.ErrAppInvalidGatewayAddress,
		},
		{
			desc:           "invalid - invalid gateway address",
			appAddress:     appAccount.Address.String(),
			gatewayAddress: "invalid address",
			err:            types.ErrAppInvalidGatewayAddress,
		},
	}

	// Initialize the App and Gateway Accounts by sending it some funds from the validator account that is part of genesis
	network.InitAccount(t, net, appAccount.Address)
	network.InitAccount(t, net, gatewayAccount.Address)

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
