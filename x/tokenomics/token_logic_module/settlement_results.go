package token_logic_module

// DEV_NOTE: This is defined in the token_logic_module package to avoid circular dependencies
// while enabling tlm_suite_test.

import (
	"errors"
	"sort"

	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// ClaimSettlementResults is a slice of ClaimSettlementResult. It implements
// methods for convenience when working with ClaimSettlementResult objects.
type ClaimSettlementResults []*tokenomicstypes.ClaimSettlementResult

// resultOption is a function which receives a ClaimSettlementResult for modification.
type resultOption func(*tokenomicstypes.ClaimSettlementResult)

// NewClaimSettlementResult returns a new ClaimSettlementResult with the given claim and options applied.
func NewClaimSettlementResult(
	claim prooftypes.Claim,
	opts ...resultOption,
) *tokenomicstypes.ClaimSettlementResult {
	result := &tokenomicstypes.ClaimSettlementResult{Claim: claim}
	for _, opt := range opts {
		opt(result)
	}
	return result
}

// GetNumComputeUnits returns the total number of claimed compute units in the results.
func (rs ClaimSettlementResults) GetNumComputeUnits() (numComputeUnits uint64, errs error) {
	for _, result := range rs {
		claimNumComputeUnits, err := result.GetNumComputeUnits()
		if err != nil {
			errs = errors.Join(err, err)
			continue
		}
		numComputeUnits += claimNumComputeUnits
	}

	return numComputeUnits, errs
}

// GetNumRelays returns the total number of relays in the combined results.
func (rs ClaimSettlementResults) GetNumRelays() (numRelays uint64, errs error) {
	for _, result := range rs {
		claimNumRelays, err := result.Claim.GetNumRelays()
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		numRelays += claimNumRelays
	}

	return numRelays, nil
}

// GetNumClaims returns the number of claims in the combined results.
func (rs ClaimSettlementResults) GetNumClaims() uint64 {
	// Each result holds a single claim.
	return uint64(len(rs))
}

// GetApplicationAddrs returns a slice of application addresses from the combined results' claims.
func (rs ClaimSettlementResults) GetApplicationAddrs() (appAddrs []string) {
	for _, result := range rs {
		appAddrs = append(appAddrs, result.GetApplicationAddr())
	}
	return appAddrs
}

// GetSupplierOperatorAddrs returns a slice of supplier addresses from the combined results' claims.
func (rs ClaimSettlementResults) GetSupplierOperatorAddrs() (supplierOperatorAddrs []string) {
	for _, result := range rs {
		supplierOperatorAddrs = append(supplierOperatorAddrs, result.GetSupplierOperatorAddr())
	}
	return supplierOperatorAddrs
}

// GetServiceIds returns a slice of service IDs from the combined results' claims.
// It is intended to be used for deterministic iterating over the map returned
// from GetRelaysPerServiceMap via the serviceId key.
func (rs ClaimSettlementResults) GetServiceIds() (serviceIds []string) {
	for _, result := range rs {
		serviceIds = append(serviceIds, result.GetServiceId())
	}

	// Sort service IDs to mitigate non-determinism.
	sort.Strings(serviceIds)

	return serviceIds
}

// GetRelaysPerServiceMap returns a map of {service_id -> total_num_relays_claimed_for_service} across all results.
// IMPORTANT: **DO NOT** directly iterate over returned map in onchain code.
// Iterating over the returned map can cause non-determinism.
// Instead, iterate over a sorted slice of the service ID keys.
// TODO_IMPROVE: Return a sorted slice of the service ID keys alongside the map for iteration.
func (rs ClaimSettlementResults) GetRelaysPerServiceMap() (map[string]uint64, error) {
	var errs error
	relaysPerServiceMap := make(map[string]uint64)

	for _, result := range rs {
		serviceId := result.Claim.GetSessionHeader().GetServiceId()
		numRelays, err := result.GetNumRelays()
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		relaysPerServiceMap[serviceId] += numRelays
	}

	return relaysPerServiceMap, errs
}

// Append appends a result to the results.
func (rs *ClaimSettlementResults) Append(result ...*tokenomicstypes.ClaimSettlementResult) {
	*rs = append(*rs, result...)
}

// WithMints returns a resultOption which sets the mints field of the ClaimSettlementResult.
func WithMints(mints []tokenomicstypes.MintBurnOp) resultOption {
	return func(r *tokenomicstypes.ClaimSettlementResult) {
		r.Mints = mints
	}
}

// WithBurns returns a resultOption which sets the burns field of the ClaimSettlementResult.
func WithBurns(burns []tokenomicstypes.MintBurnOp) resultOption {
	return func(r *tokenomicstypes.ClaimSettlementResult) {
		r.Burns = burns
	}
}

// WithModToModTransfers returns a resultOption which sets the modToModTransfers field of the ClaimSettlementResult.
func WithModToModTransfers(transfers []tokenomicstypes.ModToModTransfer) resultOption {
	return func(r *tokenomicstypes.ClaimSettlementResult) {
		r.ModToModTransfers = transfers
	}
}

// WithModToAcctTransfers returns a resultOption which sets the modToAcctTransfers field of the ClaimSettlementResult.
func WithModToAcctTransfers(transfers []tokenomicstypes.ModToAcctTransfer) resultOption {
	return func(r *tokenomicstypes.ClaimSettlementResult) {
		r.ModToAcctTransfers = transfers
	}
}
