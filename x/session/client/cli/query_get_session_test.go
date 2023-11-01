package cli_test

import (
	"fmt"
	"testing"

	sdkerrors "cosmossdk.io/errors"
	tmcli "github.com/cometbft/cometbft/libs/cli"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	"github.com/gogo/status"
	"github.com/stretchr/testify/require"

	"pocket/x/session/client/cli"
	sessiontypes "pocket/x/session/types"
)

func TestCLI_GetSession(t *testing.T) {
	// Prepare the network
	net, suppliers, applications := networkWithApplicationsAndSupplier(t, 2)
	val := net.Validators[0]
	ctx := val.ClientCtx

	// Sanity check the application configs are what we expect them to be
	appSvc0 := applications[0]
	appSvc1 := applications[1]

	require.Len(t, appSvc0.ServiceConfigs, 2)
	require.Len(t, appSvc1.ServiceConfigs, 2)

	require.Equal(t, appSvc0.ServiceConfigs[0].ServiceId.Id, "svc0")
	require.Equal(t, appSvc0.ServiceConfigs[1].ServiceId.Id, "svc00")
	require.Equal(t, appSvc1.ServiceConfigs[0].ServiceId.Id, "svc1")
	require.Equal(t, appSvc1.ServiceConfigs[1].ServiceId.Id, "svc11")

	// Sanity check the supplier configs are what we expect them to be
	supplierSvc0 := suppliers[0]
	supplierSvc1 := suppliers[1]

	require.Len(t, supplierSvc0.Services, 1)
	require.Len(t, supplierSvc1.Services, 1)

	require.Equal(t, supplierSvc0.Services[0].ServiceId.Id, "svc0")
	require.Equal(t, supplierSvc1.Services[0].ServiceId.Id, "svc1")

	// Prepare the test cases
	tests := []struct {
		desc string

		appAddress  string
		serviceId   string
		blockHeight int64

		expectedNumSuppliers int
		expectedErr          *sdkerrors.Error
	}{
		// Valid requests
		{
			desc: "valid - block height specified and is zero",

			appAddress:  appSvc0.Address,
			serviceId:   "svc0",
			blockHeight: 0,

			expectedNumSuppliers: 1,
			expectedErr:          nil,
		},
		{
			desc: "valid - block height specified and is greater than zero",

			appAddress:  appSvc1.Address,
			serviceId:   "svc1",
			blockHeight: 10, // example value; adjust as needed

			expectedNumSuppliers: 1,
			expectedErr:          nil,
		},
		{
			desc: "valid - block height unspecified and defaults to -1",

			appAddress: appSvc0.Address,
			serviceId:  "svc0",
			// blockHeight: intentionally omitted,

			expectedNumSuppliers: 1,
			expectedErr:          nil,
		},

		// Invalid requests - incompatible state
		{
			desc: "invalid - app not staked for service",

			appAddress:  appSvc0.Address,
			serviceId:   "svc9001", // appSvc0 is only staked for svc0 (has supplier) and svc00 (doesn't have supplier)
			blockHeight: 0,

			// expectedNumSuppliers:
			expectedErr: sessiontypes.ErrAppNotStakedForService,
		},
		{
			desc: "invalid - no suppliers staked for service",

			appAddress:  appSvc1.Address, // dynamically getting address from applications
			serviceId:   "svc00",         // appSvc0 is only staked for svc0 (has supplier) and svc00 (doesn't have supplier)
			blockHeight: 0,

			// expectedNumSuppliers:
			expectedErr: sessiontypes.ErrSuppliersNotFound,
		},
		{
			desc: "invalid - block height is in the future",

			appAddress:  appSvc0.Address, // dynamically getting address from applications
			serviceId:   "svc0",
			blockHeight: 100000, // example future value; adjust as needed

			// expectedNumSuppliers:
			err: sessiontypes.ErrSessionInvalidAppAddress,
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
			// appAddress: intentionally omitted
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

			expectedErr: sessiontypes.ErrSessionInvalidServiceId,
		},
		{
			desc:       "invalid - missing service ID",
			appAddress: appSvc0.Address, // dynamically getting address from applications
			// serviceId:   intentionally omitted
			blockHeight: 0,

			expectedErr: sessiontypes.ErrSessionInvalidServiceId,
		},

		// Invalid requests - bad blockHeight input
		// {
		// 	desc: "invalid - blockHeight < -1",

		// 	appAddress:  appSvc0.Address, // dynamically getting address from applications
		// 	serviceId:   "svc0",
		// 	blockHeight: -2,

		// 	err: nil,
		// },
	}

	common := []string{
		fmt.Sprintf("--%s=json", tmcli.OutputFlag),
	}

	// Run the tests
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			// Prepare the arguments for the CLI command
			args := []string{
				tt.appAddress,
				tt.serviceId,
				fmt.Sprintf("%d", tt.blockHeight),
			}
			args = append(args, common...)

			// Execute the command
			getSessionOut, err := clitestutil.ExecTestCLICmd(ctx, cli.CmdGetSession(), args)
			if tt.expectedErr != nil {
				stat, ok := status.FromError(tt.expectedErr)
				require.True(t, ok)
				require.Contains(t, stat.Message(), tt.expectedErr.Error())
				return
			}
			require.NoError(t, err)

			var getSessionRes sessiontypes.QueryGetSessionResponse
			err = net.Config.Codec.UnmarshalJSON(getSessionOut.Bytes(), &getSessionRes)
			require.NoError(t, err)
			require.NotNil(t, getSessionRes)

			session := getSessionRes.Session
			require.NotNil(t, session)

			fmt.Println("TODO sessionRes", session)

		})
	}
}
