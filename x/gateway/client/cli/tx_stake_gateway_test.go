package cli_test

import (
	"fmt"
	"os"
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
	"github.com/pokt-network/poktroll/testutil/yaml"
	"github.com/pokt-network/poktroll/x/gateway/client/cli"
	"github.com/pokt-network/poktroll/x/gateway/types"
)

func TestCLI_StakeGateway(t *testing.T) {
	net, _ := networkWithGatewayObjects(t, 2)
	val := net.Validators[0]
	ctx := val.ClientCtx

	// Create a keyring and add an account for the gateway to be staked
	kr := ctx.Keyring
	accounts := testutil.CreateKeyringAccounts(t, kr, 1)
	gatewayAccount := accounts[0]

	// Update the context with the new keyring
	ctx = ctx.WithKeyring(kr)

	// Common args used for all requests
	commonArgs := []string{
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(net.Config.BondDenom, sdkmath.NewInt(10))).String()),
	}

	tests := []struct {
		desc          string
		address       string
		inputConfig   string
		expectedError *sdkerrors.Error
	}{
		{
			desc:    "stake gateway: invalid address",
			address: "invalid",
			inputConfig: `
			  stake_amount: 1000upokt,
				`,
			expectedError: types.ErrGatewayInvalidAddress,
		},
		{
			desc: "stake gateway: missing address",
			// address:     gatewayAccount.Address.String(),
			inputConfig: `
			  stake_amount: 1000upokt,
				`,
			expectedError: types.ErrGatewayInvalidAddress,
		},
		{
			desc:    "stake gateway: invalid stake amount (zero)",
			address: gatewayAccount.Address.String(),
			inputConfig: `
			  stake_amount: 0upokt,
				`,
			expectedError: types.ErrGatewayInvalidStake,
		},
		{
			desc:    "stake gateway: invalid stake amount (negative)",
			address: gatewayAccount.Address.String(),
			inputConfig: `
			  stake_amount: -1000upokt,
				`,
			expectedError: types.ErrGatewayInvalidStake,
		},
		{
			desc:    "stake gateway: invalid stake denom",
			address: gatewayAccount.Address.String(),
			inputConfig: `
			  stake_amount: 1000invalid,
				`,
			expectedError: types.ErrGatewayInvalidStake,
		},
		{
			desc:    "stake gateway: invalid stake missing denom",
			address: gatewayAccount.Address.String(),
			inputConfig: `
			  stake_amount: 1000,
				`,
			expectedError: types.ErrGatewayInvalidStake,
		},
		{
			desc:          "stake gateway: invalid stake missing stake",
			address:       gatewayAccount.Address.String(),
			inputConfig:   ``,
			expectedError: types.ErrGatewayInvalidStake,
		},
		{
			desc:    "stake gateway: valid",
			address: gatewayAccount.Address.String(),
			inputConfig: `
			  stake_amount: 1000upokt,
				`,
		},
	}

	// Initialize the Gateway Account by sending it some funds from the validator account that is part of genesis
	network.InitAccount(t, net, gatewayAccount.Address)

	// Run the tests
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			// Wait for a new block to be committed
			require.NoError(t, net.WaitForNextBlock())

			// write the stake config to a file
			configPath := testutil.WriteToNewTempFile(t, yaml.NormalizeYAMLIndentation(tt.inputConfig)).Name()
			t.Cleanup(func() { os.Remove(configPath) })

			// Prepare the arguments for the CLI command
			args := []string{
				fmt.Sprintf("--config=%s", configPath),
				fmt.Sprintf("--%s=%s", flags.FlagFrom, tt.address),
			}
			args = append(args, commonArgs...)

			// Execute the command
			outStake, err := clitestutil.ExecTestCLICmd(ctx, cli.CmdStakeGateway(), args)
			if tt.expectedError != nil {
				stat, ok := status.FromError(tt.expectedError)
				require.True(t, ok)
				require.Contains(t, stat.Message(), tt.expectedError.Error())
				return
			}
			require.NoError(t, err)

			require.NoError(t, err)
			var resp sdk.TxResponse
			require.NoError(t, net.Config.Codec.UnmarshalJSON(outStake.Bytes(), &resp))
			require.NotNil(t, resp)
			require.NotNil(t, resp.TxHash)
			require.Equal(t, uint32(0), resp.Code)
		})
	}
}
