package query

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/pkg/cache/memory"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const (
	testFallbackServiceID = "svc1"
	testLiveCUPR          = uint64(77)
)

// unimplementedCUPRQueryClient stands in for a full node predating v0.1.35: it serves
// Service but answers codes.Unimplemented for ComputeUnitsPerRelayAtHeight.
type unimplementedCUPRQueryClient struct {
	servicetypes.QueryClient
	serviceCalls int
	cuprCalls    int
}

func (c *unimplementedCUPRQueryClient) ComputeUnitsPerRelayAtHeight(
	_ context.Context,
	_ *servicetypes.QueryComputeUnitsPerRelayAtHeightRequest,
	_ ...grpc.CallOption,
) (*servicetypes.QueryComputeUnitsPerRelayAtHeightResponse, error) {
	c.cuprCalls++
	return nil, status.Error(codes.Unimplemented, "unknown method ComputeUnitsPerRelayAtHeight")
}

func (c *unimplementedCUPRQueryClient) Service(
	_ context.Context,
	_ *servicetypes.QueryGetServiceRequest,
	_ ...grpc.CallOption,
) (*servicetypes.QueryGetServiceResponse, error) {
	c.serviceCalls++
	return &servicetypes.QueryGetServiceResponse{
		Service: sharedtypes.Service{Id: testFallbackServiceID, ComputeUnitsPerRelay: testLiveCUPR},
	}, nil
}

func newFallbackTestQuerier(t *testing.T) (*serviceQuerier, *unimplementedCUPRQueryClient) {
	t.Helper()

	servicesCache, err := memory.NewKeyValueCache[sharedtypes.Service]()
	require.NoError(t, err)
	difficultyCache, err := memory.NewKeyValueCache[servicetypes.RelayMiningDifficulty]()
	require.NoError(t, err)
	cuprCache, err := memory.NewKeyValueCache[uint64]()
	require.NoError(t, err)

	stub := &unimplementedCUPRQueryClient{}
	return &serviceQuerier{
		logger:                     polyzero.NewLogger(),
		serviceQuerier:             stub,
		servicesCache:              servicesCache,
		relayMiningDifficultyCache: difficultyCache,
		computeUnitsPerRelayCache:  cuprCache,
	}, stub
}

// TestCUPRAtHeightFallback_ColdServiceCacheDoesNotDeadlock is a REGRESSION test for a
// self-deadlock.
//
// GetServiceComputeUnitsPerRelayAtHeight holds servicesMutex while querying. The
// pre-v0.1.35 fallback reads the live cupr through GetService, which takes that SAME
// non-reentrant mutex. Running the fallback inside the locked section made the goroutine
// block on itself forever WHILE HOLDING THE LOCK, so every other service, difficulty and
// cupr lookup blocked behind it too — a total hang of serving and mining, strictly worse
// than the outage the fallback exists to prevent.
//
// A warm services cache hides it (GetService returns before locking), so this test uses a
// COLD cache deliberately. It must be time-bounded: the failure mode is a hang, not an error.
func TestCUPRAtHeightFallback_ColdServiceCacheDoesNotDeadlock(t *testing.T) {
	servq, stub := newFallbackTestQuerier(t)

	type result struct {
		cupr uint64
		err  error
	}
	done := make(chan result, 1)
	go func() {
		cupr, err := servq.GetServiceComputeUnitsPerRelayAtHeight(context.Background(), testFallbackServiceID, 100)
		done <- result{cupr, err}
	}()

	select {
	case got := <-done:
		require.NoError(t, got.err)
		require.Equal(t, testLiveCUPR, got.cupr, "the fallback must return the LIVE compute units per relay")
	case <-time.After(20 * time.Second):
		t.Fatal("DEADLOCK: the pre-v0.1.35 cupr fallback called GetService while holding " +
			"servicesMutex; the fallback must run after that lock is released")
	}

	require.Equal(t, 1, stub.serviceCalls, "the fallback should read the live service exactly once")

	// The mutex must be genuinely free afterwards, not merely un-hung for that one call.
	freed := make(chan struct{})
	go func() {
		_, _ = servq.GetService(context.Background(), testFallbackServiceID)
		close(freed)
	}()
	select {
	case <-freed:
	case <-time.After(10 * time.Second):
		t.Fatal("servicesMutex was left held by the fallback path")
	}
}

// TestCUPRAtHeightFallback_LatchesUnsupported guards the second half of the fix: once the
// node has answered Unimplemented, later calls must NOT re-issue the doomed query. Without
// the latch every relay pays a failed gRPC round trip — under servicesMutex — plus a
// warning log, in exactly the configuration the fallback exists to keep usable.
func TestCUPRAtHeightFallback_LatchesUnsupported(t *testing.T) {
	servq, stub := newFallbackTestQuerier(t)

	for i := 0; i < 5; i++ {
		cupr, err := servq.GetServiceComputeUnitsPerRelayAtHeight(
			context.Background(), testFallbackServiceID, int64(100+i),
		)
		require.NoError(t, err)
		require.Equal(t, testLiveCUPR, cupr)
	}

	require.Equal(t, 1, stub.cuprCalls,
		"the at-height query must be attempted once then latched off; got %d attempts", stub.cuprCalls)
}
