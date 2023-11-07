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
		desc             string
		address          string
		stakeString      string
		serviceIdsString string
		err              *sdkerrors.Error
	}{
		// Happy Paths
		{
			desc:             "valid",
			address:          appAccount.Address.String(),
			stakeString:      "1000upokt",
			serviceIdsString: "svc1,svc2,svc3",
			err:              nil,
		},

		// Error Paths - Address Related
		{
			desc: "address_test: missing address",
			// address:     "explicitly missing",
			stakeString:      "1000upokt",
			serviceIdsString: "svc1,svc2,svc3",
			err:              types.ErrAppInvalidAddress,
		},
		{
			desc:             "stake application: invalid address",
			address:          "invalid",
			stakeString:      "1000upokt",
			serviceIdsString: "svc1,svc2,svc3",
			err:              types.ErrAppInvalidAddress,
		},

		// Error Paths - Stake Related
		{
			desc:    "address_test: missing stake",
			address: appAccount.Address.String(),
			// stakeString: "explicitly missing",
			serviceIdsString: "svc1,svc2,svc3",
			err:              types.ErrAppInvalidStake,
		},
		{
			desc:             "address_test: invalid stake denom",
			address:          appAccount.Address.String(),
			stakeString:      "1000invalid",
			serviceIdsString: "svc1,svc2,svc3",
			err:              types.ErrAppInvalidStake,
		},
		{
			desc:             "address_test: invalid stake amount (zero)",
			address:          appAccount.Address.String(),
			stakeString:      "0upokt",
			serviceIdsString: "svc1,svc2,svc3",
			err:              types.ErrAppInvalidStake,
		},
		{
			desc:             "address_test: invalid stake amount (negative)",
			address:          appAccount.Address.String(),
			stakeString:      "-1000upokt",
			serviceIdsString: "svc1,svc2,svc3",
			err:              types.ErrAppInvalidStake,
		},

		// Error Paths - Service Related
		{
			desc:             "services_test: invalid services (empty string)",
			address:          appAccount.Address.String(),
			stakeString:      "1000upokt",
			serviceIdsString: "",
			err:              types.ErrAppInvalidServiceConfigs,
		},
		{
			desc:             "services_test: single invalid service contains spaces",
			address:          appAccount.Address.String(),
			stakeString:      "1000upokt",
			serviceIdsString: "svc1 svc1_part2 svc1_part3",
			err:              types.ErrAppInvalidServiceConfigs,
		},
		{
			desc:             "services_test: one of two services is invalid because it contains spaces",
			address:          appAccount.Address.String(),
			stakeString:      "1000upokt",
			serviceIdsString: "svc1 svc1_part2,svc2",
			err:              types.ErrAppInvalidServiceConfigs,
		},
		{
			desc:             "services_test: service ID is too long (8 chars is the max)",
			address:          appAccount.Address.String(),
			stakeString:      "1000upokt",
			serviceIdsString: "svc1,abcdefghi",
			err:              types.ErrAppInvalidServiceConfigs,
		},
	}

	// Initialize the App Account by sending it some funds from the validator account that is part of genesis
	network.InitAccount(t, net, appAccount.Address)

	// Run the tests
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			// Wait for a new block to be committed
			require.NoError(t, net.WaitForNextBlock())

			// Prepare the arguments for the CLI command
			args := []string{
				tt.stakeString,
				tt.serviceIdsString,
				fmt.Sprintf("--%s=%s", flags.FlagFrom, tt.address),
			}
			args = append(args, commonArgs...)

			// Execute the command
			outStake, err := clitestutil.ExecTestCLICmd(ctx, cli.CmdStakeApplication(), args)

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
			require.NoError(t, net.Config.Codec.UnmarshalJSON(outStake.Bytes(), &resp))
			require.NotNil(t, resp)
			require.NotNil(t, resp.TxHash)
			require.Equal(t, uint32(0), resp.Code)
		})
	}
}
