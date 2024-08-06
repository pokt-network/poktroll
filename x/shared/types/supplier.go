package types

// SupplierNotUnstaking is the value of `unstake_session_end_height` if the
// supplier is not actively in the unbonding period.
const SupplierNotUnstaking uint64 = 0

// IsUnbonding returns true if the supplier is actively unbonding.
// It determines if the supplier has submitted an unstake message, in which case
// the supplier has its UnstakeSessionEndHeight set.
func (s *Supplier) IsUnbonding() bool {
	return s.UnstakeSessionEndHeight != SupplierNotUnstaking
}

// IsActive returns whether the supplier is allowed to serve requests for the
// given serviceId and query height.
// A supplier is active for a given service starting from the session following
// the one during which the supplier staked for that service.
// A supplier that has submitted an unstake message is active until the end of
// the session containing the height at which unstake message was submitted.
func (s *Supplier) IsActive(queryHeight uint64, serviceId string) bool {
	// Service that has been staked for is not active yet.
	if s.ServicesActivationHeightsMap[serviceId] > queryHeight {
		return false
	}

	// If the supplier is not unbonding then its UnstakeSessionEndHeight is 0,
	// which returns true for all query heights.
	if s.IsUnbonding() {
		return queryHeight > s.UnstakeSessionEndHeight
	}

	return true
}

// EnsureOwner returns an error if the given address does not match supplier's owner address.
func (s *Supplier) EnsureOwner(ownerAddress string) error {
	if s.OwnerAddress != ownerAddress {
		return ErrSharedUnauthorizedSupplierUpdate.Wrapf(
			"msg.OwnerAddress %q != provided address %q",
			s.OwnerAddress,
			ownerAddress,
		)
	}

	return nil
}

// EnsureOperator returns an error if the given address does not match supplier's operator address.
func (s *Supplier) EnsureOperator(operatorAddress string) error {
	if s.OperatorAddress != operatorAddress {
		return ErrSharedUnauthorizedSupplierUpdate.Wrapf(
			"msg.OperatorAddress %q != provided address %q",
			s.OwnerAddress,
			operatorAddress,
		)
	}

	return nil
}
