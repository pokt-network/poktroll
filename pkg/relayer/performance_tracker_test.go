package relayer

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// tiny helper: monotonic > 0
func gt0(d time.Duration) bool { return d > 0 }

// ----------------------------------------------------------------------------
// Tests
// ----------------------------------------------------------------------------

func TestImmediateObserve_StartFinish(t *testing.T) {
	var (
		count    atomic.Int64
		lastName atomic.Value
		lastSID  atomic.Value
	)
	obs := func(tr *PerfTracker, name string, dur time.Duration) {
		count.Add(1)
		lastName.Store(name)
		lastSID.Store(tr.ServiceID())
		if !gt0(dur) {
			t.Errorf("duration must be > 0")
		}
	}

	ctx := WithPerfForService(context.Background(), "svcA", PerfOptions{
		RecordOnFinish: true,
		Observe:        obs,
	})
	tr, _ := FromCtx(ctx)

	tr.Start("stepA")
	time.Sleep(100 * time.Microsecond)
	tr.Finish("stepA")

	if count.Load() != 1 {
		t.Fatalf("expected 1 observation, got %d", count.Load())
	}
	if got := lastName.Load().(string); got != "stepA" {
		t.Fatalf("expected name stepA, got %q", got)
	}
	if got := lastSID.Load().(string); got != "svcA" {
		t.Fatalf("expected service_id svcA, got %q", got)
	}
}

func TestBufferedObserve_Flush_AdoptsServiceIDUpgrade(t *testing.T) {
	var (
		count atomic.Int64
		names = make(chan string, 4)
		sids  = make(chan string, 4)
	)
	obs := func(tr *PerfTracker, name string, dur time.Duration) {
		count.Add(1)
		names <- name
		sids <- tr.ServiceID()
	}

	ctx := WithPerfForService(context.Background(), "unknown", PerfOptions{
		RecordOnFinish: false,
		BufferCap:      64,
		Observe:        obs,
	})
	tr, _ := FromCtx(ctx)

	// Record a span BEFORE we know service_id.
	tr.Start("before")
	time.Sleep(50 * time.Microsecond)
	tr.Finish("before")

	// Upgrade the service id.
	tr.SetServiceID("svcB")

	// Record a span AFTER upgrade.
	tr.Start("after")
	time.Sleep(50 * time.Microsecond)
	tr.Finish("after")

	// Nothing observed yet (buffered mode).
	if count.Load() != 0 {
		t.Fatalf("expected 0 observations before Flush, got %d", count.Load())
	}

	// Flush and verify both are emitted with upgraded service_id.
	tr.Flush()

	if count.Load() != 2 {
		t.Fatalf("expected 2 observations after Flush, got %d", count.Load())
	}
	close(names)
	close(sids)

	var gotNames []string
	var gotSIDs []string
	for n := range names {
		gotNames = append(gotNames, n)
	}
	for s := range sids {
		gotSIDs = append(gotSIDs, s)
	}

	if len(gotNames) != 2 || len(gotSIDs) != 2 {
		t.Fatalf("expected 2 names and 2 sids, got %d and %d", len(gotNames), len(gotSIDs))
	}
	for i := range gotSIDs {
		if gotSIDs[i] != "svcB" {
			t.Fatalf("expected service_id svcB at index %d, got %q", i, gotSIDs[i])
		}
	}
}

func TestServiceIDUpgrade_ImmediateOnlyAffectsFutureSpans(t *testing.T) {
	var seen []string
	obs := func(tr *PerfTracker, name string, _ time.Duration) {
		seen = append(seen, tr.ServiceID()+":"+name)
	}

	ctx := WithPerfForService(context.Background(), "old", PerfOptions{
		RecordOnFinish: true,
		Observe:        obs,
	})
	tr, _ := FromCtx(ctx)

	tr.Start("a")
	tr.Finish("a") // emits with "old"
	tr.SetServiceID("new")
	tr.Start("b")
	tr.Finish("b") // emits with "new"

	if len(seen) != 2 {
		t.Fatalf("expected 2 observations, got %d", len(seen))
	}
	if seen[0] != "old:a" || seen[1] != "new:b" {
		t.Fatalf("unexpected sequence: %#v", seen)
	}
}

func TestNestedSpans_StackAndDeletion(t *testing.T) {
	var count atomic.Int64
	obs := func(tr *PerfTracker, _ string, dur time.Duration) {
		if !gt0(dur) {
			t.Errorf("duration must be > 0")
		}
		count.Add(1)
	}
	ctx := WithPerfForService(context.Background(), "svc", PerfOptions{
		RecordOnFinish: true,
		Observe:        obs,
	})
	tr, _ := FromCtx(ctx)

	tr.Start("N")
	time.Sleep(10 * time.Microsecond)
	tr.Start("N")
	time.Sleep(10 * time.Microsecond)
	tr.Finish("N")
	tr.Finish("N")

	// Unmatched finish should be ignored
	tr.Finish("N")

	if count.Load() != 2 {
		t.Fatalf("expected 2 observations, got %d", count.Load())
	}
}

func TestSpanHelper_Defer(t *testing.T) {
	var count atomic.Int64
	obs := func(tr *PerfTracker, name string, _ time.Duration) {
		if name != "X" {
			t.Fatalf("expected name X, got %q", name)
		}
		count.Add(1)
	}

	ctx := WithPerfForService(context.Background(), "svc", PerfOptions{
		RecordOnFinish: true,
		Observe:        obs,
	})
	func() {
		defer Span(ctx, "X")()
		time.Sleep(20 * time.Microsecond)
	}()

	if count.Load() != 1 {
		t.Fatalf("expected 1 observation, got %d", count.Load())
	}
}

func TestUnmatchedFinishIgnored(t *testing.T) {
	var count atomic.Int64
	obs := func(tr *PerfTracker, _ string, _ time.Duration) { count.Add(1) }

	ctx := WithPerfForService(context.Background(), "svc", PerfOptions{
		RecordOnFinish: true,
		Observe:        obs,
	})
	tr, _ := FromCtx(ctx)

	tr.Finish("no-start")
	if count.Load() != 0 {
		t.Fatalf("expected 0 observations, got %d", count.Load())
	}
}

func TestQueueFullFallback_DirectObserveOnSaturation(t *testing.T) {
	var count atomic.Int64
	obs := func(_ *PerfTracker, _ string, _ time.Duration) {
		count.Add(1)
	}

	ctx := WithPerfForService(context.Background(), "svc", PerfOptions{
		RecordOnFinish: false,
		BufferCap:      1, // saturate quickly
		Observe:        obs,
	})
	tr, _ := FromCtx(ctx)

	// First finish enqueues (no immediate observe).
	tr.Start("A")
	tr.Finish("A")
	if count.Load() != 0 {
		t.Fatalf("expected 0 immediately after first Finish (buffered), got %d", count.Load())
	}

	// Second finish can't enqueue -> falls back to direct observe.
	tr.Start("B")
	tr.Finish("B")
	if count.Load() != 1 {
		t.Fatalf("expected 1 before Flush due to fallback, got %d", count.Load())
	}

	// Flush drains the remaining one.
	tr.Flush()
	if count.Load() != 2 {
		t.Fatalf("expected 2 after Flush, got %d", count.Load())
	}
}

func TestEnsurePerfBuffered_ReusesExisting_NoOpFlush(t *testing.T) {
	var count atomic.Int64
	obs := func(_ *PerfTracker, _ string, _ time.Duration) { count.Add(1) }

	// Caller attaches once.
	ctx0 := WithPerfForService(context.Background(), "svc", PerfOptions{
		RecordOnFinish: false,
		Observe:        obs,
	})

	// First ensure (should reuse)
	ctx1, tr1, flush1 := EnsurePerfBuffered(ctx0, "svc")
	// Second ensure (should also reuse the exact same tracker)
	ctx2, tr2, flush2 := EnsurePerfBuffered(ctx1, "svc")
	if tr1 != tr2 {
		t.Fatalf("expected tracker reuse")
	}
	_ = ctx2

	// Record two buffered spans.
	tr1.Start("a")
	tr1.Finish("a")
	tr2.Start("b")
	tr2.Finish("b")

	// Both flushes should be safe; since they reuse the same tracker,
	// calling both should not double-count (queue empties once).
	flush1()
	if count.Load() != 2 {
		t.Fatalf("expected 2 after first flush, got %d", count.Load())
	}
	flush2()
	if count.Load() != 2 {
		t.Fatalf("expected still 2 after second flush, got %d", count.Load())
	}
}

// ----------------------------------------------------------------------------
// Benchmarks
// ----------------------------------------------------------------------------

func BenchmarkPerf_Instance_Buffered(b *testing.B) {
	ctx := WithPerfForService(context.Background(), "svc", PerfOptions{
		RecordOnFinish: false,
		BufferCap:      2_048,
	})
	tr, _ := FromCtx(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.Start("x")
		tr.Finish("x")
	}
	b.StopTimer()
	tr.Flush()
}

func BenchmarkPerf_Span_Buffered(b *testing.B) {
	ctx := WithPerfForService(context.Background(), "svc", PerfOptions{
		RecordOnFinish: false,
		BufferCap:      2_048,
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		func() { defer Span(ctx, "x")() }()
	}
	b.StopTimer()
	Flush(ctx)
}

func BenchmarkPerf_Pkg_Buffered(b *testing.B) {
	ctx := WithPerfForService(context.Background(), "svc", PerfOptions{
		RecordOnFinish: false,
		BufferCap:      2_048,
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Start(ctx, "x")
		Finish(ctx, "x")
	}
	b.StopTimer()
	Flush(ctx)
}

func BenchmarkPerf_Instance_Immediate(b *testing.B) {
	ctx := WithPerfForService(context.Background(), "svc", PerfOptions{
		RecordOnFinish: true,
		Observe:        func(_ *PerfTracker, _ string, _ time.Duration) {},
	})
	tr, _ := FromCtx(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.Start("x")
		tr.Finish("x")
	}
}

func BenchmarkPerf_Span_Immediate(b *testing.B) {
	ctx := WithPerfForService(context.Background(), "svc", PerfOptions{
		RecordOnFinish: true,
		Observe:        func(_ *PerfTracker, _ string, _ time.Duration) {},
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		func() { defer Span(ctx, "x")() }()
	}
}

func BenchmarkPerf_Pkg_Immediate(b *testing.B) {
	ctx := WithPerfForService(context.Background(), "svc", PerfOptions{
		RecordOnFinish: true,
		Observe:        func(_ *PerfTracker, _ string, _ time.Duration) {},
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Start(ctx, "x")
		Finish(ctx, "x")
	}
}

// Optional: measure SetServiceID overhead (atomic store)
func BenchmarkPerf_SetServiceID(b *testing.B) {
	ctx := WithPerfForService(context.Background(), "svc", PerfOptions{
		RecordOnFinish: false,
	})
	tr, _ := FromCtx(ctx)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.SetServiceID("svc2")
	}
}
