package supplier

import "github.com/pokt-network/pocket/pkg/client"

// SupplierClientMap is a helper struct needed to depinject multiple supplier clients.
// The inner structure maps a supplier operator address to a list of supplier clients for that address.
// Must be a type to successfully work with depinject.
type SupplierClientMap struct {
	SupplierClients map[string]client.SupplierClient
}

func NewSupplierClientMap() *SupplierClientMap {
	m := make(map[string]client.SupplierClient)
	return &SupplierClientMap{
		SupplierClients: m,
	}
}
