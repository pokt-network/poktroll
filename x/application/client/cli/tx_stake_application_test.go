package cli_test

import (
	"fmt"
	"pocket/x/application/client/cli"
	"pocket/x/application/types"
	"testing"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/testutil"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/status"
)

func TestCLI_StakeApplication(t *testing.T) {
	net, _ := networkWithApplicationObjects(t, 2)
	val := net.Validators[0]
	ctx := val.ClientCtx

	// Create a keyring and add an account for the application to be staked
	kr := ctx.Keyring
	accounts := testutil.CreateKeyringAccounts(t, kr, 1)
	appAccount := accounts[0]

	// Update the context with the new keyring
	ctx = ctx.WithKeyring(kr)

	// Common args used for all requests
	commonArgs := []string{
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(net.Config.BondDenom, sdkmath.NewInt(10))).String()),
	}

	tests := []struct {
		desc        string
		address     string
		stakeAmount string
		err         *errorsmod.Error
	}{
		{
			desc:        "stake application: invalid address",
			address:     "invalid",
			stakeAmount: "1000upokt",
			err:         types.ErrAppInvalidAddress,
		},
		{
			desc:        "stake application: invalid stake amount",
			address:     appAccount.Address.String(),
			stakeAmount: "1000invalid",
			err:         types.ErrAppInvalidStake,
		},
		{
			desc:        "stake application: valid",
			address:     appAccount.Address.String(),
			stakeAmount: "1000upokt",
		},
	}

	// Initialize the App Account by sending it some funds from the validator account that is part of genesis
	sendArgs := []string{
		fmt.Sprintf("--%s=%s", flags.FlagFrom, val.Address.String()),
	}
	sendArgs = append(sendArgs, commonArgs...)
	amount := sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(200)))
	_, err := clitestutil.MsgSendExec(ctx, net.Validators[0].Address, appAccount.Address, amount, sendArgs...)
	require.NoError(t, err)

	// Stake the tests
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			// Wait for a new block to be committed
			require.NoError(t, net.WaitForNextBlock())

			// Prepare the arguments for the CLI command
			args := []string{
				tt.stakeAmount,
				fmt.Sprintf("--%s=%s", flags.FlagFrom, tt.address),
			}
			args = append(args, commonArgs...)

			// Execute the command
			outStake, err := clitestutil.ExecTestCLICmd(ctx, cli.CmdStakeApplication(), args)
			if tt.err != nil {
				stat, ok := status.FromError(tt.err)
				require.True(t, ok)
				require.Contains(t, stat.Message(), tt.err.Error())
				return
			}

			require.NoError(t, err)
			var resp sdk.TxResponse
			require.NoError(t, net.Config.Codec.UnmarshalJSON(outStake.Bytes(), &resp))
			require.NotNil(t, resp)
			fmt.Println(resp)
		})
	}
}
