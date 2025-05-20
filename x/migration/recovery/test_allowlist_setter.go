//go:build test

package recovery

import "sort"

// SetRecoveryAllowlist sets the global recovery allowlist for testing purposes.
func SetRecoveryAllowlist(testRecoveryAllowlist []string) {
	// Sort the provided list to ensure correct binary search.
	sort.Strings(testRecoveryAllowlist)
	// Set the recovery allowlist to the provided list.
	lostAppStakesAllowlist = testRecoveryAllowlist
}
