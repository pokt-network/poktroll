package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "pocket/testutil/keeper"
	"pocket/x/session/keeper"
)

func TestSession_HydrateSession_Success_BaseCase(t *testing.T) {
	sessionKeeper, ctx := keepertest.SessionKeeper(t)
	blockHeight := int64(1)

	sessionHydrator := keeper.NewSessionHydrator(keepertest.TestAppAddress, keepertest.TestServiceId, blockHeight)

	session, err := sessionKeeper.HydrateSession(ctx, sessionHydrator)
	require.NoError(t, err)

	// sessionHeader := session.SessionHeader

	require.Equal(t, keepertest.TestAppAddress, session.Application.Address)
	require.Equal(t, keepertest.TestAppAddress, session.Application.Address)
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
			name: "blockHeight = 0",
		},
		{
			name: "blockHeight = 1",
		},
		{
			name: "blockHeight = sessionHeight",
		},
		{
			name: "blockHeight != sessionHeight",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// sessionKeeper, ctx := keepertest.SessionKeeper(t)
			// blockHeight := int64(1)
		})
	}
}

func TestSession_HydrateSession_SessionId(t *testing.T) {
	type test struct {
		name        string
		blockHeight int64
		appAddress  string
		serviceId   string

		expectedSessionId string
	}

	// TODO_TECHDEBT: Extend these tests once `NumBlocksPerSession` is configurable.
	// Currently assumes NumBlocksPerSession=4
	tests := []test{
		{
			name: "(app1, svc1): sessionId at first session block != sessionId at next session block",
		},
		{
			name: "app1: sessionId for svc1 != sessionId for svc2",
		},
		{
			name: "svc1: sessionId for app1 != sessionId for app2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {})
	}
}

func TestSession_HydrateSession_Application(t *testing.T) {
	type test struct {
		name       string
		appAddress string

		expectedErr error
	}

	tests := []test{
		{
			name: "app is found",
		},
		{
			name: "app is not found",
		},
		{
			name: "invalid app address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {})
	}
}

func TestSession_HydrateSession_Suppliers(t *testing.T) {
	type test struct {
		name       string
		appAddress string
		serviceId  string

		expectedErr error
	}

	// TODO_TECHDEBT: Extend these tests once `NumBlocksPerSession` is configurable.
	// Currently assumes NumSupplierPerSession=15
	tests := []test{
		{
			name: "no suppliers available",
		},
		{
			name: "num suppliers available is less than the num suppliers per session",
		},
		{
			name: "num suppliers available is greater than num suppliers per session",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {})
	}
}
