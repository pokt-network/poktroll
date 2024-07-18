package types

// SupplierNotUnstaking is the value of `unstake_session_end_height` if the
// supplier is not actively in the unbonding period.
const SupplierNotUnstaking uint64 = 0

func (s *Supplier) IsUnbonding() bool {
	return s.UnstakeSessionEndHeight != SupplierNotUnstaking
}

func (s *Supplier) IsActive(queryHeight int64) bool {
	return !s.IsUnbonding() || uint64(queryHeight) <= s.UnstakeSessionEndHeight
}
