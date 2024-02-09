package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/session/keeper"
	"github.com/pokt-network/poktroll/x/session/types"
)

// TODO_TECHDEBT(#377): All the tests in this file assume genesis has a block
// height of 0. Rewrite them in terms of height = 1 genesis.

func TestSession_HydrateSession_Success_BaseCase(t *testing.T) {
	sessionKeeper, ctx := keepertest.SessionKeeper(t)
	ctx = ctx.WithBlockHeight(100) // provide a sufficiently large block height to avoid errors
	blockHeight := int64(10)

	sessionHydrator := keeper.NewSessionHydrator(keepertest.TestApp1Address, keepertest.TestServiceId1, blockHeight)
	session, err := sessionKeeper.HydrateSession(ctx, sessionHydrator)
	require.NoError(t, err)

	// Check the header
	sessionHeader := session.Header
	require.Equal(t, keepertest.TestApp1Address, sessionHeader.ApplicationAddress)
	require.Equal(t, keepertest.TestServiceId1, sessionHeader.Service.Id)
	require.Equal(t, "", sessionHeader.Service.Name)
	require.Equal(t, int64(8), sessionHeader.SessionStartBlockHeight)
	require.Equal(t, int64(11), sessionHeader.SessionEndBlockHeight)
	require.Equal(t, "f621ad93e2d9ebf2e967eb16b8ea2bc470baa1d4e13a82f348f34c6fc93eae7f", sessionHeader.SessionId)

	// Check the session
	require.Equal(t, int64(4), session.NumBlocksPerSession)
	require.Equal(t, "f621ad93e2d9ebf2e967eb16b8ea2bc470baa1d4e13a82f348f34c6fc93eae7f", session.SessionId)
	require.Equal(t, int64(2), session.SessionNumber)

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
	type test struct {
		desc        string
		blockHeight int64

		expectedNumBlocksPerSession int64
		expectedSessionNumber       int64
		expectedSessionStartBlock   int64
		expectedSessionEndBlock     int64
		errExpected                 error
	}

	// TODO_TECHDEBT: Extend these tests once `NumBlocksPerSession` is configurable.
	// Currently assumes NumBlocksPerSession=4
	tests := []test{
		{
			desc:        "blockHeight = 0",
			blockHeight: 0,

			expectedNumBlocksPerSession: 4,
			expectedSessionNumber:       0,
			expectedSessionStartBlock:   0,
			expectedSessionEndBlock:     3,
			errExpected:                 nil,
		},
		{
			desc:        "blockHeight = 1",
			blockHeight: 1,

			expectedNumBlocksPerSession: 4,
			expectedSessionNumber:       0,
			expectedSessionStartBlock:   0,
			expectedSessionEndBlock:     3,
			errExpected:                 nil,
		},
		{
			desc:        "blockHeight = sessionHeight",
			blockHeight: 4,

			expectedNumBlocksPerSession: 4,
			expectedSessionNumber:       1,
			expectedSessionStartBlock:   4,
			expectedSessionEndBlock:     7,
			errExpected:                 nil,
		},
		{
			desc:        "blockHeight != sessionHeight",
			blockHeight: 5,

			expectedNumBlocksPerSession: 4,
			expectedSessionNumber:       1,
			expectedSessionStartBlock:   4,
			expectedSessionEndBlock:     7,
			errExpected:                 nil,
		},
		{
			desc:        "blockHeight > contextHeight",
			blockHeight: 9001, // block height over 9000 is too high given that the context height is 100

			errExpected: types.ErrSessionHydration,
		},
	}

	appAddr := keepertest.TestApp1Address
	serviceId := keepertest.TestServiceId1
	sessionKeeper, ctx := keepertest.SessionKeeper(t)
	ctx = ctx.WithBlockHeight(100) // provide a sufficiently large block height to avoid errors

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			sessionHydrator := keeper.NewSessionHydrator(appAddr, serviceId, tt.blockHeight)
			session, err := sessionKeeper.HydrateSession(ctx, sessionHydrator)

			if tt.errExpected != nil {
				require.ErrorIs(t, tt.errExpected, err)
				return
			}
			require.NoError(t, err)

			require.Equal(t, tt.expectedNumBlocksPerSession, session.NumBlocksPerSession)
			require.Equal(t, tt.expectedSessionNumber, session.SessionNumber)
			require.Equal(t, tt.expectedSessionStartBlock, session.Header.SessionStartBlockHeight)
			require.Equal(t, tt.expectedSessionEndBlock, session.Header.SessionEndBlockHeight)
		})
	}
}

func TestSession_HydrateSession_SessionId(t *testing.T) {
	type test struct {
		desc string

		blockHeight1 int64
		blockHeight2 int64

		appAddr1 string
		appAddr2 string

		serviceId1 string
		serviceId2 string

		expectedSessionId1 string
		expectedSessionId2 string
	}

	// TODO_TECHDEBT: Extend these tests once `NumBlocksPerSession` is configurable.
	// Currently assumes NumBlocksPerSession=4
	tests := []test{
		{
			desc: "(app1, svc1): sessionId at first session block != sessionId at next session block",

			blockHeight1: 4,
			blockHeight2: 8,

			appAddr1: keepertest.TestApp1Address, // app1
			appAddr2: keepertest.TestApp1Address, // app1

			serviceId1: keepertest.TestServiceId1, // svc1
			serviceId2: keepertest.TestServiceId1, // svc1

			expectedSessionId1: "8d7919d1addb175425ed99bc897aee7fecd6a184e46a0ce4119b610db9242db5",
			expectedSessionId2: "f621ad93e2d9ebf2e967eb16b8ea2bc470baa1d4e13a82f348f34c6fc93eae7f",
		},
		{
			desc: "app1: sessionId for svc1 != sessionId for svc12",

			blockHeight1: 4,
			blockHeight2: 4,

			appAddr1: keepertest.TestApp1Address, // app1
			appAddr2: keepertest.TestApp1Address, // app1

			serviceId1: keepertest.TestServiceId1,  // svc1
			serviceId2: keepertest.TestServiceId12, // svc12

			expectedSessionId1: "8d7919d1addb175425ed99bc897aee7fecd6a184e46a0ce4119b610db9242db5",
			expectedSessionId2: "40d02827f0f3f717caf844cd4cfbe1ebc2f3ada7d4e5974ed6ec795b9a715e07",
		},
		{
			desc: "svc12: sessionId for app1 != sessionId for app2",

			blockHeight1: 4,
			blockHeight2: 4,

			appAddr1: keepertest.TestApp1Address, // app1
			appAddr2: keepertest.TestApp2Address, // app2

			serviceId1: keepertest.TestServiceId12, // svc12
			serviceId2: keepertest.TestServiceId12, // svc12

			expectedSessionId1: "40d02827f0f3f717caf844cd4cfbe1ebc2f3ada7d4e5974ed6ec795b9a715e07",
			expectedSessionId2: "5c338ae8bf5138b7ca9a0ca124942984734531fb30f1670da772ef3e635939f6",
		},
	}

	sessionKeeper, ctx := keepertest.SessionKeeper(t)
	ctx = ctx.WithBlockHeight(100) // provide a sufficiently large block height to avoid errors

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			sessionHydrator1 := keeper.NewSessionHydrator(tt.appAddr1, tt.serviceId1, tt.blockHeight1)
			session1, err := sessionKeeper.HydrateSession(ctx, sessionHydrator1)
			require.NoError(t, err)

			sessionHydrator2 := keeper.NewSessionHydrator(tt.appAddr2, tt.serviceId2, tt.blockHeight2)
			session2, err := sessionKeeper.HydrateSession(ctx, sessionHydrator2)
			require.NoError(t, err)

			require.NotEqual(t, session1.Header.SessionId, session2.Header.SessionId)
			require.Equal(t, tt.expectedSessionId1, session1.Header.SessionId)
			require.Equal(t, tt.expectedSessionId2, session2.Header.SessionId)
		})
	}
}

// TODO_TECHDEBT: Expand these tests to account for application joining/leaving the network at different heights as well changing the services they support
func TestSession_HydrateSession_Application(t *testing.T) {
	type test struct {
		// Description
		desc string
		// Inputs
		appAddr   string
		serviceId string

		// Outputs
		expectedErr error
	}

	tests := []test{
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
	ctx = ctx.WithBlockHeight(100) // provide a sufficiently large block height to avoid errors

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			sessionHydrator := keeper.NewSessionHydrator(tt.appAddr, tt.serviceId, blockHeight)
			_, err := sessionKeeper.HydrateSession(ctx, sessionHydrator)
			if tt.expectedErr != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TODO_TECHDEBT: Expand these tests to account for supplier joining/leaving the network at different heights as well changing the services they support
func TestSession_HydrateSession_Suppliers(t *testing.T) {
	type test struct {
		// Description
		desc string

		// Inputs
		appAddr   string
		serviceId string

		// Outputs
		numExpectedSuppliers int
		expectedErr          error
	}

	// TODO_TECHDEBT: Extend these tests once `NumBlocksPerSession` is configurable.
	// Currently assumes NumSupplierPerSession=15
	tests := []test{
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
	ctx = ctx.WithBlockHeight(100) // provide a sufficiently large block height to avoid errors

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {})

		sessionHydrator := keeper.NewSessionHydrator(tt.appAddr, tt.serviceId, blockHeight)
		session, err := sessionKeeper.HydrateSession(ctx, sessionHydrator)

		if tt.expectedErr != nil {
			require.ErrorContains(t, err, tt.expectedErr.Error())
			continue
		}
		require.NoError(t, err)
		require.Len(t, session.Suppliers, tt.numExpectedSuppliers)
	}
}
