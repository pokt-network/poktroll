package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "pocket/testutil/keeper"
	"pocket/testutil/sample"
	"pocket/x/session/keeper"
	"pocket/x/session/types"
)

func TestSession_HydrateSession_Success_BaseCase(t *testing.T) {
	sessionKeeper, ctx := keepertest.SessionKeeper(t)
	blockHeight := int64(10)

	sessionHydrator := keeper.NewSessionHydrator(keepertest.TestApp1Address, keepertest.TestServiceId1, blockHeight)
	session, err := sessionKeeper.HydrateSession(ctx, sessionHydrator)
	require.NoError(t, err)

	// Check the header
	sessionHeader := session.Header
	require.Equal(t, keepertest.TestApp1Address, sessionHeader.ApplicationAddress)
	require.Equal(t, keepertest.TestServiceId1, sessionHeader.ServiceId.Id)
	require.Equal(t, "", sessionHeader.ServiceId.Name)
	require.Equal(t, int64(8), sessionHeader.SessionStartBlockHeight)
	require.Equal(t, "46b797bf320f813a2c420e2e6bf48cf60cef6f31c1d54d5669f8de805fbb7a29", sessionHeader.SessionId)

	// Check the session
	require.Equal(t, int64(4), session.NumBlocksPerSession)
	require.Equal(t, "46b797bf320f813a2c420e2e6bf48cf60cef6f31c1d54d5669f8de805fbb7a29", session.SessionId)
	require.Equal(t, int64(2), session.SessionNumber)

	// Check the application
	app := session.Application
	require.Equal(t, keepertest.TestApp1Address, app.Address)
	require.Len(t, app.ServiceIds, 2)

	// Check the suppliers
	suppliers := session.Suppliers
	require.Len(t, suppliers, 1)
	supplier := suppliers[0]
	require.Equal(t, keepertest.TestSupplierAddress, supplier.Address)
	require.Len(t, supplier.Services, 2)
}

func TestSession_HydrateSession_Metadata(t *testing.T) {
	type test struct {
		name        string
		blockHeight int64

		expectedNumBlocksPerSession int64
		expectedSessionNumber       int64
		expectedSessionStartBlock   int64
	}

	// TODO_TECHDEBT: Extend these tests once `NumBlocksPerSession` is configurable.
	// Currently assumes NumBlocksPerSession=4
	tests := []test{
		{
			name:        "blockHeight = 0",
			blockHeight: 0,

			expectedNumBlocksPerSession: 4,
			expectedSessionNumber:       0,
			expectedSessionStartBlock:   0,
		},
		{
			name:        "blockHeight = 1",
			blockHeight: 1,

			expectedNumBlocksPerSession: 4,
			expectedSessionNumber:       0,
			expectedSessionStartBlock:   0,
		},
		{
			name:        "blockHeight = sessionHeight",
			blockHeight: 4,

			expectedNumBlocksPerSession: 4,
			expectedSessionNumber:       1,
			expectedSessionStartBlock:   4,
		},
		{
			name:        "blockHeight != sessionHeight",
			blockHeight: 5,

			expectedNumBlocksPerSession: 4,
			expectedSessionNumber:       1,
			expectedSessionStartBlock:   4,
		},
	}

	appAddr := keepertest.TestApp1Address
	serviceId := keepertest.TestServiceId1
	sessionKeeper, ctx := keepertest.SessionKeeper(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionHydrator := keeper.NewSessionHydrator(appAddr, serviceId, tt.blockHeight)
			session, err := sessionKeeper.HydrateSession(ctx, sessionHydrator)
			require.NoError(t, err)

			require.Equal(t, tt.expectedNumBlocksPerSession, session.NumBlocksPerSession)
			require.Equal(t, tt.expectedSessionNumber, session.SessionNumber)
			require.Equal(t, tt.expectedSessionStartBlock, session.Header.SessionStartBlockHeight)
		})
	}
}

func TestSession_HydrateSession_SessionId(t *testing.T) {
	type test struct {
		name string

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
			name: "(app1, svc1): sessionId at first session block != sessionId at next session block",

			blockHeight1: 4,
			blockHeight2: 8,

			appAddr1: keepertest.TestApp1Address, // app1
			appAddr2: keepertest.TestApp1Address, // app1

			serviceId1: keepertest.TestServiceId1, // svc1
			serviceId2: keepertest.TestServiceId1, // svc1

			expectedSessionId1: "dfd35b35f37207f2a65fd03e5a78b4f637362cf30673c8c197b93fd2cceef727",
			expectedSessionId2: "46b797bf320f813a2c420e2e6bf48cf60cef6f31c1d54d5669f8de805fbb7a29",
		},
		{
			name: "app1: sessionId for svc1 != sessionId for svc2",

			blockHeight1: 4,
			blockHeight2: 4,

			appAddr1: keepertest.TestApp1Address, // app1
			appAddr2: keepertest.TestApp1Address, // app1

			serviceId1: keepertest.TestServiceId1, // svc1
			serviceId2: keepertest.TestServiceId2, // svc2

			expectedSessionId1: "dfd35b35f37207f2a65fd03e5a78b4f637362cf30673c8c197b93fd2cceef727",
			expectedSessionId2: "14346dd61c7a6d3831d17c6d3d134f61af054857373b7080f92b91ae7de505b1",
		},
		{
			name: "svc1: sessionId for app1 != sessionId for app2",

			blockHeight1: 4,
			blockHeight2: 4,

			appAddr1: keepertest.TestApp1Address, // app1
			appAddr2: keepertest.TestApp2Address, // app2

			serviceId1: keepertest.TestServiceId1, // svc1
			serviceId2: keepertest.TestServiceId1, // svc1

			expectedSessionId1: "dfd35b35f37207f2a65fd03e5a78b4f637362cf30673c8c197b93fd2cceef727",
			expectedSessionId2: "af38ae7df6bbae4669e0d12471d32fe8f5343546171675b5e3191421ba9f1df2",
		},
	}

	sessionKeeper, ctx := keepertest.SessionKeeper(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
		name    string
		appAddr string

		expectedErr error
	}

	tests := []test{
		{
			name:    "app is found",
			appAddr: keepertest.TestApp1Address,

			expectedErr: nil,
		},
		{
			name:    "app is not found",
			appAddr: sample.AccAddress(), // Generating a random address on the fly

			expectedErr: types.ErrHydratingSession,
		},
		{
			name:    "invalid app address",
			appAddr: "invalid",

			expectedErr: types.ErrHydratingSession,
		},
	}

	serviceId := keepertest.TestServiceId1
	blockHeight := int64(10)
	sessionKeeper, ctx := keepertest.SessionKeeper(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionHydrator := keeper.NewSessionHydrator(tt.appAddr, serviceId, blockHeight)
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
		name      string
		appAddr   string
		serviceId string

		numExpectedSuppliers int
	}

	// TODO_TECHDEBT: Extend these tests once `NumBlocksPerSession` is configurable.
	// Currently assumes NumSupplierPerSession=15
	tests := []test{
		{
			name:      "no suppliers available",
			appAddr:   keepertest.TestApp1Address, // app1
			serviceId: "svc_unknown",

			numExpectedSuppliers: 0,
		},
		{
			name:      "num suppliers available is less than the num suppliers per session",
			appAddr:   keepertest.TestApp1Address, // app1
			serviceId: keepertest.TestServiceId1,  // svc1

			numExpectedSuppliers: 1,
		},
		// TODO_TECHDEBT: Add this test once we make the num suppliers per session configurable
		// {
		// 	name: "num suppliers available is greater than num suppliers per session",
		// },
	}

	blockHeight := int64(10)
	sessionKeeper, ctx := keepertest.SessionKeeper(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {})

		sessionHydrator := keeper.NewSessionHydrator(tt.appAddr, tt.serviceId, blockHeight)
		session, err := sessionKeeper.HydrateSession(ctx, sessionHydrator)
		require.NoError(t, err)

		require.Len(t, session.Suppliers, tt.numExpectedSuppliers)
	}
}
