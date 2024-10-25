package token_logic_module

import (
	"errors"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
)

// PendingSettlementResults is a slice of PendingSettlementResult. It implements
// methods for convenience when working with PendingSettlementResult objects.
type PendingSettlementResults []*PendingSettlementResult

// resultOption is a function which receives a PendingSettlementResult for modification.
type resultOption func(*PendingSettlementResult)

// MintBurn holds the parameters of a mint or burn operation.
type MintBurn struct {
	TLM               TokenLogicModule
	DestinationModule string
	Coin              cosmostypes.Coin
}

// ModToAcctTransfer holds the parameters of a module to account transfer operation.
type ModToAcctTransfer struct {
	TLMName          TokenLogicModule
	SenderModule     string
	RecipientAddress cosmostypes.AccAddress
	Coin             cosmostypes.Coin
}

// ModToModTransfer holds the parameters of a module to module transfer operation.
type ModToModTransfer struct {
	TLMName         TokenLogicModule
	SenderModule    string
	RecipientModule string
	Coin            cosmostypes.Coin
}

// PendingSettlementResult holds a claim and mints, burns, and transfers that
// result from its settlement.
type PendingSettlementResult struct {
	Claim              prooftypes.Claim
	Mints              []MintBurn
	Burns              []MintBurn
	ModToModTransfers  []ModToModTransfer
	ModToAcctTransfers []ModToAcctTransfer
}

// NewPendingSettlementResult returns a new PendingSettlementResult with the given claim and options applied.
func NewPendingSettlementResult(
	claim prooftypes.Claim,
	opts ...resultOption,
) *PendingSettlementResult {
	result := &PendingSettlementResult{Claim: claim}
	for _, opt := range opts {
		opt(result)
	}
	return result
}

// GetNumComputeUnits returns the number of claimed compute units in the result's claim.
func (r *PendingSettlementResult) GetNumComputeUnits() (uint64, error) {
	return r.Claim.GetNumClaimedComputeUnits()
}

// GetNumRelays returns the number of relays in the result's claim.
func (r *PendingSettlementResult) GetNumRelays() (uint64, error) {
	return r.Claim.GetNumRelays()
}

// GetApplicationAddr returns the application address of the result's claim.
func (r *PendingSettlementResult) GetApplicationAddr() string {
	return r.Claim.GetSessionHeader().GetApplicationAddress()
}

// GetSupplierAddr returns the supplier address of the result's claim.
func (r *PendingSettlementResult) GetSupplierAddr() string {
	return r.Claim.GetSupplierOperatorAddress()
}

// GetSessionEndHeight returns the session end height of the result's claim.
func (r *PendingSettlementResult) GetSessionEndHeight() int64 {
	return r.Claim.GetSessionHeader().GetSessionEndBlockHeight()
}

// GetSessionId returns the session ID of the result's claim.
func (r *PendingSettlementResult) GetSessionId() string {
	return r.Claim.GetSessionHeader().GetSessionId()
}

// GetServiceId returns the service ID of the result's claim.
func (r *PendingSettlementResult) GetServiceId() string {
	return r.Claim.GetSessionHeader().GetServiceId()
}

// AppendMint appends a mint operation to the result.
func (r *PendingSettlementResult) AppendMint(mint MintBurn) {
	r.Mints = append(r.Mints, mint)
}

// AppendBurn appends a burn operation to the result.
func (r *PendingSettlementResult) AppendBurn(burn MintBurn) {
	r.Burns = append(r.Burns, burn)
}

// AppendModToModTransfer appends a module to module transfer operation to the result.
func (r *PendingSettlementResult) AppendModToModTransfer(transfer ModToModTransfer) {
	r.ModToModTransfers = append(r.ModToModTransfers, transfer)
}

// AppendModToAcctTransfer appends a module to account transfer operation to the result.
func (r *PendingSettlementResult) AppendModToAcctTransfer(transfer ModToAcctTransfer) {
	r.ModToAcctTransfers = append(r.ModToAcctTransfers, transfer)
}

// GetNumComputeUnits returns the total number of claimed compute units in the results.
func (rs PendingSettlementResults) GetNumComputeUnits() (numComputeUnits uint64, errs error) {
	for _, result := range rs {
		claimNumComputeUnits, err := result.GetNumComputeUnits()
		if err != nil {
			err = errors.Join(err, err)
			continue
		}
		numComputeUnits += claimNumComputeUnits
	}

	return numComputeUnits, nil
}

// GetNumRelays returns the total number of relays in the combined results.
func (rs PendingSettlementResults) GetNumRelays() (numRelays uint64, errs error) {
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
func (rs PendingSettlementResults) GetNumClaims() uint64 {
	// Each result holds a single claim.
	return uint64(len(rs))
}

// GetApplicationAddrs returns a slice of application addresses from the combined results' claims.
func (rs PendingSettlementResults) GetApplicationAddrs() (appAddrs []string) {
	for _, result := range rs {
		appAddrs = append(appAddrs, result.GetApplicationAddr())
	}
	return appAddrs
}

// GetSupplierAddrs returns a slice of supplier addresses from the combined results' claims.
func (rs PendingSettlementResults) GetSupplierAddrs() (supplierAddrs []string) {
	for _, result := range rs {
		supplierAddrs = append(supplierAddrs, result.GetSupplierAddr())
	}
	return supplierAddrs
}

// GetServiceIds returns a slice of service IDs from the combined results' claims.
// It is intended to be used for deterministic iterating over the map returned
// from GetRelaysPerServiceMap via the serviceId key.
func (rs PendingSettlementResults) GetServiceIds() (serviceIds []string) {
	for _, result := range rs {
		serviceIds = append(serviceIds, result.GetServiceId())
	}
	return serviceIds
}

// GetRelaysPerServiceMap returns a map of service IDs to the total number of relays
// claimed for that service in the combined results.
// IMPORTANT: **DO NOT** iterate over returned map in on-chain code.
func (rs PendingSettlementResults) GetRelaysPerServiceMap() (_ map[string]uint64, errs error) {
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
func (rs *PendingSettlementResults) Append(result ...*PendingSettlementResult) {
	*rs = append(*rs, result...)
}

// WithMints returns a resultOption which sets the Mints field of the PendingSettlementResult.
func WithMints(mints []MintBurn) resultOption {
	return func(r *PendingSettlementResult) {
		r.Mints = mints
	}
}

// WithBurns returns a resultOption which sets the Burns field of the PendingSettlementResult.
func WithBurns(burns []MintBurn) resultOption {
	return func(r *PendingSettlementResult) {
		r.Burns = burns
	}
}

// WithModToModTransfers returns a resultOption which sets the ModToModTransfers field of the PendingSettlementResult.
func WithModToModTransfers(transfers []ModToModTransfer) resultOption {
	return func(r *PendingSettlementResult) {
		r.ModToModTransfers = transfers
	}
}

// WithModToAcctTransfers returns a resultOption which sets the ModToAcctTransfers field of the PendingSettlementResult.
func WithModToAcctTransfers(transfers []ModToAcctTransfer) resultOption {
	return func(r *PendingSettlementResult) {
		r.ModToAcctTransfers = transfers
	}
}
