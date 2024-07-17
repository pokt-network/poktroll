package tokenomics

// This file is in place to declare the package for dynamically generated protobufs

// PendingClaimsResult encapsulates the result of settling pending claims. It is
// intended to be used to represent settled and expired results as unique instances.
type PendingClaimsResult struct {
	NumClaims           uint64
	NumComputeUnits     uint64
	NumRelays           uint64
	RelaysPerServiceMap map[string]uint64
}

// NewClaimSettlementResult creates a new PendingClaimsResult.
func NewClaimSettlementResult() PendingClaimsResult {
	return PendingClaimsResult{
		RelaysPerServiceMap: make(map[string]uint64),
	}
}
