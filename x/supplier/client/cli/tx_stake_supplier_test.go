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
	"github.com/pokt-network/poktroll/x/supplier/client/cli"
	"github.com/pokt-network/poktroll/x/supplier/client/config"
	"github.com/pokt-network/poktroll/x/supplier/types"
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

	defaultConfig := `
		- service_id: svc1
		  endpoints:
		  - url: http://pokt.network:8081
		    rpc_type: json_rpc
		`

	tests := []struct {
		desc        string
		address     string
		stakeString string
		config      string
		err         *sdkerrors.Error
	}{
		// Happy Paths
		{
			desc:        "stake supplier: valid",
			address:     supplierAccount.Address.String(),
			stakeString: "1000upokt",
			config:      defaultConfig,
		},

		// Error Paths - Address Related
		{
			desc: "stake supplier: missing address",
			// address:     "explicitly missing",
			err:         types.ErrSupplierInvalidAddress,
			stakeString: "1000upokt",
			config:      defaultConfig,
		},
		{
			desc:        "stake supplier: invalid address",
			address:     "invalid",
			stakeString: "1000upokt",
			err:         types.ErrSupplierInvalidAddress,
			config:      defaultConfig,
		},

		// Error Paths - Stake Related
		{
			desc:    "stake supplier: missing stake",
			address: supplierAccount.Address.String(),
			err:     types.ErrSupplierInvalidStake,
			// stakeString:    "explicitly missing",
			config: `
				- service_id: svc1
				  endpoints:
				  - url: http://pokt.network:8081
				    rpc_type: json_rpc
				`,
		},
		{
			desc:        "stake supplier: invalid stake denom",
			address:     supplierAccount.Address.String(),
			err:         types.ErrSupplierInvalidStake,
			stakeString: "1000invalid",
			config: `
				- service_id: svc1
				  endpoints:
				  - url: http://pokt.network:8081
				    rpc_type: json_rpc
				`,
		},
		{
			desc:        "stake supplier: invalid stake amount (zero)",
			address:     supplierAccount.Address.String(),
			err:         types.ErrSupplierInvalidStake,
			stakeString: "0upokt",
			config: `
				- service_id: svc1
				  endpoints:
				  - url: http://pokt.network:8081
				    rpc_type: json_rpc
				`,
		},
		{
			desc:        "stake supplier: invalid stake amount (negative)",
			address:     supplierAccount.Address.String(),
			err:         types.ErrSupplierInvalidStake,
			stakeString: "-1000upokt",
			config: `
				- service_id: svc1
				  endpoints:
				  - url: http://pokt.network:8081
				    rpc_type: json_rpc
				`,
		},

		// Happy Paths - Service Related
		{
			desc:        "services_test: valid multiple services",
			address:     supplierAccount.Address.String(),
			stakeString: "1000upokt",
			config: `
				- service_id: svc1
				  endpoints:
				  - url: http://pokt.network:8081
				    rpc_type: json_rpc
				- service_id: svc2
				  endpoints:
				  - url: http://pokt.network:8082
				    rpc_type: json_rpc
				`,
		},
		{
			desc:        "services_test: valid localhost",
			address:     supplierAccount.Address.String(),
			stakeString: "1000upokt",
			config: `
				- service_id: svc1
				  endpoints:
				  - url: http://127.0.0.1:8082
				    rpc_type: json_rpc
				`,
		},
		{
			desc:        "services_test: valid loopback",
			address:     supplierAccount.Address.String(),
			stakeString: "1000upokt",
			config: `
				- service_id: svc1
				  endpoints:
				  - url: http://localhost:8082
				    rpc_type: json_rpc
				`,
		},
		{
			desc:        "services_test: valid without a pork",
			address:     supplierAccount.Address.String(),
			stakeString: "1000upokt",
			config: `
				- service_id: svc1
				  endpoints:
				  - url: http://pokt.network
				    rpc_type: json_rpc
				`,
		},

		// Error Paths - Service Related
		{
			desc:    "services_test: invalid services (missing argument)",
			address: supplierAccount.Address.String(),
			err:     types.ErrSupplierInvalidServiceConfig,
			// servicesString: "explicitly omitted",
			stakeString: "1000upokt",
		},
		{
			desc:        "services_test: invalid services (empty string)",
			address:     supplierAccount.Address.String(),
			err:         types.ErrSupplierInvalidServiceConfig,
			stakeString: "1000upokt",
			config:      ``,
		},
		{
			desc:        "services_test: invalid URL",
			address:     supplierAccount.Address.String(),
			err:         types.ErrSupplierInvalidServiceConfig,
			stakeString: "1000upokt",
			config: `
				- service_id: svc1
				  endpoints:
				  - url: bad_url
				    rpc_type: json_rpc
				`,
		},
		{
			desc:        "services_test: missing URLs",
			address:     supplierAccount.Address.String(),
			err:         types.ErrSupplierInvalidServiceConfig,
			stakeString: "1000upokt",
			config: `
				- service_id: svc1
				- service_id: svc2
				`,
		},
		{
			desc:        "services_test: missing service IDs",
			address:     supplierAccount.Address.String(),
			err:         types.ErrSupplierInvalidServiceConfig,
			stakeString: "1000upokt",
			config: `
				- endpoints:
				  - url: localhost:8081
				    rpc_type: json_rpc
				- endpoints:
				  - url: localhost:8082
				    rpc_type: json_rpc
				`,
		},
		{
			desc:        "services_test: missing rpc type",
			address:     supplierAccount.Address.String(),
			err:         types.ErrSupplierInvalidServiceConfig,
			stakeString: "1000upokt",
			config: `
				- service_id: svc1
				  endpoints:
				  - url: localhost:8082
				`,
		},
	}

	// Initialize the Supplier Account by sending it some funds from the validator account that is part of genesis
	network.InitAccount(t, net, supplierAccount.Address)

	// Run the tests
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			// Wait for a new block to be committed
			require.NoError(t, net.WaitForNextBlock())

			// write the stake config to a file
			configPath := testutil.WriteToNewTempFile(t, config.NormalizeYAMLIndentation(tt.config)).Name()

			// Prepare the arguments for the CLI command
			args := []string{
				tt.stakeString,
				fmt.Sprintf("--config=%s", configPath),
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
