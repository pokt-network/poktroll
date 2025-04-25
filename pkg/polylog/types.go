package polylog

import (
	"fmt"
	"os"
	"strconv"
)

type LoggerOption func(logger Logger)

// 0.1% of the time, we want some debug logs to show up as info for visibility on the RelayMiner.
const defaultProbabilisticDebugProb = 0.001

// ProbabilisticDebugProb is the canonical value to use for probabilistic debug logging.
var ProbabilisticDebugProb float64

func init() {
	// Read "LOG_PROBABILISTIC_DEBUG_PROB" from environment variable
	// - If set, parse it as float64
	// - If not set, use defaultProbabilisticDebugProb
	// - Panic if not a float between [0, 1)
	probStr := os.Getenv("LOG_PROBABILISTIC_DEBUG_PROB")
	if probStr == "" {
		ProbabilisticDebugProb = defaultProbabilisticDebugProb
		return
	}
	val, err := strconv.ParseFloat(probStr, 64)
	if err != nil || val < 0 || val >= 1 {
		panic(fmt.Sprintf(
			`LOG_PROBABILISTIC_DEBUG_PROB must be a float in [0, 1), got "%s"`, probStr,
		))
	}
	ProbabilisticDebugProb = val
}
