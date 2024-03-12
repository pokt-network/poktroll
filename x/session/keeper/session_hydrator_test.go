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
	require.Equal(t, "20650c94d0bcc4b855654b5533bf880345ae96933c5fa7424ce59c698d208e22", sessionHeader.SessionId)

	// Check the session
	require.Equal(t, int64(4), session.NumBlocksPerSession)
	require.Equal(t, "20650c94d0bcc4b855654b5533bf880345ae96933c5fa7424ce59c698d208e22", session.SessionId)
	require.Equal(t, int64(3), session.SessionNumber)

	// Check the application
	app := session.Application
	require.Equal(t, keepertest.TestApp1Address, app.Address)
	require.Len(t, app.ServiceConfigs, 3)

	// Check the suppliers
	suppliers := session.Suppliers
	require.Len(t, suppliers, 1)

	supplier := suppliers[0]
	require.Equal(t, keepertest.TestSupplierAddress, supplier.Address)
	require.Len(t, supplier.Services, 3)
}

func TestSession_HydrateSession_Metadata(t *testing.T) {
	// TODO_TECHDEBT: Extend these tests once `NumBlocksPerSession` is configurable.
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
	// TODO_TECHDEBT: Extend these tests once `NumBlocksPerSession` is configurable.
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

			expectedSessionId1: "c8a3d0135e2d0ed3a2d579adcd3b91708e59adf558b7c8bd474473de669cb18a",
			expectedSessionId2: "20650c94d0bcc4b855654b5533bf880345ae96933c5fa7424ce59c698d208e22",
		},
		{
			desc: "app1: sessionId for svc1 != sessionId for svc12",

			blockHeight1: 5,
			blockHeight2: 5,

			appAddr1: keepertest.TestApp1Address, // app1
			appAddr2: keepertest.TestApp1Address, // app1

			serviceId1: keepertest.TestServiceId1,  // svc1
			serviceId2: keepertest.TestServiceId12, // svc12

			expectedSessionId1: "c8a3d0135e2d0ed3a2d579adcd3b91708e59adf558b7c8bd474473de669cb18a",
			expectedSessionId2: "bf37fcbc62fe728f356384e9a765584f1c9d761566b2d46c0f77297675d966c6",
		},
		{
			desc: "svc12: sessionId for app1 != sessionId for app2",

			blockHeight1: 5,
			blockHeight2: 5,

			appAddr1: keepertest.TestApp1Address, // app1
			appAddr2: keepertest.TestApp2Address, // app2

			serviceId1: keepertest.TestServiceId12, // svc12
			serviceId2: keepertest.TestServiceId12, // svc12

			expectedSessionId1: "bf37fcbc62fe728f356384e9a765584f1c9d761566b2d46c0f77297675d966c6",
			expectedSessionId2: "8806550a46a16fcb6bc0a4bd0803081a2478d25868057bcf86204c8aaefb28ac",
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

// TODO_TECHDEBT: Expand these tests to account for application joining/leaving the network at different heights as well changing the services they support
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
		// TODO_TECHDEBT: Add tests for when:
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

// TODO_TECHDEBT: Expand these tests to account for supplier joining/leaving the network at different heights as well changing the services they support
func TestSession_HydrateSession_Suppliers(t *testing.T) {
	// TODO_TECHDEBT: Extend these tests once `NumBlocksPerSession` is configurable.
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
		// TODO_TECHDEBT: Add this test once we make the num suppliers per session configurable
		// {
		// 	name: "num_suppliers_available > num_suppliers_per_session_param",
		// },
		// TODO_TECHDEBT: Add tests for when:
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
