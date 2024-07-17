package application_test

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

	"github.com/pokt-network/poktroll/proto/types/application"
	"github.com/pokt-network/poktroll/testutil/network"
	"github.com/pokt-network/poktroll/testutil/yaml"
	appmodule "github.com/pokt-network/poktroll/x/application/module"
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
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(net.Config.BondDenom, math.NewInt(10))).String()),
	}

	defaultConfig := `
		stake_amount: 1000upokt
		service_ids:
		  - svc1
		  - svc2
		  - svc3
		`

	tests := []struct {
		desc string

		appConfig string
		appAddr   string

		expectedErr *sdkerrors.Error
	}{
		// Happy Paths
		{
			desc: "valid",

			appAddr:   appAccount.Address.String(),
			appConfig: defaultConfig,

			expectedErr: nil,
		},

		// Error Paths - Address Related
		{
			desc: "invalid: missing address",
			// inputAddress explicitly omitted
			appConfig: defaultConfig,

			expectedErr: application.ErrAppInvalidAddress,
		},
		{
			desc: "invalid: invalid address",

			appAddr:   "invalid",
			appConfig: defaultConfig,

			expectedErr: application.ErrAppInvalidAddress,
		},

		// Error Paths - Stake Related
		{
			desc: "invalid: missing stake",

			appAddr: appAccount.Address.String(),
			appConfig: `
				stake_amount: # explicitly missing
				service_ids:
				  - svc1
				  - svc2
				  - svc3
				`,

			expectedErr: application.ErrAppInvalidStake,
		},
		{
			desc: "invalid: invalid stake denom",

			appAddr: appAccount.Address.String(),
			appConfig: `
				stake_amount: 1000invalid
				service_ids:
				  - svc1
				  - svc2
				  - svc3
				`,

			expectedErr: application.ErrAppInvalidStake,
		},
		{
			desc: "invalid: stake amount (zero)",

			appAddr: appAccount.Address.String(),
			appConfig: `
				stake_amount: 0upokt
				service_ids:
				  - svc1
				  - svc2
				  - svc3
				`,

			expectedErr: application.ErrAppInvalidStake,
		},
		{
			desc: "invalid: stake amount (negative)",

			appAddr: appAccount.Address.String(),
			appConfig: `
				stake_amount: -1000upokt
				service_ids:
				  - svc1
				  - svc2
				  - svc3
				`,

			expectedErr: application.ErrAppInvalidStake,
		},

		// Error Paths - Service Related
		{
			desc: "invalid: services (empty string)",

			appAddr: appAccount.Address.String(),
			appConfig: `
				stake_amount: 1000upokt
				`,

			expectedErr: application.ErrAppInvalidServiceConfigs,
		},
		{
			desc: "invalid: single invalid service contains spaces",

			appAddr: appAccount.Address.String(),
			appConfig: `
				stake_amount: 1000upokt
				service_ids:
				  - svc1 svc1_part2 svc1_part3
				`,

			expectedErr: application.ErrAppInvalidServiceConfigs,
		},
		{
			desc: "invalid: one of two services is invalid because it contains spaces",

			appAddr: appAccount.Address.String(),
			appConfig: `
				stake_amount: 1000upokt
				service_ids:
				  - svc1 svc1_part2
				  - svc2
				`,

			expectedErr: application.ErrAppInvalidServiceConfigs,
		},
		{
			desc: "invalid: service ID is too long (8 chars is the max)",

			appAddr: appAccount.Address.String(),
			appConfig: `
				stake_amount: 1000upokt
				service_ids:
				  - svc1,
				  - abcdefghi
				`,

			expectedErr: application.ErrAppInvalidServiceConfigs,
		},
	}

	// Initialize the App Account by sending it some funds from the validator account that is part of genesis
	network.InitAccount(t, net, appAccount.Address)

	// Run the tests
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			// Wait for a new block to be committed
			require.NoError(t, net.WaitForNextBlock())

			// write the stake config to a file
			configPath := testutil.WriteToNewTempFile(t, yaml.NormalizeYAMLIndentation(test.appConfig)).Name()
			t.Cleanup(func() { os.Remove(configPath) })

			// Prepare the arguments for the CLI command
			args := []string{
				fmt.Sprintf("--config=%s", configPath),
				fmt.Sprintf("--%s=%s", flags.FlagFrom, test.appAddr),
			}
			args = append(args, commonArgs...)

			// Execute the command
			outStake, err := clitestutil.ExecTestCLICmd(ctx, appmodule.CmdStakeApplication(), args)

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
			require.NoError(t, net.Config.Codec.UnmarshalJSON(outStake.Bytes(), &resp))
			require.NotNil(t, resp)
			require.NotNil(t, resp.TxHash)
			require.Equal(t, uint32(0), resp.Code)
		})
	}
}
