package types

// SupplierNotUnstaking is the value of `unstake_session_end_height` if the
// supplier is not actively in the unbonding period.
const SupplierNotUnstaking uint64 = 0

// IsUnbonding returns whether the supplier has submitted an unstake message,
// in which case the supplier has an UnstakeSessionEndHeight set.
func (s *Supplier) IsUnbonding() bool {
	return s.UnstakeSessionEndHeight != SupplierNotUnstaking
}

// IsActive returns whether the supplier is allowed to serve requests at the
// given query height.
// A supplier that has not submitted an unstake message is always active.
// A supplier that has submitted an unstake message is active until the end of
// the session containing the height at which unstake message was submitted.
func (s *Supplier) IsActive(queryHeight int64) bool {
	return !s.IsUnbonding() || uint64(queryHeight) <= s.UnstakeSessionEndHeight
}
