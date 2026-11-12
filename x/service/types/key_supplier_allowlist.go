package types

const (
	// MaxSupplierAllowlistEntriesPerMsg caps the combined number of add and remove
	// entries in one MsgUpdateServiceSupplierAllowlist.
	MaxSupplierAllowlistEntriesPerMsg = 100

	// SupplierAllowlistKeyPrefix is the prefix for a service's supplier allowlist entries.
	// Key format: SupplierAllowlistKeyPrefix | serviceId | "/" | address
	SupplierAllowlistKeyPrefix = "ServiceSupplierAllowlist/"

	// SupplierAllowlistSweepKeyPrefix is the prefix for services whose allowlist
	// changed and whose suppliers are still to be swept.
	SupplierAllowlistSweepKeyPrefix = "ServiceSupplierAllowlistSweep/"

	// SupplierAllowlistCountKeyPrefix is the prefix for the number of allowlist
	// entries of a service. Key format: SupplierAllowlistCountKeyPrefix | serviceId
	// A service has a row iff its allowlist is non-empty, so "is this service
	// permissioned" is one point read.
	SupplierAllowlistCountKeyPrefix = "ServiceSupplierAllowlistCount/"
)

// SupplierAllowlistServicePrefix returns the store key prefix (relative to
// SupplierAllowlistKeyPrefix) of all allowlist entries of a service.
func SupplierAllowlistServicePrefix(serviceId string) []byte {
	return []byte(serviceId + "/")
}

// SupplierAllowlistKey returns the store key (relative to SupplierAllowlistKeyPrefix)
// of an address on a service's allowlist.
func SupplierAllowlistKey(serviceId, address string) []byte {
	return append(SupplierAllowlistServicePrefix(serviceId), address...)
}
