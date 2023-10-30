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

	"pocket/testutil/network"
	"pocket/x/supplier/client/cli"
	"pocket/x/supplier/types"
)

func TestCLI_StakeSupplier(t *testing.T) {
	net, _ := networkWithSupplierObjects(t, 2)
	val := net.Validators[0]
	ctx := val.ClientCtx

	// Create a keyring and add an account for the supplier to be staked
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

	tests := []struct {
		desc           string
		address        string
		stakeString    string
		servicesString string
		err            *sdkerrors.Error
	}{
		// Happy Paths
		{
			desc:           "stake supplier: valid",
			address:        supplierAccount.Address.String(),
			stakeString:    "1000upokt",
			servicesString: "svc1;http://pokt.network:8081",
		},

		// Error Paths - Address Related
		{
			desc: "stake supplier: missing address",
			// address:     "explicitly missing",
			stakeString:    "1000upokt",
			servicesString: "svc1;http://pokt.network:8081",
			err:            types.ErrSupplierInvalidAddress,
		},
		{
			desc:           "stake supplier: invalid address",
			address:        "invalid",
			stakeString:    "1000upokt",
			servicesString: "svc1;http://pokt.network:8081",
			err:            types.ErrSupplierInvalidAddress,
		},

		// Error Paths - Stake Related
		{
			desc:    "stake supplier: missing stake",
			address: supplierAccount.Address.String(),
			// stakeString: "explicitly missing",
			servicesString: "svc1;http://pokt.network:8081",
			err:            types.ErrSupplierInvalidStake,
		},
		{
			desc:           "stake supplier: invalid stake denom",
			address:        supplierAccount.Address.String(),
			stakeString:    "1000invalid",
			servicesString: "svc1;http://pokt.network:8081",
			err:            types.ErrSupplierInvalidStake,
		},
		{
			desc:           "stake supplier: invalid stake amount (zero)",
			address:        supplierAccount.Address.String(),
			stakeString:    "0upokt",
			servicesString: "svc1;http://pokt.network:8081",
			err:            types.ErrSupplierInvalidStake,
		},
		{
			desc:           "stake supplier: invalid stake amount (negative)",
			address:        supplierAccount.Address.String(),
			stakeString:    "-1000upokt",
			servicesString: "svc1;http://pokt.network:8081",
			err:            types.ErrSupplierInvalidStake,
		},

		// Happy Paths - Service Related
		{
			desc:           "services_test: valid multiple services",
			address:        supplierAccount.Address.String(),
			stakeString:    "1000upokt",
			servicesString: "svc1;http://pokt.network:8081,svc2;http://pokt.network:8082",
		},
		{
			desc:           "services_test: valid localhost",
			address:        supplierAccount.Address.String(),
			stakeString:    "1000upokt",
			servicesString: "scv1;http://127.0.0.1:8082",
		},
		{
			desc:           "services_test: valid loopback",
			address:        supplierAccount.Address.String(),
			stakeString:    "1000upokt",
			servicesString: "scv1;http://localhost:8082",
		},
		{
			desc:           "services_test: valid without a pork",
			address:        supplierAccount.Address.String(),
			stakeString:    "1000upokt",
			servicesString: "scv1;http://pokt.network",
		},

		// Error Paths - Service Related
		{
			desc:        "services_test: invalid services (missing argument)",
			address:     supplierAccount.Address.String(),
			stakeString: "1000upokt",
			// servicesString: "explicitly omitted",
			err: types.ErrSupplierInvalidServiceConfig,
		},
		{
			desc:           "services_test: invalid services (empty string)",
			address:        supplierAccount.Address.String(),
			stakeString:    "1000upokt",
			servicesString: "",
			err:            types.ErrSupplierInvalidServiceConfig,
		},
		{
			desc:           "services_test: invalid because contains a space",
			address:        supplierAccount.Address.String(),
			stakeString:    "1000upokt",
			servicesString: "scv1 http://127.0.0.1:8082",
			err:            types.ErrSupplierInvalidServiceConfig,
		},
		{
			desc:           "services_test: invalid URL",
			address:        supplierAccount.Address.String(),
			stakeString:    "1000upokt",
			servicesString: "svc1;bad_url",
			err:            types.ErrSupplierInvalidServiceConfig,
		},
		{
			desc:           "services_test: missing URLs",
			address:        supplierAccount.Address.String(),
			stakeString:    "1000upokt",
			servicesString: "svc1,svc2;",
			err:            types.ErrSupplierInvalidServiceConfig,
		},
		{
			desc:           "services_test: missing service IDs",
			address:        supplierAccount.Address.String(),
			stakeString:    "1000upokt",
			servicesString: "localhost:8081,;localhost:8082",
			err:            types.ErrSupplierInvalidServiceConfig,
		},
	}

	// Initialize the Supplier Account by sending it some funds from the validator account that is part of genesis
	network.InitAccount(t, net, supplierAccount.Address)

	// Run the tests
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			// Wait for a new block to be committed
			require.NoError(t, net.WaitForNextBlock())

			// Prepare the arguments for the CLI command
			args := []string{
				tt.stakeString,
				tt.servicesString,
				fmt.Sprintf("--%s=%s", flags.FlagFrom, tt.address),
			}
			args = append(args, commonArgs...)

			// Execute the command
			outStake, err := clitestutil.ExecTestCLICmd(ctx, cli.CmdStakeSupplier(), args)

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
