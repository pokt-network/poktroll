package query

import (
	"testing"

	"github.com/stretchr/testify/require"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// TestGetSessionCacheKey_GridChangeAliasesDistinctSessions pins the reason GetSession
// resolves shared params at-height for block heights below the live grid anchor.
//
// The cache key is appAddress/serviceId/sessionStartHeight. Deriving that start height
// with params from the WRONG grid does not merely cost a cache miss -- it can map two
// heights that belong to two different real sessions onto one key, at which point
// GetSession hands back a session that does not match the request. Downstream that shows
// up as a session ID mismatch and rejects relays the chain would have paid for.
func TestGetSessionCacheKey_GridChangeAliasesDistinctSessions(t *testing.T) {
	const (
		appAddress = "pokt1testapp"
		serviceId  = "svc1"
	)

	// The grid before the change: N=10 anchored at 1. Heights 20 and 35 fall in two
	// DIFFERENT sessions under it (starts 20 and 30 respectively).
	oldGrid := &sharedtypes.Params{
		NumBlocksPerSession:     10,
		SessionGridAnchorHeight: 1,
		SessionNumberAtAnchor:   1,
	}

	// The grid after a num_blocks_per_session change re-anchors well above those heights.
	// This is what LIVE params look like while a grace-period relay for an old session is
	// still being served.
	liveGridAfterChange := &sharedtypes.Params{
		NumBlocksPerSession:     60,
		SessionGridAnchorHeight: 1000,
		SessionNumberAtAnchor:   100,
	}

	const heightInFirstSession = int64(20)
	const heightInSecondSession = int64(35)

	// Sanity: under the grid that actually governs them, the two heights are in
	// different sessions and therefore must never share a cache key.
	oldKeyFirst := getSessionCacheKey(oldGrid, appAddress, serviceId, heightInFirstSession)
	oldKeySecond := getSessionCacheKey(oldGrid, appAddress, serviceId, heightInSecondSession)
	require.NotEqual(t, oldKeyFirst, oldKeySecond,
		"test setup: the two heights must belong to different sessions under the old grid")

	// Both heights sit BELOW the live anchor, which is exactly the condition GetSession
	// now detects and resolves at-height instead of trusting live params for.
	require.Less(t, heightInFirstSession, int64(liveGridAfterChange.GetSessionGridAnchorHeight()))
	require.Less(t, heightInSecondSession, int64(liveGridAfterChange.GetSessionGridAnchorHeight()))

	// Deriving the key from LIVE params collapses them onto one key: the aliasing bug.
	// If this ever stops holding, the guard in GetSession has become unnecessary --
	// verify that deliberately rather than deleting the guard on a green test.
	liveKeyFirst := getSessionCacheKey(liveGridAfterChange, appAddress, serviceId, heightInFirstSession)
	liveKeySecond := getSessionCacheKey(liveGridAfterChange, appAddress, serviceId, heightInSecondSession)
	require.Equal(t, liveKeyFirst, liveKeySecond,
		"live params from a re-anchored grid are expected to alias these two sessions; "+
			"this is the collision GetSession's at-height fallback exists to avoid")
}

// TestGetSessionCacheKey_StableWithinASession guards the property the cache depends on:
// every height inside one session must derive the same key, so the session is fetched
// once rather than once per block.
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
