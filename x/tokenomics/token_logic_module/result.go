package token_logic_module

import (
	"errors"
	"fmt"

	"cosmossdk.io/log"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// PendingSettlementResults is a slice of PendingSettlementResult. It implements
// methods for convenience when working with PendingSettlementResult objects.
type PendingSettlementResults []*PendingSettlementResult

// resultOption is a function which receives a PendingSettlementResult for modification.
type resultOption func(*PendingSettlementResult)

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
	TLMGlobalMintReimbursementRequest_DaoReimbursementEscrow = "TLMGlobalMintReimbursementRequest_DaoReimbursementEscrow"

	// Penalization - MintBurnOperation (burn)
	TLMRelayBurnEqualsMint_SupplierSlash = "TLMRelayBurnEqualsMint_SupplierSlash"

	// Module accounting - ModToModTransfer
	TLMGlobalMint_SupplierShareholderRewardModuleTransfer               = "TLMGlobalMint_SupplierShareholderRewardModuleTransfer"
	TLMGlobalMintReimbursementRequest_ReimbursementEscrowModuleTransfer = "TLMGlobalMintReimbursementRequest_ReimbursementEscrowModuleTransfer"
)

// MintBurnOperation holds the parameters of a mint or burn operation.
type MintBurnOperation struct {
	OriginTLM         TokenLogicModuleId
	OriginReason      TokenLogicModuleReason
	DestinationModule string
	Coin              cosmostypes.Coin
}

// Validate returns an error if the MintBurnOperation has either an unspecified TLM or TLMReason.
func (m *MintBurnOperation) Validate() error {
	// TODO_IN_THIS_COMMIT: factor unspecified validation out (can build interface from proto getters) when refactoring to protobufs.
	if m.OriginTLM == UnspecifiedTLM {
		return tokenomicstypes.ErrTokenomicsModuleBurn.Wrapf("origin TLM is unspecified: %+v", m)
	}

	if m.OriginReason == UnspecifiedTLMReason {
		return tokenomicstypes.ErrTokenomicsModuleBurn.Wrapf("origin reason is unspecified: %+v", m)
	}
	return nil
}

// ModToAcctTransfer holds the parameters of a module to account transfer operation.
type ModToAcctTransfer struct {
	OriginTLM        TokenLogicModuleId
	OriginReason     TokenLogicModuleReason
	SenderModule     string
	RecipientAddress cosmostypes.AccAddress
	Coin             cosmostypes.Coin
}

// Validate returns an error if the ModToAcctTransfer has either an unspecified TLM or TLMReason.
func (m *ModToAcctTransfer) Validate() error {
	// TODO_IN_THIS_COMMIT: factor unspecified validation out (can build interface from proto getters) when refactoring to protobufs.
	if m.OriginTLM == UnspecifiedTLM {
		return tokenomicstypes.ErrTokenomicsModuleBurn.Wrapf("origin TLM is unspecified: %+v", m)
	}

	if m.OriginReason == UnspecifiedTLMReason {
		return tokenomicstypes.ErrTokenomicsModuleBurn.Wrapf("origin reason is unspecified: %+v", m)
	}
	return nil
}

// ModToModTransfer holds the parameters of a module to module transfer operation.
type ModToModTransfer struct {
	OriginTLM       TokenLogicModuleId
	OriginReason    TokenLogicModuleReason
	SenderModule    string
	RecipientModule string
	Coin            cosmostypes.Coin
}

// Validate returns an error if the ModToModTransfer has either an unspecified TLM or TLMReason.
func (m *ModToModTransfer) Validate() error {
	// TODO_IN_THIS_COMMIT: factor unspecified validation out (can build interface from proto getters) when refactoring to protobufs.
	if m.OriginTLM == UnspecifiedTLM {
		return tokenomicstypes.ErrTokenomicsModuleBurn.Wrapf("origin TLM is unspecified: %+v", m)
	}

	if m.OriginReason == UnspecifiedTLMReason {
		return tokenomicstypes.ErrTokenomicsModuleBurn.Wrapf("origin reason is unspecified: %+v", m)
	}
	return nil
}

// PendingSettlementResult holds a claim and mints, burns, and transfers that
// result from its settlement.
type PendingSettlementResult struct {
	Claim              prooftypes.Claim
	Mints              []MintBurnOperation
	Burns              []MintBurnOperation
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

// GetSupplierOperatorAddr returns the supplier address of the result's claim.
func (r *PendingSettlementResult) GetSupplierOperatorAddr() string {
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
func (r *PendingSettlementResult) AppendMint(mint MintBurnOperation) {
	r.Mints = append(r.Mints, mint)
}

// AppendBurn appends a burn operation to the result.
func (r *PendingSettlementResult) AppendBurn(burn MintBurnOperation) {
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
			errs = errors.Join(err, err)
			continue
		}
		numComputeUnits += claimNumComputeUnits
	}

	return numComputeUnits, errs
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

// GetSupplierOperatorAddrs returns a slice of supplier addresses from the combined results' claims.
func (rs PendingSettlementResults) GetSupplierOperatorAddrs() (supplierOperatorAddrs []string) {
	for _, result := range rs {
		supplierOperatorAddrs = append(supplierOperatorAddrs, result.GetSupplierOperatorAddr())
	}
	return supplierOperatorAddrs
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
func WithMints(mints []MintBurnOperation) resultOption {
	return func(r *PendingSettlementResult) {
		r.Mints = mints
	}
}

// WithBurns returns a resultOption which sets the Burns field of the PendingSettlementResult.
func WithBurns(burns []MintBurnOperation) resultOption {
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
