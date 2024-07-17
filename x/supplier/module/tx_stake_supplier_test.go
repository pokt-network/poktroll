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

	"github.com/pokt-network/poktroll/proto/types/supplier"
	"github.com/pokt-network/poktroll/testutil/network"
	"github.com/pokt-network/poktroll/testutil/yaml"
	suppliermodule "github.com/pokt-network/poktroll/x/supplier/module"
)

func TestCLI_StakeSupplier(t *testing.T) {
	net, _ := networkWithSupplierObjects(t, 2)
	val := net.Validators[0]
	ctx := val.ClientCtx

	// Create a keyring and add an account for the supplier to be staked
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

	defaultConfig := `
		stake_amount: 1000upokt
		services:
		  - service_id: svc1
		    endpoints:
		    - publicly_exposed_url: http://pokt.network:8081
		      rpc_type: json_rpc
		`

	tests := []struct {
		desc        string
		address     string
		config      string
		expectedErr *sdkerrors.Error
	}{
		// Happy Paths
		{
			desc:    "stake supplier: valid",
			address: supplierAccount.Address.String(),
			config:  defaultConfig,
		},

		// Error Paths - Address Related
		{
			desc: "stake supplier: missing address",
			// address:     "explicitly missing",
			expectedErr: supplier.ErrSupplierInvalidAddress,
			config:      defaultConfig,
		},
		{
			desc:        "stake supplier: invalid address",
			address:     "invalid",
			expectedErr: supplier.ErrSupplierInvalidAddress,
			config:      defaultConfig,
		},

		// Error Paths - Stake Related
		{
			desc:        "stake supplier: missing stake",
			address:     supplierAccount.Address.String(),
			expectedErr: supplier.ErrSupplierInvalidStake,
			config: `
				# explicitly omitted stake
				services:
				  - service_id: svc1
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				`,
		},
		{
			desc:        "stake supplier: invalid stake denom",
			address:     supplierAccount.Address.String(),
			expectedErr: supplier.ErrSupplierInvalidStake,
			config: `
				stake_amount: 1000invalid
				services:
				  - service_id: svc1
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				`,
		},
		{
			desc:        "stake supplier: invalid stake amount (zero)",
			address:     supplierAccount.Address.String(),
			expectedErr: supplier.ErrSupplierInvalidStake,
			config: `
				stake_amount: 0upokt
				services:
				  - service_id: svc1
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				`,
		},
		{
			desc:        "stake supplier: invalid stake amount (negative)",
			address:     supplierAccount.Address.String(),
			expectedErr: supplier.ErrSupplierInvalidStake,
			config: `
				stake_amount: -1000upokt
				services:
				  - service_id: svc1
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				`,
		},

		// Happy Paths - Service Related
		{
			desc:    "services_test: valid multiple services",
			address: supplierAccount.Address.String(),
			config: `
				stake_amount: 1000upokt
				services:
				  - service_id: svc1
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				  - service_id: svc2
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8082
				      rpc_type: json_rpc
				`,
		},
		{
			desc:    "services_test: valid localhost",
			address: supplierAccount.Address.String(),
			config: `
				stake_amount: 1000upokt
				services:
				  - service_id: svc1
				    endpoints:
				    - publicly_exposed_url: http://127.0.0.1:8082
				      rpc_type: json_rpc
				`,
		},
		{
			desc:    "services_test: valid loopback",
			address: supplierAccount.Address.String(),
			config: `
				stake_amount: 1000upokt
				services:
				  - service_id: svc1
				    endpoints:
				    - publicly_exposed_url: http://localhost:8082
				      rpc_type: json_rpc
				`,
		},
		{
			desc:    "services_test: valid without a port",
			address: supplierAccount.Address.String(),
			config: `
				stake_amount: 1000upokt
				services:
				  - service_id: svc1
				    endpoints:
				    - publicly_exposed_url: http://pokt.network
				      rpc_type: json_rpc
				`,
		},

		// Error Paths - Service Related
		{
			desc:        "services_test: invalid services (missing argument)",
			address:     supplierAccount.Address.String(),
			expectedErr: supplier.ErrSupplierInvalidServiceConfig,
			// servicesString explicitly omitted
			config: `
				stake_amount: 1000upokt
				`,
		},
		{
			desc:        "services_test: invalid services (empty string)",
			address:     supplierAccount.Address.String(),
			expectedErr: supplier.ErrSupplierInvalidServiceConfig,
			config: `
				stake_amount: 1000upokt
				services:
			`,
		},
		{
			desc:        "services_test: invalid URL",
			address:     supplierAccount.Address.String(),
			expectedErr: supplier.ErrSupplierInvalidServiceConfig,
			config: `
				stake_amount: 1000upokt
				services:
				  - service_id: svc1
				    endpoints:
				    - publicly_exposed_url: bad_url
				      rpc_type: json_rpc
				`,
		},
		{
			desc:        "services_test: missing URLs",
			address:     supplierAccount.Address.String(),
			expectedErr: supplier.ErrSupplierInvalidServiceConfig,
			config: `
				stake_amount: 1000upokt
				services:
				  - service_id: svc1
				  - service_id: svc2
				`,
		},
		{
			desc:        "services_test: missing service IDs",
			address:     supplierAccount.Address.String(),
			expectedErr: supplier.ErrSupplierInvalidServiceConfig,
			config: `
				stake_amount: 1000upokt
				services:
				  - endpoints:
				    - publicly_exposed_url: localhost:8081
				      rpc_type: json_rpc
				  - endpoints:
				    - publicly_exposed_url: localhost:8082
				      rpc_type: json_rpc
				`,
		},
		{
			desc:        "services_test: missing rpc type",
			address:     supplierAccount.Address.String(),
			expectedErr: supplier.ErrSupplierInvalidServiceConfig,
			config: `
				stake_amount: 1000upokt
				services:
				  - service_id: svc1
				    endpoints:
				    - publicly_exposed_url: localhost:8082
				`,
		},
	}

	// Run the tests
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			// Wait for a new block to be committed
			require.NoError(t, net.WaitForNextBlock())

			// write the stake config to a file
			configPath := testutil.WriteToNewTempFile(t, yaml.NormalizeYAMLIndentation(test.config)).Name()

			// Prepare the arguments for the CLI command
			args := []string{
				fmt.Sprintf("--config=%s", configPath),
				fmt.Sprintf("--%s=%s", flags.FlagFrom, test.address),
			}
			args = append(args, commonArgs...)

			// Execute the command
			outStake, err := clitestutil.ExecTestCLICmd(ctx, suppliermodule.CmdStakeSupplier(), args)

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
			require.NoError(t, net.Config.Codec.UnmarshalJSON(outStake.Bytes(), &resp))
			require.NotNil(t, resp)
			require.NotNil(t, resp.TxHash)
			require.Equal(t, uint32(0), resp.Code)
		})
	}
}
