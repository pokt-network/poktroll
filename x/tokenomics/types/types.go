package types

// This file is in place to declare the package for dynamically generated protobufs

type PendingClaimsResult struct {
	NumClaims           uint64
	NumComputeUnits     uint64
	NumRelays           uint64
	RelaysPerServiceMap map[string]uint64
}

func NewClaimSettlementResult() PendingClaimsResult {
	return PendingClaimsResult{
		RelaysPerServiceMap: make(map[string]uint64),
	}
}
