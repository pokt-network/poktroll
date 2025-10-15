package relayer

import (
	"context"
	"strings"
	"sync/atomic"
	"time"

	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"github.com/puzpuzpuz/xsync/v4"
)

// Overview
//
// This module provides a lightweight, span-based timing API for the relay miner.
// You can start/finish named spans anywhere in the call stack, safely nest spans
// with the same name, and record durations to Prometheus with labels
// {instruction, service_id}.
//
// Key properties
// - Per-request isolation: a tracker is attached to context via WithPerf*.
// - Low-overhead hot path: xsync.Map.Compute for atomic push/pop, optional
//   xsync.MPMCQueue for buffered emissions.
// - Nesting-safe: multiple Start/Finish pairs per name form a stack.
// - Service-aware metric: records into InstructionTimeSeconds with "service_id".
//
// Typical usage:
//
//   // after you know serviceId; attach once per request
//   ctx = relayer.WithPerfForService(ctx, serviceId, relayer.PerfOptions{
//       RecordOnFinish: false,  // buffer and Flush once at the end
//       BufferCap:      8192,   // optional
//   })
//   defer relayer.Flush(ctx)
//
//   // ergonomic timing anywhere (captures tracker internally):
//   defer relayer.Span(ctx, "build_service_backend_request")()
//   // ... code ...
//
//   // zero-lookup hot path (grab tracker once):
//   tr, _ := relayer.FromCtx(ctx)
//   tr.Start("http_client_do")
//   // ... code ...
//   tr.Finish("http_client_do")
//
// Notes:
// - context.WithValue is called once per request. Span finishes avoid ctx.Value
//   by capturing the tracker.
// - If RecordOnFinish is true, Finish() observes directly to Prometheus;
//   otherwise spans are buffered and emitted by Flush(ctx)/tr.Flush().
// - Ensure relayer.InstructionTimeSeconds has labels ("instruction","service_id")
//   and measures seconds.

// Instruction labels for metrics.
//
// When adding new instructions, maintain alphabetical order within each section
// and update the metrics documentation accordingly.
const (
	// --- General proxy sync flow ---

	InstructionProxySyncParseRequest                  string = "proxy_sync_parse_request"
	InstructionProxySyncGetServiceConfig              string = "proxy_sync_get_service_config"
	InstructionProxySyncGetSessionEntry               string = "proxy_sync_get_session_entry"
	InstructionProxySyncMarkSessionAsKnown            string = "proxy_sync_mark_session_as_known"
	InstructionProxySyncCheckSessionIsRewardable      string = "proxy_sync_check_session_is_rewardable"
	InstructionProxySyncConfigResponseController      string = "proxy_sync_config_response_controller"
	InstructionProxySyncEagerRewardRollback           string = "proxy_sync_eager_reward_rollback"
	InstructionProxySyncEagerCheckRateLimiting        string = "proxy_sync_eager_check_rate_limiting"
	InstructionProxySyncEagerRequestVerification      string = "proxy_sync_eager_request_verification"
	InstructionProxySyncBuildBackendRequest           string = "proxy_sync_build_backend_request"
	InstructionProxySyncDeriveRemainingTime           string = "proxy_sync_derive_remaining_time"
	InstructionProxySyncBackendCall                   string = "proxy_sync_backend_call"
	InstructionProxySyncSerializeBackendResponse      string = "proxy_sync_serialize_backend_response"
	InstructionProxySyncGenerateRelayResponse         string = "proxy_sync_generate_relay_response"
	InstructionProxySyncWriteResponse                 string = "proxy_sync_write_response"
	InstructionProxySyncPublishToMiningSupervisor     string = "proxy_sync_publish_to_mining_supervisor"
	InstructionProxySyncEagerCheckRelayEligibility    string = "proxy_sync_eager_check_relay_eligibility"
	InstructionProxySyncEagerCheckRewardApplicability string = "proxy_sync_eager_check_reward_applicability"

	// --- Mining Supervisor flow ---

	InstructionMiningSupervisorEnqueue            string = "mining_supervisor_enqueue"
	InstructionMiningSupervisorEnqueueEvict       string = "mining_supervisor_enqueue_evict"
	InstructionMiningSupervisorEnqueueTimeoutWait string = "mining_supervisor_enqueue_timeout_wait"
	InstructionMiningSupervisorProcessRelay       string = "mining_supervisor_process_relay"

	// --- Delayed relay flow specific at Mining Supervisor ---

	InstructionMiningSupervisorDelayedCheckRateLimiting   string = "mining_supervisor_delayed_check_rate_limiting"
	InstructionMiningSupervisorDelayedRequestVerification string = "mining_supervisor_delayed_request_verification"
	InstructionMiningSupervisorSessionUpsert              string = "mining_supervisor_session_upsert"
	InstructionMiningSupervisorDownstreamSend             string = "mining_supervisor_downstream_send"
	InstructionMiningSupervisorRewardRollback             string = "mining_supervisor_reward_rollback"
)

// SpanRecord represents a completed span (name + duration).
// Used internally when buffering spans for Flush().
type SpanRecord struct {
	Name string
	Dur  time.Duration
}

// PerfOptions configures a per-request tracker created by WithPerf().
//
// RecordOnFinish:
//   - true  => each Finish() immediately records to Prometheus
//   - false => Finish() enqueues to a lock-free buffer; caller must Flush()
//
// BufferCap:
//   - Capacity for the MPMC queue when RecordOnFinish=false (default 4_096).
//
// ConstantLabels:
//   - Constant labels attached to each observation; a "service_id" key is
//     ensured (defaults to "unknown"). Prefer WithPerfForService(...) to set it.
//
// Observe:
//   - Optional custom sink for observations. If nil, defaultObserve() records
//     into InstructionTimeSeconds with labels ("instruction","service_id").
type PerfOptions struct {
	RecordOnFinish bool
	BufferCap      int

	ConstantLabels map[string]string
	Observe        func(tr *PerfTracker, name string, dur time.Duration)
}

type perfKeyType struct{}

var perfKey perfKeyType

// PerfTracker holds per-request state. Safe for concurrent use by goroutines
// serving the same HTTP request.
//
//   - openStacks: per-name stack of start times (supports nested spans) stored in
//     an xsync.Map with atomic Compute operations.
//   - completedQueue: optional, used when buffering spans for a single Flush()
//     to reduce metric write amplification on the hot path.
type PerfTracker struct {
	openStacks     *xsync.Map[string, []int64]
	completedQueue *xsync.MPMCQueue[SpanRecord]
	serviceID      atomic.Value
	recordOnFinish bool
	constLabels    map[string]string
	observe        func(tr *PerfTracker, name string, dur time.Duration)
}

// WithPerf attaches a fresh performance tracker to the context.
// Call once per request (after you know serviceId, or use WithPerfForService).
// Derived contexts (WithTimeout/WithDeadline) inherit the value.
func WithPerf(parent context.Context, opts PerfOptions) context.Context {
	if opts.BufferCap <= 0 {
		opts.BufferCap = 4_096
	}
	// Ensure a service_id label exists.
	if opts.ConstantLabels == nil {
		opts.ConstantLabels = map[string]string{"service_id": "unknown"}
	} else if _, ok := opts.ConstantLabels["service_id"]; !ok {
		opts.ConstantLabels["service_id"] = "unknown"
	}

	tr := &PerfTracker{
		openStacks:     xsync.NewMap[string, []int64](),
		recordOnFinish: opts.RecordOnFinish,
		constLabels:    opts.ConstantLabels,
	}
	// Initialize the mutable serviceID with the requested label.
	tr.serviceID.Store(opts.ConstantLabels["service_id"])

	if !opts.RecordOnFinish {
		tr.completedQueue = xsync.NewMPMCQueue[SpanRecord](opts.BufferCap)
	}

	if opts.Observe != nil {
		tr.observe = opts.Observe
	} else {
		tr.observe = defaultObserve // emits to InstructionTimeSeconds
	}

	return context.WithValue(parent, perfKey, tr)
}

// WithPerfForService wraps WithPerf and sets the "service_id" constant label.
func WithPerfForService(parent context.Context, serviceID string, opts PerfOptions) context.Context {
	if opts.ConstantLabels == nil {
		opts.ConstantLabels = map[string]string{"service_id": serviceID}
	} else {
		opts.ConstantLabels["service_id"] = serviceID
	}
	return WithPerf(parent, opts)
}

// FromCtx extracts the per-request tracker from context.
// It returns (nil, false) if none is attached.
func FromCtx(ctx context.Context) (*PerfTracker, bool) {
	tr, ok := ctx.Value(perfKey).(*PerfTracker)
	return tr, ok
}

// EnsurePerf ensures that `ctx` carries a PerfTracker labeled with the given serviceID,
// returning (ctxWithTracker, tracker, flush). If `ctx` already has a tracker, it is reused
// and `flush` is a no-op. If serviceID is empty/blank, it defaults to "unknown".
// Options let you pick buffered vs. immediate recording.
// Typical hot path usage: buffered (RecordOnFinish=false) + single Flush at end.
func EnsurePerf(
	ctx context.Context,
	serviceID string,
	opts PerfOptions,
) (context.Context, *PerfTracker, func()) {
	// Normalize service id
	sid := strings.TrimSpace(serviceID)
	if sid == "" {
		sid = UnknownServiceID
	}

	// Reuse the existing tracker when present
	if tr, ok := FromCtx(ctx); ok {
		// Keep ctx unchanged; no flush required here
		return ctx, tr, func() { tr.Flush() }
	}

	// Attach a new tracker to this context and return a flush
	ctxWith := WithPerfForService(ctx, sid, opts)
	tr, _ := FromCtx(ctxWith) // guaranteed non-nil after WithPerfForService
	// Return idempotent flush bound to this tracker (avoids ctx.Value lookups)
	return ctxWith, tr, func() { tr.Flush() }
}

// EnsurePerfBuffered is a convenience for the common fast path:
// buffered spans, flushed by the returned flush().
// Use this in hot paths unless you truly need immediate Observe.
func EnsurePerfBuffered(ctx context.Context, serviceID string) (context.Context, *PerfTracker, func()) {
	return EnsurePerf(ctx, serviceID, PerfOptions{
		RecordOnFinish: false,
		BufferCap:      2_048,
	})
}

// SetPerfServiceID Upgrade the service_id on the tracker in ctx, if present.
func SetPerfServiceID(ctx context.Context, id string) bool {
	if tr, ok := FromCtx(ctx); ok {
		tr.SetServiceID(id)
		return true
	}
	return false
}

// -------------------------- Instance methods --------------------------------

// Start begins a span named name on this tracker.
// Safe to call multiple times for the same name (creates a nested span).
func (tr *PerfTracker) Start(name string) {
	if tr == nil || name == "" {
		return
	}
	now := time.Now().UnixNano()
	tr.openStacks.Compute(name, func(stack []int64, loaded bool) ([]int64, xsync.ComputeOp) {
		if !loaded {
			// small reusable stack with spare capacity to avoid reallocating
			s := make([]int64, 1, 2)
			s[0] = now
			return s, xsync.UpdateOp
		}
		return append(stack, now), xsync.UpdateOp
	})

}

// Finish ends the most recent span named name and records the duration.
// If RecordOnFinish was true for this tracker, the duration is observed
// immediately; otherwise it is buffered for Flush().
func (tr *PerfTracker) Finish(name string) {
	if tr == nil || name == "" {
		return
	}
	var start int64
	var haveStart bool

	tr.openStacks.Compute(name, func(stack []int64, loaded bool) ([]int64, xsync.ComputeOp) {
		if !loaded || len(stack) == 0 {
			return stack, xsync.CancelOp
		}
		start = stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		haveStart = true
		if len(stack) == 0 {
			return stack[:0], xsync.UpdateOp
		}
		return stack, xsync.UpdateOp
	})
	if !haveStart {
		return
	}

	dur := time.Duration(time.Now().UnixNano() - start)
	if tr.recordOnFinish || tr.completedQueue == nil || !tr.completedQueue.TryEnqueue(SpanRecord{Name: name, Dur: dur}) {
		tr.observe(tr, name, dur)
	}
}

// Flush drains any buffered spans and records them.
// No-op if the tracker was created with RecordOnFinish=true.
func (tr *PerfTracker) Flush() {
	if tr == nil || tr.recordOnFinish || tr.completedQueue == nil {
		return
	}
	for {
		rec, ok := tr.completedQueue.TryDequeue()
		if !ok {
			break
		}
		tr.observe(tr, rec.Name, rec.Dur)
	}
}

// SetServiceID Upgrade the service_id on the tracker.
func (tr *PerfTracker) SetServiceID(id string) {
	if id == "" {
		id = UnknownServiceID
	}
	tr.serviceID.Store(id)
}

// ServiceID returns the service_id on the tracker, if any.
func (tr *PerfTracker) ServiceID() string {
	if v := tr.serviceID.Load(); v != nil {
		return v.(string)
	}
	return UnknownServiceID
}

// ------------------------ Package-level helpers -----------------------------

// Start marks the beginning of a span on the tracker stored in ctx (if any).
func Start(ctx context.Context, name string) {
	if tr, ok := FromCtx(ctx); ok {
		tr.Start(name)
	}
}

// Finish marks the end of a span on the tracker stored in ctx (if any).
func Finish(ctx context.Context, name string) {
	if tr, ok := FromCtx(ctx); ok {
		tr.Finish(name)
	}
}

// Span returns a finisher suitable for `defer` that captures the tracker now,
// avoiding any ctx.Value lookups on the Finish path.
//
// Example:
//
//	defer relayer.Span(ctx, "serialize_http_response")()
//	// ... work ...
func Span(ctx context.Context, name string) func() {
	tr, _ := FromCtx(ctx)
	tr.Start(name)
	return func() { tr.Finish(name) }
}

// Flush drains buffered spans for the tracker in ctx (if any).
func Flush(ctx context.Context) {
	if tr, ok := FromCtx(ctx); ok {
		tr.Flush()
	}
}

// ------------------------------ Observers -----------------------------------

// obsKey is a composite key type consisting of an instruction name and a service ID, used for caching Observers.
type obsKey struct{ instr, sid string }

// instrObsCache is a cache of Observers for each instruction and service_id.
// It is shared by all PerfTrackers.
var instrObsCache = xsync.NewMap[obsKey, stdprometheus.Observer]()

// getObs returns the Observer for the given instruction and service_id.
func getObs(sid, instr string) stdprometheus.Observer {
	k := obsKey{instr: instr, sid: sid}
	if o, ok := instrObsCache.Load(k); ok {
		return o
	}
	o := InstructionTimeSeconds.WithLabelValues(instr, sid)
	if prev, ok := instrObsCache.LoadOrStore(k, o); ok {
		return prev
	}
	return o
}

// defaultObserve records into InstructionTimeSeconds with labels
// ("instruction","service_id"). It assumes the metric was defined with that
// exact label set. Durations are in seconds.
func defaultObserve(tr *PerfTracker, name string, dur time.Duration) {
	getObs(tr.ServiceID(), name).Observe(dur.Seconds())
}
