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
	"github.com/pokt-network/poktroll/x/service/client/cli"
	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestCLI_AddService(t *testing.T) {
	net, _ := networkWithSupplierObjects(t, 1)
	val := net.Validators[0]
	ctx := val.ClientCtx

	// Create a keyring and add an account for the supplier adding the service
	kr := ctx.Keyring
	accounts := testutil.CreateKeyringAccounts(t, kr, 1)
	supplierAccount := accounts[0]

	// Update the context with the new keyring
	ctx = ctx.WithKeyring(kr)

	// Common args used for all requests
	commonArgs := []string{
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(net.Config.BondDenom, sdkmath.NewInt(10))).String()),
	}

	// Initialize the Supplier account by sending it some funds from the
	// validator account that is part of genesis
	network.InitAccountWithSequence(t, net, supplierAccount.Address, 1)

	// Wait for a new block to be committed
	require.NoError(t, net.WaitForNextBlock())

	// Prepare two valid services
	srv1 := sharedtypes.Service{
		Id:   "srv1",
		Name: "service name",
	}
	srv2 := sharedtypes.Service{
		Id:   "srv2",
		Name: "service name 2",
	}
	// Add srv2 to the network
	args := []string{
		srv2.Id,
		srv2.Name,
		fmt.Sprintf("--%s=%s", flags.FlagFrom, supplierAccount.Address.String()),
	}
	args = append(args, commonArgs...)
	_, err := clitestutil.ExecTestCLICmd(ctx, cli.CmdAddService(), args)
	require.NoError(t, err)

	tests := []struct {
		desc            string
		supplierAddress string
		service         sharedtypes.Service
		err             *sdkerrors.Error
	}{
		{
			desc:            "valid - add new service",
			supplierAddress: supplierAccount.Address.String(),
			service:         srv1,
		},
		{
			desc:            "invalid - missing service id",
			supplierAddress: supplierAccount.Address.String(),
			service:         sharedtypes.Service{Name: "service name"}, // ID intentionally omitted
			err:             types.ErrServiceMissingID,
		},
		{
			desc:            "invalid - missing service name",
			supplierAddress: supplierAccount.Address.String(),
			service:         sharedtypes.Service{Id: "srv1"}, // Name intentionally omitted
			err:             types.ErrServiceMissingName,
		},
		{
			desc:            "invalid - invalid supplier address",
			supplierAddress: "invalid address",
			service:         srv1,
			err:             types.ErrServiceInvalidAddress,
		},
		{
			desc:            "invalid - service already staked",
			supplierAddress: supplierAccount.Address.String(),
			service:         srv2,
			err:             types.ErrServiceAlreadyExists,
		},
	}

	// Run the tests
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			// Wait for a new block to be committed
			require.NoError(t, net.WaitForNextBlock())

			// Prepare the arguments for the CLI command
			args := []string{
				tt.service.Id,
				tt.service.Name,
				fmt.Sprintf("--%s=%s", flags.FlagFrom, tt.supplierAddress),
			}
			args = append(args, commonArgs...)

			// Execute the command
			addServiceOutput, err := clitestutil.ExecTestCLICmd(ctx, cli.CmdAddService(), args)

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
			require.NoError(t, net.Config.Codec.UnmarshalJSON(addServiceOutput.Bytes(), &resp))
			require.NotNil(t, resp)
			require.NotNil(t, resp.TxHash)
			require.Equal(t, uint32(0), resp.Code)
		})
	}
}
