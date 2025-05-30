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
	service "github.com/pokt-network/poktroll/x/service/module"
	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestCLI_SetupService(t *testing.T) {
	net := network.New(t, network.DefaultConfig())
	val := net.Validators[0]
	ctx := val.ClientCtx

	// Create a keyring and add an account for the address adding the service
	kr := ctx.Keyring
	accounts := testutil.CreateKeyringAccounts(t, kr, 2)
	account := accounts[0]
	distinctServiceOwner := accounts[1]

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

	// Initialize the Service owner and signer accounts by sending it some funds
	// from the validator account that is part of genesis
	network.InitAccountWithSequence(t, net, account.Address, 1)
	network.InitAccountWithSequence(t, net, distinctServiceOwner.Address, 2)

	// Wait for a new block to be committed
	require.NoError(t, net.WaitForNextBlock())

	existingService := prepareService("svc0", "service 0", 1, account.Address.String())
	// Add svc0 to the network
	args := []string{
		existingService.Id,
		existingService.Name,
		fmt.Sprintf("--%s=%s", flags.FlagFrom, account.Address.String()),
	}
	args = append(args, commonArgs...)

	_, err := clitestutil.ExecTestCLICmd(ctx, service.CmdSetupService(), args)
	require.NoError(t, err)

	// Each service creation test uses a different service ID, so we don't fall
	// into the service update logic and can test the creation logic properly.
	tests := []struct {
		desc        string
		service     sharedtypes.Service
		expectedErr *sdkerrors.Error
	}{
		{
			desc:    "valid - add new service without specifying compute units per relay so that it uses the default",
			service: prepareService("svc1", "service 1", 0, account.Address.String()), // 0 means default value
		},
		{
			desc:        "invalid - missing service id",
			service:     prepareService("", "service 2", 1, account.Address.String()), // Id intentionally omitted
			expectedErr: types.ErrServiceMissingID,
		},
		{
			desc:        "invalid - invalid owner address",
			service:     prepareService("svc3", "service 3", 1, "invalid address"),
			expectedErr: types.ErrServiceInvalidAddress,
		},
		{
			desc:    "valid - missing service name",
			service: prepareService("svc4", "", 1, account.Address.String()), // Name intentionally omitted
		},
		{
			desc:    "valid - add new service",
			service: prepareService("svc5", "service 4", 1, account.Address.String()),
		},
		{
			desc:    "valid - stake for a different owner",
			service: prepareService("svc6", "service 5", 1, distinctServiceOwner.Address.String()),
		},
		{
			desc:    "valid - update an already existing service",
			service: prepareService(existingService.Id, "updated service 0", 20, account.Address.String()),
		},
		{
			desc: "valid - update the owner address of an already staked service",
			service: prepareService(
				existingService.Id,
				existingService.Name,
				existingService.ComputeUnitsPerRelay,
				distinctServiceOwner.Address.String(), // Old owner is `account.Address`, new owner is `distinctServiceOwner.Address`
			),
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
			if test.service.OwnerAddress != account.Address.String() {
				// If the owner address is different from the signing address, include it
				// as a non default owner address
				argsAndFlags = append(argsAndFlags, test.service.OwnerAddress)
			}
			argsAndFlags = append(argsAndFlags, fmt.Sprintf("--%s=%s", flags.FlagFrom, account.Address.String()))

			args := append(argsAndFlags, commonArgs...)

			// Execute the command
			setupServiceOutput, err := clitestutil.ExecTestCLICmd(ctx, service.CmdSetupService(), args)

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
			require.NoError(t, net.Config.Codec.UnmarshalJSON(setupServiceOutput.Bytes(), &resp))
			require.NotNil(t, resp)
			require.NotNil(t, resp.TxHash)
			// You can reference Cosmos SDK error codes here: https://github.com/cosmos/cosmos-sdk/blob/main/types/errors/errors.go
			require.Equal(t, uint32(0), resp.Code, "tx response failed: %v", resp)
		})
	}
}

// prepareService is a helper function to create a service instance for testing.
// It initializes a service with the given parameters and returns it.
func prepareService(id, name string, computeUnitsPerRelay uint64, ownerAddress string) sharedtypes.Service {
	return sharedtypes.Service{
		Id:                   id,
		Name:                 name,
		ComputeUnitsPerRelay: computeUnitsPerRelay,
		OwnerAddress:         ownerAddress,
	}
}
