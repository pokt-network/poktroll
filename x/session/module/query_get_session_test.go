package session_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	sdkerrors "cosmossdk.io/errors"
	cometcli "github.com/cometbft/cometbft/libs/cli"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	"github.com/gogo/status"
	"github.com/stretchr/testify/require"

	session "github.com/pokt-network/poktroll/x/session/module"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

func TestCLI_GetSession(t *testing.T) {
	if os.Getenv("INCLUDE_FLAKY_TESTS") != "true" {
		t.Skip("Skipping known flaky test: 'TestRelayerProxy'")
	} else {
		t.Log(`TODO_FLAKY: Skipping known flaky test: 'TestCLI_GetSession'

Run the following command a few times to verify it passes at least once:

$ go test -v -count=1 -run TestCLI_GetSession ./x/session/module/...`)
	}

	// Prepare the network
	net, suppliers, applications := networkWithApplicationsAndSupplier(t, 2)
	_, err := net.WaitForHeightWithTimeout(10, 30*time.Second) // Wait for a sufficiently high block height to ensure the staking transactions have been processed
	require.NoError(t, err)
	val := net.Validators[0]
	ctx := val.ClientCtx

	// Sanity check the application configs are what we expect them to be
	appSvc0 := applications[0]
	appSvc1 := applications[1]

	require.Len(t, appSvc0.ServiceConfigs, 2)
	require.Len(t, appSvc1.ServiceConfigs, 2)

	require.Equal(t, appSvc0.ServiceConfigs[0].Service.Id, "svc0")  // svc0 has a supplier
	require.Equal(t, appSvc0.ServiceConfigs[1].Service.Id, "svc00") // svc00 doesn't have a supplier
	require.Equal(t, appSvc1.ServiceConfigs[0].Service.Id, "svc1")  // svc1 has a supplier
	require.Equal(t, appSvc1.ServiceConfigs[1].Service.Id, "svc11") // svc11 doesn't have a supplier

	// Sanity check the supplier configs are what we expect them to be
	supplierSvc0 := suppliers[0] // supplier for svc0
	supplierSvc1 := suppliers[1] // supplier for svc1

	require.Len(t, supplierSvc0.Services, 1)
	require.Len(t, supplierSvc1.Services, 1)

	require.Equal(t, supplierSvc0.Services[0].Service.Id, "svc0")
	require.Equal(t, supplierSvc1.Services[0].Service.Id, "svc1")

	// Prepare the test cases
	tests := []struct {
		desc string

		appAddress  string
		serviceId   string
		blockHeight int64

		expectedErr          *sdkerrors.Error
		expectedNumSuppliers int
	}{
		// Valid requests
		{
			desc: "valid - block height specified and is zero",

			appAddress:  appSvc0.Address,
			serviceId:   "svc0",
			blockHeight: 0,

			expectedErr:          nil,
			expectedNumSuppliers: 1,
		},
		{
			desc: "valid - block height specified and is greater than zero",

			appAddress:  appSvc1.Address,
			serviceId:   "svc1",
			blockHeight: 10,

			expectedErr:          nil,
			expectedNumSuppliers: 1,
		},
		{
			desc: "valid - block height unspecified and defaults to 0",

			appAddress: appSvc0.Address,
			serviceId:  "svc0",
			// blockHeight explicitly omitted,

			expectedErr:          nil,
			expectedNumSuppliers: 1,
		},

		// Invalid requests - incompatible state
		{
			desc: "invalid - app not staked for service",

			appAddress:  appSvc0.Address,
			serviceId:   "svc9001", // appSvc0 is only staked for svc0 (has supplier) and svc00 (doesn't have supplier) and is not staked for service over 9000
			blockHeight: 0,

			expectedErr: sessiontypes.ErrSessionAppNotStakedForService,
		},
		{
			desc: "invalid - no suppliers staked for service",

			appAddress:  appSvc0.Address, // dynamically getting address from applications
			serviceId:   "svc00",         // appSvc0 is only staked for svc0 (has supplier) and svc00 (doesn't have supplier)
			blockHeight: 0,

			expectedErr: sessiontypes.ErrSessionSuppliersNotFound,
		},
		{
			desc: "invalid - block height is in the future",

			appAddress:  appSvc0.Address, // dynamically getting address from applications
			serviceId:   "svc0",
			blockHeight: 9001, // block height over 9000 is greater than the context height of 10

			expectedErr: sessiontypes.ErrSessionInvalidBlockHeight,
		},

		// Invalid requests - bad app address input
		{
			desc: "invalid - invalid appAddress",

			appAddress:  "invalidAddress", // providing a deliberately invalid address
			serviceId:   "svc0",
			blockHeight: 0,

			expectedErr: sessiontypes.ErrSessionInvalidAppAddress,
		},
		{
			desc: "invalid - missing appAddress",
			// appAddress explicitly omitted
			serviceId:   "svc0",
			blockHeight: 0,

			expectedErr: sessiontypes.ErrSessionInvalidAppAddress,
		},

		// Invalid requests - bad serviceID input
		{
			desc:        "invalid - invalid service ID",
			appAddress:  appSvc0.Address, // dynamically getting address from applications
			serviceId:   "invalidServiceId",
			blockHeight: 0,

			expectedErr: sessiontypes.ErrSessionInvalidService,
		},
		{
			desc:       "invalid - missing service ID",
			appAddress: appSvc0.Address, // dynamically getting address from applications
			// serviceId explicitly omitted
			blockHeight: 0,

			expectedErr: sessiontypes.ErrSessionInvalidService,
		},
	}

	// We want to use the `--output=json` flag for all tests so it's easy to unmarshal below
	common := []string{
		fmt.Sprintf("--%s=json", cometcli.OutputFlag),
	}

	// Run the tests
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			// Prepare the arguments for the CLI command
			args := []string{
				test.appAddress,
				test.serviceId,
				fmt.Sprintf("%d", test.blockHeight),
			}
			args = append(args, common...)

			// Execute the command
			getSessionOut, err := clitestutil.ExecTestCLICmd(ctx, session.CmdGetSession(), args)
			if test.expectedErr != nil {
				stat, ok := status.FromError(test.expectedErr)
				require.True(t, ok)
				require.Contains(t, stat.Message(), test.expectedErr.Error())
				return
			}
			require.NoError(t, err)

			var getSessionRes sessiontypes.QueryGetSessionResponse
			err = net.Config.Codec.UnmarshalJSON(getSessionOut.Bytes(), &getSessionRes)
			require.NoError(t, err)
			require.NotNil(t, getSessionRes)

			session := getSessionRes.Session
			require.NotNil(t, session)

			// Verify some data about the session
			require.Equal(t, test.appAddress, session.Application.Address)
			require.Equal(t, test.serviceId, session.Header.Service.Id)
			require.Len(t, session.Suppliers, test.expectedNumSuppliers)
		})
	}
}
