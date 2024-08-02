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
	accounts := testutil.CreateKeyringAccounts(t, kr, 2)
	operatorAccount := accounts[0]
	ownerAccount := accounts[1]

	// Initialize the Supplier operator account by sending it some funds from the
	// validator account that is part of genesis
	network.InitAccount(t, net, ownerAccount.Address)

	// Update the context with the new keyring
	ctx = ctx.WithKeyring(kr)

	// Common args used for all requests
	commonArgs := []string{
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(net.Config.BondDenom, math.NewInt(10))).String()),
	}

	tests := []struct {
		desc            string
		operatorAddress string
		ownerAddress    string
		expectedErr     *sdkerrors.Error
	}{
		{
			desc:            "unstake supplier: valid",
			operatorAddress: operatorAccount.Address.String(),
			ownerAddress:    ownerAccount.Address.String(),
		},
		{
			desc: "unstake supplier: missing owner address",
			// ownerAddress: ownerAccount.Address.String(),
			operatorAddress: operatorAccount.Address.String(),
			expectedErr:     types.ErrSupplierInvalidAddress,
		},
		{
			desc:            "unstake supplier: invalid owner address",
			ownerAddress:    "invalid",
			operatorAddress: operatorAccount.Address.String(),
			expectedErr:     types.ErrSupplierInvalidAddress,
		},
		{
			desc:         "unstake supplier: missing operator address",
			ownerAddress: ownerAccount.Address.String(),
			// operatorAddress: operatorAccount.Address.String(),
			expectedErr: types.ErrSupplierInvalidAddress,
		},
		{
			desc:            "unstake supplier: invalid operator address",
			ownerAddress:    ownerAccount.Address.String(),
			operatorAddress: "invalid",
			expectedErr:     types.ErrSupplierInvalidAddress,
		},
	}

	// Run the tests
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			// Wait for a new block to be committed
			require.NoError(t, net.WaitForNextBlock())

			// Prepare the arguments for the CLI command
			args := []string{
				test.operatorAddress,
				fmt.Sprintf("--%s=%s", flags.FlagFrom, test.ownerAddress),
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
			require.Equal(t, uint32(0), resp.Code)
		})
	}
}
