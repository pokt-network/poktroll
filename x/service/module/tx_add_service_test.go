package service_test

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

	"github.com/pokt-network/poktroll/proto/types/service"
	"github.com/pokt-network/poktroll/proto/types/shared"
	"github.com/pokt-network/poktroll/testutil/network"
	servicemodule "github.com/pokt-network/poktroll/x/service/module"
)

func TestCLI_AddService(t *testing.T) {
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
	svc1 := shared.Service{
		Id:   "svc1",
		Name: "service name",
	}
	svc2 := shared.Service{
		Id:   "svc2",
		Name: "service name 2",
	}
	// Add svc2 to the network
	args := []string{
		svc2.Id,
		svc2.Name,
		fmt.Sprintf("--%s=%s", flags.FlagFrom, account.Address.String()),
	}
	args = append(args, commonArgs...)

	_, err := clitestutil.ExecTestCLICmd(ctx, servicemodule.CmdAddService(), args)
	require.NoError(t, err)

	tests := []struct {
		desc            string
		supplierAddress string
		service         shared.Service
		expectedErr     *sdkerrors.Error
	}{
		{
			desc:            "valid - add new service",
			supplierAddress: account.Address.String(),
			service:         svc1,
		},
		{
			desc:            "invalid - missing service id",
			supplierAddress: account.Address.String(),
			service:         shared.Service{Name: "service name"}, // ID intentionally omitted
			expectedErr:     service.ErrServiceMissingID,
		},
		{
			desc:            "invalid - missing service name",
			supplierAddress: account.Address.String(),
			service:         shared.Service{Id: "svc1"}, // Name intentionally omitted
			expectedErr:     service.ErrServiceMissingName,
		},
		{
			desc:            "invalid - invalid supplier address",
			supplierAddress: "invalid address",
			service:         svc1,
			expectedErr:     service.ErrServiceInvalidAddress,
		},
		{
			desc:            "invalid - service already staked",
			supplierAddress: account.Address.String(),
			service:         svc2,
			expectedErr:     service.ErrServiceAlreadyExists,
		},
	}

	// Run the tests
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			// Wait for a new block to be committed
			require.NoError(t, net.WaitForNextBlock())

			// Prepare the arguments for the CLI command
			args := []string{
				test.service.Id,
				test.service.Name,
				fmt.Sprintf("--%s=%s", flags.FlagFrom, test.supplierAddress),
			}
			args = append(args, commonArgs...)

			// Execute the command
			addServiceOutput, err := clitestutil.ExecTestCLICmd(ctx, servicemodule.CmdAddService(), args)

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
			require.Equal(t, uint32(0), resp.Code)
		})
	}
}
