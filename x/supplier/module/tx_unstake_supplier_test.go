package supplier_test

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

	"github.com/pokt-network/poktroll/testutil/network"
	supplier "github.com/pokt-network/poktroll/x/supplier/module"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

func TestCLI_UnstakeSupplier(t *testing.T) {
	net, _ := networkWithSupplierObjects(t, 2)
	val := net.Validators[0]
	ctx := val.ClientCtx

	// Create a keyring and add an account for the supplier to be unstaked
	kr := ctx.Keyring
	accounts := testutil.CreateKeyringAccounts(t, kr, 1)
	supplierAccount := accounts[0]

	// Initialize the Supplier Account by sending it some funds from the validator account that is part of genesis
	network.InitAccount(t, net, supplierAccount.Address)

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
		expectedErr *sdkerrors.Error
	}{
		{
			desc:    "unstake supplier: valid",
			address: supplierAccount.Address.String(),
		},
		{
			desc: "unstake supplier: missing address",
			// address: supplierAccount.Address.String(),
			expectedErr: types.ErrSupplierInvalidAddress,
		},
		{
			desc:        "unstake supplier: invalid address",
			address:     "invalid",
			expectedErr: types.ErrSupplierInvalidAddress,
		},
	}

	// Run the tests
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			// Wait for a new block to be committed
			require.NoError(t, net.WaitForNextBlock())

			// Prepare the arguments for the CLI command
			args := []string{
				fmt.Sprintf("--%s=%s", flags.FlagFrom, test.address),
			}
			args = append(args, commonArgs...)

			// Execute the command
			outUnstake, err := clitestutil.ExecTestCLICmd(ctx, supplier.CmdUnstakeSupplier(), args)

			// Validate the error if one is expected
			if test.expectedErr != nil {
				stat, ok := status.FromError(test.expectedErr)
				require.True(t, ok)
				require.Contains(t, stat.Message(), test.expectedErr.Error())
				return
			}
			require.NoError(t, err)

			// Check the response, this test only asserts CLI command success and not
			// the actual supplier module state.
			var resp sdk.TxResponse
			require.NoError(t, net.Config.Codec.UnmarshalJSON(outUnstake.Bytes(), &resp))
			require.NotNil(t, resp)
			require.NotNil(t, resp.TxHash)
			// You can reference Cosmos SDK error codes here: https://github.com/cosmos/cosmos-sdk/blob/main/types/errors/errors.go
			require.Equal(t, uint32(0), resp.Code, "tx response failed: %v", resp)
		})
	}
}
