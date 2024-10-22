package token_logic_module

import (
	"errors"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
)

type PendingSettlementResults []*PendingSettlementResult

// TODO_IN_THIS_COMMIT: godoc...
type MintBurn struct {
	TLM               TokenLogicModule
	DestinationModule string
	Coin              cosmostypes.Coin
}

// TODO_IN_THIS_COMMIT: godoc... EITHER module OR addr for source and destination...
type ModToAcctTransfer struct {
	TLMName          TokenLogicModule
	SenderModule     string
	RecipientAddress cosmostypes.AccAddress
	Coin             cosmostypes.Coin
}

// TODO_IN_THIS_COMMIT: godoc... EITHER module OR addr for source and destination...
type ModToModTransfer struct {
	TLMName         TokenLogicModule
	SenderModule    string
	RecipientModule string
	Coin            cosmostypes.Coin
}

// TODO_IN_THIS_COMMIT: godoc...
type PendingSettlementResult struct {
	Claim              prooftypes.Claim
	Mints              []MintBurn
	Burns              []MintBurn
	ModToModTransfers  []ModToModTransfer
	ModToAcctTransfers []ModToAcctTransfer
}

// TODO_IN_THIS_COMMIT: godoc & move...
type resultOption func(*PendingSettlementResult)

// TODO_IN_THIS_COMMIT: godoc & move...
func WithMints(mints []MintBurn) resultOption {
	return func(r *PendingSettlementResult) {
		r.Mints = mints
	}
}

// TODO_IN_THIS_COMMIT: godoc & move...
func WithBurns(burns []MintBurn) resultOption {
	return func(r *PendingSettlementResult) {
		r.Burns = burns
	}
}

// TODO_IN_THIS_COMMIT: godoc & move...
func WithModToModTransfers(transfers []ModToModTransfer) resultOption {
	return func(r *PendingSettlementResult) {
		r.ModToModTransfers = transfers
	}
}

// TODO_IN_THIS_COMMIT: godoc & move...
func WithModToAcctTransfers(transfers []ModToAcctTransfer) resultOption {
	return func(r *PendingSettlementResult) {
		r.ModToAcctTransfers = transfers
	}
}

// TODO_IN_THIS_COMMIT: godoc...
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

// TODO_IN_THIS_COMMIT: godoc...
func (r *PendingSettlementResult) GetNumComputeUnits() (uint64, error) {
	return r.Claim.GetNumClaimedComputeUnits()
}

// TODO_IN_THIS_COMMIT: godoc...
func (r *PendingSettlementResult) GetNumRelays() (uint64, error) {
	return r.Claim.GetNumRelays()
}

// TODO_IN_THIS_COMMIT: godoc...
func (r *PendingSettlementResult) GetApplicationAddr() string {
	return r.Claim.GetSessionHeader().GetApplicationAddress()
}

// TODO_IN_THIS_COMMIT: godoc...
func (r *PendingSettlementResult) GetSupplierAddr() string {
	return r.Claim.GetSupplierOperatorAddress()
}

// TODO_IN_THIS_COMMIT: godoc...
func (r *PendingSettlementResult) GetSessionEndHeight() int64 {
	return r.Claim.GetSessionHeader().GetSessionEndBlockHeight()
}

// TODO_IN_THIS_COMMIT: godoc... use for determinstic loops
func (r *PendingSettlementResult) GetServiceId() string {
	return r.Claim.GetSessionHeader().GetServiceId()
}

// TODO_IN_THIS_COMMIT: godoc...
func (r *PendingSettlementResult) AppendMint(mint MintBurn) {
	r.Mints = append(r.Mints, mint)
}

// TODO_IN_THIS_COMMIT: godoc...
func (r *PendingSettlementResult) AppendBurn(burn MintBurn) {
	r.Burns = append(r.Burns, burn)
}

// TODO_IN_THIS_COMMIT: godoc...
func (r *PendingSettlementResult) AppendModToModTransfer(transfer ModToModTransfer) {
	r.ModToModTransfers = append(r.ModToModTransfers, transfer)
}

// TODO_IN_THIS_COMMIT: godoc...
func (r *PendingSettlementResult) AppendModToAcctTransfer(transfer ModToAcctTransfer) {
	r.ModToAcctTransfers = append(r.ModToAcctTransfers, transfer)
}

// TODO_IN_THIS_COMMIT: godoc...
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

// TODO_IN_THIS_COMMIT: godoc...
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

// TODO_IN_THIS_COMMIT: godoc...
func (rs PendingSettlementResults) GetNumClaims() uint64 {
	// Each result holds a single claim.
	return uint64(len(rs))
}

// TODO_IN_THIS_COMMIT: godoc...
func (rs PendingSettlementResults) GetApplicationAddrs() (appAddrs []string) {
	for _, result := range rs {
		appAddrs = append(appAddrs, result.GetApplicationAddr())
	}
	return appAddrs
}

// TODO_IN_THIS_COMMIT: godoc...
func (rs PendingSettlementResults) GetSupplierAddrs() (supplierAddrs []string) {
	for _, result := range rs {
		supplierAddrs = append(supplierAddrs, result.GetSupplierAddr())
	}
	return supplierAddrs
}

// TODO_IN_THIS_COMMIT: godoc... use for determinstic loops
func (rs PendingSettlementResults) GetServiceIds() (serviceIds []string) {
	for _, result := range rs {
		serviceIds = append(serviceIds, result.GetServiceId())
	}
	return serviceIds
}

// TODO_IN_THIS_COMMIT: godoc... // SHOULD NEVER be used for loops
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

// TODO_IN_THIS_COMMIT: godoc... // SHOULD NEVER be used for loops
func (rs *PendingSettlementResults) Append(result ...*PendingSettlementResult) {
	*rs = append(*rs, result...)
}
