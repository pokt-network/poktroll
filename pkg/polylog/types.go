package polylog

import (
	"fmt"
	"os"
	"strconv"
)

type LoggerOption func(logger Logger)

// 0.1% of the time, we want some debug logs to show up as info for visibility on the RelayMiner.
const defaultProbabilisticDebugInfoProb = 0.1

// ProbabilisticDebugInfoProb is the canonical value to use for probabilistic debug logging.
var ProbabilisticDebugInfoProb float64 = defaultProbabilisticDebugInfoProb

func init() {
	// Read "LOG_PROBABILISTIC_DEBUG_PROB" from environment variable
	// - If set, parse it as float64
	// - If not set, use defaultProbabilisticDebugInfoProb
	// - Panic if not a float between [0, 1)
	probStr := os.Getenv("LOG_PROBABILISTIC_DEBUG_PROB")
	if probStr != "" {
		probFloat, err := strconv.ParseFloat(probStr, 64)
		if err != nil || probFloat < 0 || probFloat >= 1 {
			panic(fmt.Sprintf(
				`LOG_PROBABILISTIC_DEBUG_PROB must be a float in [0, 1), got "%s"`, probStr,
			))
		}
		ProbabilisticDebugInfoProb = probFloat
	}
}
