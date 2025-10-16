package service_test

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

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
		fmt.Sprintf("--%s=%t", flags.FlagUnordered, true),
		fmt.Sprintf("--%s=%s", flags.TimeoutDuration, 5*time.Second),
	}

	// Initialize the Supplier account by sending it some funds from the
	// validator account that is part of genesis
	network.InitAccountWithSequence(t, net, account.Address, 1)

	// Wait for a new block to be committed
	require.NoError(t, net.WaitForNextBlock())

	// Prepare two valid services
	svc1 := sharedtypes.Service{
		Id:                   "svc1",
		Name:                 "service name",
		ComputeUnitsPerRelay: 1,
	}
	svc2 := sharedtypes.Service{
		Id:                   "svc2",
		Name:                 "service name 2",
		ComputeUnitsPerRelay: 1,
	}
	// Add svc2 to the network
	args := []string{
		svc2.Id,
		svc2.Name,
		fmt.Sprintf("--%s=%s", flags.FlagFrom, account.Address.String()),
	}
	args = append(args, commonArgs...)

	_, err := clitestutil.ExecTestCLICmd(ctx, service.CmdAddService(), args)
	require.NoError(t, err)

	tests := []struct {
		desc             string
		ownerAddress     string
		service          sharedtypes.Service
		expectedErr      *sdkerrors.Error
		metadataBase64   string
		metadataFile     []byte
		expectedCLIError string
	}{
		{
			desc:         "valid - add new service",
			ownerAddress: account.Address.String(),
			service:      svc1,
		},
		{
			desc:         "valid - add new service without specifying compute units per relay so that it uses the default",
			ownerAddress: account.Address.String(),
			service: sharedtypes.Service{
				Id:                   svc1.Id,
				Name:                 svc1.Name,
				ComputeUnitsPerRelay: 0, // this parameter is omitted when the test is run
			},
		},
		{
			desc:           "valid - metadata via base64",
			ownerAddress:   account.Address.String(),
			service:        sharedtypes.Service{Id: "svc-metadata-base64", Name: "svc base64", ComputeUnitsPerRelay: 1},
			metadataBase64: base64.StdEncoding.EncodeToString(bytes.Repeat([]byte("a"), 1024)),
		},
		{
			desc:         "valid - metadata via file",
			ownerAddress: account.Address.String(),
			service:      sharedtypes.Service{Id: "svc-metadata-file", Name: "svc file", ComputeUnitsPerRelay: 1},
			metadataFile: bytes.Repeat([]byte("b"), 1024),
		},
		{
			desc:             "invalid - metadata exceeds limit",
			ownerAddress:     account.Address.String(),
			service:          sharedtypes.Service{Id: "svc-metadata-too-big", Name: "svc large", ComputeUnitsPerRelay: 1},
			metadataBase64:   base64.StdEncoding.EncodeToString(bytes.Repeat([]byte("c"), sharedtypes.MaxServiceMetadataSizeBytes+1)),
			expectedCLIError: "experimental service metadata size",
		},
		{
			desc:         "invalid - missing service id",
			ownerAddress: account.Address.String(),
			service:      sharedtypes.Service{Name: "service name"}, // ID intentionally omitted
			expectedErr:  types.ErrServiceMissingID,
		},
		{
			desc:         "invalid - missing service name",
			ownerAddress: account.Address.String(),
			service:      sharedtypes.Service{Id: "svc1"}, // Name intentionally omitted
			expectedErr:  types.ErrServiceMissingName,
		},
		{
			desc:         "invalid - invalid owner address",
			ownerAddress: "invalid address",
			service:      svc1,
			expectedErr:  types.ErrServiceInvalidAddress,
		},
		{
			desc:         "invalid - service already staked",
			ownerAddress: account.Address.String(),
			service:      svc2,
			expectedErr:  types.ErrServiceAlreadyExists,
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
			argsAndFlags = append(argsAndFlags, fmt.Sprintf("--%s=%s", flags.FlagFrom, test.ownerAddress))

			args := append(argsAndFlags, commonArgs...)

			if test.metadataBase64 != "" {
				args = append(args, fmt.Sprintf("--%s=%s", service.FlagExperimentalMetadataBase64, test.metadataBase64))
			}

			if len(test.metadataFile) > 0 {
				f, err := os.CreateTemp(t.TempDir(), "metadata-*.json")
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(f.Name(), test.metadataFile, 0o600))
				require.NoError(t, f.Close())
				args = append(args, fmt.Sprintf("--%s=%s", service.FlagExperimentalMetadataFile, f.Name()))
			}

			// Execute the command
			addServiceOutput, err := clitestutil.ExecTestCLICmd(ctx, service.CmdAddService(), args)

			if test.expectedCLIError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.expectedCLIError)
				return
			}

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
			// You can reference Cosmos SDK error codes here: https://github.com/cosmos/cosmos-sdk/blob/main/types/errors/errors.go
			require.Equal(t, uint32(0), resp.Code, "tx response failed: %v", resp)
		})
	}
}

// TestCLI_UpdateServiceWithMetadata tests updating an existing service with metadata
func TestCLI_UpdateServiceWithMetadata(t *testing.T) {
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
		fmt.Sprintf("--%s=%t", flags.FlagUnordered, true),
		fmt.Sprintf("--%s=%s", flags.TimeoutDuration, 5*time.Second),
	}

	// Initialize the account by sending it some funds
	network.InitAccountWithSequence(t, net, account.Address, 1)

	// Wait for a new block to be committed
	require.NoError(t, net.WaitForNextBlock())

	serviceID := "svc-update-test"
	serviceName := "Update Test Service"

	tests := []struct {
		desc             string
		initialMetadata  []byte
		updatedMetadata  []byte
		computeUnits     uint64
		updatedCU        uint64
		expectedErr      bool
	}{
		{
			desc:            "add metadata to service without metadata",
			initialMetadata: nil,
			updatedMetadata: bytes.Repeat([]byte("updated"), 100),
			computeUnits:    1,
			updatedCU:       1,
			expectedErr:     false,
		},
		{
			desc:            "update metadata of service with metadata",
			initialMetadata: bytes.Repeat([]byte("initial"), 100),
			updatedMetadata: bytes.Repeat([]byte("updated"), 200),
			computeUnits:    1,
			updatedCU:       2,
			expectedErr:     false,
		},
		{
			desc:            "update compute units and add metadata",
			initialMetadata: nil,
			updatedMetadata: bytes.Repeat([]byte("added"), 100),
			computeUnits:    1,
			updatedCU:       5,
			expectedErr:     false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			// Create a unique service ID for this test
			uniqueServiceID := fmt.Sprintf("%s-%s", serviceID, test.desc)

			// Step 1: Create the initial service
			args := []string{
				uniqueServiceID,
				serviceName,
				strconv.FormatUint(test.computeUnits, 10),
				fmt.Sprintf("--%s=%s", flags.FlagFrom, account.Address.String()),
			}
			args = append(args, commonArgs...)

			// Add initial metadata if provided
			if len(test.initialMetadata) > 0 {
				f, err := os.CreateTemp(t.TempDir(), "initial-*.json")
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(f.Name(), test.initialMetadata, 0o600))
				require.NoError(t, f.Close())
				args = append(args, fmt.Sprintf("--%s=%s", service.FlagExperimentalMetadataFile, f.Name()))
			}

			_, err := clitestutil.ExecTestCLICmd(ctx, service.CmdAddService(), args)
			require.NoError(t, err)

			// Wait for the service to be created
			require.NoError(t, net.WaitForNextBlock())

			// Step 2: Update the service with new metadata and/or compute units
			updateArgs := []string{
				uniqueServiceID,
				serviceName,
				strconv.FormatUint(test.updatedCU, 10),
				fmt.Sprintf("--%s=%s", flags.FlagFrom, account.Address.String()),
			}
			updateArgs = append(updateArgs, commonArgs...)

			// Add updated metadata
			if len(test.updatedMetadata) > 0 {
				f, err := os.CreateTemp(t.TempDir(), "updated-*.json")
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(f.Name(), test.updatedMetadata, 0o600))
				require.NoError(t, f.Close())
				updateArgs = append(updateArgs, fmt.Sprintf("--%s=%s", service.FlagExperimentalMetadataFile, f.Name()))
			}

			updateOutput, err := clitestutil.ExecTestCLICmd(ctx, service.CmdAddService(), updateArgs)
			if test.expectedErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Check the response
			var resp sdk.TxResponse
			require.NoError(t, net.Config.Codec.UnmarshalJSON(updateOutput.Bytes(), &resp))
			require.NotNil(t, resp)
			require.Equal(t, uint32(0), resp.Code, "tx response failed: %v", resp)

			// Wait for the update to be committed
			require.NoError(t, net.WaitForNextBlock())

			// Step 3: Query the service to verify metadata was updated
			queryCmd := fmt.Sprintf("query service show-service %s --output json", uniqueServiceID)
			queryOutput, err := clitestutil.ExecTestCLICmd(ctx, nil, []string{queryCmd})
			require.NoError(t, err)

			var queryResp types.QueryGetServiceResponse
			require.NoError(t, net.Config.Codec.UnmarshalJSON(queryOutput.Bytes(), &queryResp))
			require.NotNil(t, queryResp.Service)

			// Verify compute units were updated
			require.Equal(t, test.updatedCU, queryResp.Service.ComputeUnitsPerRelay)

			// Verify metadata was updated if provided
			if len(test.updatedMetadata) > 0 {
				require.NotNil(t, queryResp.Service.Metadata)
				require.Equal(t, test.updatedMetadata, queryResp.Service.Metadata.ExperimentalApiSpecs)
			}
		})
	}
}

// TestServiceMetadata_CreateAndQuery tests creating a service with metadata and querying it
func TestServiceMetadata_CreateAndQuery(t *testing.T) {
	net := network.New(t, network.DefaultConfig())
	val := net.Validators[0]
	ctx := val.ClientCtx

	// Create a keyring and add an account
	kr := ctx.Keyring
	accounts := testutil.CreateKeyringAccounts(t, kr, 1)
	account := accounts[0]
	ctx = ctx.WithKeyring(kr)

	commonArgs := []string{
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf(
			"--%s=%s",
			flags.FlagFees,
			sdk.NewCoins(sdk.NewCoin(net.Config.BondDenom, math.NewInt(10))).String(),
		),
	}

	network.InitAccountWithSequence(t, net, account.Address, 1)
	require.NoError(t, net.WaitForNextBlock())

	// Test cases for various metadata payloads
	tests := []struct {
		desc         string
		serviceID    string
		metadata     []byte
		expectExists bool
	}{
		{
			desc:         "small JSON metadata",
			serviceID:    "svc-json",
			metadata:     []byte(`{"openapi":"3.0.0","info":{"title":"Test API"}}`),
			expectExists: true,
		},
		{
			desc:         "binary metadata",
			serviceID:    "svc-binary",
			metadata:     bytes.Repeat([]byte{0x00, 0x01, 0x02, 0xFF}, 256),
			expectExists: true,
		},
		{
			desc:         "max size metadata",
			serviceID:    "svc-max",
			metadata:     bytes.Repeat([]byte("x"), sharedtypes.MaxServiceMetadataSizeBytes),
			expectExists: true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			// Create service with metadata
			args := []string{
				test.serviceID,
				test.desc,
				"1",
				fmt.Sprintf("--%s=%s", flags.FlagFrom, account.Address.String()),
			}
			args = append(args, commonArgs...)

			if len(test.metadata) > 0 {
				f, err := os.CreateTemp(t.TempDir(), "metadata-*.dat")
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(f.Name(), test.metadata, 0o600))
				require.NoError(t, f.Close())
				args = append(args, fmt.Sprintf("--%s=%s", service.FlagExperimentalMetadataFile, f.Name()))
			}

			_, err := clitestutil.ExecTestCLICmd(ctx, service.CmdAddService(), args)
			require.NoError(t, err)

			require.NoError(t, net.WaitForNextBlock())

			// Query and verify metadata
			queryCmd := fmt.Sprintf("query service show-service %s --output json", test.serviceID)
			queryOutput, err := clitestutil.ExecTestCLICmd(ctx, nil, []string{queryCmd})
			require.NoError(t, err)

			var queryResp types.QueryGetServiceResponse
			require.NoError(t, net.Config.Codec.UnmarshalJSON(queryOutput.Bytes(), &queryResp))
			require.NotNil(t, queryResp.Service)

			if test.expectExists {
				require.NotNil(t, queryResp.Service.Metadata)
				require.Equal(t, test.metadata, queryResp.Service.Metadata.ExperimentalApiSpecs)
			} else {
				require.Nil(t, queryResp.Service.Metadata)
			}
		})
	}
}
