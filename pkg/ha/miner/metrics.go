package miner

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/pokt-network/poktroll/pkg/ha/observability"
)

const (
	metricsNamespace = "ha"
	metricsSubsystem = "miner"
)

var (
	// Relay consumption metrics
	relaysConsumed = observability.MinerFactory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "relays_consumed_total",
			Help:      "Total number of relays consumed from Redis streams",
		},
		[]string{"supplier", "service_id"},
	)

	relaysProcessed = observability.MinerFactory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "relays_processed_total",
			Help:      "Total number of relays successfully processed",
		},
		[]string{"supplier", "service_id"},
	)

	relaysDeduplicated = observability.MinerFactory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "relays_deduplicated_total",
			Help:      "Total number of duplicate relays filtered",
		},
		[]string{"supplier", "service_id"},
	)

	relaysRejected = observability.MinerFactory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "relays_rejected_total",
			Help:      "Total number of relays rejected due to errors",
		},
		[]string{"supplier", "reason"},
	)

	relayProcessingLatency = observability.MinerFactory.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "relay_processing_latency_seconds",
			Help:      "Time to process a single relay",
			Buckets:   []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1},
		},
		[]string{"supplier"},
	)

	// ====== OPERATOR-FOCUSED METRICS ======

	// Sessions by state - helps operators see session lifecycle status at a glance
	sessionsByState = observability.MinerFactory.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "sessions_by_state",
			Help:      "Number of sessions in each state (active, claiming, claimed, proving, proved, settled)",
		},
		[]string{"supplier", "state"},
	)

	// Session info - detailed metrics per session for debugging
	sessionRelayCount = observability.MinerFactory.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "session_relay_count",
			Help:      "Number of relays in each session",
		},
		[]string{"supplier", "session_id", "service_id"},
	)

	sessionComputeUnits = observability.MinerFactory.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "session_compute_units",
			Help:      "Total compute units in each session",
		},
		[]string{"supplier", "session_id", "service_id"},
	)

	sessionEndHeight = observability.MinerFactory.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "session_end_height",
			Help:      "End height of each active session",
		},
		[]string{"supplier", "session_id"},
	)

	// Claim timing metrics - helps operators verify timing spread
	claimScheduledHeight = observability.MinerFactory.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "claim_scheduled_height",
			Help:      "Block height when claim is scheduled to be submitted",
		},
		[]string{"supplier", "session_id"},
	)

	claimBlocksUntilSubmit = observability.MinerFactory.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "claim_blocks_until_submit",
			Help:      "Number of blocks until claim will be submitted (0 if already submitted)",
		},
		[]string{"supplier", "session_id"},
	)

	claimSubmissionLatencyBlocks = observability.MinerFactory.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "claim_submission_latency_blocks",
			Help:      "Blocks after claim window opened when claim was submitted",
			Buckets:   []float64{0, 1, 2, 3, 4, 5, 10, 15, 20},
		},
		[]string{"supplier"},
	)

	// Proof timing metrics
	proofScheduledHeight = observability.MinerFactory.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "proof_scheduled_height",
			Help:      "Block height when proof is scheduled to be submitted",
		},
		[]string{"supplier", "session_id"},
	)

	proofBlocksUntilSubmit = observability.MinerFactory.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "proof_blocks_until_submit",
			Help:      "Number of blocks until proof will be submitted (0 if already submitted)",
		},
		[]string{"supplier", "session_id"},
	)

	proofSubmissionLatencyBlocks = observability.MinerFactory.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "proof_submission_latency_blocks",
			Help:      "Blocks after proof window opened when proof was submitted",
			Buckets:   []float64{0, 1, 2, 3, 4, 5, 10, 15, 20},
		},
		[]string{"supplier"},
	)

	// Session lifecycle totals - useful for SLIs/SLOs
	sessionsCreatedTotal = observability.MinerFactory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "sessions_created_total",
			Help:      "Total number of sessions created",
		},
		[]string{"supplier", "service_id"},
	)

	sessionsSettledTotal = observability.MinerFactory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "sessions_settled_total",
			Help:      "Total number of sessions successfully settled",
		},
		[]string{"supplier", "service_id"},
	)

	sessionsFailedTotal = observability.MinerFactory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "sessions_failed_total",
			Help:      "Total number of sessions that failed (missed claim/proof window)",
		},
		[]string{"supplier", "service_id", "reason"},
	)

	// Compute units totals - helps track revenue
	computeUnitsClaimedTotal = observability.MinerFactory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "compute_units_claimed_total",
			Help:      "Total compute units claimed across all sessions",
		},
		[]string{"supplier", "service_id"},
	)

	computeUnitsSettledTotal = observability.MinerFactory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "compute_units_settled_total",
			Help:      "Total compute units settled (proven) across all sessions",
		},
		[]string{"supplier", "service_id"},
	)

	// Deduplication metrics
	dedupLocalCacheHits = observability.MinerFactory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "dedup_local_cache_hits_total",
			Help:      "Total number of deduplication local cache hits",
		},
		[]string{"session_id"},
	)

	dedupRedisCacheHits = observability.MinerFactory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "dedup_redis_cache_hits_total",
			Help:      "Total number of deduplication Redis cache hits",
		},
		[]string{"session_id"},
	)

	dedupMisses = observability.MinerFactory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "dedup_misses_total",
			Help:      "Total number of deduplication cache misses (new relays)",
		},
		[]string{"session_id"},
	)

	dedupMarked = observability.MinerFactory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "dedup_marked_total",
			Help:      "Total number of relays marked as processed",
		},
		[]string{"session_id"},
	)

	dedupErrors = observability.MinerFactory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "dedup_errors_total",
			Help:      "Total number of deduplication errors",
		},
		[]string{"session_id", "operation"},
	)

	// Session tree metrics (reserved for future instrumentation)
	_ = observability.MinerFactory.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "session_trees_active",
			Help:      "Number of active session trees",
		},
		[]string{"supplier"},
	)

	_ = observability.MinerFactory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "session_tree_updates_total",
			Help:      "Total number of session tree updates",
		},
		[]string{"supplier", "session_id"},
	)

	_ = observability.MinerFactory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "session_tree_flushes_total",
			Help:      "Total number of session tree flushes",
		},
		[]string{"supplier"},
	)

	_ = observability.MinerFactory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "session_tree_errors_total",
			Help:      "Total number of session tree errors",
		},
		[]string{"supplier", "operation"},
	)

	// Claim and proof metrics (claimsCreated reserved for future instrumentation)
	_ = observability.MinerFactory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "claims_created_total",
			Help:      "Total number of claims created",
		},
		[]string{"supplier"},
	)

	claimsSubmitted = observability.MinerFactory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "claims_submitted_total",
			Help:      "Total number of claims submitted on-chain",
		},
		[]string{"supplier"},
	)

	claimErrors = observability.MinerFactory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "claim_errors_total",
			Help:      "Total number of claim errors",
		},
		[]string{"supplier", "reason"},
	)

	// proofsCreated reserved for future instrumentation
	_ = observability.MinerFactory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "proofs_created_total",
			Help:      "Total number of proofs created",
		},
		[]string{"supplier"},
	)

	proofsSubmitted = observability.MinerFactory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "proofs_submitted_total",
			Help:      "Total number of proofs submitted on-chain",
		},
		[]string{"supplier"},
	)

	proofErrors = observability.MinerFactory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "proof_errors_total",
			Help:      "Total number of proof errors",
		},
		[]string{"supplier", "reason"},
	)

	// Redis consumer metrics (reserved for future instrumentation)
	_ = observability.MinerFactory.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "consumer_lag",
			Help:      "Number of messages pending in the consumer group",
		},
		[]string{"supplier"},
	)

	_ = observability.MinerFactory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "messages_acknowledged_total",
			Help:      "Total number of Redis messages acknowledged",
		},
		[]string{"supplier"},
	)

	// Block height
	currentBlockHeight = observability.MinerFactory.NewGauge(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "current_block_height",
			Help:      "Current block height as seen by the miner",
		},
	)

	// Leader election metrics
	leaderStatus = observability.MinerFactory.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "leader_status",
			Help:      "Whether this instance is the leader (1=leader, 0=standby)",
		},
		[]string{"supplier", "instance"},
	)

	leaderAcquisitions = observability.MinerFactory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "leader_acquisitions_total",
			Help:      "Total number of times this instance acquired leadership",
		},
		[]string{"supplier"},
	)

	leaderLosses = observability.MinerFactory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "leader_losses_total",
			Help:      "Total number of times this instance lost leadership",
		},
		[]string{"supplier"},
	)

	leaderHeartbeats = observability.MinerFactory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "leader_heartbeats_total",
			Help:      "Total number of successful leader heartbeats",
		},
		[]string{"supplier"},
	)

	// WAL metrics
	walAppends = observability.MinerFactory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "wal_appends_total",
			Help:      "Total number of entries appended to WAL",
		},
		[]string{"supplier", "session_id"},
	)

	walReplays = observability.MinerFactory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "wal_replays_total",
			Help:      "Total number of WAL entries replayed during recovery",
		},
		[]string{"supplier", "session_id"},
	)

	walCheckpoints = observability.MinerFactory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "wal_checkpoints_total",
			Help:      "Total number of WAL checkpoints created",
		},
		[]string{"supplier", "session_id"},
	)

	walSize = observability.MinerFactory.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "wal_size_entries",
			Help:      "Current number of entries in WAL",
		},
		[]string{"supplier", "session_id"},
	)

	// SMST snapshot metrics (smstSnapshots reserved for future instrumentation)
	_ = observability.MinerFactory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "smst_snapshots_total",
			Help:      "Total number of SMST snapshots saved",
		},
		[]string{"supplier", "session_id"},
	)

	smstRecoveries = observability.MinerFactory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "smst_recoveries_total",
			Help:      "Total number of SMST recoveries from snapshot",
		},
		[]string{"supplier", "session_id"},
	)

	// smstSnapshotLatency reserved for future instrumentation
	_ = observability.MinerFactory.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "smst_snapshot_latency_seconds",
			Help:      "Time to create SMST snapshot",
			Buckets:   []float64{0.01, 0.05, 0.1, 0.5, 1, 5},
		},
		[]string{"supplier"},
	)

	smstRecoveryLatency = observability.MinerFactory.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "smst_recovery_latency_seconds",
			Help:      "Time to recover SMST from snapshot + WAL",
			Buckets:   []float64{0.1, 0.5, 1, 5, 10, 30},
		},
		[]string{"supplier"},
	)

	// Session store metrics
	sessionSnapshotsSaved = observability.MinerFactory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "session_snapshots_saved_total",
			Help:      "Total number of session snapshots saved to Redis",
		},
		[]string{"supplier"},
	)

	sessionSnapshotsLoaded = observability.MinerFactory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "session_snapshots_loaded_total",
			Help:      "Total number of session snapshots loaded from Redis",
		},
		[]string{"supplier"},
	)

	sessionSnapshotsSkippedAtStartup = observability.MinerFactory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "session_snapshots_skipped_at_startup_total",
			Help:      "Total number of session snapshots skipped at startup (expired or settled)",
		},
		[]string{"supplier", "state"},
	)

	sessionStoreErrors = observability.MinerFactory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "session_store_errors_total",
			Help:      "Total number of session store errors",
		},
		[]string{"supplier", "operation"},
	)

	sessionStateTransitions = observability.MinerFactory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "session_state_transitions_total",
			Help:      "Total number of session state transitions",
		},
		[]string{"supplier", "from_state", "to_state"},
	)

	// Supplier manager metrics
	supplierManagerSuppliersActive = observability.MinerFactory.NewGauge(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "supplier_manager_suppliers_active",
			Help:      "Number of active suppliers in the supplier manager",
		},
	)

	// Supplier registry metrics
	supplierRegistryUpdatesTotal = observability.MinerFactory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "supplier_registry_updates_total",
			Help:      "Total number of supplier registry updates",
		},
		[]string{"action"},
	)
)

// =============================================
// METRICS HELPER FUNCTIONS FOR OPERATORS
// =============================================

// RecordRelayConsumed records a relay consumed from the stream.
func RecordRelayConsumed(supplier, serviceID string) {
	relaysConsumed.WithLabelValues(supplier, serviceID).Inc()
}

// RecordRelayProcessed records a relay successfully processed.
func RecordRelayProcessed(supplier, serviceID string) {
	relaysProcessed.WithLabelValues(supplier, serviceID).Inc()
}

// RecordRelayDeduplicated records a relay that was deduplicated.
func RecordRelayDeduplicated(supplier, serviceID string) {
	relaysDeduplicated.WithLabelValues(supplier, serviceID).Inc()
}

// RecordRelayRejected records a relay that was rejected.
func RecordRelayRejected(supplier, reason string) {
	relaysRejected.WithLabelValues(supplier, reason).Inc()
}

// RecordRelayProcessingLatency records how long it took to process a relay.
func RecordRelayProcessingLatency(supplier string, seconds float64) {
	relayProcessingLatency.WithLabelValues(supplier).Observe(seconds)
}

// SetSessionsByState sets the count of sessions in a given state.
func SetSessionsByState(supplier, state string, count float64) {
	sessionsByState.WithLabelValues(supplier, state).Set(count)
}

// RecordSessionCreated increments the session created counter.
func RecordSessionCreated(supplier, serviceID string) {
	sessionsCreatedTotal.WithLabelValues(supplier, serviceID).Inc()
}

// SetSessionRelayCount sets the current relay count for a session.
func SetSessionRelayCount(supplier, sessionID, serviceID string, count float64) {
	sessionRelayCount.WithLabelValues(supplier, sessionID, serviceID).Set(count)
}

// SetSessionComputeUnits sets the current compute units for a session.
func SetSessionComputeUnits(supplier, sessionID, serviceID string, units float64) {
	sessionComputeUnits.WithLabelValues(supplier, sessionID, serviceID).Set(units)
}

// SetSessionEndHeight sets the end height for a session.
func SetSessionEndHeight(supplier, sessionID string, height float64) {
	sessionEndHeight.WithLabelValues(supplier, sessionID).Set(height)
}

// ClearSessionMetrics removes session-specific metrics when session completes.
func ClearSessionMetrics(supplier, sessionID, serviceID string) {
	sessionRelayCount.DeleteLabelValues(supplier, sessionID, serviceID)
	sessionComputeUnits.DeleteLabelValues(supplier, sessionID, serviceID)
	sessionEndHeight.DeleteLabelValues(supplier, sessionID)
	claimScheduledHeight.DeleteLabelValues(supplier, sessionID)
	claimBlocksUntilSubmit.DeleteLabelValues(supplier, sessionID)
	proofScheduledHeight.DeleteLabelValues(supplier, sessionID)
	proofBlocksUntilSubmit.DeleteLabelValues(supplier, sessionID)
}

// SetClaimScheduledHeight sets when a claim is scheduled to be submitted.
func SetClaimScheduledHeight(supplier, sessionID string, height float64) {
	claimScheduledHeight.WithLabelValues(supplier, sessionID).Set(height)
}

// SetClaimBlocksUntilSubmit sets blocks remaining until claim submission.
func SetClaimBlocksUntilSubmit(supplier, sessionID string, blocks float64) {
	claimBlocksUntilSubmit.WithLabelValues(supplier, sessionID).Set(blocks)
}

// RecordClaimSubmissionLatency records how many blocks after window opened the claim was submitted.
func RecordClaimSubmissionLatency(supplier string, blocksAfterWindowOpened float64) {
	claimSubmissionLatencyBlocks.WithLabelValues(supplier).Observe(blocksAfterWindowOpened)
}

// SetProofScheduledHeight sets when a proof is scheduled to be submitted.
func SetProofScheduledHeight(supplier, sessionID string, height float64) {
	proofScheduledHeight.WithLabelValues(supplier, sessionID).Set(height)
}

// SetProofBlocksUntilSubmit sets blocks remaining until proof submission.
func SetProofBlocksUntilSubmit(supplier, sessionID string, blocks float64) {
	proofBlocksUntilSubmit.WithLabelValues(supplier, sessionID).Set(blocks)
}

// RecordProofSubmissionLatency records how many blocks after window opened the proof was submitted.
func RecordProofSubmissionLatency(supplier string, blocksAfterWindowOpened float64) {
	proofSubmissionLatencyBlocks.WithLabelValues(supplier).Observe(blocksAfterWindowOpened)
}

// RecordSessionSettled increments the settled sessions counter.
func RecordSessionSettled(supplier, serviceID string) {
	sessionsSettledTotal.WithLabelValues(supplier, serviceID).Inc()
}

// RecordSessionFailed increments the failed sessions counter.
func RecordSessionFailed(supplier, serviceID, reason string) {
	sessionsFailedTotal.WithLabelValues(supplier, serviceID, reason).Inc()
}

// RecordComputeUnitsClaimed adds to the claimed compute units total.
func RecordComputeUnitsClaimed(supplier, serviceID string, units float64) {
	computeUnitsClaimedTotal.WithLabelValues(supplier, serviceID).Add(units)
}

// RecordComputeUnitsSettled adds to the settled compute units total.
func RecordComputeUnitsSettled(supplier, serviceID string, units float64) {
	computeUnitsSettledTotal.WithLabelValues(supplier, serviceID).Add(units)
}

// RecordClaimSubmitted increments the claims submitted counter.
func RecordClaimSubmitted(supplier string) {
	claimsSubmitted.WithLabelValues(supplier).Inc()
}

// RecordClaimError increments the claim errors counter.
func RecordClaimError(supplier, reason string) {
	claimErrors.WithLabelValues(supplier, reason).Inc()
}

// RecordProofSubmitted increments the proofs submitted counter.
func RecordProofSubmitted(supplier string) {
	proofsSubmitted.WithLabelValues(supplier).Inc()
}

// RecordProofError increments the proof errors counter.
func RecordProofError(supplier, reason string) {
	proofErrors.WithLabelValues(supplier, reason).Inc()
}

// SetCurrentBlockHeight sets the current block height.
func SetCurrentBlockHeight(height float64) {
	currentBlockHeight.Set(height)
}

// RecordSessionStateTransition records a state change.
func RecordSessionStateTransition(supplier, fromState, toState string) {
	sessionStateTransitions.WithLabelValues(supplier, fromState, toState).Inc()
}
