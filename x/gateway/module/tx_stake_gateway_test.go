package gateway_test

import (
	"fmt"
	"os"
	"testing"

	sdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/testutil"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/proto/types/gateway"
	"github.com/pokt-network/poktroll/testutil/network"
	"github.com/pokt-network/poktroll/testutil/yaml"
	gatewaymodule "github.com/pokt-network/poktroll/x/gateway/module"
)

func TestCLI_StakeGateway(t *testing.T) {
	net, _ := networkWithGatewayObjects(t, 2)
	val := net.Validators[0]
	ctx := val.ClientCtx

	// Create a keyring and add an account for the gateway to be staked
	kr := ctx.Keyring
	accounts := testutil.CreateKeyringAccounts(t, kr, 1)
	gatewayAccount := accounts[0]

	// Initialize the Gateway Account by sending it some funds from the validator account that is part of genesis
	network.InitAccount(t, net, gatewayAccount.Address)

	// Update the context with the new keyring
	ctx = ctx.WithKeyring(kr)

	// Common args used for all requests
	commonArgs := []string{
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(net.Config.BondDenom, math.NewInt(10))).String()),
	}

	tests := []struct {
		desc        string
		address     string
		inputConfig string
		expectedErr *sdkerrors.Error
	}{
		{
			desc:    "stake gateway: invalid address",
			address: "invalid",
			inputConfig: `
			  stake_amount: 1000upokt
				`,
			expectedErr: gateway.ErrGatewayInvalidAddress,
		},
		{
			desc: "stake gateway: missing address",
			// address:     gatewayAccount.Address.String(),
			inputConfig: `
			  stake_amount: 1000upokt
				`,
			expectedErr: gateway.ErrGatewayInvalidAddress,
		},
		{
			desc:    "stake gateway: invalid stake amount (zero)",
			address: gatewayAccount.Address.String(),
			inputConfig: `
			  stake_amount: 0upokt
				`,
			expectedErr: gateway.ErrGatewayInvalidStake,
		},
		{
			desc:    "stake gateway: invalid stake amount (negative)",
			address: gatewayAccount.Address.String(),
			inputConfig: `
			  stake_amount: -1000upokt
				`,
			expectedErr: gateway.ErrGatewayInvalidStake,
		},
		{
			desc:    "stake gateway: invalid stake denom",
			address: gatewayAccount.Address.String(),
			inputConfig: `
			  stake_amount: 1000invalid
				`,
			expectedErr: gateway.ErrGatewayInvalidStake,
		},
		{
			desc:    "stake gateway: invalid stake missing denom",
			address: gatewayAccount.Address.String(),
			inputConfig: `
			  stake_amount: 1000
				`,
			expectedErr: gateway.ErrGatewayInvalidStake,
		},
		{
			desc:        "stake gateway: invalid stake missing stake",
			address:     gatewayAccount.Address.String(),
			inputConfig: ``,
			expectedErr: gateway.ErrGatewayInvalidStake,
		},
		{
			desc:    "stake gateway: valid",
			address: gatewayAccount.Address.String(),
			inputConfig: `
			  stake_amount: 1000upokt
				`,
		},
	}

	// Run the tests
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			// Wait for a new block to be committed
			require.NoError(t, net.WaitForNextBlock())

			// write the stake config to a file
			configPath := testutil.WriteToNewTempFile(t, yaml.NormalizeYAMLIndentation(test.inputConfig)).Name()
			t.Cleanup(func() { os.Remove(configPath) })

			// Prepare the arguments for the CLI command
			args := []string{
				fmt.Sprintf("--config=%s", configPath),
				fmt.Sprintf("--%s=%s", flags.FlagFrom, test.address),
			}
			args = append(args, commonArgs...)

			// Execute the command
			outStake, err := clitestutil.ExecTestCLICmd(ctx, gatewaymodule.CmdStakeGateway(), args)
			if test.expectedErr != nil {
				stat, ok := status.FromError(test.expectedErr)
				require.True(t, ok)
				require.Contains(t, stat.Message(), test.expectedErr.Error())
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
