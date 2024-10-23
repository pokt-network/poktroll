package service_test

import (
	"fmt"
	"strconv"
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
	"github.com/pokt-network/poktroll/testutil/sample"
	service "github.com/pokt-network/poktroll/x/service/module"
	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

func TestCLI_AddService(t *testing.T) {
	// TODO_TECHDEBT: Remove once dao reward address is promoted to a tokenomics param.
	tokenomicstypes.DaoRewardAddress = sample.AccAddress()

	net := network.New(t, network.DefaultConfig())
	val := net.Validators[0]
	ctx := val.ClientCtx

	// Create a keyring and add an account for the address adding the service
	kr := ctx.Keyring
	accounts := testutil.CreateKeyringAccounts(t, kr, 1)
	account := accounts[0]

	// Update the context with the new keyring
	ctx = ctx.WithKeyring(kr)

	// Common args used for all requests
	commonArgs := []string{
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf(
			"--%s=%s",
			flags.FlagFees,
			sdk.NewCoins(sdk.NewCoin(net.Config.BondDenom, math.NewInt(10))).String(),
		),
	}

	// Initialize the Supplier account by sending it some funds from the
	// validator account that is part of genesis
	network.InitAccountWithSequence(t, net, account.Address, 1)

	// Wait for a new block to be committed
	require.NoError(t, net.WaitForNextBlock())

	// Prepare two valid services
	svc1 := sharedtypes.Service{
		Id:                   "svc1",
		Name:                 "service name",
		ComputeUnitsPerRelay: 1,
	}
	svc2 := sharedtypes.Service{
		Id:                   "svc2",
		Name:                 "service name 2",
		ComputeUnitsPerRelay: 1,
	}
	// Add svc2 to the network
	args := []string{
		svc2.Id,
		svc2.Name,
		fmt.Sprintf("--%s=%s", flags.FlagFrom, account.Address.String()),
	}
	args = append(args, commonArgs...)

	_, err := clitestutil.ExecTestCLICmd(ctx, service.CmdAddService(), args)
	require.NoError(t, err)

	tests := []struct {
		desc         string
		ownerAddress string
		service      sharedtypes.Service
		expectedErr  *sdkerrors.Error
	}{
		{
			desc:         "valid - add new service",
			ownerAddress: account.Address.String(),
			service:      svc1,
		},
		{
			desc:         "valid - add new service without specifying compute units per relay so that it uses the default",
			ownerAddress: account.Address.String(),
			service: sharedtypes.Service{
				Id:                   svc1.Id,
				Name:                 svc1.Name,
				ComputeUnitsPerRelay: 0, // this parameter is omitted when the test is run
			},
		},
		{
			desc:         "invalid - missing service id",
			ownerAddress: account.Address.String(),
			service:      sharedtypes.Service{Name: "service name"}, // ID intentionally omitted
			expectedErr:  types.ErrServiceMissingID,
		},
		{
			desc:         "invalid - missing service name",
			ownerAddress: account.Address.String(),
			service:      sharedtypes.Service{Id: "svc1"}, // Name intentionally omitted
			expectedErr:  types.ErrServiceMissingName,
		},
		{
			desc:         "invalid - invalid owner address",
			ownerAddress: "invalid address",
			service:      svc1,
			expectedErr:  types.ErrServiceInvalidAddress,
		},
		{
			desc:         "invalid - service already staked",
			ownerAddress: account.Address.String(),
			service:      svc2,
			expectedErr:  types.ErrServiceAlreadyExists,
		},
	}

	// Run the tests
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			// Wait for a new block to be committed
			require.NoError(t, net.WaitForNextBlock())

			// Prepare the arguments for the CLI command
			argsAndFlags := []string{
				test.service.Id,
				test.service.Name,
			}
			if test.service.ComputeUnitsPerRelay > 0 {
				// Only include compute units per relay argument if provided
				argsAndFlags = append(argsAndFlags, strconv.FormatUint(test.service.ComputeUnitsPerRelay, 10))
			}
			argsAndFlags = append(argsAndFlags, fmt.Sprintf("--%s=%s", flags.FlagFrom, test.ownerAddress))

			args := append(argsAndFlags, commonArgs...)

			// Execute the command
			addServiceOutput, err := clitestutil.ExecTestCLICmd(ctx, service.CmdAddService(), args)

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
			require.NoError(t, net.Config.Codec.UnmarshalJSON(addServiceOutput.Bytes(), &resp))
			require.NotNil(t, resp)
			require.NotNil(t, resp.TxHash)
			// You can reference Cosmos SDK error codes here: https://github.com/cosmos/cosmos-sdk/blob/main/types/errors/errors.go
			require.Equal(t, uint32(0), resp.Code, "tx response failed: %v", resp)
		})
	}
}
