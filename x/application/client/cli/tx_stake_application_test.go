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
	"github.com/pokt-network/poktroll/testutil/yaml"
	"github.com/pokt-network/poktroll/x/application/client/cli"
	"github.com/pokt-network/poktroll/x/application/types"
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

	defaultConfig := `
		service_ids:
		  - svc1
		  - svc2
		  - svc3
		`

	tests := []struct {
		desc        string
		address     string
		stakeString string
		config      string
		expectedErr *sdkerrors.Error
	}{
		// Happy Paths
		{
			desc:        "valid",
			address:     appAccount.Address.String(),
			stakeString: "1000upokt",
			config:      defaultConfig,
			expectedErr: nil,
		},

		// Error Paths - Address Related
		{
			desc: "invalid: missing address",
			// address:     "explicitly missing",
			stakeString: "1000upokt",
			config:      defaultConfig,
			expectedErr: types.ErrAppInvalidAddress,
		},
		{
			desc:        "invalid: invalid address",
			address:     "invalid",
			stakeString: "1000upokt",
			config:      defaultConfig,
			expectedErr: types.ErrAppInvalidAddress,
		},

		// Error Paths - Stake Related
		{
			desc:    "invalid: missing stake",
			address: appAccount.Address.String(),
			// stakeString: "explicitly missing",
			config:      defaultConfig,
			expectedErr: types.ErrAppInvalidStake,
		},
		{
			desc:        "invalid: invalid stake denom",
			address:     appAccount.Address.String(),
			stakeString: "1000invalid",
			config:      defaultConfig,
			expectedErr: types.ErrAppInvalidStake,
		},
		{
			desc:        "invalid: stake amount (zero)",
			address:     appAccount.Address.String(),
			stakeString: "0upokt",
			config:      defaultConfig,
			expectedErr: types.ErrAppInvalidStake,
		},
		{
			desc:        "invalid: stake amount (negative)",
			address:     appAccount.Address.String(),
			stakeString: "-1000upokt",
			config:      defaultConfig,
			expectedErr: types.ErrAppInvalidStake,
		},

		// Error Paths - Service Related
		{
			desc:        "invalid: services (empty string)",
			address:     appAccount.Address.String(),
			stakeString: "1000upokt",
			config:      "",
			expectedErr: types.ErrAppInvalidServiceConfigs,
		},
		{
			desc:        "invalid: single invalid service contains spaces",
			address:     appAccount.Address.String(),
			stakeString: "1000upokt",
			config: `
				service_ids:
				  - svc1 svc1_part2 svc1_part3
				`,
			expectedErr: types.ErrAppInvalidServiceConfigs,
		},
		{
			desc:        "invalid: one of two services is invalid because it contains spaces",
			address:     appAccount.Address.String(),
			stakeString: "1000upokt",
			config: `
				service_ids:
				  - svc1 svc1_part2
				  - svc2
				`,
			expectedErr: types.ErrAppInvalidServiceConfigs,
		},
		{
			desc:        "invalid: service ID is too long (8 chars is the max)",
			address:     appAccount.Address.String(),
			stakeString: "1000upokt",
			config: `
				service_ids:
				  - svc1,
				  - abcdefghi
				`,
			expectedErr: types.ErrAppInvalidServiceConfigs,
		},
	}

	// Initialize the App Account by sending it some funds from the validator account that is part of genesis
	network.InitAccount(t, net, appAccount.Address)

	// Run the tests
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			// Wait for a new block to be committed
			require.NoError(t, net.WaitForNextBlock())

			// write the stake config to a file
			configPath := testutil.WriteToNewTempFile(t, yaml.NormalizeYAMLIndentation(tt.config)).Name()

			// Prepare the arguments for the CLI command
			args := []string{
				tt.stakeString,
				fmt.Sprintf("--config=%s", configPath),
				fmt.Sprintf("--%s=%s", flags.FlagFrom, tt.address),
			}
			args = append(args, commonArgs...)

			// Execute the command
			outStake, err := clitestutil.ExecTestCLICmd(ctx, cli.CmdStakeApplication(), args)

			// Validate the error if one is expected
			if tt.expectedErr != nil {
				stat, ok := status.FromError(tt.expectedErr)
				require.True(t, ok)
				require.Contains(t, stat.Message(), tt.expectedErr.Error())
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
