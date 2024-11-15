package types

// GetNumComputeUnits returns the number of claimed compute units in the result's claim.
func (r *SettlementResult) GetNumComputeUnits() (uint64, error) {
	return r.Claim.GetNumClaimedComputeUnits()
}

// GetNumRelays returns the number of relays in the result's claim.
func (r *SettlementResult) GetNumRelays() (uint64, error) {
	return r.Claim.GetNumRelays()
}

// GetApplicationAddr returns the application address of the result's claim.
func (r *SettlementResult) GetApplicationAddr() string {
	return r.Claim.GetSessionHeader().GetApplicationAddress()
}

// GetSupplierOperatorAddr returns the supplier address of the result's claim.
func (r *SettlementResult) GetSupplierOperatorAddr() string {
	return r.Claim.GetSupplierOperatorAddress()
}

// GetSessionEndHeight returns the session end height of the result's claim.
func (r *SettlementResult) GetSessionEndHeight() int64 {
	return r.Claim.GetSessionHeader().GetSessionEndBlockHeight()
}

// GetSessionId returns the session ID of the result's claim.
func (r *SettlementResult) GetSessionId() string {
	return r.Claim.GetSessionHeader().GetSessionId()
}

// GetServiceId returns the service ID of the result's claim.
func (r *SettlementResult) GetServiceId() string {
	return r.Claim.GetSessionHeader().GetServiceId()
}

// AppendMint appends a mint operation to the result.
func (r *SettlementResult) AppendMint(mint MintBurnOp) {
	r.Mints = append(r.Mints, mint)
}

// AppendBurn appends a burn operation to the result.
func (r *SettlementResult) AppendBurn(burn MintBurnOp) {
	r.Burns = append(r.Burns, burn)
}

// AppendModToModTransfer appends a module to module transfer operation to the result.
func (r *SettlementResult) AppendModToModTransfer(transfer ModToModTransfer) {
	r.ModToModTransfers = append(r.ModToModTransfers, transfer)
}

// AppendModToAcctTransfer appends a module to account transfer operation to the result.
func (r *SettlementResult) AppendModToAcctTransfer(transfer ModToAcctTransfer) {
	r.ModToAcctTransfers = append(r.ModToAcctTransfers, transfer)
}

// Validate returns an error if the MintBurnOperation has either an unspecified TLM or TLMReason.
func (m *MintBurnOp) Validate() error {
	// TODO_IN_THIS_COMMIT: factor common unspecified validation out (can build interface from proto getters) when refactoring to protobufs.
	if m.OpReason == SettlementOpReason_UNSPECIFIED {
		return ErrTokenomicsModuleBurn.Wrapf("origin reason is unspecified: %+v", m)
	}
	return nil
}

// Validate returns an error if the ModToAcctTransfer has either an unspecified TLM or TLMReason.
func (m *ModToAcctTransfer) Validate() error {
	// TODO_IN_THIS_COMMIT: factor common unspecified validation out (can build interface from proto getters) when refactoring to protobufs.
	if m.OpReason == SettlementOpReason_UNSPECIFIED {
		return ErrTokenomicsModuleBurn.Wrapf("origin reason is unspecified: %+v", m)
	}
	return nil
}

// Validate returns an error if the ModToModTransfer has either an unspecified TLM or TLMReason.
func (m *ModToModTransfer) Validate() error {
	// TODO_IN_THIS_COMMIT: factor common unspecified validation out (can build interface from proto getters) when refactoring to protobufs.
	if m.OpReason == SettlementOpReason_UNSPECIFIED {
		return ErrTokenomicsModuleBurn.Wrapf("origin reason is unspecified: %+v", m)
	}
	return nil
}
