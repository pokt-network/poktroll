package cli_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	sdkerrors "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/testutil"
	testcli "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/testutil/network"
	"github.com/pokt-network/poktroll/testutil/network/sessionnet"
	"github.com/pokt-network/poktroll/testutil/yaml"
	"github.com/pokt-network/poktroll/x/application/client/cli"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
)

func TestCLI_StakeApplication(t *testing.T) {
	ctx := context.Background()
	memnet := sessionnet.NewInMemoryNetworkWithSessions(
		t, &network.InMemoryNetworkConfig{
			NumSuppliers:            2,
			AppSupplierPairingRatio: 1,
		},
	)
	memnet.Start(ctx, t)

	appGenesisState := network.GetGenesisState[*apptypes.GenesisState](t, apptypes.ModuleName, memnet)
	applications := appGenesisState.ApplicationList
	appAddr := applications[0].GetAddress()

	net := memnet.GetNetwork(t)

	// Common args used for all requests
	commonArgs := []string{
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(net.Config.BondDenom, sdkmath.NewInt(10))).String()),
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

		inputConfig  string
		inputAddress string

		expectedError *sdkerrors.Error
	}{
		// Happy Paths
		{
			desc: "valid",

			inputAddress: appAddr,
			inputConfig:  defaultConfig,

			expectedError: nil,
		},

		// Error Paths - Address Related
		{
			desc: "invalid: missing address",
			// inputAddress:     "explicitly missing",
			inputConfig: defaultConfig,

			expectedError: apptypes.ErrAppInvalidAddress,
		},
		{
			desc: "invalid: invalid address",

			inputAddress: "invalid",
			inputConfig:  defaultConfig,

			expectedError: apptypes.ErrAppInvalidAddress,
		},

		// Error Paths - Stake Related
		{
			desc: "invalid: missing stake",

			inputAddress: appAddr,
			inputConfig: `
				stake_amount: # explicitly missing
				service_ids:
				  - svc1
				  - svc2
				  - svc3
				`,

			expectedError: apptypes.ErrAppInvalidStake,
		},
		{
			desc: "invalid: invalid stake denom",

			inputAddress: appAddr,
			inputConfig: `
				stake_amount: 1000invalid
				service_ids:
				  - svc1
				  - svc2
				  - svc3
				`,

			expectedError: apptypes.ErrAppInvalidStake,
		},
		{
			desc: "invalid: stake amount (zero)",

			inputAddress: appAddr,
			inputConfig: `
				stake_amount: 0upokt
				service_ids:
				  - svc1
				  - svc2
				  - svc3
				`,

			expectedError: apptypes.ErrAppInvalidStake,
		},
		{
			desc: "invalid: stake amount (negative)",

			inputAddress: appAddr,
			inputConfig: `
				stake_amount: -1000upokt
				service_ids:
				  - svc1
				  - svc2
				  - svc3
				`,

			expectedError: apptypes.ErrAppInvalidStake,
		},

		// Error Paths - Service Related
		{
			desc: "invalid: services (empty string)",

			inputAddress: appAddr,
			inputConfig: `
				stake_amount: 1000upokt
				`,

			expectedError: apptypes.ErrAppInvalidServiceConfigs,
		},
		{
			desc: "invalid: single invalid service contains spaces",

			inputAddress: appAddr,
			inputConfig: `
				stake_amount: 1000upokt
				service_ids:
				  - svc1 svc1_part2 svc1_part3
				`,

			expectedError: apptypes.ErrAppInvalidServiceConfigs,
		},
		{
			desc: "invalid: one of two services is invalid because it contains spaces",

			inputAddress: appAddr,
			inputConfig: `
				stake_amount: 1000upokt
				service_ids:
				  - svc1 svc1_part2
				  - svc2
				`,

			expectedError: apptypes.ErrAppInvalidServiceConfigs,
		},
		{
			desc: "invalid: service ID is too long (8 chars is the max)",

			inputAddress: appAddr,
			inputConfig: `
				stake_amount: 1000upokt
				service_ids:
				  - svc1,
				  - abcdefghi
				`,

			expectedError: apptypes.ErrAppInvalidServiceConfigs,
		},
	}

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
				fmt.Sprintf("--%s=%s", flags.FlagFrom, tt.inputAddress),
			}
			args = append(args, commonArgs...)

			// Execute the command
			outStake, err := testcli.ExecTestCLICmd(memnet.GetClientCtx(t), cli.CmdStakeApplication(), args)

			// Validate the error if one is expected
			if tt.expectedError != nil {
				stat, ok := status.FromError(tt.expectedError)
				require.True(t, ok)
				require.Contains(t, stat.Message(), tt.expectedError.Error())
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
