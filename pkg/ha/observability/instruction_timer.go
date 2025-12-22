package observability

import "time"

// Instruction labels for HA Relayer metrics.
const (
	// Relay processing flow
	InstructionRelayReceived         string = "relay_received"
	InstructionParseRelayRequest     string = "parse_relay_request"
	InstructionValidateRelayRequest  string = "validate_relay_request"
	InstructionCheckRateLimiting     string = "check_rate_limiting"
	InstructionGetSupplierConfig     string = "get_supplier_config"
	InstructionBuildBackendRequest   string = "build_backend_request"
	InstructionForwardToBackend      string = "forward_to_backend"
	InstructionSerializeResponse     string = "serialize_response"
	InstructionSignResponse          string = "sign_response"
	InstructionCheckMiningDifficulty string = "check_mining_difficulty"
	InstructionPublishToRedis        string = "publish_to_redis"
	InstructionSendClientResponse    string = "send_client_response"

	// Session validation flow
	InstructionGetSession             string = "get_session"
	InstructionValidateSessionHeader  string = "validate_session_header"
	InstructionVerifySignature        string = "verify_signature"
	InstructionCheckSupplierInSession string = "check_supplier_in_session"

	// Redis operations
	InstructionRedisXAdd   string = "redis_xadd"
	InstructionRedisXRead  string = "redis_xread"
	InstructionRedisXAck   string = "redis_xack"
	InstructionRedisGet    string = "redis_get"
	InstructionRedisSet    string = "redis_set"
	InstructionRedisPubSub string = "redis_pubsub"
)

// Instruction labels for HA Miner metrics.
const (
	// Relay consumption flow
	InstructionConsumeFromStream      string = "consume_from_stream"
	InstructionDeserializeRelay       string = "deserialize_relay"
	InstructionCheckDuplicateRelay    string = "check_duplicate_relay"
	InstructionGetOrCreateSessionTree string = "get_or_create_session_tree"
	InstructionUpdateSessionTree      string = "update_session_tree"
	InstructionMarkRelayProcessed     string = "mark_relay_processed"
	InstructionAcknowledgeMessage     string = "acknowledge_message"

	// Claim flow
	InstructionWaitForClaimWindow string = "wait_for_claim_window"
	InstructionFlushSessionTree   string = "flush_session_tree"
	InstructionCalculateClaimRoot string = "calculate_claim_root"
	InstructionBuildClaimMessage  string = "build_claim_message"
	InstructionSubmitClaim        string = "submit_claim"
	InstructionWaitForClaimTx     string = "wait_for_claim_tx"

	// Proof flow
	InstructionWaitForProofWindow   string = "wait_for_proof_window"
	InstructionGetProofPath         string = "get_proof_path"
	InstructionGenerateClosestProof string = "generate_closest_proof"
	InstructionBuildProofMessage    string = "build_proof_message"
	InstructionSubmitProof          string = "submit_proof"
	InstructionWaitForProofTx       string = "wait_for_proof_tx"

	// Recovery flow
	InstructionLoadSMSTSnapshot    string = "load_smst_snapshot"
	InstructionReplayWALEntries    string = "replay_wal_entries"
	InstructionRestoreSessionState string = "restore_session_state"

	// Leader election
	InstructionAcquireLock string = "acquire_lock"
	InstructionRenewLock   string = "renew_lock"
	InstructionReleaseLock string = "release_lock"
)

// InstructionTimestamp represents a single timing measurement for an instruction.
type InstructionTimestamp struct {
	instruction string
	timestamp   time.Time
}

// InstructionTimer tracks a collection of instruction timing measurements.
type InstructionTimer struct {
	Timestamps []*InstructionTimestamp
}

// NewInstructionTimer creates a new instruction timer.
func NewInstructionTimer() *InstructionTimer {
	return &InstructionTimer{
		Timestamps: make([]*InstructionTimestamp, 0, 16),
	}
}

// Record adds a new instruction timing entry with current timestamp.
func (it *InstructionTimer) Record(instruction string) {
	it.Timestamps = append(it.Timestamps, &InstructionTimestamp{
		instruction: instruction,
		timestamp:   time.Now(),
	})
}

// RecordWithTimestamp adds a new instruction timing entry with a specific timestamp.
func (it *InstructionTimer) RecordWithTimestamp(instruction string, ts time.Time) {
	it.Timestamps = append(it.Timestamps, &InstructionTimestamp{
		instruction: instruction,
		timestamp:   ts,
	})
}

// GetDurations returns a map of instruction names to their durations.
func (it *InstructionTimer) GetDurations() map[string]time.Duration {
	durations := make(map[string]time.Duration)
	var lastTime time.Time

	for i, inst := range it.Timestamps {
		if i == 0 {
			lastTime = inst.timestamp
			continue
		}

		durations[inst.instruction] = inst.timestamp.Sub(lastTime)
		lastTime = inst.timestamp
	}

	return durations
}

// TotalDuration returns the total duration from first to last recorded timestamp.
func (it *InstructionTimer) TotalDuration() time.Duration {
	if len(it.Timestamps) < 2 {
		return 0
	}
	return it.Timestamps[len(it.Timestamps)-1].timestamp.Sub(it.Timestamps[0].timestamp)
}

// Reset clears all recorded timestamps.
func (it *InstructionTimer) Reset() {
	it.Timestamps = it.Timestamps[:0]
}

// RecordDurations calculates and records durations between consecutive instructions.
// This function observes durations in the InstructionTimeSeconds Prometheus histogram.
func RecordDurations(component string, instructionTimestamps []*InstructionTimestamp) {
	var lastTime time.Time
	for i, inst := range instructionTimestamps {
		if i == 0 {
			lastTime = inst.timestamp
			continue
		}

		instructionTimeSeconds := inst.timestamp.Sub(lastTime).Seconds()
		InstructionTimeSeconds.WithLabelValues(component, inst.instruction).Observe(instructionTimeSeconds)
		lastTime = inst.timestamp
	}
}
