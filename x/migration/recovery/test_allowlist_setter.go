//go:build test

package recovery

// SetRecoveryAllowlist sets the global recovery allowlist for testing purposes.
func SetRecoveryAllowlist(testRecoveryAllowlist []string) {
	// Set the recovery allowlist to the provided list.
	lostAppStakesAllowlist = testRecoveryAllowlist
}
