package proxy

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/puzpuzpuz/xsync/v4"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/config"
	"github.com/pokt-network/poktroll/pkg/relayer/relay_authenticator"
	"github.com/pokt-network/poktroll/x/service/types"
)

// SessionCache stores fast-path per-session state.
// Rewardable starts true and is only ever downgraded to false.
type SessionCache struct {
	EndHeight  int64
	Rewardable atomic.Bool
}

// RelayMiningSupervisor runs delayed validation/rewardability and forwards rewardable
// relays to the downstream "observable"/miner. It uses one bounded queue, a worker pool,
// and an xsync-backed session cache.
type RelayMiningSupervisor struct {
	logger     polylog.Logger
	downstream chan<- *types.Relay // we don't own/close this

	queue chan *types.Relay // internal bounded queue
	wg    sync.WaitGroup

	ctx     context.Context
	cancel  context.CancelFunc
	stopped atomic.Bool

	// Options
	dropOldest          bool
	enqueueTimeout      time.Duration
	gaugeSampleInterval time.Duration
	dropLogInterval     time.Duration

	// Fast state
	downstreamClosed atomic.Bool
	lastDropLogNs    atomic.Int64 // rate-limit downstream drop logs

	// Dependencies
	relayMeter         relayer.RelayMeter
	relayAuthenticator relayer.RelayAuthenticator

	// Sessions: sessionID -> *SessionCache
	knownSessions *xsync.Map[string, *SessionCache]
}

// NewRelayMiningSupervisor creates a new instance and starts workers + gauge sampler.
// Panics if downstream is nil.
func NewRelayMiningSupervisor(
	logger polylog.Logger,
	downstream chan<- *types.Relay,
	cfg *config.MiningSupervisorConfig,
	relayMeter relayer.RelayMeter,
	relayAuthenticator relayer.RelayAuthenticator,
) *RelayMiningSupervisor {
	if downstream == nil {
		// Panic is appropriate here since this is a programmer error (invalid constructor arg).
		// TODO_FUTURE: Consider returning (nil, error) for better testability.
		panic("RelayMiningSupervisor: downstream channel must not be nil")
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 10_000
	}
	if cfg.Workers <= 0 {
		cfg.Workers = uint8(runtime.GOMAXPROCS(0))
	}
	switch cfg.DropPolicy {
	case "drop-oldest", "drop-new":
	default:
		logger.Warn().Msgf("unknown mining_supervisor.drop_policy = %q; defaulting to drop-new", cfg.DropPolicy)
		cfg.DropPolicy = config.DefaultMSDropPolicy
	}
	if cfg.GaugeSampleInterval <= 0 {
		cfg.GaugeSampleInterval = 200 * time.Millisecond
	}
	if cfg.DropLogInterval <= 0 {
		cfg.DropLogInterval = 2 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	s := &RelayMiningSupervisor{
		logger:              logger.With("component", "mining_supervisor"),
		downstream:          downstream,
		queue:               make(chan *types.Relay, cfg.QueueSize),
		ctx:                 ctx,
		cancel:              cancel,
		dropOldest:          cfg.DropPolicy == "drop-oldest",
		enqueueTimeout:      cfg.EnqueueTimeout,
		gaugeSampleInterval: cfg.GaugeSampleInterval,
		dropLogInterval:     cfg.DropLogInterval,
		knownSessions:       xsync.NewMap[string, *SessionCache](),
		relayMeter:          relayMeter,
		relayAuthenticator:  relayAuthenticator,
	}

	relayer.SetMiningQueueLen(0)
	s.logger.Info().
		Int("queue_cap", cap(s.queue)).
		Int("downstream_cap", cap(downstream)).
		Uint8("workers", cfg.Workers).
		Str("drop_policy", cfg.DropPolicy).
		Dur("enqueue_timeout", cfg.EnqueueTimeout).
		Dur("gauge_sample_interval", cfg.GaugeSampleInterval).
		Dur("drop_log_interval", cfg.DropLogInterval).
		Msg("relay mining supervisor started")

	// workers
	for i := 0; i < int(cfg.Workers); i++ {
		s.wg.Add(1)
		go s.worker(i)
	}

	// gauge sampler (keeps hot path clean)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		t := time.NewTicker(s.gaugeSampleInterval)
		defer t.Stop()
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-t.C:
				relayer.SetMiningQueueLen(len(s.queue))
			}
		}
	}()

	return s
}

// Stop stops workers and the sampler, drains the internal queue, and returns.
// Does NOT close downstream.
func (s *RelayMiningSupervisor) Stop() {
	if s.stopped.Swap(true) {
		return
	}
	s.cancel()
	close(s.queue) // signal workers to exit after draining
	s.wg.Wait()
	s.logger.Info().Msg("relay mining supervisor stopped")
}

// Publish enqueues a relay. Default is non-blocking; with EnqueueTimeout > 0,
// waits up to that timeout and (if drop-oldest) evicts one entry before a final retry.
func (s *RelayMiningSupervisor) Publish(ctx context.Context, r *types.Relay) bool {
	if r == nil {
		return false
	}

	if s.stopped.Load() || s.ctx.Err() != nil || s.downstreamClosed.Load() {
		relayer.CaptureMiningQueueDropped(serviceIDOf(r), "stopped_or_closed")
		return false
	}

	sid := serviceIDOf(r)

	// Reuse caller tracker if present; otherwise create a buffered one here.
	ctx, tr, flush := relayer.EnsurePerfBuffered(ctx, sid)
	defer flush()

	tr.Start(relayer.InstructionMiningSupervisorEnqueue)
	defer tr.Finish(relayer.InstructionMiningSupervisorEnqueue)

	// Non-blocking fast path
	if s.enqueueTimeout <= 0 {
		select {
		case s.queue <- r:
			relayer.CaptureMiningQueueEnqueued(sid)
			return true
		default:
			// optional drop-oldest policy
			if s.dropOldest {
				tr.Start(relayer.InstructionMiningSupervisorEnqueueEvict)
				var evicted *types.Relay
				select { // try evict one
				case evicted = <-s.queue:
				default:
				}
				tr.Finish(relayer.InstructionMiningSupervisorEnqueueEvict)
				if evicted != nil {
					relayer.CaptureMiningQueueDropped(serviceIDOf(evicted), "evicted")
				}
				select { // retry once
				case s.queue <- r:
					relayer.CaptureMiningQueueEnqueued(sid)
					return true
				default:
				}
			}
			relayer.CaptureMiningQueueDropped(sid, "full")
			s.maybeLogDrop("queue_full", sid, len(s.queue), cap(s.queue))
			return false
		}
	}

	// Small grace wait path
	timer := time.NewTimer(s.enqueueTimeout)
	defer timer.Stop()

	// Measure ONLY the waiting time in this select.
	tr.Start(relayer.InstructionMiningSupervisorEnqueueTimeoutWait)
	waitDone := func() { tr.Finish(relayer.InstructionMiningSupervisorEnqueueTimeoutWait) }

	select {
	case s.queue <- r:
		waitDone() // woke because queue became writable
		relayer.CaptureMiningQueueEnqueued(sid)
		return true

	case <-timer.C:
		waitDone() // waited full timeout
		if s.dropOldest {
			tr.Start(relayer.InstructionMiningSupervisorEnqueueEvict)
			var evicted *types.Relay
			select {
			case evicted = <-s.queue:
			default:
			}
			tr.Finish(relayer.InstructionMiningSupervisorEnqueueEvict)

			if evicted != nil {
				relayer.CaptureMiningQueueDropped(serviceIDOf(evicted), "evicted")
			}
			// retry is non-blocking; not part of the "wait" span
			select {
			case s.queue <- r:
				relayer.CaptureMiningQueueEnqueued(sid)
				return true
			default:
				relayer.CaptureMiningQueueDropped(sid, "timeout_full_after_evict")
				s.maybeLogDrop("queue_timeout_full_after_evict", sid, len(s.queue), cap(s.queue))
				return false
			}
		}
		relayer.CaptureMiningQueueDropped(sid, "timeout")
		s.maybeLogDrop("queue_timeout", sid, len(s.queue), cap(s.queue))
		return false

	case <-s.ctx.Done():
		waitDone() // unblocked by supervisor shutdown/cancel
		relayer.CaptureMiningQueueDropped(sid, "context")
		return false
	}
}

// worker drains the internal queue, performs delayed validation/rewardability,
// updates session cache, and forwards rewardable relays to downstream.
func (s *RelayMiningSupervisor) worker(id int) {
	defer s.wg.Done()
	defer func() {
		if rec := recover(); rec != nil {
			relayer.CaptureMiningWorkerPanic(id)
			s.logger.Error().
				Fields(map[string]any{"panic": rec}).
				Int("worker_id", id).
				Msg("mining worker recovered; restarting")
			time.Sleep(50 * time.Millisecond)
			if s.ctx.Err() == nil {
				s.wg.Add(1)
				go s.worker(id) // self-heal
			}
		}
	}()

	for {
		select {
		case <-s.ctx.Done():
			return
		case r, ok := <-s.queue:
			if !ok {
				return
			}
			s.processRelay(r)
		}
	}
}

// processRelay performs delayed validation/rewardability, updates session cache,
// and forwards rewardable relays to downstream.
func (s *RelayMiningSupervisor) processRelay(r *types.Relay) {
	meta := r.Req.GetMeta()
	sess := meta.GetSessionHeader()
	serviceID := sess.GetServiceId()
	sessionID := sess.GetSessionId()

	// One buffered tracker per processed relay; flush at the end.
	ctxPerf, tr, flush := relayer.EnsurePerfBuffered(s.ctx, serviceID)
	defer flush()

	// Overall processing span
	tr.Start(relayer.InstructionMiningSupervisorProcessRelay)
	defer tr.Finish(relayer.InstructionMiningSupervisorProcessRelay)

	// 1) Over-servicing snapshot.
	tr.Start(relayer.InstructionMiningSupervisorDelayedCheckRateLimiting)
	isOver := s.relayMeter.IsOverServicing(ctxPerf, meta)
	rewardable := !isOver || s.relayMeter.AllowOverServicing()
	tr.Finish(relayer.InstructionMiningSupervisorDelayedCheckRateLimiting)

	// 2) Signature/session verification.
	tr.Start(relayer.InstructionMiningSupervisorDelayedRequestVerification)
	err := s.relayAuthenticator.VerifyRelayRequest(ctxPerf, r.Req, serviceID)
	tr.Finish(relayer.InstructionMiningSupervisorDelayedRequestVerification)
	if err != nil {
		// invalid session -> permanently not rewardable
		if errors.Is(err, relay_authenticator.ErrRelayAuthenticatorInvalidSession) {
			rewardable = false
		}
		s.logger.Error().Err(err).Str("service_id", serviceID).
			Msg("dropped relay during delayed verification")
	}

	// 3) Update session cache (known and rewardable downgrade).
	tr.Start(relayer.InstructionMiningSupervisorSessionUpsert)
	s.upsertSessionState(sessionID, sess.GetSessionEndBlockHeight(), rewardable)
	tr.Finish(relayer.InstructionMiningSupervisorSessionUpsert)

	// 4) Forward only if rewardable; else revert optimistic accounting.
	if rewardable {
		tr.Start(relayer.InstructionMiningSupervisorDownstreamSend)
		ok := s.safeSendDownstream(r)
		tr.Finish(relayer.InstructionMiningSupervisorDownstreamSend)

		if !ok {
			relayer.CaptureMiningQueueDropped(serviceID, "downstream_closed_or_full")
			s.maybeLogDrop("downstream_full", serviceID, len(s.downstream), cap(s.downstream))
			s.downstreamClosed.Store(true)
			rewardable = ok // rollback to non-rewardable since we were not able to send downstream
		}
	}

	if !rewardable {
		tr.Start(relayer.InstructionMiningSupervisorRewardRollback)
		s.relayMeter.SetNonApplicableRelayReward(ctxPerf, meta)
		tr.Finish(relayer.InstructionMiningSupervisorRewardRollback)
	}
}

// safeSendDownstream uses a non-blocking send to protect worker latency.
func (s *RelayMiningSupervisor) safeSendDownstream(r *types.Relay) (ok bool) {
	defer func() {
		if rec := recover(); rec != nil {
			s.logger.Warn().Msg("downstream channel closed; dropping future relays")
			ok = false
		}
	}()
	select {
	case s.downstream <- r:
		return true
	default:
		return false // buffer full / no reader
	}
}

// --- Session cache (handler fast-path helpers) ---

// GetSessionEntry returns cached state if present.
func (s *RelayMiningSupervisor) GetSessionEntry(sessionID string) (*SessionCache, bool) {
	return s.knownSessions.Load(sessionID)
}

// MarkSessionAsKnown inserts/updates a session as known; rewardable stays true unless previously downgraded.
func (s *RelayMiningSupervisor) MarkSessionAsKnown(sessionID string, endHeight int64) *SessionCache {
	return s.upsertSessionState(sessionID, endHeight, true)
}

// MarkSessionAsNonRewardable permanently downgrades rewardability for a session.
func (s *RelayMiningSupervisor) MarkSessionAsNonRewardable(sessionID string) (*SessionCache, error) {
	st, ok := s.knownSessions.Load(sessionID)
	if !ok {
		return nil, fmt.Errorf("session %q not found", sessionID)
	}
	st.Rewardable.Store(false)
	return st, nil
}

// upsertSessionState inserts or updates session state and only downgrades rewardability.
//
// Note: This is intentionally optimistic. A TOCTOU race exists where:
//   - Thread A loads session with Rewardable=true
//   - Thread B downgrades to Rewardable=false
//   - Thread A acts on stale Rewardable=true
//
// This is acceptable because:
//   - Worst case: one extra relay is temporarily counted as rewardable
//   - The mining supervisor will rollback via SetNonApplicableRelayReward
//   - The atomic bool ensures eventual consistency
func (s *RelayMiningSupervisor) upsertSessionState(sessionID string, endHeight int64, rewardable bool) *SessionCache {
	if st, ok := s.knownSessions.Load(sessionID); ok {
		if endHeight > st.EndHeight {
			st.EndHeight = endHeight
		}
		if !rewardable {
			st.Rewardable.Store(false)
		}
		return st
	}
	st := &SessionCache{EndHeight: endHeight}
	st.Rewardable.Store(rewardable)
	s.knownSessions.Store(sessionID, st)
	return st
}

// PruneOutdatedKnownSessions removes sessions whose EndHeight is before the current height (with +1 guard).
// Call periodically (e.g., on new block events). Consider adding a grace window if late requests are common.
func (s *RelayMiningSupervisor) PruneOutdatedKnownSessions(_ context.Context, block client.Block) {
	s.knownSessions.Range(func(sessionID string, st *SessionCache) bool {
		if st.EndHeight+1 < block.Height() {
			s.knownSessions.Delete(sessionID)
		}
		return true
	})
}

// serviceIDOf returns a non-empty service id for metrics.
func serviceIDOf(r *types.Relay) string {
	if r == nil || r.Req == nil || r.Req.Meta.SessionHeader == nil {
		return "unknown"
	}
	sid := r.Req.Meta.SessionHeader.ServiceId
	if sid == "" {
		return "unknown"
	}
	return sid
}

// maybeLogDrop rate-limits drop logs to avoid log spam under backpressure.
func (s *RelayMiningSupervisor) maybeLogDrop(reason, serviceID string, length, capacity int) {
	now := time.Now().UnixNano()
	next := s.lastDropLogNs.Load()
	if next == 0 || now >= next {
		if s.lastDropLogNs.CompareAndSwap(next, now+s.dropLogInterval.Nanoseconds()) {
			s.logger.Warn().
				Str("reason", reason).
				Str("service_id", serviceID).
				Int("len", length).
				Int("cap", capacity).
				Msg("dropping relay due to backpressure")
		}
	}
}
