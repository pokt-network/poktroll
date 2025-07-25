package types

import cosmostypes "github.com/cosmos/cosmos-sdk/types"

// GetNumComputeUnits returns the number of claimed compute units in the result's claim.
func (r *ClaimSettlementResult) GetNumComputeUnits() (uint64, error) {
	return r.Claim.GetNumClaimedComputeUnits()
}

// GetNumRelays returns the number of relays in the result's claim.
func (r *ClaimSettlementResult) GetNumRelays() (uint64, error) {
	return r.Claim.GetNumRelays()
}

// GetApplicationAddr returns the application address of the result's claim.
func (r *ClaimSettlementResult) GetApplicationAddr() string {
	return r.Claim.GetSessionHeader().GetApplicationAddress()
}

// GetSupplierOperatorAddr returns the supplier address of the result's claim.
func (r *ClaimSettlementResult) GetSupplierOperatorAddr() string {
	return r.Claim.GetSupplierOperatorAddress()
}

// GetSessionEndHeight returns the session end height of the result's claim.
func (r *ClaimSettlementResult) GetSessionEndHeight() int64 {
	return r.Claim.GetSessionHeader().GetSessionEndBlockHeight()
}

// GetSessionId returns the session ID of the result's claim.
func (r *ClaimSettlementResult) GetSessionId() string {
	return r.Claim.GetSessionHeader().GetSessionId()
}

// GetServiceId returns the service ID of the result's claim.
func (r *ClaimSettlementResult) GetServiceId() string {
	return r.Claim.GetSessionHeader().GetServiceId()
}

// AppendMint appends a mint operation to the result.
func (r *ClaimSettlementResult) AppendMint(mint MintBurnOp) {
	r.Mints = append(r.Mints, mint)
}

// AppendBurn appends a burn operation to the result.
func (r *ClaimSettlementResult) AppendBurn(burn MintBurnOp) {
	r.Burns = append(r.Burns, burn)
}

// AppendModToModTransfer appends a module to module transfer operation to the result.
func (r *ClaimSettlementResult) AppendModToModTransfer(transfer ModToModTransfer) {
	r.ModToModTransfers = append(r.ModToModTransfers, transfer)
}

// AppendModToAcctTransfer appends a module to account transfer operation to the result.
func (r *ClaimSettlementResult) AppendModToAcctTransfer(transfer ModToAcctTransfer) {
	r.ModToAcctTransfers = append(r.ModToAcctTransfers, transfer)
}

// GetRewardDistribution returns a map of recipient addresses to their total reward amounts
// as strings. This aggregates all module-to-account transfers for each recipient address
// in the settlement result, providing a consolidated view of reward distribution.
//
// The returned map contains:
// - Key: recipient address (string)
// - Value: total reward amount as a coin string (e.g. "1000upokt")
//
// This is primarily used for event emission and observability purposes.
func (r *ClaimSettlementResult) GetRewardDistribution() map[string]string {
	// Start with a coin map to support arithmatic.
	rewardDistributionCoin := make(map[string]cosmostypes.Coin)

	for _, transfer := range r.ModToAcctTransfers {
		rewardCoin, hasAcctReward := rewardDistributionCoin[transfer.RecipientAddress]

		if !hasAcctReward {
			rewardDistributionCoin[transfer.RecipientAddress] = transfer.GetCoin()
			continue
		}

		rewardDistributionCoin[transfer.RecipientAddress] = rewardCoin.Add(transfer.GetCoin())
	}

	// Convert coin map to a string map.
	rewardDistribution := make(map[string]string)
	for address, coin := range rewardDistributionCoin {
		rewardDistribution[address] = coin.String()
	}

	return rewardDistribution
}

// Validate returns an error if the MintBurnOperation has either an unspecified TLM or TLMReason.
func (m *MintBurnOp) Validate() error {
	return validateOpReason(m.OpReason, m)
}

// Validate returns an error if the ModToAcctTransfer has either an unspecified TLM or TLMReason.
func (m *ModToAcctTransfer) Validate() error {
	return validateOpReason(m.OpReason, m)
}

// Validate returns an error if the ModToModTransfer has either an unspecified TLM or TLMReason.
func (m *ModToModTransfer) Validate() error {
	return validateOpReason(m.OpReason, m)
}

func validateOpReason(opReason SettlementOpReason, op any) error {
	if opReason == SettlementOpReason_UNSPECIFIED {
		return ErrTokenomicsSettlementBurn.Wrapf("Settlement operation reason is unspecified: %+v", op)
	}
	return nil
}
