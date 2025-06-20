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
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/yaml"
	supplier "github.com/pokt-network/poktroll/x/supplier/module"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

func TestCLI_StakeSupplier(t *testing.T) {
	net, _ := networkWithSupplierObjects(t, 2)
	val := net.Validators[0]
	ctx := val.ClientCtx

	// Create a keyring and add an account for the supplier to be staked
	kr := ctx.Keyring
	accounts := testutil.CreateKeyringAccounts(t, kr, 2)
	ownerAccount := accounts[0]
	operatorAccount := accounts[1]

	// Initialize the Supplier Account by sending it some funds from the validator account that is part of genesis
	network.InitAccount(t, net, ownerAccount.Address)
	err := net.WaitForNextBlock()
	require.NoError(t, err)
	network.InitAccount(t, net, operatorAccount.Address)

	// Update the context with the new keyring
	ctx = ctx.WithKeyring(kr)

	// Common args used for all requests
	commonArgs := []string{
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(net.Config.BondDenom, math.NewInt(10))).String()),
	}

	defaultConfig := fmt.Sprintf(`
		owner_address: %s
		operator_address: %s
		stake_amount: 1000upokt
		services:
		  - service_id: svc1
		    endpoints:
		    - publicly_exposed_url: http://pokt.network:8081
		      rpc_type: json_rpc
		`, ownerAccount.Address.String(), operatorAccount.Address.String())

	stakeOnlyConfig := fmt.Sprintf(`
		owner_address: %s
		operator_address: %s
		stake_amount: 1000upokt
		`, ownerAccount.Address.String(), operatorAccount.Address.String())

	servicesOnlyConfig := fmt.Sprintf(`
		owner_address: %s
		operator_address: %s
		services:
		  - service_id: svc1
		    endpoints:
		    - publicly_exposed_url: http://pokt.network:8081
		      rpc_type: json_rpc
		`, ownerAccount.Address.String(), operatorAccount.Address.String())

	tests := []struct {
		desc            string
		ownerAddress    string
		signerAddress   string
		config          string
		additionalFlags []string
		expectedErr     *sdkerrors.Error
	}{
		// Happy Paths
		{
			desc:         "owner stake supplier: valid stake only config",
			ownerAddress: ownerAccount.Address.String(),
			config:       stakeOnlyConfig,
			additionalFlags: []string{
				fmt.Sprintf("--%s=true", "stake-only"),
			},
		},
		{
			desc:          "operator stake supplier: valid stake only config",
			ownerAddress:  ownerAccount.Address.String(),
			signerAddress: operatorAccount.Address.String(),
			config:        stakeOnlyConfig,
			additionalFlags: []string{
				fmt.Sprintf("--%s=true", "stake-only"),
			},
		},
		{
			desc:          "operator stake supplier: valid services only configs",
			ownerAddress:  ownerAccount.Address.String(),
			signerAddress: operatorAccount.Address.String(),
			config:        servicesOnlyConfig,
			additionalFlags: []string{
				fmt.Sprintf("--%s=true", "services-only"),
			},
		},
		{
			desc:         "stake supplier: valid, omitted operator address",
			ownerAddress: ownerAccount.Address.String(),
			config: fmt.Sprintf(`
		owner_address: %s
		stake_amount: 1000upokt
		services:
		  - service_id: svc1
		    endpoints:
		    - publicly_exposed_url: http://pokt.network:8081
		      rpc_type: json_rpc
		`, ownerAccount.Address.String()),
		},

		// Error Paths - Address Related
		{
			desc: "stake supplier: missing owner address",
			// ownerAddress:     "explicitly missing",
			expectedErr: types.ErrSupplierInvalidAddress,
			config:      defaultConfig,
		},
		{
			desc:         "stake supplier: invalid owner address",
			ownerAddress: "invalid",
			expectedErr:  types.ErrSupplierInvalidAddress,
			config:       defaultConfig,
		},

		// Error Paths - Stake Related
		{
			desc:         "stake supplier: missing stake",
			ownerAddress: ownerAccount.Address.String(),
			expectedErr:  types.ErrSupplierInvalidStake,
			config: fmt.Sprintf(`
				owner_address: %s
				operator_address: %s
				# explicitly omitted stake
				services:
				  - service_id: svc1
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				`, ownerAccount.Address.String(), operatorAccount.Address.String()),
		},
		{
			desc:         "stake supplier: invalid stake denom",
			ownerAddress: ownerAccount.Address.String(),
			expectedErr:  types.ErrSupplierInvalidStake,
			config: fmt.Sprintf(`
				owner_address: %s
				operator_address: %s
				stake_amount: 1000invalid
				services:
				  - service_id: svc1
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				`, ownerAccount.Address.String(), operatorAccount.Address.String()),
		},
		{
			desc:         "stake supplier: invalid stake amount (zero)",
			ownerAddress: ownerAccount.Address.String(),
			expectedErr:  types.ErrSupplierInvalidStake,
			config: fmt.Sprintf(`
				owner_address: %s
				operator_address: %s
				stake_amount: 0upokt
				services:
				  - service_id: svc1
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				`, ownerAccount.Address.String(), operatorAccount.Address.String()),
		},
		{
			desc:         "stake supplier: invalid stake amount (negative)",
			ownerAddress: ownerAccount.Address.String(),
			expectedErr:  types.ErrSupplierInvalidStake,
			config: fmt.Sprintf(`
				owner_address: %s
				operator_address: %s
				stake_amount: -1000upokt
				services:
				  - service_id: svc1
				    endpoints:
				    - publicly_exposed_url: http://pokt.network:8081
				      rpc_type: json_rpc
				`, ownerAccount.Address.String(), operatorAccount.Address.String()),
		},

		// Happy Paths - Service Related
		{
			desc:          "services_test: valid multiple services",
			ownerAddress:  ownerAccount.Address.String(),
			signerAddress: operatorAccount.Address.String(),
			config: fmt.Sprintf(`
				owner_address: %s
				operator_address: %s
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
				`, ownerAccount.Address.String(), operatorAccount.Address.String()),
		},
		{
			desc:          "services_test: valid localhost",
			ownerAddress:  ownerAccount.Address.String(),
			signerAddress: operatorAccount.Address.String(),
			config: fmt.Sprintf(`
				owner_address: %s
				operator_address: %s
				stake_amount: 1000upokt
				services:
				  - service_id: svc1
				    endpoints:
				    - publicly_exposed_url: http://127.0.0.1:8082
				      rpc_type: json_rpc
				`, ownerAccount.Address.String(), operatorAccount.Address.String()),
		},
		{
			desc:          "services_test: valid loopback",
			ownerAddress:  ownerAccount.Address.String(),
			signerAddress: operatorAccount.Address.String(),
			config: fmt.Sprintf(`
				owner_address: %s
				operator_address: %s
				stake_amount: 1000upokt
				services:
				  - service_id: svc1
				    endpoints:
				    - publicly_exposed_url: http://localhost:8082
				      rpc_type: json_rpc
				`, ownerAccount.Address.String(), operatorAccount.Address.String()),
		},
		{
			desc:          "services_test: valid without a port",
			ownerAddress:  ownerAccount.Address.String(),
			signerAddress: operatorAccount.Address.String(),
			config: fmt.Sprintf(`
				owner_address: %s
				operator_address: %s
				stake_amount: 1000upokt
				services:
				  - service_id: svc1
				    endpoints:
				    - publicly_exposed_url: http://pokt.network
				      rpc_type: json_rpc
				`, ownerAccount.Address.String(), operatorAccount.Address.String()),
		},

		// Error Paths - Service Related
		{
			desc:         "owner stake supplier: unauthorized valid service configs",
			ownerAddress: ownerAccount.Address.String(),
			config:       defaultConfig,
			expectedErr:  types.ErrSupplierInvalidServiceConfig,
		},
		{
			desc:         "owner stake supplier: missing --stake-only flag",
			ownerAddress: ownerAccount.Address.String(),
			config:       stakeOnlyConfig,
			expectedErr:  types.ErrSupplierInvalidServiceConfig,
		},
		{
			desc:          "operator stake supplier: missing --stake-only flag",
			signerAddress: operatorAccount.Address.String(),
			ownerAddress:  ownerAccount.Address.String(),
			config:        stakeOnlyConfig,
			expectedErr:   types.ErrSupplierInvalidServiceConfig,
		},
		{
			desc:          "operator stake supplier: missing --services-only flag",
			signerAddress: operatorAccount.Address.String(),
			ownerAddress:  ownerAccount.Address.String(),
			config:        servicesOnlyConfig,
			expectedErr:   types.ErrSupplierInvalidServiceConfig,
		},
		{
			desc:         "services_test: invalid services (missing argument)",
			ownerAddress: ownerAccount.Address.String(),
			expectedErr:  types.ErrSupplierInvalidServiceConfig,
			// servicesString explicitly omitted
			config: fmt.Sprintf(`
				owner_address: %s
				operator_address: %s
				stake_amount: 1000upokt
				`, ownerAccount.Address.String(), sample.AccAddress()),
		},
		{
			desc:         "services_test: invalid services (empty string)",
			ownerAddress: ownerAccount.Address.String(),
			expectedErr:  types.ErrSupplierInvalidServiceConfig,
			config: fmt.Sprintf(`
				owner_address: %s
				operator_address: %s
				stake_amount: 1000upokt
				services:
			`, ownerAccount.Address.String(), sample.AccAddress()),
		},
		{
			desc:         "services_test: invalid URL",
			ownerAddress: ownerAccount.Address.String(),
			expectedErr:  types.ErrSupplierInvalidServiceConfig,
			config: fmt.Sprintf(`
				owner_address: %s
				operator_address: %s
				stake_amount: 1000upokt
				services:
				  - service_id: svc1
				    endpoints:
				    - publicly_exposed_url: bad_url
				      rpc_type: json_rpc
				`, ownerAccount.Address.String(), sample.AccAddress()),
		},
		{
			desc:         "services_test: missing URLs",
			ownerAddress: ownerAccount.Address.String(),
			expectedErr:  types.ErrSupplierInvalidServiceConfig,
			config: fmt.Sprintf(`
				owner_address: %s
				operator_address: %s
				stake_amount: 1000upokt
				services:
				  - service_id: svc1
				  - service_id: svc2
				`, ownerAccount.Address.String(), sample.AccAddress()),
		},
		{
			desc:         "services_test: missing service IDs",
			ownerAddress: ownerAccount.Address.String(),
			expectedErr:  types.ErrSupplierInvalidServiceConfig,
			config: fmt.Sprintf(`
				owner_address: %s
				operator_address: %s
				stake_amount: 1000upokt
				services:
				  - endpoints:
				    - publicly_exposed_url: localhost:8081
				      rpc_type: json_rpc
				  - endpoints:
				    - publicly_exposed_url: localhost:8082
				      rpc_type: json_rpc
				`, ownerAccount.Address.String(), sample.AccAddress()),
		},
		{
			desc:         "services_test: missing rpc type",
			ownerAddress: ownerAccount.Address.String(),
			expectedErr:  types.ErrSupplierInvalidServiceConfig,
			config: fmt.Sprintf(`
				owner_address: %s
				operator_address: %s
				stake_amount: 1000upokt
				services:
				  - service_id: svc1
				    endpoints:
				    - publicly_exposed_url: localhost:8082
				`, ownerAccount.Address.String(), sample.AccAddress()),
		},
	}

	// Run the tests
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			// Wait for a new block to be committed
			require.NoError(t, net.WaitForNextBlock())

			signerAddress := test.signerAddress
			if signerAddress == "" {
				signerAddress = test.ownerAddress
			}
			args := []string{
				fmt.Sprintf("--%s=%s", flags.FlagFrom, signerAddress),
			}

			if test.config != "" {
				// write the stake config to a file
				configPath := testutil.WriteToNewTempFile(t, yaml.NormalizeYAMLIndentation(test.config)).Name()

				// Append the config file path to the arguments
				args = append(args, fmt.Sprintf("--config=%s", configPath))
			}

			if len(test.additionalFlags) > 0 {
				args = append(args, test.additionalFlags...)
			}

			args = append(args, commonArgs...)

			// Execute the command
			outStake, err := clitestutil.ExecTestCLICmd(ctx, supplier.CmdStakeSupplier(), args)

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
			// You can reference Cosmos SDK error codes here: https://github.com/cosmos/cosmos-sdk/blob/main/types/errors/errors.go
			require.Equal(t, uint32(0), resp.Code, "tx response failed: %v", resp)
		})
	}
}
