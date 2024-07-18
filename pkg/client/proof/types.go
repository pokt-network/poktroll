package proof

import "github.com/pokt-network/poktroll/pkg/client"

// SupplierClientMap is a helper struct needed to depinject multiple supplier clients.
// The inner structure maps a supplier address to a list of supplier clients for that address.
// Must be a type to successfully work with depinject.
type SupplierClientMap struct {
	SupplierClients map[string]client.ProofClient
}

func NewSupplierClientMap() *SupplierClientMap {
	m := make(map[string]client.ProofClient)
	return &SupplierClientMap{
		SupplierClients: m,
	}
}
