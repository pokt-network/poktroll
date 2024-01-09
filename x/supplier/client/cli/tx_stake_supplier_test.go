package cli_test

import (
	"context"
	"fmt"
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
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/client/cli"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

func TestCLI_StakeSupplier(t *testing.T) {
	ctx := context.Background()
	memnet := sessionnet.NewInMemoryNetworkWithSessions(t, &network.InMemoryNetworkConfig{})
	memnet.Start(ctx, t)

	clientCtx := memnet.GetClientCtx(t)
	net := memnet.GetNetwork(t)

	preGeneratedAcct := memnet.CreateNewOnChainAccount(t)
	supplier := sharedtypes.Supplier{
		Address: preGeneratedAcct.Address.String(),
	}

	// Common args used for all requests
	commonArgs := []string{
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(net.Config.BondDenom, sdkmath.NewInt(10))).String()),
	}

	defaultConfig := `
		stake_amount: 1000upokt
		services:
		  - service_id: svc1
		    endpoints:
		    - url: http://pokt.network:8081
		      rpc_type: json_rpc
		`

	tests := []struct {
		desc    string
		address string
		config  string
		err     *sdkerrors.Error
	}{
		// Happy Paths
		{
			desc:    "stake supplier: valid",
			address: supplier.Address,
			config:  defaultConfig,
		},

		// Error Paths - Address Related
		{
			desc: "stake supplier: missing address",
			// address:     "explicitly omitted",
			err:    suppliertypes.ErrSupplierInvalidAddress,
			config: defaultConfig,
		},
		{
			desc:    "stake supplier: invalid address",
			address: "invalid",
			err:     suppliertypes.ErrSupplierInvalidAddress,
			config:  defaultConfig,
		},

		// Error Paths - Stake Related
		{
			desc:    "stake supplier: missing stake",
			address: supplier.GetAddress(),
			err:     suppliertypes.ErrSupplierInvalidStake,
			// stakeString:    "explicitly omitted",
			config: `
				# explicitly omitted stake
				services:
				  - service_id: svc1
				    endpoints:
				    - url: http://pokt.network:8081
				      rpc_type: json_rpc
				`,
		},
		{
			desc:    "stake supplier: invalid stake denom",
			address: supplier.GetAddress(),
			err:     suppliertypes.ErrSupplierInvalidStake,
			config: `
				stake_amount: 1000invalid
				services:
				  - service_id: svc1
				    endpoints:
				    - url: http://pokt.network:8081
				      rpc_type: json_rpc
				`,
		},
		{
			desc:    "stake supplier: invalid stake amount (zero)",
			address: supplier.GetAddress(),
			err:     suppliertypes.ErrSupplierInvalidStake,
			config: `
				stake_amount: 0upokt
				services:
				  - service_id: svc1
				    endpoints:
				    - url: http://pokt.network:8081
				      rpc_type: json_rpc
				`,
		},
		{
			desc:    "stake supplier: invalid stake amount (negative)",
			address: supplier.GetAddress(),
			err:     suppliertypes.ErrSupplierInvalidStake,
			config: `
				stake_amount: -1000upokt
				services:
				  - service_id: svc1
				    endpoints:
				    - url: http://pokt.network:8081
				      rpc_type: json_rpc
				`,
		},

		// Happy Paths - Service Related
		{
			desc:    "services_test: valid multiple services",
			address: supplier.GetAddress(),
			config: `
				stake_amount: 1000upokt
				services:
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
			desc:    "services_test: valid localhost",
			address: supplier.GetAddress(),
			config: `
				stake_amount: 1000upokt
				services:
				  - service_id: svc1
				    endpoints:
				    - url: http://127.0.0.1:8082
				      rpc_type: json_rpc
				`,
		},
		{
			desc:    "services_test: valid loopback",
			address: supplier.GetAddress(),
			config: `
				stake_amount: 1000upokt
				services:
				  - service_id: svc1
				    endpoints:
				    - url: http://localhost:8082
				      rpc_type: json_rpc
				`,
		},
		{
			desc:    "services_test: valid without a pork",
			address: supplier.GetAddress(),
			config: `
				stake_amount: 1000upokt
				services:
				  - service_id: svc1
				    endpoints:
				    - url: http://pokt.network
				      rpc_type: json_rpc
				`,
		},

		// Error Paths - Service Related
		{
			desc:    "services_test: invalid services (missing argument)",
			address: supplier.GetAddress(),
			err:     suppliertypes.ErrSupplierInvalidServiceConfig,
			// servicesString: "explicitly omitted",
			config: `
				stake_amount: 1000upokt
				`,
		},
		{
			desc:    "services_test: invalid services (empty string)",
			address: supplier.GetAddress(),
			err:     suppliertypes.ErrSupplierInvalidServiceConfig,
			config: `
				stake_amount: 1000upokt
				services:
			`,
		},
		{
			desc:    "services_test: invalid URL",
			address: supplier.GetAddress(),
			err:     suppliertypes.ErrSupplierInvalidServiceConfig,
			config: `
				stake_amount: 1000upokt
				services:
				  - service_id: svc1
				    endpoints:
				    - url: bad_url
				      rpc_type: json_rpc
				`,
		},
		{
			desc:    "services_test: missing URLs",
			address: supplier.GetAddress(),
			err:     suppliertypes.ErrSupplierInvalidServiceConfig,
			config: `
				stake_amount: 1000upokt
				services:
				  - service_id: svc1
				  - service_id: svc2
				`,
		},
		{
			desc:    "services_test: missing service IDs",
			address: supplier.GetAddress(),
			err:     suppliertypes.ErrSupplierInvalidServiceConfig,
			config: `
				stake_amount: 1000upokt
				services:
				  - endpoints:
				    - url: localhost:8081
				      rpc_type: json_rpc
				  - endpoints:
				    - url: localhost:8082
				      rpc_type: json_rpc
				`,
		},
		{
			desc:    "services_test: missing rpc type",
			address: supplier.GetAddress(),
			err:     suppliertypes.ErrSupplierInvalidServiceConfig,
			config: `
				stake_amount: 1000upokt
				services:
				  - service_id: svc1
				    endpoints:
				    - url: localhost:8082
				`,
		},
	}

	// Run the tests
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			// Wait for a new block to be committed
			require.NoError(t, net.WaitForNextBlock())

			// write the stake config to a file
			configPath := testutil.WriteToNewTempFile(t, yaml.NormalizeYAMLIndentation(tt.config)).Name()

			// Prepare the arguments for the CLI command
			args := []string{
				fmt.Sprintf("--config=%s", configPath),
				fmt.Sprintf("--%s=%s", flags.FlagFrom, tt.address),
			}
			args = append(args, commonArgs...)

			// Execute the command
			outStake, err := testcli.ExecTestCLICmd(clientCtx, cli.CmdStakeSupplier(), args)

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
