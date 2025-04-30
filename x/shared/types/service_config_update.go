package types

const (
	// NoDeactivationHeight represents that a service configuration has no deactivation
	// height and is considered active indefinitely.
	NoDeactivationHeight = iota // 0
)

// IsActive checks if the ServiceConfigUpdate is active at the given block height.
//
// A service configuration is considered active when the following conditions are met:
// 1. The block height is greater than or equal to the activation height
// 2. Either:
//   - The deactivation height is 0 (indicating no scheduled deactivation), OR
//   - The block height is less than the deactivation height
//
// This determines whether a supplier's service configuration should be considered
// active for session hydration  at the specified block height.
func (s *ServiceConfigUpdate) IsActive(queryHeight int64) bool {
	// If activation height is in the future, the config is not active yet
	if s.ActivationHeight > queryHeight {
		return false
	}

	// If no deactivation is scheduled (value 0), the config is active indefinitely
	if s.DeactivationHeight == NoDeactivationHeight {
		return true
	}

	// If deactivation height has been reached or passed, the config is no longer active
	if s.DeactivationHeight <= queryHeight {
		return false
	}

	// The config is active (activation height reached but deactivation height not yet reached)
	return true
}
