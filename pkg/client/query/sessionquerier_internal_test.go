package query

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/pokt-network/poktroll/pkg/cache/memory"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// TestGetSessionCacheKey_GridChangeAliasesDistinctSessions pins the reason GetSession
// resolves shared params at-height for block heights below the live grid anchor.
//
// The cache key is appAddress/serviceId/sessionStartHeight. Deriving that start height
// with params from the WRONG grid does not merely cost a cache miss -- it can map two
// heights belonging to two different real sessions onto one key, at which point
// GetSession hands back a session that does not match the request. Downstream that shows
// up as a session ID mismatch and rejects relays the chain would have paid for.
func TestGetSessionCacheKey_GridChangeAliasesDistinctSessions(t *testing.T) {
	const (
		appAddress = "pokt1testapp"
		serviceId  = "svc1"
	)

	oldGrid := &sharedtypes.Params{
		NumBlocksPerSession:     10,
		SessionGridAnchorHeight: 1,
		SessionNumberAtAnchor:   1,
	}
	liveGridAfterChange := &sharedtypes.Params{
		NumBlocksPerSession:     60,
		SessionGridAnchorHeight: 1000,
		SessionNumberAtAnchor:   100,
	}

	const heightInFirstSession = int64(20)
	const heightInSecondSession = int64(35)

	// Sanity: under the grid that actually governs them, these are different sessions and
	// must never share a cache key.
	require.NotEqual(t,
		getSessionCacheKey(oldGrid, appAddress, serviceId, heightInFirstSession),
		getSessionCacheKey(oldGrid, appAddress, serviceId, heightInSecondSession),
		"test setup: the two heights must belong to different sessions under the old grid")

	// Both sit BELOW the live anchor -- the condition GetSession detects.
	require.Less(t, heightInFirstSession, int64(liveGridAfterChange.GetSessionGridAnchorHeight()))
	require.Less(t, heightInSecondSession, int64(liveGridAfterChange.GetSessionGridAnchorHeight()))

	// Deriving from LIVE params collapses them onto one key: the aliasing bug itself.
	require.Equal(t,
		getSessionCacheKey(liveGridAfterChange, appAddress, serviceId, heightInFirstSession),
		getSessionCacheKey(liveGridAfterChange, appAddress, serviceId, heightInSecondSession),
		"live params from a re-anchored grid are expected to alias these two sessions; "+
			"this is the collision GetSession's at-height fallback exists to avoid")
}

// TestGetSessionCacheKey_StableWithinASession guards the property the cache depends on:
// every height inside one session derives the same key, so the session is fetched once
// rather than once per block.
func TestGetSessionCacheKey_StableWithinASession(t *testing.T) {
	params := &sharedtypes.Params{
		NumBlocksPerSession:     10,
		SessionGridAnchorHeight: 1,
		SessionNumberAtAnchor:   1,
	}

	firstKey := getSessionCacheKey(params, "pokt1testapp", "svc1", 21)
	for height := int64(22); height <= 30; height++ {
		require.Equal(t, firstKey, getSessionCacheKey(params, "pokt1testapp", "svc1", height),
			"height %d is in the same session as 21 and must share its cache key", height)
	}
}

// heightEchoSessionQueryClient returns a session whose SessionId encodes the requested
// height, so a caller can tell WHICH session came back rather than merely that one did.
type heightEchoSessionQueryClient struct {
	sessiontypes.QueryClient
}

func (c *heightEchoSessionQueryClient) GetSession(
	_ context.Context,
	req *sessiontypes.QueryGetSessionRequest,
	_ ...grpc.CallOption,
) (*sessiontypes.QueryGetSessionResponse, error) {
	return &sessiontypes.QueryGetSessionResponse{
		Session: &sessiontypes.Session{
			SessionId: fmt.Sprintf("session-for-height-%d", req.BlockHeight),
			Header:    &sessiontypes.SessionHeader{SessionId: fmt.Sprintf("session-for-height-%d", req.BlockHeight)},
		},
	}, nil
}

// TestGetSession_BelowAnchorDoesNotAliasSessions is the REGRESSION test for the fix
// itself, exercising GetSession rather than the key helper in isolation.
//
// The sibling helper tests above pin only the PREMISE (that live params can alias two
// grids). Deleting the at-height guard in GetSession leaves those green, because they
// never call it. This one goes through the real path: with the guard removed, both
// heights derive the same key from live params and the SECOND call returns the FIRST
// call's cached session.
func TestGetSession_BelowAnchorDoesNotAliasSessions(t *testing.T) {
	const (
		appAddress = "pokt1testapp"
		serviceId  = "svc1"
	)

	// Live params describe a re-anchored 60-block grid; the queried heights predate it.
	// The at-height answer is the real 10-block grid those heights belong to.
	fake := &fakeSharedQueryClient{
		liveParams: sharedtypes.Params{
			NumBlocksPerSession:     60,
			SessionGridAnchorHeight: 1000,
			SessionNumberAtAnchor:   100,
		},
		historicalParam: sharedtypes.Params{
			NumBlocksPerSession:     10,
			SessionGridAnchorHeight: 1,
			SessionNumberAtAnchor:   1,
		},
	}

	sessionsCache, err := memory.NewKeyValueCache[*sessiontypes.Session]()
	require.NoError(t, err)

	sessq := &sessionQuerier{
		logger:            polyzero.NewLogger(),
		sessionQuerier:    &heightEchoSessionQueryClient{},
		sharedQueryClient: newTestSharedQuerier(fake),
		sessionsCache:     sessionsCache,
	}

	// Heights 20 and 35 are in DIFFERENT sessions on the 10-block grid (starts 20 and 30),
	// but collapse to one key under the live 60-block grid.
	first, err := sessq.GetSession(context.Background(), appAddress, serviceId, 20)
	require.NoError(t, err)
	second, err := sessq.GetSession(context.Background(), appAddress, serviceId, 35)
	require.NoError(t, err)

	require.Equal(t, "session-for-height-20", first.SessionId)
	require.Equal(t, "session-for-height-35", second.SessionId,
		"height 35 returned the session cached for height 20: the live-params cache key "+
			"aliased two different sessions, which is what the at-height guard prevents")

	require.Positive(t, fake.numParamsAtHeightCalls,
		"heights below the live grid anchor must resolve shared params at-height")
}

// TestGetSession_AtOrAboveAnchorUsesLiveParams guards the other half: the common path must
// NOT query at-height. GetSession is called with the CURRENT block height on the relay hot
// path, so querying per height would add a memo entry per block and trip the at-height
// memo's drop-everything eviction, degrading every other at-height caller.
func TestGetSession_AtOrAboveAnchorUsesLiveParams(t *testing.T) {
	fake := &fakeSharedQueryClient{
		liveParams: sharedtypes.Params{
			NumBlocksPerSession:     10,
			SessionGridAnchorHeight: 1,
			SessionNumberAtAnchor:   1,
		},
		historicalParam: sharedtypes.Params{
			NumBlocksPerSession:     60,
			SessionGridAnchorHeight: 1,
			SessionNumberAtAnchor:   1,
		},
	}

	sessionsCache, err := memory.NewKeyValueCache[*sessiontypes.Session]()
	require.NoError(t, err)

	sessq := &sessionQuerier{
		logger:            polyzero.NewLogger(),
		sessionQuerier:    &heightEchoSessionQueryClient{},
		sharedQueryClient: newTestSharedQuerier(fake),
		sessionsCache:     sessionsCache,
	}

	_, err = sessq.GetSession(context.Background(), "pokt1testapp", "svc1", 500)
	require.NoError(t, err)

	require.Zero(t, fake.numParamsAtHeightCalls,
		"a height at or above the live grid anchor must use live params, not an at-height query")
}
