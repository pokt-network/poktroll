package keeper

// TODO_BLOCKER(#21): Make these configurable governance param
const (
	// TODO_BLOCKER: Remove direct usage of these constants in helper functions
	// when they will be replaced by governance params
	NumBlocksPerSession = 4
	// Duration of the grace period in number of sessions
	SessionGracePeriod          = 1
	NumSupplierPerSession       = 15
	SessionIDComponentDelimiter = "."
)
