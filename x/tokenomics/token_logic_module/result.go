package token_logic_module

import (
	"errors"
	"fmt"

	"cosmossdk.io/log"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// SettlementResults is a slice of SettlementResult. It implements
// methods for convenience when working with SettlementResult objects.
type SettlementResults []*SettlementResult

// resultOption is a function which receives a SettlementResult for modification.
type resultOption func(*SettlementResult)

// TODO_IN_THIS_COMMIT: promot to protobuf & godoc...
type TokenLogicModuleReason string

const (
	// UnspecifiedTLMReason is the default value for TokenLogicModuleReason, it is used as a field
	// type for objects which need to distinguish whether a TLMReason has been specified it or not.
	UnspecifiedTLMReason TokenLogicModuleReason = ""

	// Value transfer (credit/debit) - MintBurnOperation (mint/burn)
	TLMRelayBurnEqualsMint_SupplierStakeMint    = "TLMRelayBurnEqualsMint_SupplierStakeMint"
	TLMRelayBurnEqualsMint_ApplicationStakeBurn = "TLMGlobalMint_ApplicationBurn"

	// Inflation - MintBurnOperation (mint)
	TLMGlobalMint_Inflation = "TLMGlobalMint_Inflation"

	// Reward distribution - ModToAcctTransfer
	TLMRelayBurnEqualsMint_SupplierShareholderRewardDistribution = "TLMRelayBurnEqualsMint_SupplierShareholderRewardDistribution"
	TLMGlobalMint_DaoRewardDistribution                          = "TLMRelayBurnEqualsMint_DaoRewardDistribution"
	TLMGlobalMint_ProposerRewardDistribution                     = "TLMGlobalMint_ProposerRewardDistribution"
	TLMGlobalMint_SupplierShareholderRewardDistribution          = "TLMGlobalMint_SupplierShareholderRewardDistribution"
	TLMGlobalMint_SourceOwnerRewardDistribution                  = "TLMGlobalMint_SourceOwnerRewardDistribution"
	TLMGlobalMint_ApplicationRewardDistribution                  = "TLMGlobalMint_ApplicationRewardDistribution"

	// Self-servicing mitigation - MintBurnOperation (burn)
	TLMGlobalMintReimbursementRequest_AppReimbursementEscrow = "TLMGlobalMintReimbursementRequest_AppReimbursementEscrow"

	// Penalization - MintBurnOperation (burn)
	UnspecifiedTLM_SupplierSlash = "TLMRelayBurnEqualsMint_SupplierSlash"

	// Module accounting - ModToModTransfer
	TLMGlobalMint_SupplierShareholderRewardModuleTransfer               = "TLMGlobalMint_SupplierShareholderRewardModuleTransfer"
	TLMGlobalMintReimbursementRequest_ReimbursementEscrowModuleTransfer = "TLMGlobalMintReimbursementRequest_ReimbursementEscrowModuleTransfer"
)

// MintBurnOperation holds the parameters of a mint or burn operation.
type MintBurnOperation struct {
	TLMReason         TokenLogicModuleReason
	DestinationModule string
	Coin              cosmostypes.Coin
}

// Validate returns an error if the MintBurnOperation has either an unspecified TLM or TLMReason.
func (m *MintBurnOperation) Validate() error {
	// TODO_IN_THIS_COMMIT: factor common unspecified validation out (can build interface from proto getters) when refactoring to protobufs.
	if m.TLMReason == UnspecifiedTLMReason {
		return tokenomicstypes.ErrTokenomicsModuleBurn.Wrapf("origin reason is unspecified: %+v", m)
	}
	return nil
}

// ModToAcctTransfer holds the parameters of a module to account transfer operation.
type ModToAcctTransfer struct {
	TLMReason        TokenLogicModuleReason
	SenderModule     string
	RecipientAddress cosmostypes.AccAddress
	Coin             cosmostypes.Coin
}

// Validate returns an error if the ModToAcctTransfer has either an unspecified TLM or TLMReason.
func (m *ModToAcctTransfer) Validate() error {
	// TODO_IN_THIS_COMMIT: factor common unspecified validation out (can build interface from proto getters) when refactoring to protobufs.
	if m.TLMReason == UnspecifiedTLMReason {
		return tokenomicstypes.ErrTokenomicsModuleBurn.Wrapf("origin reason is unspecified: %+v", m)
	}
	return nil
}

// ModToModTransfer holds the parameters of a module to module transfer operation.
type ModToModTransfer struct {
	TLMReason       TokenLogicModuleReason
	SenderModule    string
	RecipientModule string
	Coin            cosmostypes.Coin
}

// Validate returns an error if the ModToModTransfer has either an unspecified TLM or TLMReason.
func (m *ModToModTransfer) Validate() error {
	// TODO_IN_THIS_COMMIT: factor common unspecified validation out (can build interface from proto getters) when refactoring to protobufs.
	if m.TLMReason == UnspecifiedTLMReason {
		return tokenomicstypes.ErrTokenomicsModuleBurn.Wrapf("origin reason is unspecified: %+v", m)
	}
	return nil
}

// SettlementResult holds a claim and mints, burns, and transfers that result from its settlement.
type SettlementResult struct {
	claim              prooftypes.Claim
	mints              []MintBurnOperation
	burns              []MintBurnOperation
	modToModTransfers  []ModToModTransfer
	modToAcctTransfers []ModToAcctTransfer
	//supplierOperatorAddrToSlash string
}

// NewSettlementResult returns a new SettlementResult with the given claim and options applied.
func NewSettlementResult(
	claim prooftypes.Claim,
	opts ...resultOption,
) *SettlementResult {
	result := &SettlementResult{claim: claim}
	for _, opt := range opts {
		opt(result)
	}
	return result
}

// GetClaim returns the claim associated with the result.
func (r *SettlementResult) GetClaim() *prooftypes.Claim {
	// Copy claim to prevent callers from mutating the result's claim.
	claimCopy := r.claim
	return &claimCopy
}

// GetMints returns the mints associated with the result.
func (r *SettlementResult) GetMints() []MintBurnOperation {
	return r.mints
}

// GetBurns returns the burns associated with the result.
func (r *SettlementResult) GetBurns() []MintBurnOperation {
	return r.burns
}

// GetModToModTransfers returns the modToModTransfers associated with the result.
func (r *SettlementResult) GetModToModTransfers() []ModToModTransfer {
	return r.modToModTransfers
}

// GetModToAcctTransfers returns the modToAcctTransfers associated with the result.
func (r *SettlementResult) GetModToAcctTransfers() []ModToAcctTransfer {
	return r.modToAcctTransfers
}

// GetNumComputeUnits returns the number of claimed compute units in the result's claim.
func (r *SettlementResult) GetNumComputeUnits() (uint64, error) {
	return r.GetClaim().GetNumClaimedComputeUnits()
}

// GetNumRelays returns the number of relays in the result's claim.
func (r *SettlementResult) GetNumRelays() (uint64, error) {
	return r.GetClaim().GetNumRelays()
}

// GetApplicationAddr returns the application address of the result's claim.
func (r *SettlementResult) GetApplicationAddr() string {
	return r.GetClaim().GetSessionHeader().GetApplicationAddress()
}

// GetSupplierOperatorAddr returns the supplier address of the result's claim.
func (r *SettlementResult) GetSupplierOperatorAddr() string {
	return r.GetClaim().GetSupplierOperatorAddress()
}

// GetSessionEndHeight returns the session end height of the result's claim.
func (r *SettlementResult) GetSessionEndHeight() int64 {
	return r.GetClaim().GetSessionHeader().GetSessionEndBlockHeight()
}

// GetSessionId returns the session ID of the result's claim.
func (r *SettlementResult) GetSessionId() string {
	return r.GetClaim().GetSessionHeader().GetSessionId()
}

// GetServiceId returns the service ID of the result's claim.
func (r *SettlementResult) GetServiceId() string {
	return r.GetClaim().GetSessionHeader().GetServiceId()
}

// AppendMint appends a mint operation to the result.
func (r *SettlementResult) AppendMint(mint MintBurnOperation) {
	r.mints = append(r.mints, mint)
}

// AppendBurn appends a burn operation to the result.
func (r *SettlementResult) AppendBurn(burn MintBurnOperation) {
	r.burns = append(r.burns, burn)
}

// AppendModToModTransfer appends a module to module transfer operation to the result.
func (r *SettlementResult) AppendModToModTransfer(transfer ModToModTransfer) {
	r.modToModTransfers = append(r.modToModTransfers, transfer)
}

// AppendModToAcctTransfer appends a module to account transfer operation to the result.
func (r *SettlementResult) AppendModToAcctTransfer(transfer ModToAcctTransfer) {
	r.modToAcctTransfers = append(r.modToAcctTransfers, transfer)
}

//// TODO_IN_THIS_COMMIT: update godoc...
//// WillSlashSupplier appends a supplier slash operation to the result.
//func (r *SettlementResult) WillSlashSupplier() {
//	r.supplierOperatorAddrToSlash = r.GetClaim().GetSupplierOperatorAddress()
//}

// GetNumComputeUnits returns the total number of claimed compute units in the results.
func (rs SettlementResults) GetNumComputeUnits() (numComputeUnits uint64, errs error) {
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
func (rs SettlementResults) GetNumRelays() (numRelays uint64, errs error) {
	for _, result := range rs {
		claimNumRelays, err := result.GetClaim().GetNumRelays()
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		numRelays += claimNumRelays
	}

	return numRelays, nil
}

// GetNumClaims returns the number of claims in the combined results.
func (rs SettlementResults) GetNumClaims() uint64 {
	// Each result holds a single claim.
	return uint64(len(rs))
}

// GetApplicationAddrs returns a slice of application addresses from the combined results' claims.
func (rs SettlementResults) GetApplicationAddrs() (appAddrs []string) {
	for _, result := range rs {
		appAddrs = append(appAddrs, result.GetApplicationAddr())
	}
	return appAddrs
}

// GetSupplierOperatorAddrs returns a slice of supplier addresses from the combined results' claims.
func (rs SettlementResults) GetSupplierOperatorAddrs() (supplierOperatorAddrs []string) {
	for _, result := range rs {
		supplierOperatorAddrs = append(supplierOperatorAddrs, result.GetSupplierOperatorAddr())
	}
	return supplierOperatorAddrs
}

// GetServiceIds returns a slice of service IDs from the combined results' claims.
// It is intended to be used for deterministic iterating over the map returned
// from GetRelaysPerServiceMap via the serviceId key.
func (rs SettlementResults) GetServiceIds() (serviceIds []string) {
	for _, result := range rs {
		serviceIds = append(serviceIds, result.GetServiceId())
	}
	return serviceIds
}

// GetRelaysPerServiceMap returns a map of service IDs to the total number of relays
// claimed for that service in the combined results.
// IMPORTANT: **DO NOT** iterate over returned map in on-chain code.
func (rs SettlementResults) GetRelaysPerServiceMap() (_ map[string]uint64, errs error) {
	relaysPerServiceMap := make(map[string]uint64)

	for _, result := range rs {
		serviceId := result.GetClaim().GetSessionHeader().GetServiceId()
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
func (rs *SettlementResults) Append(result ...*SettlementResult) {
	*rs = append(*rs, result...)
}

// WithMints returns a resultOption which sets the mints field of the SettlementResult.
func WithMints(mints []MintBurnOperation) resultOption {
	return func(r *SettlementResult) {
		r.mints = mints
	}
}

// WithBurns returns a resultOption which sets the burns field of the SettlementResult.
func WithBurns(burns []MintBurnOperation) resultOption {
	return func(r *SettlementResult) {
		r.burns = burns
	}
}

// WithModToModTransfers returns a resultOption which sets the modToModTransfers field of the SettlementResult.
func WithModToModTransfers(transfers []ModToModTransfer) resultOption {
	return func(r *SettlementResult) {
		r.modToModTransfers = transfers
	}
}

// WithModToAcctTransfers returns a resultOption which sets the modToAcctTransfers field of the SettlementResult.
func WithModToAcctTransfers(transfers []ModToAcctTransfer) resultOption {
	return func(r *SettlementResult) {
		r.modToAcctTransfers = transfers
	}
}

// logRewardOperation logs (at the info level) whether a particular reward operation
// was queued or not by appending a corresponding prefix to the given message.
func logRewardOperation(logger log.Logger, msg string, reward *cosmostypes.Coin) {
	var opMsgPrefix string
	if reward.IsZero() {
		opMsgPrefix = "operation skipped:"
	} else {
		opMsgPrefix = "operation queued:"
	}
	logger.Info(fmt.Sprintf("%s: %s", opMsgPrefix, msg))
}
