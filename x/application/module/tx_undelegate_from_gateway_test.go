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

	"github.com/pokt-network/pocket/testutil/network"
	application "github.com/pokt-network/pocket/x/application/module"
	"github.com/pokt-network/pocket/x/application/types"
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
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(net.Config.BondDenom, math.NewInt(10))).String()),
	}

	tests := []struct {
		desc           string
		appAddress     string
		gatewayAddress string
		expectedErr    *sdkerrors.Error
	}{
		{
			desc:           "undelegate from gateway: valid",
			appAddress:     appAccount.Address.String(),
			gatewayAddress: gatewayAccount.Address.String(),
		},
		{
			desc: "invalid - missing app address",
			// appAddress explicitly omitted
			gatewayAddress: gatewayAccount.Address.String(),
			expectedErr:    types.ErrAppInvalidAddress,
		},
		{
			desc:           "invalid - invalid app address",
			appAddress:     "invalid address",
			gatewayAddress: gatewayAccount.Address.String(),
			expectedErr:    types.ErrAppInvalidAddress,
		},
		{
			desc:       "invalid - missing gateway address",
			appAddress: appAccount.Address.String(),
			// gatewayAddress explicitly omitted
			expectedErr: types.ErrAppInvalidGatewayAddress,
		},
		{
			desc:           "invalid - invalid gateway address",
			appAddress:     appAccount.Address.String(),
			gatewayAddress: "invalid address",
			expectedErr:    types.ErrAppInvalidGatewayAddress,
		},
	}

	// Initialize the App and Gateway Accounts by sending it some funds from the validator account that is part of genesis
	network.InitAccountWithSequence(t, net, appAccount.Address, 1)
	network.InitAccountWithSequence(t, net, gatewayAccount.Address, 2)

	// Run the tests
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			// Wait for a new block to be committed
			require.NoError(t, net.WaitForNextBlock())

			// Prepare the arguments for the CLI command
			args := []string{
				test.gatewayAddress,
				fmt.Sprintf("--%s=%s", flags.FlagFrom, test.appAddress),
			}
			args = append(args, commonArgs...)

			// Execute the command
			undelegateOutput, err := clitestutil.ExecTestCLICmd(ctx, application.CmdUndelegateFromGateway(), args)

			// Validate the error if one is expected
			if test.expectedErr != nil {
				stat, ok := status.FromError(test.expectedErr)
				require.True(t, ok)
				require.Contains(t, stat.Message(), test.expectedErr.Error())
				return
			}
			require.NoError(t, err)

			// Check the response
			var resp sdk.TxResponse
			require.NoError(t, net.Config.Codec.UnmarshalJSON(undelegateOutput.Bytes(), &resp))
			require.NotNil(t, resp)
			require.NotNil(t, resp.TxHash)
			// You can reference Cosmos SDK error codes here: https://github.com/cosmos/cosmos-sdk/blob/main/types/errors/errors.go
			require.Equal(t, uint32(0), resp.Code, "tx response failed: %v", resp)
		})
	}
}
