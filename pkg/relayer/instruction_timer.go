package relayer

import "time"

// Instruction labels for metrics.
const (
	// General proxy sync flow
	InstructionInitRequestLogger                      string = "init_request_logger"
	InstructionGetStartBlock                          string = "get_start_block"
	InstructionSetContextValueComponentKind           string = "set_context_value_component_kind"
	InstructionChainHeadProbabilisticDebugInfo        string = "chain_head_probabilistic_debug_info"
	InstructionHandlingRequestProbabilisticDebugInfo  string = "handling_request_probabilistic_debug_info"
	InstructionDebugRelayRequestExtraction            string = "debug_relay_request_extraction"
	InstructionNewRelayRequest                        string = "new_relay_request"
	InstructionRelayRequestBasicValidation            string = "relay_request_basic_validation"
	InstructionLoggerWithRequestDetails               string = "logger_with_request_details"
	InstructionGetAvailableSuppliers                  string = "get_available_suppliers"
	InstructionCheckSupplierAvailable                 string = "check_supplier_available"
	InstructionDetermineRequestTimeoutForServiceID    string = "determine_request_timeout_for_service_id"
	InstructionSetContextDeadline                     string = "set_context_deadline"
	InstructionSetResponseControllerWriteDeadline     string = "set_response_controller_write_deadline"
	InstructionEagerCheckRateLimiting                 string = "eager_check__rate_limiting"
	InstructionGetServiceConfig                       string = "get_service_config"
	InstructionLoggerWithServiceDetails               string = "logger_with_service_details"
	InstructionPreRequestVerification                 string = "pre_request_verification"
	InstructionPostRequestVerification                string = "post_request_verification"
	InstructionBuildServiceBackendRequest             string = "build_service_backend_request"
	InstructionSetRequestTimeoutWithRemainingTime     string = "set_request_timeout_with_remaining_time"
	InstructionHTTPClientDo                           string = "http_client_do"
	InstructionDeferCloseResponseBodyAndCaptureSvcDur string = "defer_close_response_body_and_capture_service_duration"
	InstructionSerializeHTTPResponse                  string = "serialize_http_response"
	InstructionCheckDeadlineBeforeResponse            string = "check_deadline_before_response"
	InstructionRelayResponseGenerated                 string = "relay_response_generated"
	InstructionLoggerWithResponsePreparation          string = "logger_with_response_preparation"
	InstructionResponseSent                           string = "response_sent"
)

// InstructionTimestamp represents a single timing measurement for an instruction.
// - Captures instruction identifier and timestamp during relay processing
type InstructionTimestamp struct {
	instruction string
	timestamp   time.Time
}

// InstructionTimer tracks a collection of instruction timing measurements.
// - Maintains slice of timestamps for relay processing instructions
type InstructionTimer struct {
	Timestamps []*InstructionTimestamp
}

// Record adds a new instruction timing entry with current timestamp.
func (it *InstructionTimer) Record(instruction string) {
	it.Timestamps = append(it.Timestamps, &InstructionTimestamp{
		instruction: instruction,
		timestamp:   time.Now(),
	})
}

// RecordDurations calculates and records durations between consecutive instructions.
// - First instruction establishes baseline (no metric recorded)
// - Subsequent instructions record delta from previous instruction
// - Observes durations in InstructionTimeSeconds Prometheus histogram
func RecordDurations(instructionTimestamps []*InstructionTimestamp) {
	var lastTime time.Time
	for i, inst := range instructionTimestamps {
		if i == 0 {
			lastTime = inst.timestamp
			continue
		}

		instructionTimeSeconds := inst.timestamp.Sub(lastTime).Seconds()
		InstructionTimeSeconds.With("instruction", inst.instruction).Observe(float64(instructionTimeSeconds))
		lastTime = inst.timestamp
	}
}
