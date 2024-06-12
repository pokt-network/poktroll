package supplier

// SupplierClientMap is a helper struct needed to depinject multiple supplier clients.
// The inner structure maps a supplier address to a list of supplier clients for that address.
// Must be a type to successfully work with depinject.
type SupplierClientMap struct {
	SupplierClients map[string]SupplierClient
}

func NewSupplierClientMap() *SupplierClientMap {
	m := make(map[string]SupplierClient)
	return &SupplierClientMap{
		SupplierClients: m,
	}
}
