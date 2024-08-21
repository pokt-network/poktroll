package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/session/keeper"
	"github.com/pokt-network/poktroll/x/session/types"
)

func TestSession_HydrateSession_Success_BaseCase(t *testing.T) {
	sessionKeeper, ctx := keepertest.SessionKeeper(t)

	ctx = sdk.UnwrapSDKContext(ctx).WithBlockHeight(100) // provide a sufficiently large block height to avoid errors
	blockHeight := int64(10)

	sessionHydrator := keeper.NewSessionHydrator(keepertest.TestApp1Address, keepertest.TestServiceId1, blockHeight)
	session, err := sessionKeeper.HydrateSession(ctx, sessionHydrator)
	require.NoError(t, err)

	// Check the header
	sessionHeader := session.Header
	require.Equal(t, keepertest.TestApp1Address, sessionHeader.ApplicationAddress)
	require.Equal(t, keepertest.TestServiceId1, sessionHeader.Service.Id)
	require.Equal(t, "", sessionHeader.Service.Name)
	require.Equal(t, int64(9), sessionHeader.SessionStartBlockHeight)
	require.Equal(t, int64(12), sessionHeader.SessionEndBlockHeight)
	require.Equal(t, "fea5d6f7544ff6d8af5c22529b2ccf01ed7930b3d454d42dda5ccc0b65b6ebfd", sessionHeader.SessionId)

	// Check the session
	require.Equal(t, int64(4), session.NumBlocksPerSession)
	require.Equal(t, "fea5d6f7544ff6d8af5c22529b2ccf01ed7930b3d454d42dda5ccc0b65b6ebfd", session.SessionId)
	require.Equal(t, int64(3), session.SessionNumber)

	// Check the application
	app := session.Application
	require.Equal(t, keepertest.TestApp1Address, app.Address)
	require.Len(t, app.ServiceConfigs, 3)

	// Check the suppliers
	suppliers := session.Suppliers
	require.Len(t, suppliers, 1)

	supplier := suppliers[0]
	require.Equal(t, keepertest.TestSupplierOperatorAddress, supplier.OperatorAddress)
	require.Len(t, supplier.Services, 3)
}

func TestSession_HydrateSession_Metadata(t *testing.T) {
	// TODO_TEST: Extend these tests once `NumBlocksPerSession` is configurable.
	// Currently assumes NumBlocksPerSession=4
	tests := []struct {
		desc        string
		blockHeight int64

		expectedNumBlocksPerSession int64
		expectedSessionNumber       int64
		expectedSessionStartBlock   int64
		expectedSessionEndBlock     int64
		expectedErr                 error
	}{
		{
			desc:        "blockHeight = 0",
			blockHeight: 0,

			expectedNumBlocksPerSession: 4,
			expectedSessionNumber:       0,
			expectedSessionStartBlock:   0,
			expectedSessionEndBlock:     0,
			expectedErr:                 nil,
		},
		{
			desc:        "blockHeight = 1",
			blockHeight: 1,

			expectedNumBlocksPerSession: 4,
			expectedSessionNumber:       1,
			expectedSessionStartBlock:   1,
			expectedSessionEndBlock:     4,
			expectedErr:                 nil,
		},
		{
			desc:        "blockHeight = sessionHeight",
			blockHeight: 5,

			expectedNumBlocksPerSession: 4,
			expectedSessionNumber:       2,
			expectedSessionStartBlock:   5,
			expectedSessionEndBlock:     8,
			expectedErr:                 nil,
		},
		{
			desc:        "blockHeight != sessionHeight",
			blockHeight: 6,

			expectedNumBlocksPerSession: 4,
			expectedSessionNumber:       2,
			expectedSessionStartBlock:   5,
			expectedSessionEndBlock:     8,
			expectedErr:                 nil,
		},
		{
			desc:        "blockHeight > contextHeight",
			blockHeight: 9001, // block height over 9000 is too high given that the context height is 100

			expectedErr: types.ErrSessionHydration,
		},
	}

	appAddr := keepertest.TestApp1Address
	serviceId := keepertest.TestServiceId1
	sessionKeeper, ctx := keepertest.SessionKeeper(t)
	ctx = sdk.UnwrapSDKContext(ctx).WithBlockHeight(100) // provide a sufficiently large block height to avoid errors

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			sessionHydrator := keeper.NewSessionHydrator(appAddr, serviceId, test.blockHeight)
			session, err := sessionKeeper.HydrateSession(ctx, sessionHydrator)

			if test.expectedErr != nil {
				require.ErrorIs(t, test.expectedErr, err)
				return
			}
			require.NoError(t, err)

			require.Equal(t, test.expectedNumBlocksPerSession, session.NumBlocksPerSession)
			require.Equal(t, test.expectedSessionNumber, session.SessionNumber)
			require.Equal(t, test.expectedSessionStartBlock, session.Header.SessionStartBlockHeight)
			require.Equal(t, test.expectedSessionEndBlock, session.Header.SessionEndBlockHeight)
		})
	}
}

func TestSession_HydrateSession_SessionId(t *testing.T) {
	// TODO_TEST: Extend these tests once `NumBlocksPerSession` is configurable.
	// Currently assumes NumBlocksPerSession=4
	tests := []struct {
		desc string

		blockHeight1 int64
		blockHeight2 int64

		appAddr1 string
		appAddr2 string

		serviceId1 string
		serviceId2 string

		expectedSessionId1 string
		expectedSessionId2 string
	}{
		{
			desc: "(app1, svc1): sessionId at first session block != sessionId at next session block",

			blockHeight1: 5,
			blockHeight2: 9,

			appAddr1: keepertest.TestApp1Address, // app1
			appAddr2: keepertest.TestApp1Address, // app1

			serviceId1: keepertest.TestServiceId1, // svc1
			serviceId2: keepertest.TestServiceId1, // svc1

			expectedSessionId1: "e161348f2153bb41092040c3c287596f8daf98e90986475be21412a1ded945ed",
			expectedSessionId2: "fea5d6f7544ff6d8af5c22529b2ccf01ed7930b3d454d42dda5ccc0b65b6ebfd",
		},
		{
			desc: "app1: sessionId for svc1 != sessionId for svc12",

			blockHeight1: 5,
			blockHeight2: 5,

			appAddr1: keepertest.TestApp1Address, // app1
			appAddr2: keepertest.TestApp1Address, // app1

			serviceId1: keepertest.TestServiceId1,  // svc1
			serviceId2: keepertest.TestServiceId12, // svc12

			expectedSessionId1: "e161348f2153bb41092040c3c287596f8daf98e90986475be21412a1ded945ed",
			expectedSessionId2: "c01eb8924dbb9dae7cab8ed56016ce8fdd2d23542f9e0b7ca8d6972d3fb45ce5",
		},
		{
			desc: "svc12: sessionId for app1 != sessionId for app2",

			blockHeight1: 5,
			blockHeight2: 5,

			appAddr1: keepertest.TestApp1Address, // app1
			appAddr2: keepertest.TestApp2Address, // app2

			serviceId1: keepertest.TestServiceId12, // svc12
			serviceId2: keepertest.TestServiceId12, // svc12

			expectedSessionId1: "c01eb8924dbb9dae7cab8ed56016ce8fdd2d23542f9e0b7ca8d6972d3fb45ce5",
			expectedSessionId2: "f6dbe4961afd8ff0e444a1159d02bc645f9ded3daaa605c187e6bd7ee2232a68",
		},
	}

	sessionKeeper, ctx := keepertest.SessionKeeper(t)
	ctx = sdk.UnwrapSDKContext(ctx).WithBlockHeight(100) // provide a sufficiently large block height to avoid errors

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			sessionHydrator1 := keeper.NewSessionHydrator(test.appAddr1, test.serviceId1, test.blockHeight1)
			session1, err := sessionKeeper.HydrateSession(ctx, sessionHydrator1)
			require.NoError(t, err)

			sessionHydrator2 := keeper.NewSessionHydrator(test.appAddr2, test.serviceId2, test.blockHeight2)
			session2, err := sessionKeeper.HydrateSession(ctx, sessionHydrator2)
			require.NoError(t, err)

			require.NotEqual(t, session1.Header.SessionId, session2.Header.SessionId)
			require.Equal(t, test.expectedSessionId1, session1.Header.SessionId)
			require.Equal(t, test.expectedSessionId2, session2.Header.SessionId)
		})
	}
}

// TODO_TEST: Expand these tests to account for application joining/leaving the network at different heights as well changing the services they support
func TestSession_HydrateSession_Application(t *testing.T) {
	tests := []struct {
		// Description
		desc string
		// Inputs
		appAddr   string
		serviceId string

		// Outputs
		expectedErr error
	}{
		{
			desc: "app is found",

			appAddr:   keepertest.TestApp1Address,
			serviceId: keepertest.TestServiceId1,

			expectedErr: nil,
		},
		{
			desc: "app is not found",

			appAddr:   sample.AccAddress(), // Generating a random address on the fly
			serviceId: keepertest.TestServiceId1,

			expectedErr: types.ErrSessionHydration,
		},
		{
			desc: "invalid app address",

			appAddr:   "invalid",
			serviceId: keepertest.TestServiceId1,

			expectedErr: types.ErrSessionHydration,
		},
		{
			desc: "invalid - app not staked for service",

			appAddr:   keepertest.TestApp1Address, // app1
			serviceId: "svc9001",                  // app1 is only stake for svc1 and svc11

			expectedErr: types.ErrSessionHydration,
		},
		// TODO_TEST: Add tests for when:
		// - Application join/leaves (stakes/unstakes) altogether
		// - Application adds/removes certain services mid-session
		// - Application increases stakes mid-session
	}

	blockHeight := int64(10)
	sessionKeeper, ctx := keepertest.SessionKeeper(t)
	ctx = sdk.UnwrapSDKContext(ctx).WithBlockHeight(100) // provide a sufficiently large block height to avoid errors

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			sessionHydrator := keeper.NewSessionHydrator(test.appAddr, test.serviceId, blockHeight)
			_, err := sessionKeeper.HydrateSession(ctx, sessionHydrator)
			if test.expectedErr != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TODO_TEST: Expand these tests to account for supplier joining/leaving the network at different heights as well changing the services they support
func TestSession_HydrateSession_Suppliers(t *testing.T) {
	// TODO_TEST: Extend these tests once `NumBlocksPerSession` is configurable.
	// Currently assumes NumSupplierPerSession=15
	tests := []struct {
		// Description
		desc string

		// Inputs
		appAddr   string
		serviceId string

		// Outputs
		numExpectedSuppliers int
		expectedErr          error
	}{
		{
			desc: "num_suppliers_available = 0",

			appAddr:   keepertest.TestApp1Address, // app1
			serviceId: keepertest.TestServiceId11,

			numExpectedSuppliers: 0,
			expectedErr:          types.ErrSessionSuppliersNotFound,
		},
		{
			desc: "num_suppliers_available < num_suppliers_per_session_param",

			appAddr:   keepertest.TestApp1Address, // app1
			serviceId: keepertest.TestServiceId1,  // svc1

			numExpectedSuppliers: 1,
			expectedErr:          nil,
		},
		// TODO_TEST: Add this test once we make the num suppliers per session configurable
		// {
		// 	name: "num_suppliers_available > num_suppliers_per_session_param",
		// },
		// TODO_TEST: Add tests for when:
		// - Supplier join/leaves (stakes/unstakes) altogether
		// - Supplier adds/removes certain services mid-session
		// - Supplier increases stakes mid-session
	}

	blockHeight := int64(10)
	sessionKeeper, ctx := keepertest.SessionKeeper(t)
	ctx = sdk.UnwrapSDKContext(ctx).WithBlockHeight(100) // provide a sufficiently large block height to avoid errors

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {})

		sessionHydrator := keeper.NewSessionHydrator(test.appAddr, test.serviceId, blockHeight)
		session, err := sessionKeeper.HydrateSession(ctx, sessionHydrator)

		if test.expectedErr != nil {
			require.ErrorContains(t, err, test.expectedErr.Error())
			continue
		}
		require.NoError(t, err)
		require.Len(t, session.Suppliers, test.numExpectedSuppliers)
	}
}
